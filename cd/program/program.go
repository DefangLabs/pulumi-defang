package program

import (
	"fmt"
	"os"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"gopkg.in/yaml.v3"
)

func parseCompose(path string) (*compose.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading compose file: %w", err)
	}
	var cf compose.Project
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}
	if len(cf.Services) == 0 {
		return nil, fmt.Errorf("no services found in compose file")
	}
	return &cf, nil
}

// Run is the Pulumi inline program that deploys a compose project.
func Run(ctx *pulumi.Context) error {
	cfg := config.New(ctx, "defang")

	composePath := cfg.Require("compose")
	provider := cfg.Require("provider") // "aws", "gcp", or "azure"
	domain := cfg.Get("domain")         // optional project domain

	cf, err := parseCompose(composePath)
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
