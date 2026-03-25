package aws

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const autonamingSuffixLen = 5 // "-" + hex(4)
const tgMaxNameLen = 32

// targetGroupName builds a logical Pulumi resource name for a TargetGroup.
// The physical name (with its 32-char AWS limit) is handled by autonaming config.
// We keep the logical name short enough that autonaming stays within budget.
func targetGroupName(service string, port int, appProtocol string) string {
	suffix := fmt.Sprintf("-%d", port)
	if appProtocol != "" && appProtocol != "http" {
		suffix += appProtocol
	}

	maxService := tgMaxNameLen - autonamingSuffixLen - len(suffix)
	if len(service) > maxService {
		service = service[:maxService]
	}

	return service + suffix
}

var randomNamingPattern = regexp.MustCompile(`\$\{((hex|alphanum|string|num)\((\d+)\)|uuid)}`)

// autonamePrefix reads the pulumi:autonaming pattern from stack config and
// resolves it for use as a resource name prefix (stripping random suffixes).
// Falls back to the provided name if no pattern is configured.
func autonamePrefix(ctx *pulumi.Context, name string) string {
	var autonaming struct {
		Pattern string `json:"pattern"`
	}
	if err := config.New(ctx, "pulumi").TryObject("autonaming", &autonaming); err != nil || autonaming.Pattern == "" {
		return name
	}

	pattern := autonaming.Pattern

	// Substitute known variables
	pattern = strings.ReplaceAll(pattern, "${organization}", ctx.Organization())
	pattern = strings.ReplaceAll(pattern, "${project}", ctx.Project())
	pattern = strings.ReplaceAll(pattern, "${stack}", ctx.Stack())
	pattern = strings.ReplaceAll(pattern, "${name}", name)

	// Strip random suffix patterns
	pattern = randomNamingPattern.ReplaceAllLiteralString(pattern, "")

	// Clean up trailing/leading/double hyphens
	pattern = strings.Trim(pattern, "-")

	return pattern
}
