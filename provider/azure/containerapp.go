package azure

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type containerAppResult struct {
	app *app.ContainerApp
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

// createContainerApp creates an Azure Container App.
func createContainerApp(
	ctx *pulumi.Context,
	serviceName string,
	svc common.ServiceConfig,
	infra *sharedInfra,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*containerAppResult, error) {
	image := svc.GetImage()

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
	if recipe.MaxReplicas > 0 {
		maxReplicas = recipe.MaxReplicas
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
		ResourceGroupName:    infra.resourceGroup.Name,
		ManagedEnvironmentId: infra.environment.ID().ToStringOutput(),
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
					Image:   pulumi.String(image),
					Command: pulumi.ToStringArray(svc.Entrypoint),
					Args:    pulumi.ToStringArray(svc.Command),
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

	return &containerAppResult{app: containerApp}, nil
}
