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

// CloudRunResult holds the per-service outputs for a Cloud Run service.
type CloudRunResult struct {
	Endpoint pulumi.StringOutput
}

// CloudSQLResult holds the per-service outputs for a Cloud SQL Postgres instance.
type CloudSQLResult struct {
	Endpoint pulumi.StringOutput
}

// gcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func gcpRegion(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	if r := cfg.Get("region"); r != "" {
		return r
	}
	return defaultGCPRegion
}

// Build creates all GCP resources for the project.
// The GCP provider must be passed via the parent chain (pulumi.Providers on the parent component).
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	region := gcpRegion(ctx)
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
			if err := ctx.RegisterComponentResource("defang-gcp:index:GcpCloudSql", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			sqlResult, err := createCloudSQL(ctx, svcName, svc, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress)
		} else {
			// Container service → Cloud Run
			if err := ctx.RegisterComponentResource("defang-gcp:index:GcpCloudRunService", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Cloud Run component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			crResult, err := createCloudRunService(ctx, svcName, svc, region, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
			}
			endpoints[svcName] = crResult.service.Uri
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

// BuildStandaloneCloudRun creates GCP resources for a single standalone Cloud Run service.
// The GCP provider must be passed via opts (pulumi.Providers on the parent component).
func BuildStandaloneCloudRun(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, opts ...pulumi.ResourceOption) (*CloudRunResult, error) {
	region := gcpRegion(ctx)
	recipe := LoadRecipe(ctx)

	crResult, err := createCloudRunService(ctx, serviceName, svc, region, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service %s: %w", serviceName, err)
	}

	return &CloudRunResult{
		Endpoint: crResult.service.Uri,
	}, nil
}

// BuildStandaloneCloudSQL creates GCP resources for a single standalone Cloud SQL Postgres instance.
// The GCP provider must be passed via opts (pulumi.Providers on the parent component).
func BuildStandaloneCloudSQL(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, opts ...pulumi.ResourceOption) (*CloudSQLResult, error) {
	recipe := LoadRecipe(ctx)

	sqlResult, err := createCloudSQL(ctx, serviceName, svc, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud SQL for %s: %w", serviceName, err)
	}

	return &CloudSQLResult{
		Endpoint: pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress),
	}, nil
}
