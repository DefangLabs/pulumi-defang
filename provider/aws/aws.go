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

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

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
		if err := ctx.RegisterComponentResource("defang:index:AwsService", svcName, comp, opts[0]); err != nil {
			return nil, fmt.Errorf("registering service component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(comp), pulumi.Provider(awsProv)}

		result, err := CreateOneService(ctx, svcName, OneServiceArgs{
			Svc:              svc,
			Cluster:          cluster,
			ExecRole:         execRole,
			LogGroup:         logGroup,
			VpcID:            vpcID,
			SubnetIDs:        subnetIDs,
			PrivateSubnetIDs: privateSubnetIDs,
			SG:               sg,
			Listener:         httpListener,
			ALB:              alb,
			Region:           region.Name,
			ImgInfra:         imgInfra,
		}, recipe, svcOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating service %s: %w", svcName, err)
		}

		endpoints[svcName] = result.Endpoint

		if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
			"endpoint": result.Endpoint,
		}); err != nil {
			return nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: albDNS,
	}, nil
}

// BuildStandalone creates all AWS resources for a single standalone service.
// Creates its own shared infra (cluster, VPC, ALB, etc.) then calls CreateOneService.
func BuildStandalone(ctx *pulumi.Context, serviceName string, svc common.ServiceConfig, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*OneServiceResult, error) {
	recipe := LoadRecipe(ctx)

	// Create explicit AWS provider
	awsProvArgs := &aws.ProviderArgs{
		DefaultTags: &aws.ProviderDefaultTagsArgs{
			Tags: pulumi.StringMap{
				"defang:project": pulumi.String(serviceName),
				"defang:stack":   pulumi.String(ctx.Stack()),
			},
		},
	}
	if awsCfg != nil && awsCfg.Region != "" {
		awsProvArgs.Region = pulumi.String(awsCfg.Region)
	}
	awsProv, err := aws.NewProvider(ctx, "aws", awsProvArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating AWS provider: %w", err)
	}
	provOpts := append(opts, pulumi.Provider(awsProv))

	region, err := aws.GetRegion(ctx, nil, pulumi.Provider(awsProv))
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	// Create or use existing VPC and subnets
	net, err := resolveNetworking(ctx, awsCfg, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	// Create security group
	sg, err := ec2.NewSecurityGroup(ctx, "sg", &ec2.SecurityGroupArgs{
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
	}, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	// For postgres, we don't need cluster/ALB
	if svc.Postgres != nil {
		rdsResult, err := createRDS(ctx, serviceName, svc, net.vpcID, net.privateSubnetIDs, sg, recipe, provOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating RDS: %w", err)
		}
		return &OneServiceResult{
			Endpoint: pulumi.Sprintf("%s:%d", rdsResult.instance.Address, 5432),
		}, nil
	}

	// Create ECS cluster, log group, execution role
	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(recipe.LogRetentionDays),
	}, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := createExecutionRole(ctx, provOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	// Create image build infra if needed
	var imgInfra *imageInfra
	if svc.NeedsBuild() {
		imgInfra, err = createImageInfra(ctx, logGroup, region.Name, provOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating image build infrastructure: %w", err)
		}
	}

	// Create ALB if service has ingress ports
	var httpListener *lb.Listener
	var svcALB *lb.LoadBalancer
	if svc.HasIngressPorts() {
		albResult, err := createALB(ctx, net.vpcID, net.publicSubnetIDs, sg, recipe, provOpts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albResult.httpListener
		svcALB = albResult.alb
	}

	return CreateOneService(ctx, serviceName, OneServiceArgs{
		Svc:              svc,
		Cluster:          cluster,
		ExecRole:         execRole,
		LogGroup:         logGroup,
		VpcID:            net.vpcID,
		SubnetIDs:        net.publicSubnetIDs,
		PrivateSubnetIDs: net.privateSubnetIDs,
		SG:               sg,
		Listener:         httpListener,
		ALB:              svcALB,
		Region:           region.Name,
		ImgInfra:         imgInfra,
	}, recipe, provOpts...)
}
