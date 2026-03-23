package azure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type containerAppResult struct {
	App *app.ContainerApp
}

// containerAppCpuMemory snaps requested CPU/memory to Azure Container Apps fixed tiers.
func containerAppCpuMemory(cpus float64, memMiB int) (float64, string) {
	memGi := float64(memMiB) / 1024.0
	options := []struct {
		cpu float64
		mem float64
	}{
		{0.25, 0.5},
		{0.5, 1.0},
		{0.75, 1.5},
		{1.0, 2.0},
		{1.25, 2.5},
		{1.5, 3.0},
		{1.75, 3.5},
		{2.0, 4.0},
	}
	for _, opt := range options {
		if cpus <= opt.cpu && memGi <= opt.mem {
			return opt.cpu, fmt.Sprintf("%.2fGi", opt.mem)
		}
	}
	return 2.0, "4.00Gi"
}

// CreateContainerApp creates an Azure Container App.
func CreateContainerApp(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*containerAppResult, error) {
	// Build environment variables
	envs := app.EnvironmentVarArray{
		app.EnvironmentVarArgs{
			Name:  pulumi.String("DEFANG_SERVICE"),
			Value: pulumi.String(serviceName),
		},
	}
	for k, v := range svc.Environment {
		envs = append(envs, app.EnvironmentVarArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(v),
		})
	}

	// Resource limits
	cpu, mem := containerAppCpuMemory(svc.GetCPUs(), svc.GetMemoryMiB())

	// Scale config
	minReplicas := svc.GetReplicas()
	maxReplicas := minReplicas
	if MaxReplicas.Get(ctx) > 0 {
		maxReplicas = MaxReplicas.Get(ctx)
	}

	// Ingress config
	var ingress *app.IngressArgs
	if len(svc.Ports) > 0 {
		external := false
		for _, p := range svc.Ports {
			if p.Mode == "ingress" {
				external = true
				break
			}
		}
		ingress = &app.IngressArgs{
			External:   pulumi.Bool(external),
			TargetPort: pulumi.Int(svc.Ports[0].Target),
		}
	}

	// Health check probes
	var probes app.ContainerAppProbeArray
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 && len(svc.Ports) > 0 {
		probe := app.ContainerAppProbeArgs{
			Type: pulumi.String("Liveness"),
			HttpGet: &app.ContainerAppProbeHttpGetArgs{
				Port: pulumi.Int(svc.Ports[0].Target),
				Path: pulumi.String("/"),
			},
		}
		if svc.HealthCheck.IntervalSeconds != nil {
			probe.PeriodSeconds = pulumi.Int(*svc.HealthCheck.IntervalSeconds)
		}
		if svc.HealthCheck.TimeoutSeconds != nil {
			probe.TimeoutSeconds = pulumi.Int(*svc.HealthCheck.TimeoutSeconds)
		}
		if svc.HealthCheck.Retries != nil {
			probe.FailureThreshold = pulumi.Int(*svc.HealthCheck.Retries)
		}
		if svc.HealthCheck.StartPeriodSeconds != nil {
			probe.InitialDelaySeconds = pulumi.Int(*svc.HealthCheck.StartPeriodSeconds)
		}
		probes = append(probes, probe)
	}

	containerApp, err := app.NewContainerApp(ctx, serviceName, &app.ContainerAppArgs{
		ResourceGroupName:    infra.ResourceGroup.Name,
		ManagedEnvironmentId: infra.Environment.ID().ToStringOutput(),
		Configuration: &app.ConfigurationArgs{
			Ingress: ingress,
		},
		Template: &app.TemplateArgs{
			Scale: &app.ScaleArgs{
				MinReplicas: pulumi.Int(minReplicas),
				MaxReplicas: pulumi.Int(maxReplicas),
			},
			Containers: app.ContainerArray{
				app.ContainerArgs{
					Name:    pulumi.String(serviceName),
					Image:   pulumi.String(*svc.Image), // TODO: support build config
					Command: compose.ToPulumiStringArray(svc.Entrypoint),
					Args:    compose.ToPulumiStringArray(svc.Command),
					Env:     envs,
					Probes:  probes,
					Resources: &app.ContainerResourcesArgs{
						Cpu:    pulumi.Float64(cpu),
						Memory: pulumi.String(mem),
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Container App: %w", err)
	}

	return &containerAppResult{App: containerApp}, nil
}
