package program

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"

	awss3 "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	azurestorage "github.com/pulumi/pulumi-azure-native-sdk/storage/v3"
	gcpstorage "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// projectPbKey returns the object-store key for the ProjectUpdate protobuf:
// `projects/{project}/{stack}/project.pb`. For Azure, the `projects/` prefix
// is stripped by the caller since Azure uses a dedicated `projects` container.
func projectPbKey(ctx *pulumi.Context) string {
	return fmt.Sprintf("projects/%s/%s/project.pb", ctx.Project(), ctx.Stack())
}

// saveProjectPbAWS uploads data as a Pulumi-managed S3 object at the key
// derived from DEFANG_STATE_URL. Gated on dep so the upload only runs after
// the project component (and its services) have been created successfully —
// matching the pattern used by the legacy defang-mvp CD pipeline.
//
// extraOpts should include pulumi.Provider(...) because
// pulumi:disable-default-providers excludes aws (see cd/main.go projectConfig).
func saveProjectPbAWS(ctx *pulumi.Context, data []byte, dep pulumi.Resource, extraOpts ...pulumi.ResourceOption) error {
	stateURL := os.Getenv("DEFANG_STATE_URL")
	if stateURL == "" {
		return fmt.Errorf("DEFANG_STATE_URL is not set; cannot upload project.pb")
	}
	u, err := url.Parse(stateURL)
	if err != nil {
		return fmt.Errorf("invalid DEFANG_STATE_URL %q: %w", stateURL, err)
	}
	if u.Scheme != "s3" || u.Host == "" {
		return fmt.Errorf("DEFANG_STATE_URL must be an s3:// URL with a bucket for AWS uploads, got %q", stateURL)
	}
	opts := append([]pulumi.ResourceOption{pulumi.DependsOn([]pulumi.Resource{dep})}, extraOpts...)
	// ContentBase64 preserves binary bytes; Content (string) would fail gRPC
	// marshaling because protobuf is not valid UTF-8. The provider decodes
	// the base64 server-side and stores raw bytes in S3.
	_, err = awss3.NewBucketObjectv2(ctx, "project-pb", &awss3.BucketObjectv2Args{
		Bucket:        pulumi.String(u.Host),
		Key:           pulumi.String(projectPbKey(ctx)),
		ContentBase64: pulumi.String(base64.StdEncoding.EncodeToString(data)),
		ContentType:   pulumi.String("application/protobuf"),
	}, opts...)
	return err
}

// saveProjectPbGCP uploads data as a Pulumi-managed GCS object at the key
// derived from DEFANG_STATE_URL. See saveProjectPbAWS for semantics.
func saveProjectPbGCP(ctx *pulumi.Context, data []byte, dep pulumi.Resource, extraOpts ...pulumi.ResourceOption) error {
	stateURL := os.Getenv("DEFANG_STATE_URL")
	if stateURL == "" {
		return fmt.Errorf("DEFANG_STATE_URL is not set; cannot upload project.pb")
	}
	u, err := url.Parse(stateURL)
	if err != nil {
		return fmt.Errorf("invalid DEFANG_STATE_URL %q: %w", stateURL, err)
	}
	if u.Scheme != "gs" || u.Host == "" {
		return fmt.Errorf("DEFANG_STATE_URL must be a gs:// URL with a bucket for GCP uploads, got %q", stateURL)
	}
	opts := append([]pulumi.ResourceOption{pulumi.DependsOn([]pulumi.Resource{dep})}, extraOpts...)
	asset, cleanup, err := binaryFileAsset(data, "project-pb-*.pb")
	if err != nil {
		return err
	}
	obj, err := gcpstorage.NewBucketObject(ctx, "project-pb", &gcpstorage.BucketObjectArgs{
		Bucket:      pulumi.String(u.Host),
		Name:        pulumi.String(projectPbKey(ctx)),
		Source:      asset,
		ContentType: pulumi.String("application/protobuf"),
	}, opts...)
	if err != nil {
		cleanup()
		return err
	}
	// Remove the temp file after the blob has been created (Crc32c resolves post-create).
	obj.Crc32c.ApplyT(func(string) error { cleanup(); return nil })
	return nil
}

// saveProjectPbAzure uploads data as a Pulumi-managed Azure Blob in the CD
// storage account's `projects` container. See saveProjectPbAWS for semantics.
func saveProjectPbAzure(ctx *pulumi.Context, data []byte, dep pulumi.Resource, extraOpts ...pulumi.ResourceOption) error {
	stateURL := os.Getenv("DEFANG_STATE_URL")
	if stateURL == "" {
		return fmt.Errorf("DEFANG_STATE_URL is not set; cannot upload project.pb")
	}
	u, err := url.Parse(stateURL)
	if err != nil {
		return fmt.Errorf("invalid DEFANG_STATE_URL %q: %w", stateURL, err)
	}
	if u.Scheme != "azblob" {
		return fmt.Errorf("DEFANG_STATE_URL must be an azblob:// URL for Azure uploads, got %q", stateURL)
	}
	account := u.Query().Get("storage_account")
	if account == "" {
		return fmt.Errorf("DEFANG_STATE_URL %q missing storage_account", stateURL)
	}
	// The CD storage account lives in the shared CD resource group, named
	// `defang-cd-<location>` by convention (see defang/src/pkg/clouds/azure/cd/driver.go).
	location := os.Getenv("AZURE_LOCATION")
	if location == "" {
		return fmt.Errorf("AZURE_LOCATION must be set to derive the CD resource group")
	}
	cdRG := "defang-cd-" + location

	// Azure uses a dedicated `projects` container, so strip the AWS/GCS-style
	// `projects/` prefix from the object key — otherwise the blob lands at
	// `projects/projects/<project>/<stack>/project.pb`.
	blobName := fmt.Sprintf("%s/%s/project.pb", ctx.Project(), ctx.Stack())

	opts := append([]pulumi.ResourceOption{pulumi.DependsOn([]pulumi.Resource{dep})}, extraOpts...)
	asset, cleanup, err := binaryFileAsset(data, "project-pb-*.pb")
	if err != nil {
		return err
	}
	blob, err := azurestorage.NewBlob(ctx, "project-pb", &azurestorage.BlobArgs{
		ResourceGroupName: pulumi.String(cdRG),
		AccountName:       pulumi.String(account),
		ContainerName:     pulumi.String("projects"),
		BlobName:          pulumi.String(blobName),
		Source:            asset,
		ContentType:       pulumi.StringPtr("application/protobuf"),
	}, opts...)
	if err != nil {
		cleanup()
		return err
	}
	blob.Url.ApplyT(func(string) error { cleanup(); return nil })
	return nil
}

// binaryFileAsset writes data to a temp file and returns a FileAsset referencing it.
// Pulumi's StringAsset rejects non-UTF-8 data (gRPC marshal error), so binary
// protobufs must go through the filesystem. Returned cleanup should be invoked
// after the consuming resource has been created.
func binaryFileAsset(data []byte, pattern string) (pulumi.Asset, func(), error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, func() {}, fmt.Errorf("creating temp file for project.pb: %w", err)
	}
	cleanup := func() { _ = os.Remove(f.Name()) }
	if _, err := f.Write(data); err != nil {
		f.Close()
		cleanup()
		return nil, func() {}, fmt.Errorf("writing temp file for project.pb: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("closing temp file for project.pb: %w", err)
	}
	return pulumi.NewFileAsset(f.Name()), cleanup, nil
}
