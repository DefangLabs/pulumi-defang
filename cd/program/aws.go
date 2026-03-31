package program

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	awscompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Version is set by main to the build version string.
var Version = "development"

func deployAWS(ctx *pulumi.Context, cf *compose.Project, domain string) (pulumi.StringMapOutput, pulumi.StringPtrOutput, error) {
	cfg := config.New(ctx, "aws")

	providerArgs := &aws.ProviderArgs{
		Region: pulumi.StringPtr(cfg.Require("region")),
		DefaultTags: &aws.ProviderDefaultTagsArgs{
			Tags: pulumi.StringMap{
				"defang:org":     pulumi.String(ctx.Organization()),
				"defang:project": pulumi.String(ctx.Project()),
				"defang:stack":   pulumi.String(ctx.Stack()),
				"defang:version": pulumi.String(Version),
			},
		},
	}
	if profile := cfg.Get("profile"); profile != "" {
		providerArgs.Profile = pulumi.StringPtr(profile)
	}

	awsProvider, err := aws.NewProvider(ctx, "aws", providerArgs)
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	args := toAWSArgs(cf)
	if domain != "" {
		args.Aws = defangaws.AWSConfigArgs{
			ProjectDomain: pulumi.StringPtr(domain),
		}
	}

	project, err := defangaws.NewProject(ctx, ctx.Project(), args, pulumi.Providers(awsProvider))
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}
	return project.Endpoints, project.LoadBalancerDns, nil
}

func toAWSArgs(cf *compose.Project) *defangaws.ProjectArgs {
	args := &defangaws.ProjectArgs{
		Services: toAWSServices(cf.Services),
	}
	if len(cf.Networks) > 0 {
		nm := make(awscompose.NetworkConfigMap, len(cf.Networks))
		for k, v := range cf.Networks {
			nm[string(k)] = awscompose.NetworkConfigArgs{Internal: pulumi.Bool(v.Internal)}
		}
		args.Networks = nm
	}
	return args
}

func toAWSServices(services compose.Services) awscompose.ServiceConfigMap {
	m := make(awscompose.ServiceConfigMap, len(services))
	for name, svc := range services {
		m[name] = toAWSServiceArgs(svc)
	}
	return m
}

func toAWSServiceArgs(svc compose.ServiceConfig) awscompose.ServiceConfigArgs {
	args := awscompose.ServiceConfigArgs{
		Image:       pulumi.StringPtrFromPtr(svc.Image),
		Platform:    pulumi.StringPtrFromPtr(svc.Platform),
		Environment: pulumi.ToStringMap(svc.Environment),
		Command:     pulumi.ToStringArray(svc.Command),
		Entrypoint:  pulumi.ToStringArray(svc.Entrypoint),
	}
	if svc.DomainName != "" {
		args.DomainName = pulumi.StringPtr(svc.DomainName)
	}
	if svc.Build != nil {
		args.Build = awscompose.BuildConfigArgs{
			Context:    svc.Build.Context,
			Dockerfile: pulumi.StringPtrFromPtr(svc.Build.Dockerfile),
			Args:       pulumi.ToStringMap(svc.Build.Args),
			ShmSize:    pulumi.StringPtrFromPtr(svc.Build.ShmSize),
			Target:     pulumi.StringPtrFromPtr(svc.Build.Target),
		}
	}
	if len(svc.Ports) > 0 {
		ports := make(awscompose.ServicePortConfigArray, 0, len(svc.Ports))
		for _, p := range svc.Ports {
			ports = append(ports, awscompose.ServicePortConfigArgs{
				Target:      pulumi.Int(int(p.Target)),
				Mode:        pulumi.String(p.Mode),
				Protocol:    pulumi.String(p.Protocol),
				AppProtocol: pulumi.String(p.AppProtocol),
			})
		}
		args.Ports = ports
	}
	if svc.Deploy != nil {
		da := awscompose.DeployConfigArgs{}
		if svc.Deploy.Replicas != nil {
			da.Replicas = pulumi.IntPtr(int(*svc.Deploy.Replicas))
		}
		if svc.Deploy.Resources != nil && svc.Deploy.Resources.Reservations != nil {
			r := svc.Deploy.Resources.Reservations
			ra := awscompose.ResourceConfigArgs{
				Cpus:   pulumi.Float64PtrFromPtr(r.CPUs),
				Memory: pulumi.StringPtrFromPtr(r.Memory),
			}
			da.Resources = awscompose.ResourcesArgs{Reservations: ra}
		}
		args.Deploy = da
	}
	if svc.Postgres != nil {
		args.Postgres = awscompose.PostgresConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Postgres.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Postgres.FromSnapshot),
		}
	}
	if svc.Redis != nil {
		args.Redis = awscompose.RedisConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Redis.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Redis.FromSnapshot),
		}
	}
	if svc.HealthCheck != nil {
		args.HealthCheck = awscompose.HealthCheckConfigArgs{
			Test:               pulumi.ToStringArray(svc.HealthCheck.Test),
			IntervalSeconds:    pulumi.IntPtr(int(svc.HealthCheck.IntervalSeconds)),
			TimeoutSeconds:     pulumi.IntPtr(int(svc.HealthCheck.TimeoutSeconds)),
			Retries:            pulumi.IntPtr(int(svc.HealthCheck.Retries)),
			StartPeriodSeconds: pulumi.IntPtr(int(svc.HealthCheck.StartPeriodSeconds)),
		}
	}
	if len(svc.Networks) > 0 {
		nm := make(awscompose.ServiceNetworkConfigMap, len(svc.Networks))
		for k, v := range svc.Networks {
			nm[string(k)] = awscompose.ServiceNetworkConfigArgs{Aliases: pulumi.ToStringArray(v.Aliases)}
		}
		args.Networks = nm
	}
	if len(svc.DependsOn) > 0 {
		dm := make(awscompose.ServiceDependencyMap, len(svc.DependsOn))
		for k, v := range svc.DependsOn {
			dm[k] = awscompose.ServiceDependencyArgs{Condition: pulumi.String(v.Condition), Required: pulumi.Bool(v.Required)}
		}
		args.DependsOn = dm
	}
	if svc.LLM != nil {
		args.Llm = awscompose.LlmConfigArgs{}
	}
	return args
}
