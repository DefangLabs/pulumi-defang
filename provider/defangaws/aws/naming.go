package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

const autonamingSuffixLen = 5 // "-" + hex(4) from autonaming stack config
const tgMaxNameLen = 32

// targetGroupName builds a logical Pulumi resource name for a TargetGroup.
// The physical name (with its 32-char AWS limit) is handled by autonaming config.
// We keep the logical name short enough that autonaming stays within budget.
func targetGroupName(service string, port int, appProtocol compose.PortAppProtocol) string {
	suffix := fmt.Sprintf("-%d", port)
	if appProtocol != "" && appProtocol != "http" {
		suffix += string(appProtocol)
	}

	maxService := tgMaxNameLen - autonamingSuffixLen - len(suffix)
	if len(service) > maxService {
		service = service[:maxService]
	}

	return service + suffix
}
