package common

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMocks struct{}

func (testMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (testMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// Sanity check: when no autonaming pattern is configured, AutonamingPrefix
// should return the provided name unchanged.
func TestAutonamingPrefix(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		assert.Equal(t, "postgres", AutonamingPrefix(ctx, "postgres"))
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
}

// withConfig seeds pulumi config for the run. Config values for objects must
// be JSON-encoded strings because TryObject json.Unmarshals the raw value.
func withConfig(cfg map[string]string) pulumi.RunOption {
	return func(info *pulumi.RunInfo) { info.Config = cfg }
}

// Exercises an exhaustive pattern like the one used in examples/crewai/Pulumi.yaml:
// "Defang-${project}-${stack}-${name}-${hex(7)}". All variables are substituted,
// the hex random suffix is stripped, and the result is trimmed of trailing hyphens.
func TestAutonamingPrefix_ExhaustivePattern(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		assert.Equal(t, "Defang-proj-stack-postgres", AutonamingPrefix(ctx, "postgres"))
		return nil
	},
		pulumi.WithMocks("proj", "stack", testMocks{}),
		withConfig(map[string]string{
			"pulumi:autonaming": `{"pattern":"Defang-${project}-${stack}-${name}-${hex(7)}"}`,
		}),
	)
	require.NoError(t, err)
}
