package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Build creates all AWS resources for the project.
func Build(ctx *pulumi.Context, projectName string, args common.BuildArgs, parentOpt pulumi.ResourceOption) (*common.BuildResult, error) {
	// Create explicit AWS provider to pin the version used by all child resources
	awsProvArgs := &aws.ProviderArgs{
		DefaultTags: &aws.ProviderDefaultTagsArgs{
			Tags: pulumi.StringMap{
				"defang:project": pulumi.String(projectName),
				"defang:stack":   pulumi.String(ctx.Stack()),
			},
		},
	}
	if args.AWS != nil && args.AWS.Region != "" {
		awsProvArgs.Region = pulumi.String(args.AWS.Region)
	}
	awsProv, err := aws.NewProvider(ctx, "aws", awsProvArgs, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating AWS provider: %w", err)
	}
	opts := []pulumi.ResourceOption{parentOpt, pulumi.Provider(awsProv)}

	// Look up current AWS region
	region, err := aws.GetRegion(ctx, nil, pulumi.Provider(awsProv))
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	// Create ECS cluster
	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	recipe := args.AWSRecipe

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
	net, err := resolveNetworking(ctx, args.AWS, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}
	vpcID := net.vpcID
	subnetIDs := net.publicSubnetIDs
	privateSubnetIDs := net.privateSubnetIDs

	// Create security group for services
	sg, err := ec2.NewSecurityGroup(ctx, "sg", &ec2.SecurityGroupArgs{
		VpcId:       vpcID,
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

	// Deploy each service
	endpoints := pulumi.StringMap{}

	for svcName, svc := range args.Services {
		if svc.Postgres != nil {
			// Create managed Postgres (RDS)
			rdsResult, err := createRDS(ctx, svcName, svc, vpcID, privateSubnetIDs, sg, recipe, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating RDS for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:%d", rdsResult.instance.Address, 5432)
		} else {
			// Create ECS service
			ecsResult, err := createECSService(ctx, svcName, svc, &ecsServiceArgs{
				cluster:   cluster,
				execRole:  execRole,
				logGroup:  logGroup,
				vpcID:     vpcID,
				subnetIDs: subnetIDs,
				sg:        sg,
				listener:  httpListener,
				alb:       alb,
				region:    region.Name,
			}, recipe, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating ECS service %s: %w", svcName, err)
			}
			if ecsResult.hasIngress {
				endpoints[svcName] = ecsResult.endpoint
			} else {
				endpoints[svcName] = pulumi.Sprintf("%s (no ingress)", svcName)
			}
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: albDNS,
	}, nil
}

// BuildService creates AWS resources for a single standalone service.
func BuildService(ctx *pulumi.Context, serviceName string, args common.ServiceBuildArgs, parentOpt pulumi.ResourceOption) (*common.ServiceBuildResult, error) {
	svc := args.Service
	recipe := args.AWSRecipe

	// Create explicit AWS provider to pin the version used by all child resources
	awsProvArgs := &aws.ProviderArgs{
		DefaultTags: &aws.ProviderDefaultTagsArgs{
			Tags: pulumi.StringMap{
				"defang:project": pulumi.String(serviceName),
				"defang:stack":   pulumi.String(ctx.Stack()),
			},
		},
	}
	if args.AWS != nil && args.AWS.Region != "" {
		awsProvArgs.Region = pulumi.String(args.AWS.Region)
	}
	awsProv, err := aws.NewProvider(ctx, "aws", awsProvArgs, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating AWS provider: %w", err)
	}
	opts := []pulumi.ResourceOption{parentOpt, pulumi.Provider(awsProv)}

	region, err := aws.GetRegion(ctx, nil, pulumi.Provider(awsProv))
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	// Create or use existing VPC and subnets
	net, err := resolveNetworking(ctx, args.AWS, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}
	vpcID := net.vpcID
	subnetIDs := net.publicSubnetIDs
	privateSubnetIDs := net.privateSubnetIDs

	// Create security group
	sg, err := ec2.NewSecurityGroup(ctx, "sg", &ec2.SecurityGroupArgs{
		VpcId:       vpcID,
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

	if svc.Postgres != nil {
		rdsResult, err := createRDS(ctx, serviceName, svc, vpcID, privateSubnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating RDS: %w", err)
		}
		return &common.ServiceBuildResult{
			Endpoint: pulumi.Sprintf("%s:%d", rdsResult.instance.Address, 5432),
		}, nil
	}

	// Create ECS cluster, log group, execution role for standalone service
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

	// Create ALB if service has ingress ports
	var httpListener *lb.Listener
	var svcALB *lb.LoadBalancer
	if svc.HasIngressPorts() {
		albResult, err := createALB(ctx, vpcID, subnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albResult.httpListener
		svcALB = albResult.alb
	}

	ecsResult, err := createECSService(ctx, serviceName, svc, &ecsServiceArgs{
		cluster:   cluster,
		execRole:  execRole,
		logGroup:  logGroup,
		vpcID:     vpcID,
		subnetIDs: subnetIDs,
		sg:        sg,
		listener:  httpListener,
		alb:       svcALB,
		region:    region.Name,
	}, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service: %w", err)
	}

	var endpoint pulumi.StringOutput
	if ecsResult.hasIngress {
		endpoint = ecsResult.endpoint
	} else {
		endpoint = pulumi.Sprintf("%s (no ingress)", serviceName)
	}

	return &common.ServiceBuildResult{
		Endpoint: endpoint,
	}, nil
}
