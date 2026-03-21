package gcp

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const defaultGCPRegion = "us-central1"

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

// GcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func GcpRegion(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	if r := cfg.Get("region"); r != "" {
		return r
	}
	return defaultGCPRegion
}

// Build creates all GCP resources for the project.
// The GCP provider must be passed via the parent chain (pulumi.Providers on the parent component).
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	region := GcpRegion(ctx)
	opts := []pulumi.ResourceOption{parentOpt}

	// Create Artifact Registry repository for container images
	ar, err := artifactregistry.NewRepository(ctx, "repo", &artifactregistry.RepositoryArgs{
		RepositoryId: pulumi.String(strings.ToLower(projectName)),
		Description:  pulumi.String(fmt.Sprintf("Container images for %s", projectName)),
		Format:       pulumi.String("DOCKER"),
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

		if svc.Postgres != nil {
			// Managed Postgres → Cloud SQL
			if err := ctx.RegisterComponentResource("defang-gcp:index:GcpCloudSql", svcName, comp, opts...); err != nil {
				return nil, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			configProvider := NewConfigProvider(projectName)
			sqlResult, err := CreateCloudSQL(ctx, configProvider, svcName, svc, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", sqlResult.Instance.PublicIpAddress)
		} else {
			// Container service → Cloud Run
			if err := ctx.RegisterComponentResource("defang-gcp:index:GcpCloudRunService", svcName, comp, opts...); err != nil {
				return nil, fmt.Errorf("registering Cloud Run component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			crResult, err := CreateCloudRunService(ctx, svcName, svc, region, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
			}
			endpoints[svcName] = crResult.Service.Uri
		}

		if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
			"endpoint": endpoints[svcName],
		}); err != nil {
			return nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}
