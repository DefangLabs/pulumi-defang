package program

import (
	"errors"
	"fmt"
	"log"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"google.golang.org/protobuf/encoding/protowire"
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
		cfg := config.New(ctx, "defang")

		provider := cfg.Require("provider") // "aws", "gcp", or "azure"
		domain := cfg.Get("domain")         // optional project domain

		composeYaml, err := extractComposeYaml(projectPb)
		if err != nil {
			log.Fatalf("failed to extract compose: %v", err)
		}
		cf, err := parseCompose(composeYaml, ctx.Project())
		if err != nil {
			return err
		}

		var endpoints pulumi.StringMapOutput
		var loadBalancerDns pulumi.StringPtrOutput

		switch provider {
		case "aws":
			endpoints, loadBalancerDns, err = deployAWS(ctx, cf, domain, projectPb)
		case "gcp":
			endpoints, loadBalancerDns, err = deployGCP(ctx, cf, projectPb)
		case "azure":
			endpoints, loadBalancerDns, err = deployAzure(ctx, cf, projectPb)
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

// extractComposeYaml extracts the compose bytes (field 4) from a ProjectUpdate protobuf
// without importing the full defang CLI proto package.
func extractComposeYaml(projectUpdate []byte) ([]byte, error) {
	for len(projectUpdate) > 0 {
		num, typ, n := protowire.ConsumeTag(projectUpdate)
		if n < 0 {
			return nil, errors.New("invalid protobuf tag")
		}
		projectUpdate = projectUpdate[n:]
		switch typ {
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(projectUpdate)
			if n < 0 {
				return nil, errors.New("invalid protobuf bytes field")
			}
			if num == 4 {
				return v, nil
			}
			projectUpdate = projectUpdate[n:]
		case protowire.VarintType:
			_, n := protowire.ConsumeVarint(projectUpdate)
			if n < 0 {
				return nil, errors.New("invalid protobuf varint field")
			}
			projectUpdate = projectUpdate[n:]
		default:
			return nil, fmt.Errorf("unexpected protobuf wire type %d", typ)
		}
	}
	return nil, errors.New("ProjectUpdate has no compose field")
}
