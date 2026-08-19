package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoundedName(t *testing.T) {
	t.Run("leaves short names unchanged", func(t *testing.T) {
		require.Equal(t, "api-backend", BoundedName("api", "-backend", 32))
	})

	t.Run("bounds long names and retains role", func(t *testing.T) {
		name := BoundedName(strings.Repeat("service", 10), "-backend", 32)
		require.Len(t, name, 32)
		require.True(t, strings.HasSuffix(name, "-backend"))
	})

	t.Run("does not collapse equal prefixes", func(t *testing.T) {
		first := BoundedName(strings.Repeat("a", 50)+"one", "-backend", 32)
		second := BoundedName(strings.Repeat("a", 50)+"two", "-backend", 32)
		require.NotEqual(t, first, second)
	})
}
