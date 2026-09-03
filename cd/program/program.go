package program

import (
	"errors"
	"fmt"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"go.yaml.in/yaml/v4"
)

var (
	errMigrationAliasUnknownService = errors.New("migration alias targets an unknown service")
	errMigrationAliasConflict       = errors.New("migration alias conflicts with x-defang-aliases")
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
	return NewRunWithAliases(projectUpdate, nil)
}

// ServiceAliases maps a compose service and child-resource kind to a Pulumi
// URN that the child should adopt. The CD's migration preflight builds this
// from the existing state; keeping it out of ProjectUpdate means internal
// migration metadata is neither written back to project.pb nor exposed as a
// customer-facing compose mutation.
type ServiceAliases map[string]map[string]string

// NewRunWithAliases is NewRun with state-derived aliases applied after parsing
// compose and before the provider inputs are built. Callers must finish
// populating aliases before the returned program starts.
func NewRunWithAliases(projectUpdate *defangv1.ProjectUpdate, aliases ServiceAliases) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		defangCfg := config.New(ctx, "defang")

		provider := defangCfg.Require("provider") // "aws", "gcp", or "azure"
		domain := defangCfg.Get("domain")         // optional project domain
		etag := projectUpdate.GetEtag()           // deployment identifier
		if etag == "" {
			etag = defangCfg.Get("etag")
		}
		ttl, err := parseTTL(defangCfg.Get("ttl")) // optional self-destruct; 0 = never
		if err != nil {
			return err
		}

		if len(projectUpdate.GetCompose()) == 0 {
			return errors.New("ProjectUpdate has no compose field")
		}

		project, err := parseCompose(projectUpdate.GetCompose(), ctx.Project())
		if err != nil {
			return err
		}
		if err := applyServiceAliases(project, aliases); err != nil {
			return err
		}

		var endpoints pulumi.StringMapOutput
		var loadBalancerDns pulumi.StringPtrOutput

		switch provider {
		case "aws":
			endpoints, loadBalancerDns, err = deployAWS(ctx, project, domain, etag, ttl, projectUpdate)
		case "gcp":
			endpoints, loadBalancerDns, err = deployGCP(ctx, project, etag, ttl, defangCfg.Get("cdImage"), projectUpdate)
		case "azure":
			endpoints, loadBalancerDns, err = deployAzure(ctx, project, domain, etag, ttl, projectUpdate)
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

func applyServiceAliases(project *compose.Project, aliases ServiceAliases) error {
	for serviceName, detected := range aliases {
		svc, ok := project.Services[serviceName]
		if !ok {
			return fmt.Errorf("%w %q", errMigrationAliasUnknownService, serviceName)
		}
		if svc.Aliases == nil {
			svc.Aliases = make(map[string]string, len(detected))
		}
		for kind, urn := range detected {
			if existing := svc.Aliases[kind]; existing != "" && existing != urn {
				return fmt.Errorf("%w: service %q kind %q", errMigrationAliasConflict, serviceName, kind)
			}
			svc.Aliases[kind] = urn
		}
		project.Services[serviceName] = svc
	}
	return nil
}
