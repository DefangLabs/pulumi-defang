package program

import (
	"errors"
	"fmt"
	"log"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"google.golang.org/protobuf/proto"
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
// projectPb is the raw ProjectUpdate protobuf; each per-provider deploy
// function uploads it as a Pulumi-managed blob at the end of the deploy
// (gated on the project component so the upload only happens on success).
func NewRun(projectPb []byte) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		defangCfg := config.New(ctx, "defang")

		provider := defangCfg.Require("provider") // "aws", "gcp", or "azure"
		domain := defangCfg.Get("domain")         // optional project domain

		composeYaml, err := extractComposeYaml(projectPb)
		if err != nil {
			log.Fatalf("failed to extract compose: %v", err)
		}
		project, err := parseCompose(composeYaml, ctx.Project())
		if err != nil {
			return err
		}

		var endpoints pulumi.StringMapOutput
		var loadBalancerDns pulumi.StringPtrOutput

		switch provider {
		case "aws":
			endpoints, loadBalancerDns, err = deployAWS(ctx, project, domain, projectPb)
		case "gcp":
			endpoints, loadBalancerDns, err = deployGCP(ctx, project, projectPb)
		case "azure":
			endpoints, loadBalancerDns, err = deployAzure(ctx, project, projectPb)
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

// extractComposeYaml unmarshals a ProjectUpdate protobuf and returns its
// embedded compose YAML bytes. Errors if the protobuf is malformed or the
// Compose field is empty (e.g. the CLI sent a ProjectUpdate without a
// compose file).
func extractComposeYaml(projectUpdate []byte) ([]byte, error) {
	var pu defangv1.ProjectUpdate
	if err := proto.Unmarshal(projectUpdate, &pu); err != nil {
		return nil, fmt.Errorf("unmarshaling ProjectUpdate: %w", err)
	}
	if len(pu.Compose) == 0 {
		return nil, errors.New("ProjectUpdate has no compose field")
	}
	return pu.Compose, nil
}
