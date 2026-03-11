package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v8/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

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

	// Deploy each service
	endpoints := pulumi.StringMap{}
	var hasPostgres bool

	// Check if any service needs postgres (for VPC/service connection)
	for _, svc := range args.Services {
		if svc.Postgres != nil {
			hasPostgres = true
			break
		}
	}
	_ = hasPostgres

	recipe := LoadRecipe(ctx)

	for svcName, svc := range args.Services {
		if svc.Postgres != nil {
			// Create managed Cloud SQL Postgres
			sqlResult, err := createCloudSQL(ctx, svcName, svc, recipe, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress)
		} else {
			// Create Cloud Run service
			crResult, err := createCloudRunService(ctx, svcName, svc, recipe, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
			}
			endpoints[svcName] = crResult.service.Uri
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}

// BuildService creates GCP resources for a single standalone service.
func BuildService(ctx *pulumi.Context, serviceName string, args common.ServiceBuildArgs, parentOpt pulumi.ResourceOption) (*common.ServiceBuildResult, error) {
	svc := args.Service

	// Create explicit GCP provider to pin the version used by all child resources
	gcpProvArgs := &gcp.ProviderArgs{
		DefaultLabels: pulumi.StringMap{
			"defang-project": pulumi.String(serviceName),
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

	recipe := LoadRecipe(ctx)

	if svc.Postgres != nil {
		sqlResult, err := createCloudSQL(ctx, serviceName, svc, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating Cloud SQL: %w", err)
		}
		return &common.ServiceBuildResult{
			Endpoint: pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress),
		}, nil
	}

	crResult, err := createCloudRunService(ctx, serviceName, svc, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service: %w", err)
	}

	return &common.ServiceBuildResult{
		Endpoint: crResult.service.Uri,
	}, nil
}
