package gcp

import (
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func strPtr(s string) *string { return &s }

type testMocks struct{}

func (testMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (testMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// buildDepsSpy records the Pulumi dependency URNs of the Build custom resource.
type buildDepsSpy struct {
	mu        sync.Mutex
	buildDeps []string
}

func (m *buildDepsSpy) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "defang-gcp:defanggcp:Build" && args.RegisterRPC != nil {
		m.mu.Lock()
		m.buildDeps = args.RegisterRPC.GetDependencies()
		m.mu.Unlock()
	}
	return args.Name + "_id", args.Inputs, nil
}

func (m *buildDepsSpy) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestCloudBuildMachineType(t *testing.T) {
	MiB := 1024 * 1024
	tests := []struct {
		name     string
		shmBytes int
		want     string
	}{
		{"zero defaults to 8 GiB -> E2_HIGHCPU_8", 0, "E2_HIGHCPU_8"},
		{"4096 MiB -> E2_MEDIUM", 4096 * MiB, "E2_MEDIUM"},
		{"4097 MiB -> E2_HIGHCPU_8", 4097 * MiB, "E2_HIGHCPU_8"},
		{"8192 MiB -> E2_HIGHCPU_8", 8192 * MiB, "E2_HIGHCPU_8"},
		{"8193 MiB -> E2_HIGHCPU_32", 8193 * MiB, "E2_HIGHCPU_32"},
		{"32768 MiB -> E2_HIGHCPU_32", 32768 * MiB, "E2_HIGHCPU_32"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cloudBuildMachineType(tt.shmBytes))
		})
	}
}

func TestCloudBuildDiskSizeGb(t *testing.T) {
	MiB := 1024 * 1024
	tests := []struct {
		name     string
		shmBytes int
		want     int
	}{
		{"zero defaults to 8 GiB -> 16 GB", 0, 16},
		{"4096 MiB -> 8 GB, clamped to 16", 4096 * MiB, 16},
		{"8192 MiB -> 16 GB", 8192 * MiB, 16},
		{"16384 MiB -> 32 GB", 16384 * MiB, 32},
		{"32768 MiB -> 64 GB", 32768 * MiB, 64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cloudBuildDiskSizeGb(tt.shmBytes))
		})
	}
}

func TestIsCloudRunSupportedRegistry(t *testing.T) {
	tests := []struct {
		registry string
		want     bool
	}{
		{"", true},          // implicit docker.io
		{"docker.io", true}, // explicit docker.io
		{"gcr.io", true},
		{"us.gcr.io", true},
		{"us-central1.gcr.io", true},
		{"docker.pkg.dev", true}, // no region prefix
		{"us-central1-docker.pkg.dev", true},
		{"europe-west1-docker.pkg.dev", true},
		{"quay.io", false},
		{"ghcr.io", false},
		{"registry.example.com", false},
		{"my-registry.io", false},
	}
	for _, tt := range tests {
		t.Run(tt.registry, func(t *testing.T) {
			assert.Equal(t, tt.want, isCloudRunSupportedRegistry(tt.registry))
		})
	}
}

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"quay.io", "quay-io"},
		{"ghcr.io", "ghcr-io"},
		{"my.registry.example.com", "my-registry-example-com"},
		{"UPPER.CASE.IO", "upper-case-io"},
		{"-leading-dash", "leading-dash"},
		{"trailing-dash-", "trailing-dash"},
		// 64-char input should be trimmed to 63
		{
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeRepoName(tt.input))
		})
	}
}

func TestGetServiceImage(t *testing.T) {
	fakeInfra := &BuildInfra{
		GcpProject: "my-gcp-project",
		Region:     "us-central1",
	}

	tests := []struct {
		name      string
		svc       compose.ServiceConfig
		infra     *BuildInfra
		repos     map[string]*artifactregistry.Repository
		wantImage string
		wantErr   bool
	}{
		{
			name:    "no image or build returns error",
			svc:     compose.ServiceConfig{},
			wantErr: true,
		},
		{
			name:      "docker.io image returned as-is",
			svc:       compose.ServiceConfig{Image: strPtr("nginx:latest")},
			infra:     fakeInfra,
			wantImage: "nginx:latest",
		},
		{
			name:      "explicit docker.io image returned as-is",
			svc:       compose.ServiceConfig{Image: strPtr("docker.io/library/nginx:1.25")},
			infra:     fakeInfra,
			wantImage: "docker.io/library/nginx:1.25",
		},
		{
			name:      "gcr.io image returned as-is",
			svc:       compose.ServiceConfig{Image: strPtr("gcr.io/my-project/app:v1")},
			infra:     fakeInfra,
			wantImage: "gcr.io/my-project/app:v1",
		},
		{
			name:      "artifact registry image returned as-is",
			svc:       compose.ServiceConfig{Image: strPtr("us-central1-docker.pkg.dev/proj/repo/app:v1")},
			infra:     fakeInfra,
			wantImage: "us-central1-docker.pkg.dev/proj/repo/app:v1",
		},
		{
			name:  "quay.io image rewritten to artifact registry",
			svc:   compose.ServiceConfig{Image: strPtr("quay.io/prometheus/node-exporter:v1.8.0")},
			infra: fakeInfra,
			repos: map[string]*artifactregistry.Repository{
				"quay.io": {Name: pulumi.String("quay-io").ToStringOutput()},
			},
			wantImage: "us-central1-docker.pkg.dev/my-gcp-project/quay-io/prometheus/node-exporter:v1.8.0",
		},
		{
			name:  "ghcr.io image rewritten to artifact registry",
			svc:   compose.ServiceConfig{Image: strPtr("ghcr.io/owner/image:sha-abc123")},
			infra: fakeInfra,
			repos: map[string]*artifactregistry.Repository{
				"ghcr.io": {Name: pulumi.String("ghcr-io").ToStringOutput()},
			},
			wantImage: "us-central1-docker.pkg.dev/my-gcp-project/ghcr-io/owner/image:sha-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotImage string
			var wg sync.WaitGroup
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := GetServiceImage(ctx, "svc", tt.svc, tt.repos, tt.infra)
				if tt.wantErr {
					assert.Error(t, err)
					return nil
				}
				require.NoError(t, err)
				wg.Add(1)
				got.ToStringOutput().ApplyT(func(s string) string {
					defer wg.Done()
					gotImage = s
					return s
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
			wg.Wait()
			if !tt.wantErr {
				assert.Equal(t, tt.wantImage, gotImage)
			}
		})
	}
}

func TestGenerateBuildSteps(t *testing.T) {
	const dest = "us-central1-docker.pkg.dev/my-project/my-repo/app:latest"

	var steps []buildStep
	var unmarshalErr error
	var wg sync.WaitGroup
	wg.Add(1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		out := generateBuildSteps(pulumi.String(dest).ToStringOutput())
		out.ApplyT(func(stepsYAML string) string {
			defer wg.Done()
			unmarshalErr = yaml.Unmarshal([]byte(stepsYAML), &steps)
			return stepsYAML
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
	wg.Wait()
	require.NoError(t, unmarshalErr)
	require.Len(t, steps, 2, "expected exactly 2 build steps")

	// Step 0: buildx create with docker-container driver
	assert.Equal(t, "gcr.io/cloud-builders/docker", steps[0].Name)
	assert.Contains(t, steps[0].Args, "create")
	assert.Contains(t, steps[0].Args, "docker-container")

	// Step 1: buildx build with --load (not --push)
	assert.Equal(t, "gcr.io/cloud-builders/docker", steps[1].Name)
	assert.Contains(t, steps[1].Args, "build")
	assert.Contains(t, steps[1].Args, "--load")
	assert.NotContains(t, steps[1].Args, "--push")
	assert.Contains(t, steps[1].Args, dest)
}

func TestBuildSourceDigest(t *testing.T) {
	ptr := strPtr

	tests := []struct {
		name        string
		build       compose.BuildConfig
		wantChanged *compose.BuildConfig // if set, assert digest differs from this
	}{
		{
			name:  "context only is deterministic",
			build: compose.BuildConfig{Context: pulumi.String("./app")},
		},
		{
			name: "dockerfile is included",
			build: compose.BuildConfig{
				Context:    pulumi.String("./app"),
				Dockerfile: ptr("Dockerfile.prod"),
			},
			wantChanged: &compose.BuildConfig{Context: pulumi.String("./app")},
		},
		{
			name: "target is included",
			build: compose.BuildConfig{
				Context: pulumi.String("./app"),
				Target:  ptr("release"),
			},
			wantChanged: &compose.BuildConfig{Context: pulumi.String("./app")},
		},
		{
			name: "args are included",
			build: compose.BuildConfig{
				Context: pulumi.String("./app"),
				Args:    map[string]string{"ENV": "prod"},
			},
			wantChanged: &compose.BuildConfig{Context: pulumi.String("./app")},
		},
		{
			name: "arg order does not affect digest",
			build: compose.BuildConfig{
				Context: pulumi.String("./app"),
				Args:    map[string]string{"A": "1", "B": "2"},
			},
			// Same args, different insertion order — Go maps are unordered but
			// json.Marshal sorts keys, so both should produce the same digest.
			wantChanged: nil,
		},
		{
			name:        "different context produces different digest",
			build:       compose.BuildConfig{Context: pulumi.String("./other")},
			wantChanged: &compose.BuildConfig{Context: pulumi.String("./app")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d1, d2 string
			var wg sync.WaitGroup
			wg.Add(2)
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				buildSourceDigest(&tt.build).ApplyT(func(s string) string {
					defer wg.Done()
					d1 = s
					return s
				})
				if tt.wantChanged != nil {
					buildSourceDigest(tt.wantChanged).ApplyT(func(s string) string {
						defer wg.Done()
						d2 = s
						return s
					})
				} else {
					// Idempotency: same config called twice yields same digest.
					buildSourceDigest(&tt.build).ApplyT(func(s string) string {
						defer wg.Done()
						d2 = s
						return s
					})
				}
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
			wg.Wait()
			if tt.wantChanged != nil {
				assert.NotEqual(t, d1, d2, "expected digests to differ")
			} else {
				assert.Equal(t, d1, d2, "expected identical digests")
			}
		})
	}
}

// TestBuildServiceImageDependsOnBucketIAMMember verifies that the Build custom
// resource has an explicit Pulumi dependency on the BucketIAMMember that grants
// the build service account read access to the artifacts bucket. Without this
// dependency Cloud Build can be submitted before GCP IAM has propagated the
// binding (~60 s window), causing a 403 on first deploy.
func TestBuildServiceImageDependsOnBucketIAMMember(t *testing.T) {
	spy := &buildDepsSpy{}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		iamMember, err := storage.NewBucketIAMMember(ctx, "artifacts-viewer", &storage.BucketIAMMemberArgs{
			Bucket: pulumi.String("my-artifacts-bucket"),
			Role:   pulumi.String("roles/storage.objectViewer"),
			Member: pulumi.String("serviceAccount:build@proj.iam.gserviceaccount.com"),
		})
		if err != nil {
			return err
		}

		sa, err := serviceaccount.NewAccount(ctx, "build-sa", &serviceaccount.AccountArgs{
			AccountId: pulumi.String("build-sa"),
		})
		if err != nil {
			return err
		}

		infra := &BuildInfra{
			ServiceAccount:  sa,
			BucketIAMMember: iamMember,
			RepositoryURL:   pulumi.String("us-central1-docker.pkg.dev/proj/repo").ToStringOutput(),
			Region:          "us-central1",
			GcpProject:      "proj",
		}

		svc := compose.ServiceConfig{
			Build: &compose.BuildConfig{
				// Use a gs:// URI so resolveSourceURI skips the BucketObject creation
				// and we don't need a real BuildBucket in infra.
				Context: pulumi.String("gs://my-artifacts-bucket/context.zip"),
			},
		}

		_, err = buildServiceImage(ctx, "my-svc", svc, infra)
		return err
	}, pulumi.WithMocks("proj", "stack", spy))

	require.NoError(t, err)

	// The Build resource must list the BucketIAMMember as a dependency so that
	// Pulumi waits for the IAM binding before submitting the Cloud Build job.
	var hasBucketIAMDep bool
	for _, dep := range spy.buildDeps {
		if strings.Contains(dep, "BucketIAMMember") {
			hasBucketIAMDep = true
			break
		}
	}
	assert.True(t, hasBucketIAMDep,
		"Build resource must depend on BucketIAMMember (got deps: %v)", spy.buildDeps)
}
