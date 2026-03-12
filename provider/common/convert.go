package common

import (
	"github.com/DefangLabs/pulumi-defang/provider/shared"
)

// ToServices converts the Pulumi-facing ServiceInput map to resolved common.ServiceConfig.
func ToServices(services map[string]shared.ServiceInput) map[string]ServiceConfig {
	result := make(map[string]ServiceConfig, len(services))
	for name, svc := range services {
		cfg := ServiceConfig{
			Build:       ToBuild(svc.Build),
			Image:       svc.Image,
			Ports:       ToPorts(svc.Ports),
			Deploy:      ToDeploy(svc.Deploy),
			Environment: svc.Environment,
			Command:     svc.Command,
			Entrypoint:  svc.Entrypoint,
			Postgres:    ToPostgres(svc.Postgres, svc.Image, svc.Environment),
			HealthCheck: ToHealthCheck(svc.HealthCheck),
			DomainName:  svc.DomainName,
		}
		if svc.Platform != nil {
			cfg.Platform = *svc.Platform
		}
		result[name] = cfg
	}
	return result
}

// ToPorts converts Pulumi PortConfig to common.ServicePortConfig.
func ToPorts(ports []shared.PortConfig) []ServicePortConfig {
	result := make([]ServicePortConfig, len(ports))
	for i, p := range ports {
		result[i] = ServicePortConfig{
			Target:      p.Target,
			Mode:        p.Mode,
			Protocol:    shared.GetPortProtocol(p),
			AppProtocol: shared.GetAppProtocol(p),
		}
	}
	return result
}

// ToBuild converts Pulumi BuildInput to common.BuildConfig.
func ToBuild(b *shared.BuildInput) *BuildConfig {
	if b == nil {
		return nil
	}
	cfg := &BuildConfig{
		Context: b.Context,
		Args:    b.Args,
	}
	if b.Dockerfile != nil {
		cfg.Dockerfile = *b.Dockerfile
	}
	if b.ShmSize != nil {
		cfg.ShmSize = shared.ParseMemoryMiB(*b.ShmSize) * 1024 * 1024 // store as bytes
	}
	if b.Target != nil {
		cfg.Target = *b.Target
	}
	return cfg
}

// ToDeploy converts Pulumi DeployConfig to common.DeployConfig.
func ToDeploy(d *shared.DeployConfig) *DeployConfig {
	if d == nil {
		return nil
	}
	result := &DeployConfig{
		Replicas: d.Replicas,
	}
	if d.Resources != nil && d.Resources.Reservations != nil {
		r := d.Resources.Reservations
		res := &ResourceConfig{
			CPUs: r.CPUs,
		}
		if r.Memory != nil {
			m := shared.ParseMemoryMiB(*r.Memory)
			res.MemoryMiB = &m
		}
		result.Resources = &ResourcesConfig{
			Reservations: res,
		}
	}
	return result
}

// ToPostgres derives PostgresConfig from the x-defang-postgres extension, image tag, and env vars.
func ToPostgres(p *shared.PostgresInput, image *string, env map[string]string) *PostgresConfig {
	if p == nil {
		return nil
	}

	// Derive version from image tag (e.g. "postgres:16" → 16)
	version := 0
	if image != nil {
		version = shared.GetPostgresVersion(shared.ParseImageTag(*image))
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

	return &PostgresConfig{
		Version:       version,
		DBName:        dbName,
		Username:      username,
		Password:      password,
		AllowDowntime: allowDowntime,
		FromSnapshot:  fromSnapshot,
	}
}

// ToHealthCheck converts Pulumi HealthCheckConfig to common.HealthCheckConfig.
func ToHealthCheck(h *shared.HealthCheckConfig) *HealthCheckConfig {
	if h == nil {
		return nil
	}
	return &HealthCheckConfig{
		Test:               h.Test,
		IntervalSeconds:    h.IntervalSeconds,
		TimeoutSeconds:     h.TimeoutSeconds,
		Retries:            h.Retries,
		StartPeriodSeconds: h.StartPeriodSeconds,
	}
}

// ToAWSConfig converts Pulumi AWSConfigInput to common.AWSConfig.
func ToAWSConfig(a *shared.AWSConfigInput) *AWSConfig {
	if a == nil {
		return nil
	}
	return &AWSConfig{
		VpcID:            a.VpcID,
		SubnetIDs:        a.SubnetIDs,
		PrivateSubnetIDs: a.PrivateSubnetIDs,
	}
}

// ToServiceConfig converts a single Pulumi ServiceInput to common.ServiceConfig.
func ToServiceConfig(svc shared.ServiceInput) ServiceConfig {
	cfg := ServiceConfig{
		Build:       ToBuild(svc.Build),
		Image:       svc.Image,
		Ports:       ToPorts(svc.Ports),
		Deploy:      ToDeploy(svc.Deploy),
		Environment: svc.Environment,
		Command:     svc.Command,
		Entrypoint:  svc.Entrypoint,
		Postgres:    ToPostgres(svc.Postgres, svc.Image, svc.Environment),
		HealthCheck: ToHealthCheck(svc.HealthCheck),
		DomainName:  svc.DomainName,
	}
	if svc.Platform != nil {
		cfg.Platform = *svc.Platform
	}
	return cfg
}
