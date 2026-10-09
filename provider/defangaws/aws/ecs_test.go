package aws

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ssmParameterType = "aws:ssm/parameter:Parameter"

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
	mu        sync.Mutex
	created   []pulumi.MockResourceArgs
	ssmParams int // count of SSM parameters created so far, used to fake a distinct Version per one
}

func (m *composedSecretMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created = append(m.created, args)
	outputs := args.Inputs
	if args.TypeToken == ssmParameterType {
		outputs["arn"] = resource.NewStringProperty("arn:aws:ssm:us-west-2:123456789012:parameter" + args.Name)
		m.ssmParams++
		outputs["version"] = resource.NewNumberProperty(float64(m.ssmParams))
	}
	return args.Name + "_id", outputs, nil
}

// Parent names do not appear in child URNs, so identical sidecar names in
// separate Service components still need distinct logical resource names.
func TestComposedSidecarSecretsHaveUniqueURNs(t *testing.T) {
	mocks := &composedSecretMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		for _, name := range []string{"api", "worker", "proxy"} {
			var parent pulumi.ResourceState
			if err := ctx.RegisterComponentResource("defang-aws:index:Service", name, &parent); err != nil {
				return err
			}
			env := compose.Environment{"AUTH": pulumi.String("Bearer ${TOKEN}")}
			args := &ECSServiceArgs{
				ImageURI: pulumi.String("app:latest"),
				Sidecars: map[string]compose.ServiceConfig{
					"proxy": {Image: pulumi.String("proxy:latest"), Environment: env},
				},
			}
			_, err := CreateECSService(ctx, &fakeConfigProvider{values: map[string]string{"TOKEN": "value"}},
				name, compose.ServiceConfig{Environment: env}, args, nil, pulumi.Parent(&parent))
			if err != nil {
				return err
			}
		}
		return nil
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)

	seen := make(map[resource.URN]bool)
	for _, r := range mocks.created {
		if r.TypeToken != ssmParameterType {
			continue
		}
		parentType := resource.URN(r.RegisterRPC.GetParent()).QualifiedType()
		urn := resource.NewURN("stack", "project", parentType, tokens.Type(r.TypeToken), r.Name)
		assert.False(t, seen[urn], "duplicate SSM parameter URN: %s", urn)
		seen[urn] = true
	}
	assert.Len(t, seen, 6, "each main container and sidecar needs its own parameter")
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
		arnOutput, versionOutput, err := newComposedEnvSecret(ctx, "myservice", "DATABASE_URL", value, nil)
		require.NoError(t, err)

		arnOutput.ApplyT(func(arn string) string {
			assert.True(t, strings.HasPrefix(arn, "arn:aws:ssm:"), "got %q", arn)
			assert.NotContains(t, arn, "hunter2", "the ARN must never embed the secret value")
			return arn
		})
		versionOutput.ApplyT(func(version int) int {
			assert.Positive(t, version, "expected the parameter's Version output to be populated")
			return version
		})
		return nil
	}, pulumi.WithMocks("myproject", "stack", mocks))
	require.NoError(t, err)

	require.Len(t, mocks.created, 1, "expected exactly one SSM parameter to be created")
	param := mocks.created[0]
	assert.Equal(t, ssmParameterType, param.TypeToken)
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
		case ssmParameterType:
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
	require.True(t, triggers.IsObject(), "expected a composedSecretsVersion trigger since a composed secret exists")
	version, ok := triggers.ObjectValue()["composedSecretsVersion"]
	require.True(t, ok, "expected a composedSecretsVersion trigger since a composed secret exists")
	assert.NotContains(t, version.StringValue(), password,
		"the trigger must be a version number, never the plaintext value")

	// Triggers alone has no effect on ECS — ForceNewDeployment must also be
	// set, or a rotated composed secret's value is never picked up despite
	// the trigger changing. See the AWS provider's "Redeploy Service On
	// Every Apply" example, which pairs the two.
	forceNewDeployment, ok := serviceInputs["forceNewDeployment"]
	require.True(t, ok, "expected forceNewDeployment to be set since a composed secret exists")
	assert.True(t, forceNewDeployment.BoolValue())
}

// TestMergeComposedSecretsTrigger verifies the Triggers helper: it merges the
// composed secrets' SSM parameter Versions into any caller-supplied triggers,
// and the trigger value changes when a version changes — which is what makes
// a value rotation force an ECS redeployment despite the task definition's
// own JSON staying the same (its Secrets entry holds a stable parameter Arn,
// not the value; see newComposedEnvSecret). See DefangLabs/pulumi-defang#638.
func TestMergeComposedSecretsTrigger(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		v1 := pulumi.Int(1).ToIntOutput()
		v2 := pulumi.Int(2).ToIntOutput()

		noCaller := mergeComposedSecretsTrigger(nil, []pulumi.IntOutput{v1}).ToStringMapOutput()
		withCaller := mergeComposedSecretsTrigger(
			pulumi.StringMap{"other": pulumi.String("kept")}, []pulumi.IntOutput{v1},
		).ToStringMapOutput()
		changed := mergeComposedSecretsTrigger(nil, []pulumi.IntOutput{v2}).ToStringMapOutput()

		pulumi.All(noCaller, withCaller, changed).ApplyT(func(all []any) error {
			noCallerMap := all[0].(map[string]string)
			withCallerMap := all[1].(map[string]string)
			changedMap := all[2].(map[string]string)

			version, ok := noCallerMap["composedSecretsVersion"]
			require.True(t, ok)
			assert.Equal(t, "1", version)

			assert.Equal(t, "kept", withCallerMap["other"], "caller-supplied triggers must be preserved")
			assert.Equal(t, version, withCallerMap["composedSecretsVersion"],
				"merging must not change the computed trigger")

			assert.NotEqual(t, version, changedMap["composedSecretsVersion"],
				"a changed composed secret's version must change the trigger, so a rotation still forces a redeploy")
			return nil
		})
		return nil
	}, pulumi.WithMocks("myproject", "stack", &composedSecretMocks{}))
	require.NoError(t, err)
}
