package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

// Build creates all GCP resources for the project.
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	// Create explicit GCP provider to pin the version used by all child resources
	gcpProvArgs := &gcp.ProviderArgs{
		DefaultLabels: pulumi.StringMap{
			"defang-project": pulumi.String(projectName),
			"defang-stack":   pulumi.String(ctx.Stack()),
		},
	}
	if args.GCP != nil {
		if args.GCP.Project != "" {
			gcpProvArgs.Project = pulumi.StringPtr(args.GCP.Project)
		}
		if args.GCP.Region != "" {
			gcpProvArgs.Region = pulumi.StringPtr(args.GCP.Region)
		}
	}
	gcpProv, err := gcp.NewProvider(ctx, "gcp", gcpProvArgs, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating GCP provider: %w", err)
	}
	opts := []pulumi.ResourceOption{parentOpt, pulumi.Provider(gcpProv)}

	// Create Artifact Registry repository for container images
	ar, err := artifactregistry.NewRepository(ctx, "repo", &artifactregistry.RepositoryArgs{
		Description: pulumi.String(fmt.Sprintf("Container images for %s", projectName)),
		Format:      pulumi.String("DOCKER"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry: %w", err)
	}
	_ = ar

	recipe := LoadRecipe(ctx)

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}

	for svcName, svc := range args.Services {
		comp := &serviceComponent{}
		if err := ctx.RegisterComponentResource("defang:index:GcpService", svcName, comp, opts[0]); err != nil {
			return nil, fmt.Errorf("registering service component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp), pulumi.Provider(gcpProv)}

		result, err := CreateOneService(ctx, svcName, svc, recipe, svcOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating service %s: %w", svcName, err)
		}

		endpoints[svcName] = result.Endpoint

		if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
			"endpoint": result.Endpoint,
		}); err != nil {
			return nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}

// BuildStandalone creates GCP resources for a single standalone service.
func BuildStandalone(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, gcpCfg *common.GCPConfig, opts ...pulumi.ResourceOption) (*OneServiceResult, error) {
	// Create explicit GCP provider
	gcpProvArgs := &gcp.ProviderArgs{
		DefaultLabels: pulumi.StringMap{
			"defang-project": pulumi.String(serviceName),
			"defang-stack":   pulumi.String(ctx.Stack()),
		},
	}
	if gcpCfg != nil {
		if gcpCfg.Project != "" {
			gcpProvArgs.Project = pulumi.StringPtr(gcpCfg.Project)
		}
		if gcpCfg.Region != "" {
			gcpProvArgs.Region = pulumi.StringPtr(gcpCfg.Region)
		}
	}
	gcpProv, err := gcp.NewProvider(ctx, "gcp", gcpProvArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating GCP provider: %w", err)
	}
	provOpts := append(opts, pulumi.Provider(gcpProv))

	recipe := LoadRecipe(ctx)

	return CreateOneService(ctx, serviceName, svc, recipe, provOpts...)
}
