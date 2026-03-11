package provider

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/aws"
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
	}

	var result *common.BuildResult
	var err error

	switch inputs.Provider {
	case "aws":
		result, err = provideraws.Build(ctx, name, args, childOpt)
	case "gcp":
		result, err = providergcp.Build(ctx, name, args, childOpt)
	default:
		return nil, fmt.Errorf("unsupported provider %q: must be \"aws\" or \"gcp\"", inputs.Provider)
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
		result[name] = common.ServiceConfig{
			Image:       svc.Image,
			Ports:       toPorts(svc.Ports),
			Deploy:      toDeploy(svc.Deploy),
			Environment: svc.Environment,
			Command:     svc.Command,
			Entrypoint:  svc.Entrypoint,
			Postgres:    toPostgres(svc.Postgres),
			HealthCheck: toHealthCheck(svc.HealthCheck),
			DomainName:  svc.DomainName,
			CloudRun:    toCloudRun(svc.CloudRun),
		}
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

func toDeploy(d *DeployConfig) *common.DeployConfig {
	if d == nil {
		return nil
	}
	return &common.DeployConfig{
		Replicas:  getReplicas(d),
		CPUs:      getCPUs(d),
		MemoryMiB: getMemoryMiB(d),
	}
}

func toPostgres(p *PostgresConfig) *common.PostgresConfig {
	if p == nil {
		return nil
	}
	version := 17
	if p.Version != nil {
		version = *p.Version
	}
	dbName := "postgres"
	if p.DBName != nil {
		dbName = *p.DBName
	}
	username := "postgres"
	if p.Username != nil {
		username = *p.Username
	}
	availabilityType := "ZONAL"
	if p.AvailabilityType != nil {
		availabilityType = *p.AvailabilityType
	}
	backupEnabled := false
	if p.BackupEnabled != nil {
		backupEnabled = *p.BackupEnabled
	}
	pointInTimeRecovery := false
	if p.PointInTimeRecovery != nil {
		pointInTimeRecovery = *p.PointInTimeRecovery
	}
	sslMode := "ALLOW_UNENCRYPTED_AND_ENCRYPTED"
	if p.SslMode != nil {
		sslMode = *p.SslMode
	}
	deletionProtection := false
	if p.DeletionProtection != nil {
		deletionProtection = *p.DeletionProtection
	}
	allowBurstable := true
	if p.AllowBurstable != nil {
		allowBurstable = *p.AllowBurstable
	}
	return &common.PostgresConfig{
		Version:             version,
		DBName:              dbName,
		Username:            username,
		Password:            p.Password,
		AvailabilityType:    availabilityType,
		BackupEnabled:       backupEnabled,
		PointInTimeRecovery: pointInTimeRecovery,
		SslMode:             sslMode,
		DeletionProtection:  deletionProtection,
		AllowBurstable:      allowBurstable,
	}
}

func toCloudRun(c *CloudRunConfig) *common.CloudRunConfig {
	if c == nil {
		return nil
	}
	ingress := "INGRESS_TRAFFIC_ALL"
	if c.Ingress != nil {
		ingress = *c.Ingress
	}
	launchStage := "BETA"
	if c.LaunchStage != nil {
		launchStage = *c.LaunchStage
	}
	var maxReplicas int
	if c.MaxReplicas != nil {
		maxReplicas = *c.MaxReplicas
	}
	return &common.CloudRunConfig{
		Ingress:     ingress,
		LaunchStage: launchStage,
		MaxReplicas: maxReplicas,
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
