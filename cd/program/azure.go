package program

import (
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	defangazure "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure"
	azurecompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-azure/compose"
	pulumiazure "github.com/pulumi/pulumi-azure-native-sdk/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func deployAzure(ctx *pulumi.Context, cf *compose.Project, projectPb []byte) (pulumi.StringMapOutput, pulumi.StringPtrOutput, error) {
	cfg := config.New(ctx, "azure-native")

	providerArgs := &pulumiazure.ProviderArgs{
		Location:                  pulumi.StringPtr(cfg.Require("location")),
		UseDefaultAzureCredential: pulumi.BoolPtr(true),
	}
	if subID := cfg.Get("subscriptionId"); subID != "" {
		providerArgs.SubscriptionId = pulumi.StringPtr(subID)
	}
	azureProvider, err := pulumiazure.NewProvider(ctx, "azure", providerArgs)
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	project, err := defangazure.NewProject(ctx, cf.Name, toAzureArgs(cf), pulumi.Providers(azureProvider))
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	// Upload ProjectUpdate protobuf as a Pulumi-managed Azure Blob, gated on
	// the project component so it only runs after all services are created.
	if len(projectPb) > 0 {
		if err := saveProjectPbAzure(ctx, projectPb, project); err != nil {
			return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
		}
	}
	return project.Endpoints, project.LoadBalancerDns, nil
}

func toAzureArgs(cf *compose.Project) *defangazure.ProjectArgs {
	args := &defangazure.ProjectArgs{
		Services: toAzureServices(cf.Services),
	}
	if len(cf.Networks) > 0 {
		nm := make(azurecompose.NetworkConfigMap, len(cf.Networks))
		for k, v := range cf.Networks {
			nm[string(k)] = azurecompose.NetworkConfigArgs{Internal: pulumi.Bool(v.Internal)}
		}
		args.Networks = nm
	}
	return args
}

func toAzureServices(services compose.Services) azurecompose.ServiceConfigMap {
	m := make(azurecompose.ServiceConfigMap, len(services))
	for name, svc := range services {
		m[name] = toAzureServiceArgs(svc)
	}
	return m
}

func toAzureServiceArgs(svc compose.ServiceConfig) azurecompose.ServiceConfigArgs {
	args := azurecompose.ServiceConfigArgs{
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
		args.Build = azurecompose.BuildConfigArgs{
			Context:    svc.Build.Context,
			Dockerfile: pulumi.StringPtrFromPtr(svc.Build.Dockerfile),
			Args:       pulumi.ToStringMap(svc.Build.Args),
			ShmSize:    pulumi.StringPtrFromPtr(svc.Build.ShmSize),
			Target:     pulumi.StringPtrFromPtr(svc.Build.Target),
		}
	}
	if len(svc.Ports) > 0 {
		ports := make(azurecompose.ServicePortConfigArray, 0, len(svc.Ports))
		for _, p := range svc.Ports {
			ports = append(ports, azurecompose.ServicePortConfigArgs{
				Target:      pulumi.Int(int(p.Target)),
				Mode:        pulumi.String(p.Mode),
				Protocol:    pulumi.String(p.Protocol),
				AppProtocol: pulumi.String(p.AppProtocol),
			})
		}
		args.Ports = ports
	}
	if svc.Deploy != nil {
		da := azurecompose.DeployConfigArgs{}
		if svc.Deploy.Replicas != nil {
			da.Replicas = pulumi.IntPtr(int(*svc.Deploy.Replicas))
		}
		if svc.Deploy.Resources != nil && svc.Deploy.Resources.Reservations != nil {
			r := svc.Deploy.Resources.Reservations
			ra := azurecompose.ResourceConfigArgs{
				Cpus:   pulumi.Float64PtrFromPtr(r.CPUs),
				Memory: pulumi.StringPtrFromPtr(r.Memory),
			}
			da.Resources = azurecompose.ResourcesArgs{Reservations: ra}
		}
		args.Deploy = da
	}
	if svc.Postgres != nil {
		args.Postgres = azurecompose.PostgresConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Postgres.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Postgres.FromSnapshot),
		}
	}
	if svc.Redis != nil {
		args.Redis = azurecompose.RedisConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Redis.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Redis.FromSnapshot),
		}
	}
	if svc.HealthCheck != nil {
		args.HealthCheck = azurecompose.HealthCheckConfigArgs{
			Test:               pulumi.ToStringArray(svc.HealthCheck.Test),
			IntervalSeconds:    pulumi.IntPtr(int(svc.HealthCheck.IntervalSeconds)),
			TimeoutSeconds:     pulumi.IntPtr(int(svc.HealthCheck.TimeoutSeconds)),
			Retries:            pulumi.IntPtr(int(svc.HealthCheck.Retries)),
			StartPeriodSeconds: pulumi.IntPtr(int(svc.HealthCheck.StartPeriodSeconds)),
		}
	}
	if len(svc.Networks) > 0 {
		nm := make(azurecompose.ServiceNetworkConfigMap, len(svc.Networks))
		for k, v := range svc.Networks {
			nm[string(k)] = azurecompose.ServiceNetworkConfigArgs{Aliases: pulumi.ToStringArray(v.Aliases)}
		}
		args.Networks = nm
	}
	if len(svc.DependsOn) > 0 {
		dm := make(azurecompose.ServiceDependencyMap, len(svc.DependsOn))
		for k, v := range svc.DependsOn {
			dm[k] = azurecompose.ServiceDependencyArgs{Condition: pulumi.String(v.Condition), Required: pulumi.Bool(v.Required)}
		}
		args.DependsOn = dm
	}
	if svc.LLM != nil {
		args.Llm = azurecompose.LlmConfigArgs{}
	}
	return args
}
