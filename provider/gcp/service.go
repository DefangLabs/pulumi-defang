package gcp

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// OneServiceResult holds the per-service outputs.
type OneServiceResult struct {
	Endpoint pulumi.StringOutput
}

// CreateOneService creates all resources for a single GCP service (Cloud Run or Cloud SQL).
func CreateOneService(ctx *pulumi.Context, name string, svc common.ServiceConfig, recipe Recipe, opts ...pulumi.ResourceOption) (*OneServiceResult, error) {
	if svc.Postgres != nil {
		sqlResult, err := createCloudSQL(ctx, name, svc, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating Cloud SQL for %s: %w", name, err)
		}
		return &OneServiceResult{
			Endpoint: pulumi.Sprintf("%s:5432", sqlResult.instance.PublicIpAddress),
		}, nil
	}

	crResult, err := createCloudRunService(ctx, name, svc, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run service %s: %w", name, err)
	}
	return &OneServiceResult{
		Endpoint: crResult.service.Uri,
	}, nil
}
