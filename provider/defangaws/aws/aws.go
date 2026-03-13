package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
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

// EcsServiceResult holds the per-service outputs for an ECS service.
type EcsServiceResult struct {
	Endpoint   pulumi.StringOutput
	HasIngress bool
}

// PostgresResult holds the per-service outputs for an RDS Postgres instance.
type PostgresResult struct {
	Endpoint pulumi.StringOutput
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
	net, err := resolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}
	vpcID := net.vpcID
	subnetIDs := net.publicSubnetIDs
	privateSubnetIDs := net.privateSubnetIDs

	// Create security group for services
	sg, err := ec2.NewSecurityGroup(ctx, "svc-sg", &ec2.SecurityGroupArgs{
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

	for svcName, svc := range args.Services {
		comp := &serviceComponent{}

		if svc.Postgres != nil {
			// Managed Postgres → RDS
			if err := ctx.RegisterComponentResource("defang-aws:index:AwsPostgres", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering postgres component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			rdsResult, err := createRDS(ctx, svcName, svc, vpcID, privateSubnetIDs, sg, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating RDS for %s: %w", svcName, err)
			}
			endpoints[svcName] = pulumi.Sprintf("%s:%d", rdsResult.instance.Address, 5432)
		} else {
			// Container service → ECS
			if err := ctx.RegisterComponentResource("defang-aws:index:AwsEcsService", svcName, comp, opts[0]); err != nil {
				return nil, fmt.Errorf("registering ECS service component %s: %w", svcName, err)
			}
			svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}

			imageURI, err := getServiceImage(ctx, svcName, svc, imgInfra, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("resolving image for %s: %w", svcName, err)
			}

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
				imageURI:  imageURI,
			}, recipe, svcOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating ECS service %s: %w", svcName, err)
			}

			if ecsResult.hasIngress {
				endpoints[svcName] = ecsResult.endpoint
			} else {
				endpoints[svcName] = pulumi.Sprintf("%s (no ingress)", svcName)
			}
		}

		if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
			"endpoint": endpoints[svcName],
		}); err != nil {
			return nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: albDNS,
	}, nil
}

// BuildStandaloneECS creates all AWS resources for a single standalone ECS service.
// The AWS provider must be passed via opts (pulumi.Providers on the parent component).
func BuildStandaloneECS(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*EcsServiceResult, error) {
	recipe := LoadRecipe(ctx)

	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := resolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       net.vpcID,
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
		albResult, err := createALB(ctx, net.vpcID, net.publicSubnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albResult.httpListener
		svcALB = albResult.alb
	}

	imageURI, err := getServiceImage(ctx, serviceName, svc, imgInfra, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", serviceName, err)
	}

	ecsResult, err := createECSService(ctx, serviceName, svc, &ecsServiceArgs{
		cluster:   cluster,
		execRole:  execRole,
		logGroup:  logGroup,
		vpcID:     net.vpcID,
		subnetIDs: net.publicSubnetIDs,
		sg:        sg,
		listener:  httpListener,
		alb:       svcALB,
		region:    region.Name,
		imageURI:  imageURI,
	}, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service %s: %w", serviceName, err)
	}

	endpoint := ecsResult.endpoint
	if !ecsResult.hasIngress {
		endpoint = pulumi.Sprintf("%s (no ingress)", serviceName)
	}

	return &EcsServiceResult{
		Endpoint:   endpoint,
		HasIngress: ecsResult.hasIngress,
	}, nil
}

// BuildStandalonePostgres creates AWS resources for a standalone RDS Postgres instance.
// The AWS provider must be passed via opts (pulumi.Providers on the parent component).
func BuildStandalonePostgres(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*PostgresResult, error) {
	recipe := LoadRecipe(ctx)

	net, err := resolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       net.vpcID,
		Description: pulumi.String("Security group for Postgres"),
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

	rdsResult, err := createRDS(ctx, serviceName, svc, net.vpcID, net.privateSubnetIDs, sg, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating RDS: %w", err)
	}

	return &PostgresResult{
		Endpoint: pulumi.Sprintf("%s:%d", rdsResult.instance.Address, 5432),
	}, nil
}
