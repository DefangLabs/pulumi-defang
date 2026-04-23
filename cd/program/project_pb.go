package program

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"

	awss3 "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	azurestorage "github.com/pulumi/pulumi-azure-native-sdk/storage/v3"
	gcpstorage "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// saveProjectPb uploads the ProjectUpdate protobuf as a Pulumi-managed
// blob on the provider's object store, derived from DEFANG_STATE_URL. The
// upload DependsOn dep so it runs only after the project component (and
// its services) have been created successfully — matching the pattern used
// by the legacy defang-mvp CD pipeline (pulumi-gcp's BucketObject,
// pulumi-aws's s3.BucketObject).
func saveProjectPb(ctx *pulumi.Context, provider string, data []byte, dep pulumi.Resource) error {
	stateURL := os.Getenv("DEFANG_STATE_URL")
	if stateURL == "" {
		return fmt.Errorf("DEFANG_STATE_URL is not set; cannot upload project.pb")
	}
	key := fmt.Sprintf("projects/%s/%s/project.pb", ctx.Project(), ctx.Stack())
	opts := []pulumi.ResourceOption{pulumi.DependsOn([]pulumi.Resource{dep})}

	switch provider {
	case "aws":
		return saveProjectPbAWS(ctx, stateURL, key, data, opts)
	case "gcp":
		return saveProjectPbGCP(ctx, stateURL, key, data, opts)
	case "azure":
		return saveProjectPbAzure(ctx, stateURL, key, data, opts)
	}
	return fmt.Errorf("unsupported provider for project.pb upload: %q", provider)
}

func saveProjectPbAWS(ctx *pulumi.Context, stateURL, key string, data []byte, opts []pulumi.ResourceOption) error {
	bucket := strings.SplitN(strings.TrimPrefix(stateURL, "s3://"), "/", 2)[0]
	bucket = strings.SplitN(bucket, "?", 2)[0]
	if bucket == "" {
		return fmt.Errorf("cannot extract bucket from DEFANG_STATE_URL %q", stateURL)
	}
	// ContentBase64 preserves binary bytes; Content (string) would fail gRPC
	// marshaling because protobuf is not valid UTF-8. The provider decodes
	// the base64 server-side and stores raw bytes in S3.
	_, err := awss3.NewBucketObjectv2(ctx, "project-pb", &awss3.BucketObjectv2Args{
		Bucket:        pulumi.String(bucket),
		Key:           pulumi.String(key),
		ContentBase64: pulumi.String(base64.StdEncoding.EncodeToString(data)),
		ContentType:   pulumi.String("application/protobuf"),
	}, opts...)
	return err
}

func saveProjectPbGCP(ctx *pulumi.Context, stateURL, key string, data []byte, opts []pulumi.ResourceOption) error {
	bucket := strings.SplitN(strings.TrimPrefix(stateURL, "gs://"), "/", 2)[0]
	bucket = strings.SplitN(bucket, "?", 2)[0]
	if bucket == "" {
		return fmt.Errorf("cannot extract bucket from DEFANG_STATE_URL %q", stateURL)
	}
	asset, cleanup, err := binaryFileAsset(data, "project-pb-*.pb")
	if err != nil {
		return err
	}
	obj, err := gcpstorage.NewBucketObject(ctx, "project-pb", &gcpstorage.BucketObjectArgs{
		Bucket:      pulumi.String(bucket),
		Name:        pulumi.String(key),
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

func saveProjectPbAzure(ctx *pulumi.Context, stateURL, key string, data []byte, opts []pulumi.ResourceOption) error {
	// azblob://<container>?storage_account=<acct>
	u, err := url.Parse(stateURL)
	if err != nil {
		return fmt.Errorf("parsing DEFANG_STATE_URL %q: %w", stateURL, err)
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

	// Azure uses a dedicated `projects` container, so strip the `projects/`
	// prefix that AWS/GCS key convention requires (they use a shared bucket).
	blobName := strings.TrimPrefix(key, "projects/")

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
