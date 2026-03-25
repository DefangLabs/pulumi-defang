package main

import (
	"fmt"
	"os"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/defangazure/azure"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"go.yaml.in/yaml/v3"
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
	return &compose.Project{Services: cf.Services}, nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "defang")

		composePath := cfg.Require("compose")
		provider := cfg.Require("provider") // "aws", "gcp", or "azure"

		project, err := parseCompose(composePath)
		if err != nil {
			return err
		}

		args := defangaws.ProjectInputs{
			Project: *project,
		}

		var result *common.BuildResult
		switch provider {
		case "aws":
			result, err = provideraws.Build(ctx, ctx.Project(), args, nil, nil)
		case "gcp":
			result, err = providergcp.Build(ctx, ctx.Project(), args, nil)
		case "azure":
			result, err = providerazure.Build(ctx, ctx.Project(), args, nil)
		default:
			return fmt.Errorf("unsupported provider: %s (must be aws, gcp, or azure)", provider)
		}
		if err != nil {
			return err
		}

		ctx.Export("endpoints", result.Endpoints)
		ctx.Export("loadBalancerDns", result.LoadBalancerDNS)

		return nil
	})
}
