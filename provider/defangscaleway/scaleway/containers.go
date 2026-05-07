package scaleway

import (
	"errors"
	"fmt"
	"math"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	scalewayconfig "github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/config"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/containers"
)

var (
	ErrContainerImageMissing     = errors.New("Scaleway Serverless Containers require a pre-built image")
	ErrContainerNamespaceMissing = errors.New("Scaleway Serverless Containers require a namespace")
)

type ContainerResult struct {
	Container *containers.Container
	Domain    *containers.Domain
	Endpoint  pulumi.StringOutput
}

func NewStandaloneInfra(ctx *pulumi.Context, projectName string) *SharedInfra {
	return &SharedInfra{
		Region:    scalewayconfig.GetRegion(ctx),
		Zone:      scalewayconfig.GetZone(ctx),
		ProjectID: scalewayconfig.GetProjectId(ctx),
	}
}

func CreateContainerNamespace(
	ctx *pulumi.Context,
	projectName string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*containers.Namespace, error) {
	if infra == nil {
		infra = NewStandaloneInfra(ctx, projectName)
	}
	args := &containers.NamespaceArgs{
		Name: pulumi.StringPtr(projectName),
		Tags: pulumi.StringArray{
			pulumi.String("defang"),
			pulumi.String(projectName),
		},
	}
	if infra.Region != "" {
		args.Region = pulumi.StringPtr(infra.Region)
	}
	if infra.ProjectID != "" {
		args.ProjectId = pulumi.StringPtr(infra.ProjectID)
	}
	namespace, err := containers.NewNamespace(ctx, projectName+"-namespace", args, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway Serverless Containers namespace: %w", err)
	}
	return namespace, nil
}

func containerCPULimit(cpus float64) int {
	if cpus <= 0 {
		return 140
	}
	return int(math.Ceil(cpus * 1000))
}

func containerMemoryLimit(memMiB int) int {
	if memMiB <= 0 {
		return 256
	}
	return max(memMiB, 128)
}

func containerProtocol(svc compose.ServiceConfig) string {
	if len(svc.Ports) == 0 {
		return "http1"
	}
	switch svc.Ports[0].GetAppProtocol() {
	case compose.PortAppProtocolHTTP2, compose.PortAppProtocolGRPC:
		return "h2c"
	default:
		return "http1"
	}
}

func containerPort(svc compose.ServiceConfig) pulumi.IntPtrInput {
	for _, p := range svc.Ports {
		if p.IsIngress() && p.Target > 0 {
			return pulumi.IntPtr(int(p.Target))
		}
	}
	return nil
}

func containerMinScale(svc compose.ServiceConfig) pulumi.IntPtrInput {
	if svc.GetReplicas() <= 1 {
		return pulumi.IntPtr(0)
	}
	return pulumi.IntPtr(1)
}

func containerMaxScale(svc compose.ServiceConfig) pulumi.IntPtrInput {
	return pulumi.IntPtr(int(svc.GetReplicas()))
}

func containerEnvironment(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.InvokeOption,
) (pulumi.StringMap, pulumi.StringMap) {
	env := pulumi.StringMap{
		"DEFANG_SERVICE": pulumi.String(serviceName),
	}
	if infra != nil && infra.Etag != "" {
		env["DEFANG_ETAG"] = pulumi.String(infra.Etag)
	}
	secrets := pulumi.StringMap{}
	for k, v := range common.Sorted(svc.Environment) {
		if secretVar := compose.GetConfigName2(k, v); secretVar != "" && configProvider != nil {
			secrets[k] = configProvider.GetConfigValue(ctx, secretVar, opts...)
			continue
		}
		raw := ""
		if v != nil {
			raw = *v
		}
		env[k] = compose.InterpolateEnvironmentVariable(ctx, configProvider, raw, opts...)
	}
	return env, secrets
}

func CreateContainerService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*ContainerResult, error) {
	if image == nil {
		return nil, ErrContainerImageMissing
	}
	if infra == nil || infra.Namespace == nil {
		return nil, ErrContainerNamespaceMissing
	}
	env, secrets := containerEnvironment(ctx, configProvider, serviceName, svc, infra)
	args := &containers.ContainerArgs{
		Name:                       pulumi.StringPtr(serviceName),
		NamespaceId:                infra.Namespace.ID(),
		RegistryImage:              image.ToStringOutput().ToStringPtrOutput(),
		Port:                       containerPort(svc),
		CpuLimit:                   pulumi.IntPtr(containerCPULimit(svc.GetCPUs())),
		MemoryLimit:                pulumi.IntPtr(containerMemoryLimit(svc.GetMemoryMiB())),
		MinScale:                   containerMinScale(svc),
		MaxScale:                   containerMaxScale(svc),
		Privacy:                    pulumi.StringPtr("public"),
		Protocol:                   pulumi.StringPtr(containerProtocol(svc)),
		Deploy:                     pulumi.BoolPtr(true),
		Commands:                   compose.ToPulumiStringArray(svc.Entrypoint),
		Args:                       compose.ToPulumiStringArray(svc.Command),
		EnvironmentVariables:       env,
		SecretEnvironmentVariables: secrets,
	}
	if infra.Region != "" {
		args.Region = pulumi.StringPtr(infra.Region)
	}
	if infra.PrivateNetwork != nil {
		args.PrivateNetworkId = infra.PrivateNetwork.ID().ToStringOutput().ToStringPtrOutput()
	}

	container, err := containers.NewContainer(ctx, serviceName, args, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway Serverless Container: %w", err)
	}

	var domain *containers.Domain
	if svc.DomainName != "" {
		domain, err = containers.NewDomain(ctx, serviceName+"-domain", &containers.DomainArgs{
			ContainerId: container.ID(),
			Hostname:    pulumi.String(svc.DomainName),
			Region:      args.Region,
		}, append(opts, pulumi.Parent(container))...)
		if err != nil {
			return nil, fmt.Errorf("creating Scaleway Serverless Container domain: %w", err)
		}
	}

	return &ContainerResult{Container: container, Domain: domain, Endpoint: container.DomainName}, nil
}
