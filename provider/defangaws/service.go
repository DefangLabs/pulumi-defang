package defangaws

import (
	"fmt"
	"sort"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	awsroute53 "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Service is the controller struct for the defang-aws:index:Service component.
type Service struct{}

// ServiceInputs defines the inputs for a standalone AWS ECS service.
// Scalar fields use pulumi.String / *pulumi.String so the generated SDK
// wraps them in pulumi.Input (matching the Node.js SDK behaviour).
type ServiceInputs struct {
	Image       *string                     `pulumi:"image,optional"`
	Platform    *string                     `pulumi:"platform,optional"`
	ProjectName string                      `pulumi:"project_name"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]string           `pulumi:"environment,optional"`
	Command     []string                    `pulumi:"command,optional"`
	Entrypoint  []string                    `pulumi:"entrypoint,optional"`
	HealthCheck *compose.HealthCheckConfig  `pulumi:"healthCheck,optional"`
	DomainName  string                      `pulumi:"domainName,optional"`

	AWS *provideraws.SharedInfra `pulumi:"aws,optional"`
}

// ServiceOutputs holds the outputs of a Service component.
type ServiceOutputs struct {
	pulumi.ResourceState
	Endpoint pulumix.Output[string] `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for Service.
func (*Service) Construct(
	ctx *pulumi.Context, name, typ string, inputs ServiceInputs, opts pulumi.ResourceOption,
) (*ServiceOutputs, error) {
	comp := &ServiceOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := compose.ServiceConfig{
		Image:       inputs.Image,
		Platform:    inputs.Platform,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
		Command:     inputs.Command,
		Entrypoint:  inputs.Entrypoint,
		HealthCheck: inputs.HealthCheck,
		DomainName:  inputs.DomainName,
	}

	configProvider := provideraws.NewConfigProvider(inputs.ProjectName)
	infra := inputs.AWS
	// infra, err := provideraws.BuildSharedInfra(ctx, name, svc, inputs.AWS, childOpt)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to build AWS ECS infrastructure: %w", err)
	// }

	var buildInfra *provideraws.BuildInfra
	if infra != nil {
		buildInfra = infra.BuildInfra
	}
	imageURI, err := provideraws.GetServiceImage(ctx, name, svc, buildInfra, childOpt)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", name, err)
	}

	ecsResult, err := NewECSServiceComponent(ctx, configProvider, name, svc, &provideraws.ECSServiceArgs{
		Infra:    infra,
		ImageURI: imageURI,
	}, nil, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECS service: %w", err)
	}

	endpoint := ecsResult.Endpoint
	comp.Endpoint = pulumix.Output[string](endpoint)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// serviceComponent is a local component resource used to group per-service resources in the tree.
type serviceComponent struct {
	pulumi.ResourceState
}

type ECSResult struct {
	Endpoint   pulumi.StringOutput
	Dependency pulumi.Resource // the ECS service, for dependees
}

// NewECSServiceComponent registers a component resource for a container service,
// creates its ECS children, registers outputs, and returns the endpoint.
func NewECSServiceComponent(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	args *provideraws.ECSServiceArgs,
	deps []pulumi.Resource,
	parentOpt pulumi.ResourceOption,
) (*ECSResult, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:Service", serviceName, comp, parentOpt); err != nil {
		return nil, fmt.Errorf("registering ECS service component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	ecsResult, err := provideraws.CreateECSService(ctx, configProvider, serviceName, svc, args, deps, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service %s: %w", serviceName, err)
	}

	// Create per-service BYOD certificates as children of this component
	if err := createServiceBYODCerts(ctx, serviceName, svc, args, opts); err != nil {
		return nil, err
	}

	endpoint := pulumi.StringOutput(ecsResult.Endpoint)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return nil, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return &ECSResult{
		Endpoint:   endpoint,
		Dependency: ecsResult.Service,
	}, nil
}

// createServiceBYODCerts creates BYOD (Bring Your Own Domain) ACM certificates,
// listener attachments, and DNS records as children of the service component.
// Hostnames that share the same DNS validation record are grouped onto one cert
// (e.g. example.com and *.example.com).
func createServiceBYODCerts(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	args *provideraws.ECSServiceArgs,
	opts []pulumi.ResourceOption,
) error {
	infra := args.Infra
	if infra == nil || infra.HttpsListener == nil || infra.Alb == nil {
		return nil
	}
	if svc.DomainName == "" {
		return nil
	}

	// Collect all BYOD hostnames: domainname + network aliases
	hostnames := append([]string{svc.DomainName}, svc.DefaultNetwork().Aliases...)

	// Group hostnames that share the same validation record onto one cert
	certGroups := provideraws.GroupHostnamesByCert(hostnames)

	albAliases := awsroute53.RecordAliasArray{
		&awsroute53.RecordAliasArgs{
			EvaluateTargetHealth: pulumi.Bool(true),
			Name:                 infra.Alb.DnsName,
			ZoneId:               infra.Alb.ZoneId,
		},
	}

	// Iterate groups in sorted order for deterministic resource names
	bases := make([]string, 0, len(certGroups))
	for base := range certGroups {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	for _, base := range bases {
		group := certGroups[base]

		// Look up the Route53 hosted zone for this domain
		zone, err := provideraws.GetHostedZoneForHost(ctx, base)
		if err != nil {
			return fmt.Errorf("finding hosted zone for %s: %w", base, err)
		}

		certArn, err := provideraws.CreateCertificateDNS(ctx, group, provideraws.CertificateDnsArgs{
			CaaIssuer: []string{"amazon.com", "letsencrypt.org"},
			ZoneId:    pulumi.String(zone.Id),
			Tags: pulumi.StringMap{
				"defang:service": pulumi.String(serviceName),
			},
		}, common.MergeOptions(opts,
			pulumi.RetainOnDelete(true),
		)...)
		if err != nil {
			return fmt.Errorf("creating BYOD certificate for %s: %w", base, err)
		}

		_, err = lb.NewListenerCertificate(ctx, serviceName+"-"+base+"-cert", &lb.ListenerCertificateArgs{
			ListenerArn:    infra.HttpsListener.Arn,
			CertificateArn: certArn,
		}, opts...)
		if err != nil {
			return fmt.Errorf("attaching BYOD certificate for %s: %w", base, err)
		}

		// Create ALIAS DNS records for each hostname in this group → ALB
		zoneId := pulumi.String(zone.Id)
		for _, hostname := range group {
			_, err = provideraws.CreateRecord(ctx, hostname, common.RecordTypeA, &awsroute53.RecordArgs{
				Aliases: albAliases,
				ZoneId:  zoneId,
			}, opts...)
			if err != nil {
				return fmt.Errorf("creating DNS record for %s: %w", hostname, err)
			}
		}
	}
	return nil
}
