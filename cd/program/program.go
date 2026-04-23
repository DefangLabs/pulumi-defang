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
// projectPb is the raw ProjectUpdate protobuf; it's uploaded as a
// Pulumi-managed blob at the end of the deploy (gated on the project
// component so the upload only happens on success).
func NewRun(composeYaml, projectPb []byte) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "defang")

		provider := cfg.Require("provider") // "aws", "gcp", or "azure"
		domain := cfg.Get("domain")         // optional project domain

		cf, err := parseCompose(composeYaml, ctx.Project())
		if err != nil {
			return err
		}

		var projectRes pulumi.Resource
		var endpoints pulumi.StringMapOutput
		var loadBalancerDns pulumi.StringPtrOutput

		switch provider {
		case "aws":
			projectRes, endpoints, loadBalancerDns, err = deployAWS(ctx, cf, domain)
		case "gcp":
			projectRes, endpoints, loadBalancerDns, err = deployGCP(ctx, cf)
		case "azure":
			projectRes, endpoints, loadBalancerDns, err = deployAzure(ctx, cf)
		default:
			return fmt.Errorf("unsupported provider: %q (must be aws, gcp, or azure)", provider)
		}
		if err != nil {
			return err
		}

		// Upload ProjectUpdate protobuf as a Pulumi-managed blob, gated on the
		// project component so it only runs after all services are created.
		// The CLI reads this file via `provider.GetProjectUpdate` for
		// `defang compose ps`, log filtering, etc.
		if len(projectPb) > 0 {
			if err := saveProjectPb(ctx, provider, projectPb, projectRes); err != nil {
				return err
			}
		}

		ctx.Export("endpoints", endpoints)
		ctx.Export("loadBalancerDns", loadBalancerDns)

		return nil
	}
}
