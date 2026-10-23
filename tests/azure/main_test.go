package azure

import (
	"os"
	"testing"
)

// TestMain keeps the suite hermetic: a subscription ID leaking in from the
// developer's shell would make readLiveCustomDomains attempt a real ARM read
// during Construct instead of taking its no-subscription skip path.
func TestMain(m *testing.M) {
	_ = os.Unsetenv("ARM_SUBSCRIPTION_ID")
	_ = os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	os.Exit(m.Run())
}
