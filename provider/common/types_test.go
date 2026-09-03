package common

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeOptionsKeepsOptionsAnInvokeCanUse(t *testing.T) {
	// pulumi.Parent and pulumi.Provider are ResourceOrInvokeOption, so they
	// survive a round trip through []pulumi.ResourceOption. An invoke resolves
	// its provider from either one, which is what the CD needs because it runs
	// with pulumi:disable-default-providers.
	parent := pulumi.Parent(nil)
	provider := pulumi.Provider(nil)

	got := InvokeOptions([]pulumi.ResourceOption{parent, provider})

	require.Len(t, got, 2)
}

func TestInvokeOptionsDropsResourceOnlyOptions(t *testing.T) {
	// DeleteBeforeReplace is a plain ResourceOption; an invoke has no use for
	// it and it must not be forwarded.
	got := InvokeOptions([]pulumi.ResourceOption{pulumi.DeleteBeforeReplace(true)})

	assert.Empty(t, got)
}

func TestInvokeOptionsOnEmptyInput(t *testing.T) {
	assert.Empty(t, InvokeOptions(nil))
}
