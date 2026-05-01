package program

import (
	"fmt"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp"
	gcpcompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/config"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"google.golang.org/protobuf/proto"
)

func deployGCP(ctx *pulumi.Context, cf *compose.Project, etag string, projectUpdate *defangv1.ProjectUpdate) (pulumi.StringMapOutput, pulumi.StringPtrOutput, error) {
	gcpProvider, err := gcp.NewProvider(ctx, "gcp", &gcp.ProviderArgs{
		Project: pulumi.String(config.GetProject(ctx)),
		Region:  pulumi.String(config.GetRegion(ctx)),
		DefaultLabels: pulumi.StringMap{
			"defang-etag":    pulumi.String(etag),
			"defang-org":     pulumi.String(ctx.Organization()), // TODO: doesn't work with DIY backends
			"defang-project": pulumi.String(ctx.Project()),
			"defang-stack":   pulumi.String(ctx.Stack()),
			// "defang-version": pulumi.String(Version), FIXME: cannot have dots
		},
	})
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	project, err := defanggcp.NewProject(ctx, cf.Name, toGCPArgs(cf, etag), pulumi.Providers(gcpProvider))
	if err != nil {
		return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
	}

	// Upload ProjectUpdate protobuf as a Pulumi-managed GCS object, gated on
	// the project component so it only runs after all services are created.
	// pulumi.Provider(gcpProvider) is required because
	// pulumi:disable-default-providers excludes gcp (see cd/main.go projectConfig).
	if projectUpdate != nil {
		updatedPb := project.Endpoints.ApplyT(func(endpoints map[string]string) ([]byte, error) {
			for _, svc := range projectUpdate.Services {
				svc.Status = "PROVISIONING"
				svc.State = defangv1.ServiceState_DEPLOYMENT_COMPLETED
				if ep, ok := endpoints[svc.GetService().GetName()]; ok {
					svc.Endpoints = []string{ep}
				}
			}
			return proto.Marshal(projectUpdate)
		}).(pulumi.AnyOutput)

		if err := saveProjectPbGCP(ctx, updatedPb, project, pulumi.Provider(gcpProvider)); err != nil {
			return pulumi.StringMapOutput{}, pulumi.StringPtrOutput{}, err
		}
	}
	return project.Endpoints, project.LoadBalancerDns, nil
}

func toGCPArgs(cf *compose.Project, etag string) *defanggcp.ProjectArgs {
	args := &defanggcp.ProjectArgs{
		Services: toGCPServices(cf.Services),
	}
	if len(cf.Networks) > 0 {
		nm := make(gcpcompose.NetworkConfigMap, len(cf.Networks))
		for k, v := range cf.Networks {
			nm[string(k)] = gcpcompose.NetworkConfigArgs{Internal: pulumi.Bool(v.Internal)}
		}
		args.Networks = nm
	}
	if etag != "" {
		e := etag
		args.Etag = &e
	}
	return args
}

func toGCPServices(services compose.Services) gcpcompose.ServiceConfigMap {
	m := make(gcpcompose.ServiceConfigMap, len(services))
	for name, svc := range services {
		m[name] = toGCPServiceArgs(svc)
	}
	return m
}

func toGCPServiceArgs(svc compose.ServiceConfig) gcpcompose.ServiceConfigArgs {
	args := gcpcompose.ServiceConfigArgs{
		Image:       pulumi.StringPtrFromPtr(svc.Image),
		Platform:    pulumi.StringPtrFromPtr(svc.Platform),
		Environment: pulumi.ToStringMap(svc.ResolvedEnvironment()),
		Command:     pulumi.ToStringArray(svc.Command),
		Entrypoint:  pulumi.ToStringArray(svc.Entrypoint),
	}
	if svc.DomainName != "" {
		args.DomainName = pulumi.StringPtr(svc.DomainName)
	}
	if svc.Build != nil {
		args.Build = gcpcompose.BuildConfigArgs{
			Context:    svc.Build.Context,
			Dockerfile: pulumi.StringPtrFromPtr(svc.Build.Dockerfile),
			Args:       pulumi.ToStringMap(svc.Build.Args),
			ShmSize:    pulumi.StringPtrFromPtr(svc.Build.ShmSize),
			Target:     pulumi.StringPtrFromPtr(svc.Build.Target),
		}
	}
	if len(svc.Ports) > 0 {
		ports := make(gcpcompose.ServicePortConfigArray, 0, len(svc.Ports))
		for _, p := range svc.Ports {
			ports = append(ports, gcpcompose.ServicePortConfigArgs{
				Target:      pulumi.Int(int(p.Target)),
				Mode:        pulumi.String(p.Mode),
				Protocol:    pulumi.String(p.Protocol),
				AppProtocol: pulumi.String(p.AppProtocol),
			})
		}
		args.Ports = ports
	}
	if svc.Deploy != nil {
		da := gcpcompose.DeployConfigArgs{}
		if svc.Deploy.Replicas != nil {
			da.Replicas = pulumi.IntPtr(int(*svc.Deploy.Replicas))
		}
		if svc.Deploy.Resources != nil && svc.Deploy.Resources.Reservations != nil {
			r := svc.Deploy.Resources.Reservations
			ra := gcpcompose.ResourceConfigArgs{
				Cpus:   pulumi.Float64PtrFromPtr(r.CPUs),
				Memory: pulumi.StringPtrFromPtr(r.Memory),
			}
			da.Resources = gcpcompose.ResourcesArgs{Reservations: ra}
		}
		args.Deploy = da
	}
	if svc.Postgres != nil {
		args.Postgres = gcpcompose.PostgresConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Postgres.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Postgres.FromSnapshot),
		}
	}
	if svc.Redis != nil {
		args.Redis = gcpcompose.RedisConfigArgs{
			AllowDowntime: pulumi.BoolPtrFromPtr(svc.Redis.AllowDowntime),
			FromSnapshot:  pulumi.StringPtrFromPtr(svc.Redis.FromSnapshot),
		}
	}
	if svc.HealthCheck != nil {
		args.HealthCheck = gcpcompose.HealthCheckConfigArgs{
			Test:               pulumi.ToStringArray(svc.HealthCheck.Test),
			IntervalSeconds:    pulumi.IntPtr(int(svc.HealthCheck.IntervalSeconds)),
			TimeoutSeconds:     pulumi.IntPtr(int(svc.HealthCheck.TimeoutSeconds)),
			Retries:            pulumi.IntPtr(int(svc.HealthCheck.Retries)),
			StartPeriodSeconds: pulumi.IntPtr(int(svc.HealthCheck.StartPeriodSeconds)),
		}
	}
	if len(svc.Networks) > 0 {
		nm := make(gcpcompose.ServiceNetworkConfigMap, len(svc.Networks))
		for k, v := range svc.Networks {
			nm[string(k)] = gcpcompose.ServiceNetworkConfigArgs{Aliases: pulumi.ToStringArray(v.Aliases)}
		}
		args.Networks = nm
	}
	if len(svc.DependsOn) > 0 {
		dm := make(gcpcompose.ServiceDependencyMap, len(svc.DependsOn))
		for k, v := range svc.DependsOn {
			dm[k] = gcpcompose.ServiceDependencyArgs{Condition: pulumi.String(v.Condition), Required: pulumi.Bool(v.Required)}
		}
		args.DependsOn = dm
	}
	if svc.LLM != nil {
		args.Llm = gcpcompose.LlmConfigArgs{}
	}
	return args
}

// saveProjectPbGCP uploads data as a Pulumi-managed GCS object at the key
// derived from DEFANG_STATE_URL. See saveProjectPbAWS for semantics.
// data is a pulumi.AnyOutput wrapping []byte so callers can produce the bytes
// inside an ApplyT (e.g. after updating endpoints) without creating resources
// inside that callback.
func saveProjectPbGCP(ctx *pulumi.Context, data pulumi.AnyOutput, dep pulumi.Resource, opts ...pulumi.ResourceOption) error {
	u, err := parseStateURL(ctx)
	if err != nil || u == nil {
		return err
	}
	if u.Scheme != "gs" || u.Host == "" {
		return fmt.Errorf("DEFANG_STATE_URL must be a gs:// URL with a bucket for GCP uploads, got %q", u.String())
	}

	source := data.ApplyT(func(v any) (pulumi.Asset, error) {
		return NewTempFileAsset("defang-cd-*-project.pb", v.([]byte))
	}).(pulumi.AssetOutput)

	_, err = storage.NewBucketObject(ctx, "project-pb", &storage.BucketObjectArgs{
		Bucket:      pulumi.String(u.Host),
		Name:        pulumi.String(projectPbKey(ctx)),
		Source:      source,
		ContentType: pulumi.String(protobufContentType),
	}, common.MergeOptions(opts, pulumi.DependsOn([]pulumi.Resource{dep}))...)
	return err
}
