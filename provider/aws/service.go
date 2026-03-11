package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// OneServiceArgs holds shared infra refs + per-service config for creating a single service.
type OneServiceArgs struct {
	Svc              common.ServiceConfig
	Cluster          *ecs.Cluster
	ExecRole         *iam.Role
	LogGroup         *cloudwatch.LogGroup
	VpcID            pulumi.StringOutput
	SubnetIDs        pulumi.StringArrayOutput
	PrivateSubnetIDs pulumi.StringArrayOutput
	SG               *ec2.SecurityGroup
	Listener         *lb.Listener     // nil if no ALB
	ALB              *lb.LoadBalancer // nil if no ALB
	Region           string
	ImgInfra         *imageInfra // nil if no builds needed
}

// OneServiceResult holds the per-service outputs.
type OneServiceResult struct {
	Endpoint   pulumi.StringOutput
	HasIngress bool
}

// CreateOneService creates all resources for a single service (ECS, RDS, or CodeBuild+ECS).
// Shared infra is passed in; per-service resources are created as children.
func CreateOneService(ctx *pulumi.Context, name string, args OneServiceArgs, recipe Recipe, opts ...pulumi.ResourceOption) (*OneServiceResult, error) {
	svc := args.Svc

	if svc.Postgres != nil {
		rdsResult, err := createRDS(ctx, name, svc, args.VpcID, args.PrivateSubnetIDs, args.SG, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating RDS for %s: %w", name, err)
		}
		return &OneServiceResult{
			Endpoint: pulumi.Sprintf("%s:%d", rdsResult.instance.Address, 5432),
		}, nil
	}

	// Resolve container image (build or pre-built)
	imageURI, err := getServiceImage(ctx, name, svc, args.ImgInfra, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", name, err)
	}

	// Create ECS service
	ecsResult, err := createECSService(ctx, name, svc, &ecsServiceArgs{
		cluster:   args.Cluster,
		execRole:  args.ExecRole,
		logGroup:  args.LogGroup,
		vpcID:     args.VpcID,
		subnetIDs: args.SubnetIDs,
		sg:        args.SG,
		listener:  args.Listener,
		alb:       args.ALB,
		region:    args.Region,
		imageURI:  imageURI,
	}, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service %s: %w", name, err)
	}

	if ecsResult.hasIngress {
		return &OneServiceResult{Endpoint: ecsResult.endpoint, HasIngress: true}, nil
	}
	return &OneServiceResult{Endpoint: pulumi.Sprintf("%s (no ingress)", name)}, nil
}
