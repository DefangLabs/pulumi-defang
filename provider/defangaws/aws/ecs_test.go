package aws

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConfigProvider is a minimal compose.ConfigProvider stub: GetConfigValue
// resolves from a fixed map, GetSecretRef returns a deterministic fake ARN.
type fakeConfigProvider struct {
	values map[string]string
}

func (f *fakeConfigProvider) GetConfigValue(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) pulumi.StringOutput {
	return pulumi.String(f.values[key]).ToStringOutput()
}

func (f *fakeConfigProvider) GetSecretRef(_ *pulumi.Context, key string, _ ...pulumi.InvokeOption) (string, error) {
	return "arn:aws:ssm:us-west-2:123456789012:parameter/Defang/myproject/stack/" + key, nil
}

// composedSecretMocks is a minimal pulumi.MockResourceMonitor that answers the
// two invokes newComposedEnvSecret needs to build an ARN (getCallerRegion,
// getCallerAccountId) and records every resource it creates.
type composedSecretMocks struct {
	created []pulumi.MockResourceArgs
}

func (m *composedSecretMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.created = append(m.created, args)
	return args.Name + "_id", args.Inputs, nil
}

func (m *composedSecretMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	switch args.Token {
	case "aws:index/getCallerIdentity:getCallerIdentity":
		return resource.PropertyMap{"accountId": resource.NewStringProperty("123456789012")}, nil
	case "aws:index/getRegion:getRegion":
		return resource.PropertyMap{"region": resource.NewStringProperty("us-west-2")}, nil
	}
	return resource.PropertyMap{}, nil
}

// TestNewComposedEnvSecret verifies that a composite env value (one that
// embeds a config-provided secret via interpolation, e.g. a DSN built from
// ${POSTGRES_PASSWORD}, rather than a bare ${VAR} reference) is stored as a
// new SSM SecureString parameter, and that the caller only ever gets back an
// ARN pointing at it — never the plaintext value itself. Keeping that value
// out of the ECS task definition's plaintext Environment (readable via
// ecs:DescribeTaskDefinition) is the whole point of the fix.
// See DefangLabs/pulumi-defang#638 (follow-up to the Azure fix for #637 /
// DefangLabs/station#198).
func TestNewComposedEnvSecret(t *testing.T) {
	const secretValue = "postgresql://postgres:hunter2@db:5432/mydb" //nolint:gosec // test fixture, not a real credential

	mocks := &composedSecretMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		value := pulumi.String(secretValue).ToStringOutput()
		arn, err := newComposedEnvSecret(ctx, "myservice", "DATABASE_URL", value, nil)
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(arn, "arn:aws:ssm:us-west-2:123456789012:parameter/Defang/"),
			"got %q", arn)
		assert.Contains(t, arn, "/_env/myservice/DATABASE_URL")
		assert.NotContains(t, arn, "hunter2", "the ARN must never embed the secret value")
		return nil
	}, pulumi.WithMocks("myproject", "stack", mocks))
	require.NoError(t, err)

	require.Len(t, mocks.created, 1, "expected exactly one SSM parameter to be created")
	param := mocks.created[0]
	assert.Equal(t, "aws:ssm/parameter:Parameter", param.TypeToken)
	assert.Equal(t, "SecureString", param.Inputs["type"].StringValue())
	// The pulumi-aws schema marks ssm.Parameter's Value as sensitive, so it
	// arrives wrapped in a Secret property value.
	valueInput := param.Inputs["value"]
	require.True(t, valueInput.IsSecret(), "parameter value should be marked secret")
	assert.Equal(t, secretValue, valueInput.SecretValue().Element.StringValue())
	assert.Equal(t, "/Defang/myproject/stack/_env/myservice/DATABASE_URL", param.Inputs["name"].StringValue())
}

// TestCreateECSServiceKeepsComposedSecretOutOfPlaintext is the end-to-end
// complement to TestNewComposedEnvSecret: it drives CreateECSService (the
// compose-shaped path used by `defang compose up` via project.go) with a
// DATABASE_URL built from ${PASSWORD} — a composite value, not a bare ${VAR}
// reference — and checks the resulting ECS TaskDefinition JSON directly.
// Before this fix, such a value landed as a plaintext `environment` entry
// with the password in clear, readable via ecs:DescribeTaskDefinition.
// See DefangLabs/pulumi-defang#638.
func TestCreateECSServiceKeepsComposedSecretOutOfPlaintext(t *testing.T) {
	const password = "hunter2"

	mocks := &composedSecretMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		svc := compose.ServiceConfig{
			Environment: compose.Environment{
				"DATABASE_URL": pulumi.String("postgresql://postgres:${PASSWORD}@db:5432/mydb"),
			},
		}
		configProvider := &fakeConfigProvider{values: map[string]string{"PASSWORD": password}}
		args := &ECSServiceArgs{ImageURI: pulumi.String("myapp:latest")}
		_, err := CreateECSService(ctx, configProvider, "myservice", svc, args, nil, nil)
		return err
	}, pulumi.WithMocks("myproject", "stack", mocks))
	require.NoError(t, err)

	var taskDefInputs resource.PropertyMap
	var ssmParams int
	for _, r := range mocks.created {
		switch r.TypeToken {
		case "aws:ecs/taskDefinition:TaskDefinition":
			taskDefInputs = r.Inputs
		case "aws:ssm/parameter:Parameter":
			ssmParams++
		}
	}
	require.NotNil(t, taskDefInputs, "expected a TaskDefinition to be created")
	assert.Equal(t, 1, ssmParams, "expected exactly one new SSM parameter for the composed secret")

	var defs []ContainerDefinition
	require.NoError(t, json.Unmarshal([]byte(taskDefInputs["containerDefinitions"].StringValue()), &defs))
	require.Len(t, defs, 1)
	def := defs[0]

	for _, e := range def.Environment {
		assert.NotEqual(t, "DATABASE_URL", e.Name,
			"composite secret-bearing value must not land in plaintext environment")
		assert.NotContains(t, e.Value, password, "the password must never appear in a plaintext env value")
	}

	var secretRef string
	for _, s := range def.Secrets {
		if s.Name == "DATABASE_URL" {
			secretRef = s.ValueFrom
		}
	}
	require.NotEmpty(t, secretRef, "DATABASE_URL must appear in container 'secrets' as a valueFrom ref")
	assert.True(t, strings.HasPrefix(secretRef, "arn:aws:ssm:"))
	assert.NotContains(t, secretRef, password, "valueFrom must be a reference, never the plaintext secret")
}
