package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/aws"
	providerazure "github.com/DefangLabs/pulumi-defang/provider/azure"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang:index:Project component.
type Project struct{}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumi.StringMapOutput `pulumi:"endpoints"`

	// Load balancer DNS name (AWS ALB or GCP forwarding rule IP)
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	args := common.BuildArgs{
		Services: toServices(inputs.Services),
		AWS:      toAWSConfig(inputs.AWS),
		GCP:      toGCPConfig(inputs.GCP),
		Azure:    toAzureConfig(inputs.Azure),
	}

	var result *common.BuildResult
	var err error

	switch inputs.Provider {
	case "aws":
		result, err = provideraws.Build(ctx, name, args, childOpt)
	case "gcp":
		result, err = providergcp.Build(ctx, name, args, childOpt)
	case "azure":
		result, err = providerazure.Build(ctx, name, args, childOpt)
	default:
		return nil, fmt.Errorf("unsupported provider %q: must be \"aws\", \"gcp\", or \"azure\"", inputs.Provider)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build %s resources: %w", inputs.Provider, err)
	}

	comp.Endpoints = result.Endpoints
	comp.LoadBalancerDNS = result.LoadBalancerDNS

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// toServices converts the Pulumi-facing ServiceInput map to resolved common.ServiceConfig.
func toServices(services map[string]ServiceInput) map[string]common.ServiceConfig {
	result := make(map[string]common.ServiceConfig, len(services))
	for name, svc := range services {
		cfg := common.ServiceConfig{
			Build:       toBuild(svc.Build),
			Image:       svc.Image,
			Ports:       toPorts(svc.Ports),
			Deploy:      toDeploy(svc.Deploy),
			Environment: svc.Environment,
			Command:     svc.Command,
			Entrypoint:  svc.Entrypoint,
			Postgres:    toPostgres(svc.Postgres, svc.Image, svc.Environment),
			HealthCheck: toHealthCheck(svc.HealthCheck),
			DomainName:  svc.DomainName,
		}
		if svc.Platform != nil {
			cfg.Platform = *svc.Platform
		}
		result[name] = cfg
	}
	return result
}

func toPorts(ports []PortConfig) []common.ServicePortConfig {
	result := make([]common.ServicePortConfig, len(ports))
	for i, p := range ports {
		result[i] = common.ServicePortConfig{
			Target:      p.Target,
			Mode:        p.Mode,
			Protocol:    getPortProtocol(p),
			AppProtocol: getAppProtocol(p),
		}
	}
	return result
}

func toBuild(b *BuildInput) *common.BuildConfig {
	if b == nil {
		return nil
	}
	cfg := &common.BuildConfig{
		Context: b.Context,
		Args:    b.Args,
	}
	if b.Dockerfile != nil {
		cfg.Dockerfile = *b.Dockerfile
	}
	if b.ShmSize != nil {
		cfg.ShmSize = parseMemoryMiB(*b.ShmSize) * 1024 * 1024 // store as bytes
	}
	if b.Target != nil {
		cfg.Target = *b.Target
	}
	return cfg
}

func toDeploy(d *DeployConfig) *common.DeployConfig {
	if d == nil {
		return nil
	}
	result := &common.DeployConfig{
		Replicas: d.Replicas,
	}
	if d.Resources != nil && d.Resources.Reservations != nil {
		r := d.Resources.Reservations
		res := &common.ResourceConfig{
			CPUs: r.CPUs,
		}
		if r.Memory != nil {
			m := parseMemoryMiB(*r.Memory)
			res.MemoryMiB = &m
		}
		result.Resources = &common.ResourcesConfig{
			Reservations: res,
		}
	}
	return result
}

// toPostgres derives PostgresConfig from the x-defang-postgres extension, image tag, and env vars.
func toPostgres(p *PostgresInput, image *string, env map[string]string) *common.PostgresConfig {
	if p == nil {
		return nil
	}

	// Derive version from image tag (e.g. "postgres:16" → 16)
	version := 0
	if image != nil {
		version = getPostgresVersion(parseImageTag(*image))
	}

	// Derive credentials from env vars, matching defang-mvp behavior
	dbName := "postgres"
	if v, ok := env["POSTGRES_DB"]; ok && v != "" {
		dbName = v
	}
	username := "postgres"
	if v, ok := env["POSTGRES_USER"]; ok && v != "" {
		username = v
	}
	password := env["POSTGRES_PASSWORD"]

	allowDowntime := false
	if p.AllowDowntime != nil {
		allowDowntime = *p.AllowDowntime
	}
	fromSnapshot := ""
	if p.FromSnapshot != nil {
		fromSnapshot = *p.FromSnapshot
	}

	return &common.PostgresConfig{
		Version:       version,
		DBName:        dbName,
		Username:      username,
		Password:      password,
		AllowDowntime: allowDowntime,
		FromSnapshot:  fromSnapshot,
	}
}

func toHealthCheck(h *HealthCheckConfig) *common.HealthCheckConfig {
	if h == nil {
		return nil
	}
	return &common.HealthCheckConfig{
		Test:               h.Test,
		IntervalSeconds:    h.IntervalSeconds,
		TimeoutSeconds:     h.TimeoutSeconds,
		Retries:            h.Retries,
		StartPeriodSeconds: h.StartPeriodSeconds,
	}
}

func toAWSConfig(a *AWSConfigInput) *common.AWSConfig {
	if a == nil {
		return nil
	}
	return &common.AWSConfig{
		VpcID:            a.VpcID,
		SubnetIDs:        a.SubnetIDs,
		PrivateSubnetIDs: a.PrivateSubnetIDs,
		Region:           a.Region,
	}
}

func toGCPConfig(g *GCPConfigInput) *common.GCPConfig {
	if g == nil {
		return nil
	}
	return &common.GCPConfig{
		Project: g.Project,
		Region:  g.Region,
	}
}

func toAzureConfig(a *AzureConfigInput) *common.AzureConfig {
	if a == nil {
		return nil
	}
	return &common.AzureConfig{
		SubscriptionID: a.SubscriptionID,
		Location:       a.Location,
	}
}
