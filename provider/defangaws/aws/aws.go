package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

// Build creates all AWS resources for the project.
// The AWS provider must be passed via the parent chain (pulumi.Providers on the parent component).
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, awsCfg *common.AWSConfig, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	opts := []pulumi.ResourceOption{parentOpt}

	// Look up current AWS region from the inherited provider
	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	// Create ECS cluster
	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	recipe := LoadRecipe(ctx)

	// Create CloudWatch log group
	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(recipe.LogRetentionDays),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	// Create execution role (shared by all task definitions)
	execRole, err := createExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	// Create or use existing VPC and subnets
	net, err := ResolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}
	vpcID := net.VpcID
	subnetIDs := net.PublicSubnetIDs
	privateSubnetIDs := net.PrivateSubnetIDs

	// Create security group for services
	sg, err := ec2.NewSecurityGroup(ctx, "svc-sg", &ec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(vpcID),
		Description: pulumi.String(fmt.Sprintf("Security group for %s services", projectName)),
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	// Create ALB if any service has ingress ports
	var albDNS pulumi.StringPtrOutput
	var httpListener *lb.Listener
	needsALB := common.NeedIngress(args.Services)

	var alb *lb.LoadBalancer
	if needsALB {
		albResult, err := createALB(ctx, vpcID, subnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albResult.httpListener
		alb = albResult.alb
		albDNS = alb.DnsName.ToStringPtrOutput()
	} else {
		albDNS = pulumi.StringPtr("").ToStringPtrOutput()
	}

	// Create shared image build infrastructure if any service needs a build
	var imgInfra *imageInfra
	for _, svc := range args.Services {
		if svc.NeedsBuild() {
			imgInfra, err = createImageInfra(ctx, logGroup, region.Name, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating image build infrastructure: %w", err)
			}
			break
		}
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}

	configProvider := NewConfigProvider(projectName)
	for svcName, svc := range args.Services {
		var endpoint pulumi.StringOutput
		var err error

		switch {
		case svc.Postgres != nil:
			// Managed Postgres → RDS
			endpoint, err = newPostgresComponent(ctx, configProvider, svcName, svc, vpcID, privateSubnetIDs, sg, recipe, opts[0])
		case svc.Redis != nil:
			// Managed Redis → ElastiCache
			endpoint, err = newRedisComponent(ctx, configProvider, svcName, svc, vpcID, privateSubnetIDs, sg, recipe, opts[0])
		default:
			// Container service → ECS
			imageURI, imgErr := getServiceImage(ctx, svcName, svc, imgInfra, opts[0])
			if imgErr != nil {
				return nil, fmt.Errorf("resolving image for %s: %w", svcName, imgErr)
			}
			endpoint, err = NewECSServiceComponent(ctx, configProvider, svcName, svc, &ECSServiceArgs{
				Cluster:   cluster,
				ExecRole:  execRole,
				LogGroup:  logGroup,
				VpcID:     vpcID,
				SubnetIDs: subnetIDs,
				Sg:        sg,
				Listener:  httpListener,
				Alb:       alb,
				Region:    region.Name,
				ImageURI:  imageURI,
			}, recipe, opts[0])
		}
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: albDNS,
	}, nil
}

// BuildECSArgs creates all shared AWS infrastructure for a standalone ECS service and
// returns the args to pass to NewECSServiceComponent.
// The AWS provider must be passed via opts (pulumi.Providers on the parent component).
func BuildECSArgs(ctx *pulumi.Context, serviceName string, svc shared.ServiceInput, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*ECSServiceArgs, error) {
	recipe := LoadRecipe(ctx)

	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := ResolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(net.VpcID),
		Description: pulumi.String("Security group for services"),
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(recipe.LogRetentionDays),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := createExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	var imgInfra *imageInfra
	if svc.NeedsBuild() {
		imgInfra, err = createImageInfra(ctx, logGroup, region.Name, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating image build infrastructure: %w", err)
		}
	}

	var httpListener *lb.Listener
	var svcALB *lb.LoadBalancer
	if svc.HasIngressPorts() {
		albRes, err := createALB(ctx, net.VpcID, net.PublicSubnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albRes.httpListener
		svcALB = albRes.alb
	}

	imageURI, err := getServiceImage(ctx, serviceName, svc, imgInfra, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", serviceName, err)
	}

	return &ECSServiceArgs{
		Cluster:   cluster,
		ExecRole:  execRole,
		LogGroup:  logGroup,
		VpcID:     net.VpcID,
		SubnetIDs: net.PublicSubnetIDs,
		Sg:        sg,
		Listener:  httpListener,
		Alb:       svcALB,
		Region:    region.Name,
		ImageURI:  imageURI,
	}, nil
}

