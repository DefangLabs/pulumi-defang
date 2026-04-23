package common

import (
	"regexp"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

var randomNamingPattern = regexp.MustCompile(`\$\{((hex|alphanum|string|num)\((\d+)\)|uuid)}`)

// AutonamingPrefix reads the pulumi:autonaming pattern from stack config and
// resolves it for use as a resource name prefix (stripping random suffixes).
// Falls back to the provided name if no pattern is configured.
func AutonamingPrefix(ctx *pulumi.Context, name string) string {
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
