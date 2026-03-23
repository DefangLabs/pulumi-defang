package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
	"github.com/stretchr/testify/require"
)

func TestProviderStarts(t *testing.T) {
	server := testutil.MakeTestServer()
	require.NotNil(t, server)
}
