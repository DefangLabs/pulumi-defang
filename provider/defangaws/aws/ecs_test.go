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

// composedSecretMocks is a minimal pulumi.MockResourceMonitor that records
// every resource it creates and synthesizes a fake Arn for any SSM parameter
// (an output-only field newComposedEnvSecret's caller relies on — nothing in
// resource.Inputs sets it) so assertions against the real code path can check
// its shape.
type composedSecretMocks struct {
	created []pulumi.MockResourceArgs
}

func (m *composedSecretMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.created = append(m.created, args)
	outputs := args.Inputs
	if args.TypeToken == "aws:ssm/parameter:Parameter" {
		outputs["arn"] = resource.NewStringProperty("arn:aws:ssm:us-west-2:123456789012:parameter" + args.Name)
	}
	return args.Name + "_id", outputs, nil
}

func (m *composedSecretMocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// TestNewComposedEnvSecret verifies that a composite env value (one that
// embeds a config-provided secret via interpolation, e.g. a DSN built from
// ${POSTGRES_PASSWORD}, rather than a bare ${VAR} reference) is stored as a
// new SSM SecureString parameter, and that the caller only ever gets back an
// Arn Output pointing at it — never the plaintext value itself. Keeping that
// value out of the ECS task definition's plaintext Environment (readable via
// ecs:DescribeTaskDefinition) is the whole point of the fix. No physical
// parameter name is set, so AWS assigns one — this only checks the Arn's
// shape and that it's independent of the Pulumi logical resource name.
// See DefangLabs/pulumi-defang#638 (follow-up to the Azure fix for #637 /
// DefangLabs/station#198).
func TestNewComposedEnvSecret(t *testing.T) {
	const secretValue = "postgresql://postgres:hunter2@db:5432/mydb" //nolint:gosec // test fixture, not a real credential

	mocks := &composedSecretMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		value := pulumi.String(secretValue).ToStringOutput()
		arnOutput, err := newComposedEnvSecret(ctx, "myservice", "DATABASE_URL", value, nil)
		require.NoError(t, err)

		arnOutput.ApplyT(func(arn string) string {
			assert.True(t, strings.HasPrefix(arn, "arn:aws:ssm:"), "got %q", arn)
			assert.NotContains(t, arn, "hunter2", "the ARN must never embed the secret value")
			return arn
		})
		return nil
	}, pulumi.WithMocks("myproject", "stack", mocks))
	require.NoError(t, err)

	require.Len(t, mocks.created, 1, "expected exactly one SSM parameter to be created")
	param := mocks.created[0]
	assert.Equal(t, "aws:ssm/parameter:Parameter", param.TypeToken)
	assert.Equal(t, "SecureString", param.Inputs["type"].StringValue())
	_, hasName := param.Inputs["name"]
	assert.False(t, hasName, "no physical name should be set; let AWS assign one")
	// The pulumi-aws schema marks ssm.Parameter's Value as sensitive, so it
	// arrives wrapped in a Secret property value.
	valueInput := param.Inputs["value"]
	require.True(t, valueInput.IsSecret(), "parameter value should be marked secret")
	assert.Equal(t, secretValue, valueInput.SecretValue().Element.StringValue())
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

	var serviceInputs resource.PropertyMap
	for _, r := range mocks.created {
		if r.TypeToken == "aws:ecs/service:Service" {
			serviceInputs = r.Inputs
		}
	}
	require.NotNil(t, serviceInputs, "expected an ECS Service to be created")
	triggers := serviceInputs["triggers"]
	require.True(t, triggers.IsObject(), "expected a composedSecretsSha256 trigger since a composed secret exists")
	hash, ok := triggers.ObjectValue()["composedSecretsSha256"]
	require.True(t, ok, "expected a composedSecretsSha256 trigger since a composed secret exists")
	assert.NotContains(t, hash.StringValue(), password, "the trigger must be a hash, never the plaintext value")
}

// TestMergeComposedSecretsTrigger verifies the Triggers-hash helper: it
// merges a hash of the composed secret values into any caller-supplied
// triggers without ever exposing the plaintext values themselves, and the
// hash changes when a value changes — which is what makes a value rotation
// force an ECS redeployment despite the task definition's own JSON staying
// the same (its Secrets entry holds a stable parameter Arn, not the value;
// see newComposedEnvSecret). See DefangLabs/pulumi-defang#638.
func TestMergeComposedSecretsTrigger(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		v1 := pulumi.String("hunter2").ToStringOutput()
		v2 := pulumi.String("hunter2-updated").ToStringOutput()

		noCaller := mergeComposedSecretsTrigger(nil, []pulumi.StringOutput{v1}).ToStringMapOutput()
		withCaller := mergeComposedSecretsTrigger(
			pulumi.StringMap{"other": pulumi.String("kept")}, []pulumi.StringOutput{v1},
		).ToStringMapOutput()
		changed := mergeComposedSecretsTrigger(nil, []pulumi.StringOutput{v2}).ToStringMapOutput()

		pulumi.All(noCaller, withCaller, changed).ApplyT(func(all []any) error {
			noCallerMap := all[0].(map[string]string)
			withCallerMap := all[1].(map[string]string)
			changedMap := all[2].(map[string]string)

			hash, ok := noCallerMap["composedSecretsSha256"]
			require.True(t, ok)
			assert.NotContains(t, hash, "hunter2", "the trigger must be a hash, never the plaintext value")

			assert.Equal(t, "kept", withCallerMap["other"], "caller-supplied triggers must be preserved")
			assert.Equal(t, hash, withCallerMap["composedSecretsSha256"],
				"merging must not change the computed hash")

			assert.NotEqual(t, hash, changedMap["composedSecretsSha256"],
				"a changed composed value must change the hash, so a rotation still forces a redeploy")
			return nil
		})
		return nil
	}, pulumi.WithMocks("myproject", "stack", &composedSecretMocks{}))
	require.NoError(t, err)
}
