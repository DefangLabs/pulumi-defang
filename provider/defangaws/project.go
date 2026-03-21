package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Project is the controller struct for the defang-aws:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the AWS Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services map[string]shared.ServiceInput       `pulumi:"services" yaml:"services"`
	Networks map[string]shared.NetworkConfigInput `pulumi:"networks,optional" yaml:"networks,omitempty"`

	// AWS-specific infrastructure configuration (VPC, subnets)
	AWS *shared.AWSConfigInput `pulumi:"aws,optional"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumix.Output[map[string]string] `pulumi:"endpoints"`

	// Load balancer DNS name (AWS ALB)
	LoadBalancerDNS pulumix.Output[*string] `pulumi:"loadBalancerDns,optional"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	args := common.BuildArgs{
		Services: inputs.Services,
	}

	result, err := Build(ctx, name, args, inputs.AWS, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS resources: %w", err)
	}

	comp.Endpoints = pulumix.Output[map[string]string](result.Endpoints)
	comp.LoadBalancerDNS = pulumix.Output[*string](result.LoadBalancerDNS)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
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

	recipe := provideraws.LoadRecipe(ctx)

	// Create CloudWatch log group
	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(recipe.LogRetentionDays),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	// Create execution role (shared by all task definitions)
	execRole, err := provideraws.CreateExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	// Create or use existing VPC and subnets
	net, err := provideraws.ResolveNetworking(ctx, awsCfg, recipe, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}
	vpcID := net.VpcID
	subnetIDs := net.PublicSubnetIDs
	privateSubnetIDs := net.PrivateSubnetIDs
	privateZoneID := net.PrivateZoneID
	privateDomain := net.PrivateDomain

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
		albResult, err := provideraws.CreateALB(ctx, vpcID, subnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albResult.HttpListener
		alb = albResult.Alb
		albDNS = alb.DnsName.ToStringPtrOutput()
	} else {
		albDNS = pulumi.StringPtr("").ToStringPtrOutput()
	}

	// Create shared image build infrastructure if any service needs a build
	var imgInfra *provideraws.ImageInfra
	for _, svc := range args.Services {
		if svc.NeedsBuild() {
			imgInfra, err = provideraws.CreateImageInfra(ctx, logGroup, region.Name, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating image build infrastructure: %w", err)
			}
			break
		}
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → dependency resource for dependees

	configProvider := provideraws.NewConfigProvider(projectName)

	services := common.TopologicalSort(args.Services)
	for _, svcName := range services {
		svc := args.Services[svcName]

		// Collect dependency resources from services this one depends on
		var deps []pulumi.Resource
		for dep := range svc.DependsOn {
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			}
		}

		var dependency pulumi.Resource
		var endpoint pulumi.StringOutput
		var err error

		var privateFqdn string
		if privateDomain != "" {
			privateFqdn = svcName + "." + privateDomain
		}

		switch {
		case svc.Postgres != nil:
			// Managed Postgres → RDS
			var pgResult *PostgresResult
			pgResult, err = newPostgresComponent(ctx, configProvider, svcName, svc, vpcID, privateSubnetIDs, sg, privateZoneID, privateFqdn, recipe, deps, opts[0])
			if pgResult != nil {
				dependency = pgResult.Dependency
				endpoint = pgResult.Endpoint
			}
		case svc.Redis != nil:
			// Managed Redis → ElastiCache
			var redisResult *RedisResult
			redisResult, err = newRedisComponent(ctx, configProvider, svcName, svc, vpcID, privateSubnetIDs, sg, privateZoneID, privateFqdn, recipe, deps, opts[0])
			if redisResult != nil {
				dependency = redisResult.Dependency
				endpoint = redisResult.Endpoint
			}
		default:
			// Container service → ECS
			imageURI, imgErr := provideraws.GetServiceImage(ctx, svcName, svc, imgInfra, opts[0])
			if imgErr != nil {
				return nil, fmt.Errorf("resolving image for %s: %w", svcName, imgErr)
			}
			var ecsResult *ECSResult
			ecsResult, err = NewECSServiceComponent(ctx, configProvider, svcName, svc, &provideraws.ECSServiceArgs{
				Cluster:         cluster,
				ExecRole:        execRole,
				LogGroup:        logGroup,
				VpcID:           vpcID,
				PublicSubnetIDs: subnetIDs,
				Sg:              sg,
				Listener:        httpListener,
				Alb:             alb,
				Region:          region.Name,
				ImageURI:        imageURI,
			}, recipe, deps, opts[0])
			if ecsResult != nil {
				dependency = ecsResult.Dependency
				endpoint = ecsResult.Endpoint
			}
		}
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
		if dependency != nil {
			dependencies[svcName] = dependency
		}
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: albDNS,
	}, nil
}
