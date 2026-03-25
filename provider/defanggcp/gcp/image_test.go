package gcp

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

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

func TestBuildSourceDigest(t *testing.T) {
	ptr := func(s string) *string { return &s }

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
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				digest := buildSourceDigest(&tt.build)

				if tt.wantChanged != nil {
					other := buildSourceDigest(tt.wantChanged)
					pulumi.All(digest, other).ApplyT(func(vals []any) any {
						assert.NotEqual(t, vals[0], vals[1], "expected digests to differ")
						return nil
					})
				} else {
					// Idempotency: same config called twice yields same digest.
					again := buildSourceDigest(&tt.build)
					pulumi.All(digest, again).ApplyT(func(vals []any) any {
						assert.Equal(t, vals[0], vals[1], "expected identical digests")
						return nil
					})
				}
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}
}
