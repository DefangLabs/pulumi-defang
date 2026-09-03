package common

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Both widenings must be lossless: an option list valid for resources and
// invokes alike keeps every element on the way to either API.
func TestOptionWideningKeepsEveryElement(t *testing.T) {
	opts := []pulumi.ResourceOrInvokeOption{pulumi.Parent(nil), pulumi.Provider(nil)}

	require.Len(t, ResourceOptions(opts), len(opts))
	require.Len(t, InvokeOptions(opts), len(opts))
}

func TestOptionWideningOnEmptyInput(t *testing.T) {
	assert.Empty(t, ResourceOptions(nil))
	assert.Empty(t, InvokeOptions(nil))
}
