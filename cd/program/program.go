package program

import (
	"errors"
	"fmt"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"gopkg.in/yaml.v3"
)

func parseCompose(data []byte, projectName string) (*compose.Project, error) {
	cf := compose.Project{Name: projectName}
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}
	return &cf, nil
}

// NewRun returns a Pulumi inline program that deploys the given compose YAML.
// projectUpdate is the ProjectUpdate protobuf; each per-provider deploy
// function uploads it as a Pulumi-managed blob at the end of the deploy
// (gated on the project component so the upload only happens on success).
func NewRun(projectUpdate *defangv1.ProjectUpdate) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		defangCfg := config.New(ctx, "defang")

		provider := defangCfg.Require("provider") // "aws", "gcp", or "azure"
		domain := defangCfg.Get("domain")         // optional project domain
		etag := projectUpdate.Etag                // deployment identifier
		if etag == "" {
			etag = defangCfg.Get("etag")
		}

		if len(projectUpdate.Compose) == 0 {
			return errors.New("ProjectUpdate has no compose field")
		}

		project, err := parseCompose(projectUpdate.Compose, ctx.Project())
		if err != nil {
			return err
		}

		var endpoints pulumi.StringMapOutput
		var loadBalancerDns pulumi.StringPtrOutput

		switch provider {
		case "aws":
			endpoints, loadBalancerDns, err = deployAWS(ctx, project, domain, etag, projectUpdate)
		case "gcp":
			endpoints, loadBalancerDns, err = deployGCP(ctx, project, etag, projectUpdate)
		case "azure":
			endpoints, loadBalancerDns, err = deployAzure(ctx, project, etag, projectUpdate)
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
