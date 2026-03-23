package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderStarts(t *testing.T) {
	server := makeTestServer()
	require.NotNil(t, server)
}
