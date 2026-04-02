package program

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"gopkg.in/yaml.v3"
)

// Version is set by main to the build version string.
var Version = "development"

func parseCompose(data []byte, projectName string) (*compose.Project, error) {
	cf := compose.Project{Name: projectName}
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}
	return &cf, nil
}

// NewRun returns a Pulumi inline program that deploys the given compose YAML.
func NewRun(composeYaml []byte) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "defang")

		provider := cfg.Require("provider") // "aws", "gcp", or "azure"
		domain := cfg.Get("domain")         // optional project domain

		cf, err := parseCompose(composeYaml, ctx.Project())
		if err != nil {
			return err
		}

		var endpoints pulumi.StringMapOutput
		var loadBalancerDns pulumi.StringPtrOutput

		switch provider {
		case "aws":
			endpoints, loadBalancerDns, err = deployAWS(ctx, cf, domain)
		case "gcp":
			endpoints, loadBalancerDns, err = deployGCP(ctx, cf)
		case "azure":
			endpoints, loadBalancerDns, err = deployAzure(ctx, cf)
		default:
			return fmt.Errorf("unsupported provider: %q (must be aws, gcp, or azure)", provider)
		}
		if err != nil {
			return err
		}

		ctx.Export("endpoints", endpoints)
		ctx.Export("loadBalancerDns", loadBalancerDns)

		return nil
	}
}
