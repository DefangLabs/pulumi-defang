package gcp

import (
	"sync"
	"testing"

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

func TestParseImage(t *testing.T) {
	tests := []struct {
		input            string
		wantRegistry     string
		wantRepo         string
		wantTag          string
		wantRoundTripped string // fullImage() result; defaults to input if empty
	}{
		{
			input:    "nginx",
			wantRepo: "nginx",
		},
		{
			input:    "nginx:latest",
			wantRepo: "nginx",
			wantTag:  "latest",
		},
		{
			input:        "gcr.io/my-project/myapp:v1",
			wantRegistry: "gcr.io",
			wantRepo:     "my-project/myapp",
			wantTag:      "v1",
		},
		{
			input:        "us-central1-docker.pkg.dev/proj/repo/img:tag",
			wantRegistry: "us-central1-docker.pkg.dev",
			wantRepo:     "proj/repo/img",
			wantTag:      "tag",
		},
		{
			input:        "quay.io/prometheus/node-exporter:v1.8.0",
			wantRegistry: "quay.io",
			wantRepo:     "prometheus/node-exporter",
			wantTag:      "v1.8.0",
		},
		{
			input:        "ghcr.io/owner/image:sha-abc123",
			wantRegistry: "ghcr.io",
			wantRepo:     "owner/image",
			wantTag:      "sha-abc123",
		},
		{
			input:        "docker.io/library/nginx:1.25",
			wantRegistry: "docker.io",
			wantRepo:     "library/nginx",
			wantTag:      "1.25",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseImage(tt.input)
			assert.Equal(t, tt.wantRegistry, got.registry, "registry")
			assert.Equal(t, tt.wantRepo, got.repo, "repo")
			assert.Equal(t, tt.wantTag, got.tag, "tag")
			want := tt.wantRoundTripped
			if want == "" {
				want = tt.input
			}
			assert.Equal(t, want, got.fullImage(), "fullImage round-trip")
		})
	}
}

func TestIsCloudRunSupportedRegistry(t *testing.T) {
	tests := []struct {
		registry string
		want     bool
	}{
		{"", true},           // implicit docker.io
		{"docker.io", true},  // explicit docker.io
		{"gcr.io", true},
		{"us.gcr.io", true},
		{"us-central1.gcr.io", true},
		{"docker.pkg.dev", true},                    // no region prefix
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
			name:      "quay.io image rewritten to artifact registry",
			svc:       compose.ServiceConfig{Image: strPtr("quay.io/prometheus/node-exporter:v1.8.0")},
			infra:     fakeInfra,
			wantImage: "us-central1-docker.pkg.dev/my-gcp-project/quay-io/prometheus/node-exporter:v1.8.0",
		},
		{
			name:      "ghcr.io image rewritten to artifact registry",
			svc:       compose.ServiceConfig{Image: strPtr("ghcr.io/owner/image:sha-abc123")},
			infra:     fakeInfra,
			wantImage: "us-central1-docker.pkg.dev/my-gcp-project/ghcr-io/owner/image:sha-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotImage string
			var wg sync.WaitGroup
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := GetServiceImage(ctx, "svc", tt.svc, tt.infra)
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
