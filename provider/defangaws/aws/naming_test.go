package aws

import (
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/require"
)

func TestTargetGroupName(t *testing.T) {
	t.Run("short service remains readable", func(t *testing.T) {
		require.Equal(t, "api-8080", targetGroupName("api", 8080, compose.PortAppProtocolHTTP, ""))
	})

	t.Run("long services are bounded and collision resistant", func(t *testing.T) {
		prefix := strings.Repeat("a", 40)
		first := targetGroupName(prefix+"one", 8080, compose.PortAppProtocolHTTP, "")
		second := targetGroupName(prefix+"two", 8080, compose.PortAppProtocolHTTP, "")

		require.Len(t, first, tgMaxNameLen-autonamingSuffixLen)
		require.Len(t, second, tgMaxNameLen-autonamingSuffixLen)
		require.NotEqual(t, first, second)
		require.True(t, strings.HasSuffix(first, "-8080"))
	})
}
