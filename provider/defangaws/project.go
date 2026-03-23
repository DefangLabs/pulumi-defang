package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Project is the controller struct for the defang-aws:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the AWS Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services map[string]compose.ServiceConfig      `pulumi:"services"          yaml:"services"`
	Networks map[string]compose.NetworkConfigInput `pulumi:"networks,optional" yaml:"networks,omitempty"`

	// AWS-specific infrastructure configuration (VPC, subnets)
	AWS *compose.AWSConfigInput `pulumi:"aws,optional"`
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
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
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
func Build(
	ctx *pulumi.Context,
	projectName string,
	args common.BuildArgs,
	awsCfg *common.AWSConfig,
	parentOpt pulumi.ResourceOption,
) (*common.BuildResult, error) {
	opts := []pulumi.ResourceOption{parentOpt}

	infra, err := provideraws.BuildProjectInfra(ctx, projectName, args.Services, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating shared infrastructure: %w", err)
	}

	var albDNS pulumi.StringPtrOutput
	if infra.Alb != nil {
		albDNS = infra.Alb.DnsName.ToStringPtrOutput()
	} else {
		albDNS = pulumi.StringPtr("").ToStringPtrOutput()
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
		endpoint, dependency, err = buildService(ctx, configProvider, svcName, svc, infra, deps, opts[0])
		if err != nil {
			return nil, fmt.Errorf("building service %s: %w", svcName, err)
		}

		// Create private DNS CNAME for managed services (Postgres, Redis)
		if dependency != nil && infra.PrivateDomain != "" {
			privateFqdn := svcName + "." + infra.PrivateDomain
			record, cnameErr := provideraws.CreateRecord(ctx, privateFqdn, provideraws.RecordTypeCNAME, route53.RecordArgs{
				ZoneId:  infra.PrivateZoneID.ToIDOutput().ToStringOutput(),
				Records: pulumi.StringArray{endpoint},
				Ttl:     pulumi.Int(300),
			}, pulumi.DependsOn([]pulumi.Resource{dependency}), opts[0])
			if cnameErr != nil {
				return nil, fmt.Errorf("creating CNAME for %s: %w", svcName, cnameErr)
			}
			dependency = record
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

func buildService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	infra *provideraws.SharedInfra,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOption,
) (pulumi.StringOutput, pulumi.Resource, error) {
	var endpoint pulumi.StringOutput
	var dependency pulumi.Resource
	var err error
	switch {
	case svc.Postgres != nil:
		// Managed Postgres → RDS
		var pgResult *PostgresResult
		pgResult, err = newPostgresComponent(ctx, configProvider, svcName, svc, infra, deps, parentOpt)
		if pgResult != nil {
			dependency = pgResult.Dependency
			endpoint = pgResult.Endpoint
		}
	case svc.Redis != nil:
		// Managed Redis → ElastiCache
		var redisResult *RedisResult
		redisResult, err = newRedisComponent(ctx, configProvider, svcName, svc, infra, deps, parentOpt)
		if redisResult != nil {
			dependency = redisResult.Dependency
			endpoint = redisResult.Endpoint
		}
	default:
		// TODO: detect sidecar services (network_mode: "service:<name>", volumes_from)
		// and add them as additional containers in the parent's task definition
		// instead of creating a separate ECS service.

		// Container service → ECS
		imageURI, imgErr := provideraws.GetServiceImage(ctx, svcName, svc, infra.ImageInfra, parentOpt)
		if imgErr != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("resolving image for %s: %w", svcName, imgErr)
		}
		var ecsResult *ECSResult
		ecsResult, err = NewECSServiceComponent(ctx, configProvider, svcName, svc, &provideraws.ECSServiceArgs{
			Infra:    infra,
			ImageURI: imageURI,
		}, deps, parentOpt)
		if ecsResult != nil {
			dependency = ecsResult.Dependency
			endpoint = ecsResult.Endpoint
		}
	}
	if err != nil {
		return pulumi.StringOutput{}, nil, err
	}
	return endpoint, dependency, nil
}
