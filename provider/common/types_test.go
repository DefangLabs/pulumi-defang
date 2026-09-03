package common

import (
	"reflect"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/internals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parentComponent struct{ pulumi.ResourceState }

type childComponent struct{ pulumi.ResourceState }

// The widenings have to preserve what each option actually does, not merely how
// many there are. Registering a resource through ResourceOptions and reading the
// resulting URN proves the parent survived: a child's URN embeds its parent's
// type, so a dropped or replaced option changes it.
func TestResourceOptionsKeepsTheOptionsEffect(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		parent := &parentComponent{}
		require.NoError(t, ctx.RegisterComponentResource("test:index:Parent", "mum", parent))

		child := &childComponent{}
		opts := []pulumi.ResourceOrInvokeOption{pulumi.Parent(parent)}
		require.NoError(t,
			ctx.RegisterComponentResource("test:index:Child", "kid", child, ResourceOptions(opts)...))

		urn, err := internals.UnsafeAwaitOutput(ctx.Context(), child.URN())
		require.NoError(t, err)
		assert.Contains(t, urn.Value.(pulumi.URN), "test:index:Parent$test:index:Child::kid",
			"child lost the parent carried by the widened option")
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
}

// The InvokeOptions half is covered end-to-end by
// TestCloudSQLResolvesConfigPasswordWithExplicitProvider in provider/defanggcp/gcp,
// which asserts the widened option still reaches the Secret Manager invoke as a
// provider. Here we only pin the mechanical guarantee: nothing is added, dropped
// or reordered on the way through.
func TestWideningIsElementWise(t *testing.T) {
	opts := []pulumi.ResourceOrInvokeOption{pulumi.Parent(nil), pulumi.Provider(nil), pulumi.Parent(nil)}

	resourceOpts := ResourceOptions(opts)
	invokeOpts := InvokeOptions(opts)

	require.Len(t, resourceOpts, len(opts))
	require.Len(t, invokeOpts, len(opts))
	// Options are func values, so identity is the code pointer behind them.
	// Parent and Provider are distinct functions, which is what makes a
	// reordering or a substitution visible here.
	for i, opt := range opts {
		want := reflect.ValueOf(opt).Pointer()
		assert.Equal(t, want, reflect.ValueOf(resourceOpts[i]).Pointer(),
			"resource option %d is not the input option", i)
		assert.Equal(t, want, reflect.ValueOf(invokeOpts[i]).Pointer(),
			"invoke option %d is not the input option", i)
	}
}

func TestOptionWideningOnEmptyInput(t *testing.T) {
	assert.Empty(t, ResourceOptions(nil))
	assert.Empty(t, InvokeOptions(nil))
}
