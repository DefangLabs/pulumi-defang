package main

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// defangProjectName is passed as the `name` argument to the defang
// NewProject call. It also determines the secret path the defang config
// provider reads (e.g. AWS SSM `/Defang/<defangProjectName>/<stack>/<KEY>`),
// so the secret we pre-create in each cloud must use this same name.
const (
	defangProjectName = "config-example"
	configKey         = "CONFIG"
	configValue       = "secr3t"
)

// testEnvironment is the interpolation matrix that every cloud variant
// uses verbatim — literal values, secret references, modifiers with and
// without defaults. All three defang providers share the same compose
// type shape, so a single pulumi.StringMap works across them.
var testEnvironment = pulumi.StringMap{
	"LITERAL":           pulumi.String("verbatim"),
	"CONFIG":            pulumi.String("${CONFIG}"), // secret
	"OTHER":             pulumi.String("${CONFIG}"), // secret
	"INTERPOLATED":      pulumi.String("prefix${CONFIG}suffix"),
	"EMPTY":             pulumi.String(""),                       // empty literal
	"MODIFIER_REQUIRED": pulumi.String("${CONFIG?required}"),     // secret, with modifier
	"MODIFIER_DEFAULT":  pulumi.String("${CONFIG-defaultValue}"), // secret, with modifier and default
	"MODIFIER_ALT":      pulumi.String("${CONFIG+altValue}"),     // secret, with modifier and alt
}

// testCommand serves the container's env vars on port 8080 so the
// interpolated values can be inspected via the ingress endpoint.
var testCommand = pulumi.StringArray{
	pulumi.String("sh"), pulumi.String("-c"),
	pulumi.String("mkdir -p /www && env > /www/index.html && httpd -f -p 8080 -h /www"),
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Pick the cloud with `pulumi config set defang:provider {aws|gcp|azure}`.
		cloud := config.New(ctx, "defang").Get("provider")
		switch cloud {
		case "aws":
			return runAws(ctx)
		case "gcp":
			return runGcp(ctx)
		case "azure":
			return runAzure(ctx)
		case "all":
			return errors.Join(
				runAws(ctx),
				runGcp(ctx),
				runAzure(ctx),
			)
		default:
			return fmt.Errorf("unsupported defang:provider %q (want aws|gcp|azure|all)", cloud)
		}
	})
}
