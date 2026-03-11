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

// CloudRunResult holds the per-service outputs for a Cloud Run service.
type CloudRunResult struct {
	Endpoint pulumi.StringOutput
}

// CloudSQLResult holds the per-service outputs for a Cloud SQL Postgres instance.
type CloudSQLResult struct {
	Endpoint pulumi.StringOutput
}

// Build creates all GCP resources for the project.
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	// Create explicit GCP provider to pin the version used by all child resources
	gcpProv, err := createGCPProvider(ctx, projectName, args.GCP, parentOpt)
	if err != nil {
		return nil, err
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

		if svc.Postgres != nil {
			// Managed Postgres → Cloud SQL
			if err := ctx.RegisterComponentResource("defang:index:GcpCloudSql", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp), pulumi.Provider(gcpProv)}

			sqlResult, err := createCloudSQL(ctx, svcName, svc, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress)
		} else {
			// Container service → Cloud Run
			if err := ctx.RegisterComponentResource("defang:index:GcpCloudRunService", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering Cloud Run component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp), pulumi.Provider(gcpProv)}

			crResult, err := createCloudRunService(ctx, svcName, svc, recipe, svcOpts...)
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
func BuildStandaloneCloudRun(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, gcpCfg *common.GCPConfig, opts ...pulumi.ResourceOption) (*CloudRunResult, error) {
	gcpProv, err := createGCPProvider(ctx, serviceName, gcpCfg, opts...)
	if err != nil {
		return nil, err
	}
	provOpts := append(opts, pulumi.Provider(gcpProv))

	recipe := LoadRecipe(ctx)

	crResult, err := createCloudRunService(ctx, serviceName, svc, recipe, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service %s: %w", serviceName, err)
	}

	return &CloudRunResult{
		Endpoint: crResult.service.Uri,
	}, nil
}

// BuildStandaloneCloudSQL creates GCP resources for a single standalone Cloud SQL Postgres instance.
func BuildStandaloneCloudSQL(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, gcpCfg *common.GCPConfig, opts ...pulumi.ResourceOption) (*CloudSQLResult, error) {
	gcpProv, err := createGCPProvider(ctx, serviceName, gcpCfg, opts...)
	if err != nil {
		return nil, err
	}
	provOpts := append(opts, pulumi.Provider(gcpProv))

	recipe := LoadRecipe(ctx)

	sqlResult, err := createCloudSQL(ctx, serviceName, svc, recipe, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud SQL for %s: %w", serviceName, err)
	}

	return &CloudSQLResult{
		Endpoint: pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress),
	}, nil
}

// createGCPProvider creates a GCP provider with default labels.
func createGCPProvider(ctx *pulumi.Context, projectName string, gcpCfg *common.GCPConfig, opts ...pulumi.ResourceOption) (*gcp.Provider, error) {
	gcpProvArgs := &gcp.ProviderArgs{
		DefaultLabels: pulumi.StringMap{
			"defang-project": pulumi.String(projectName),
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
	return gcpProv, nil
}
