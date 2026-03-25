package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// BuildInfra holds GCP infrastructure shared across all services with build configs.
// Created once per project when at least one service defines a build context.
type BuildInfra struct {
	Repository     *artifactregistry.Repository
	ServiceAccount *serviceaccount.Account
	BuildBucket    *storage.Bucket // GCS bucket for uploading local build contexts
	RepositoryURL  pulumi.StringOutput // e.g. "us-central1-docker.pkg.dev/project/repo"
	Region         string
	GcpProject     string
}

// hasBuildConfig reports whether any service in the map defines a build context.
func hasBuildConfig(services map[string]compose.ServiceConfig) bool {
	for _, svc := range services {
		if svc.Build != nil {
			return true
		}
	}
	return false
}

// gcpProjectId reads the GCP project ID from Pulumi stack config.
func gcpProjectId(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	return cfg.Get("project")
}

// createBuildInfra creates the shared GCP infrastructure required to build container images:
// an Artifact Registry repository, a GCS bucket for build artifacts, a build service account,
// and the associated IAM bindings.
func createBuildInfra(
	ctx *pulumi.Context,
	projectName string,
	opts ...pulumi.ResourceOption,
) (*BuildInfra, error) {
	region := GcpRegion(ctx)
	gcpProject := gcpProjectId(ctx)

	bsa, err := serviceaccount.NewAccount(ctx, projectName+"-build", &serviceaccount.AccountArgs{
		AccountId:   pulumi.String(sanitizeAccountId(projectName + "-build")),
		DisplayName: pulumi.String("Image build service account for " + projectName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating build service account: %w", err)
	}

	ar, err := artifactregistry.NewRepository(ctx, projectName+"-repo", &artifactregistry.RepositoryArgs{
		Location:     pulumi.String(region),
		RepositoryId: pulumi.String(sanitizeAccountId(projectName) + "-repo"),
		Description:  pulumi.String("Docker images for " + projectName),
		Format:       pulumi.String("DOCKER"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry repository: %w", err)
	}

	bucket, err := storage.NewBucket(ctx, projectName+"-build-artifacts", &storage.BucketArgs{
		Location:                 pulumi.String(region),
		ForceDestroy:             pulumi.Bool(true),
		UniformBucketLevelAccess: pulumi.Bool(true),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating build artifacts bucket: %w", err)
	}

	saOpts := make([]pulumi.ResourceOption, 0, len(opts)+2)
	saOpts = append(saOpts, opts...)
	saOpts = append(saOpts, pulumi.DeletedWith(bsa), pulumi.DeleteBeforeReplace(true))

	repoIAMArgs := &artifactregistry.RepositoryIamBindingArgs{
		Location:   pulumi.String(region),
		Project:    pulumi.String(gcpProject),
		Repository: ar.Name,
		Role:       pulumi.String("roles/artifactregistry.admin"),
		Members:    pulumi.StringArray{pulumi.Sprintf("serviceAccount:%v", bsa.Email)},
	}
	if _, err := artifactregistry.NewRepositoryIamBinding(
		ctx, projectName+"-repo-writer", repoIAMArgs, saOpts...,
	); err != nil {
		return nil, fmt.Errorf("binding artifact registry admin role: %w", err)
	}

	if _, err := storage.NewBucketIAMMember(ctx, projectName+"-build-bucket-viewer", &storage.BucketIAMMemberArgs{
		Bucket: bucket.Name,
		Role:   pulumi.String("roles/storage.objectViewer"),
		Member: pulumi.Sprintf("serviceAccount:%v", bsa.Email),
	}, saOpts...); err != nil {
		return nil, fmt.Errorf("binding storage.objectViewer role: %w", err)
	}

	if _, err := projects.NewIAMMember(ctx, projectName+"-build-log-writer", &projects.IAMMemberArgs{
		Project: pulumi.String(gcpProject),
		Role:    pulumi.String("roles/logging.logWriter"),
		Member:  pulumi.Sprintf("serviceAccount:%v", bsa.Email),
	}, saOpts...); err != nil {
		return nil, fmt.Errorf("binding logging.logWriter role: %w", err)
	}

	if _, err := projects.NewIAMMember(ctx, projectName+"-build-log-bucket-writer", &projects.IAMMemberArgs{
		Project: pulumi.String(gcpProject),
		Role:    pulumi.String("roles/logging.bucketWriter"),
		Member:  pulumi.Sprintf("serviceAccount:%v", bsa.Email),
	}, saOpts...); err != nil {
		return nil, fmt.Errorf("binding logging.bucketWriter role: %w", err)
	}

	repoURL := pulumi.Sprintf("%s-docker.pkg.dev/%s/%s", region, gcpProject, ar.Name)

	return &BuildInfra{
		Repository:     ar,
		ServiceAccount: bsa,
		BuildBucket:    bucket,
		RepositoryURL:  repoURL,
		Region:         region,
		GcpProject:     gcpProject,
	}, nil
}
