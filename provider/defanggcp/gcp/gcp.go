package gcp

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// GlobalConfig holds project-level GCP resources shared across all services.
type GlobalConfig struct {
	Region string
}

// BuildGlobalConfig creates shared GCP infrastructure for a multi-service project.
func BuildGlobalConfig(
	ctx *pulumi.Context,
	projectName string,
	services map[string]compose.ServiceConfig,
	opts ...pulumi.ResourceOption,
) (*GlobalConfig, error) {
	region := GcpRegion(ctx)

	return &GlobalConfig{
		Region: region,
	}, nil
}

const defaultGCPRegion = "us-central1"

// GcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func GcpRegion(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	if r := cfg.Get("region"); r != "" {
		return r
	}
	return defaultGCPRegion
}
