package provider

import (
	"strconv"
	"strings"
)

// AWSConfigInput defines optional AWS-specific configuration.
type AWSConfigInput struct {
	VpcID            string   `pulumi:"vpcId,optional"`
	SubnetIDs        []string `pulumi:"subnetIds,optional"`
	PrivateSubnetIDs []string `pulumi:"privateSubnetIds,optional"`
	Region           string   `pulumi:"region,optional"`
}

// GCPConfigInput defines optional GCP-specific configuration.
type GCPConfigInput struct {
	Project string `pulumi:"project,optional"`
	Region  string `pulumi:"region,optional"`
}

// ProjectInputs defines the top-level inputs for the defang:index:Project component.
type ProjectInputs struct {
	// Cloud provider: "aws" or "gcp"
	Provider string `pulumi:"provider" yaml:"provider"`

	// Services map: name -> service config
	Services map[string]ServiceInput `pulumi:"services" yaml:"services"`

	// AWS-specific configuration
	AWS *AWSConfigInput `pulumi:"aws,optional"`

	// GCP-specific configuration
	GCP *GCPConfigInput `pulumi:"gcp,optional"`
}

// ServiceInput defines the configuration for a single service.
// YAML tags are aligned with Docker Compose service spec where possible.
type ServiceInput struct {
	// Container image to deploy (required if no postgres config)
	Image *string `pulumi:"image,optional" yaml:"image,omitempty"`

	// Port configurations
	Ports []PortConfig `pulumi:"ports,optional" yaml:"ports,omitempty"`

	// Deployment configuration (replicas, resources)
	Deploy *DeployConfig `pulumi:"deploy,optional" yaml:"deploy,omitempty"`

	// Environment variables
	Environment map[string]string `pulumi:"environment,optional" yaml:"environment,omitempty"`

	// Command to run
	Command []string `pulumi:"command,optional" yaml:"command,omitempty"`

	// Entrypoint override
	Entrypoint []string `pulumi:"entrypoint,optional" yaml:"entrypoint,omitempty"`

	// Managed Postgres configuration
	Postgres *PostgresConfig `pulumi:"postgres,optional" yaml:"x-defang-postgres,omitempty"`

	// Health check configuration
	HealthCheck *HealthCheckConfig `pulumi:"healthCheck,optional" yaml:"healthcheck,omitempty"`

	// Custom domain name
	DomainName *string `pulumi:"domainName,optional" yaml:"domainname,omitempty"`

	// GCP Cloud Run-specific configuration
	CloudRun *CloudRunConfig `pulumi:"cloudRun,optional" yaml:"x-defang-cloudrun,omitempty"`
}

// PortConfig defines a port mapping for a service.
type PortConfig struct {
	// Container port
	Target int `pulumi:"target" yaml:"target"`

	// Port mode: "host" or "ingress" (default: "host")
	Mode string `pulumi:"mode,optional" yaml:"mode,omitempty"`

	// Transport protocol: "tcp" or "udp" (default: "tcp")
	Protocol string `pulumi:"protocol,optional" yaml:"protocol,omitempty"`

	// Application protocol: "http", "http2", "grpc" (default: "http")
	AppProtocol string `pulumi:"appProtocol,optional" yaml:"app_protocol,omitempty"`
}

// DeployConfig defines deployment parameters.
// YAML tags match Docker Compose deploy spec.
type DeployConfig struct {
	// Number of replicas (default: 1)
	Replicas *int `pulumi:"replicas,optional" yaml:"replicas,omitempty"`

	// Resource reservations and limits
	Resources *ResourcesConfig `pulumi:"resources,optional" yaml:"resources,omitempty"`
}

// ResourcesConfig defines resource reservations and limits.
// Mirrors Docker Compose deploy.resources spec.
type ResourcesConfig struct {
	// Resource reservations (guaranteed minimums)
	Reservations *ResourceConfig `pulumi:"reservations,optional" yaml:"reservations,omitempty"`

	// Resource limits (hard caps)
	Limits *ResourceConfig `pulumi:"limits,optional" yaml:"limits,omitempty"`
}

// ResourceConfig defines CPU and memory for a single resource bound.
// Mirrors Docker Compose deploy.resources.reservations spec.
type ResourceConfig struct {
	// CPU units (e.g., 0.25, 0.5, 1, 2, 4)
	CPUs *float64 `pulumi:"cpus,optional" yaml:"cpus,omitempty"`

	// Memory as a string (e.g., "512Mi", "2Gi") or raw bytes number.
	// Compose-go normalizes to bytes; we also accept "Mi"/"Gi" suffixes.
	Memory *string `pulumi:"memory,optional" yaml:"memory,omitempty"`
}

// PostgresConfig enables a managed Postgres instance for this service.
type PostgresConfig struct {
	// Postgres major version: 14, 15, 16, 17 (default: 17)
	Version *int `pulumi:"version,optional" yaml:"version,omitempty"`

	// Database name (default: "postgres")
	DBName *string `pulumi:"dbName,optional" yaml:"dbname,omitempty"`

	// Database user (default: "postgres")
	Username *string `pulumi:"username,optional" yaml:"username,omitempty"`

	// Database password (required)
	Password string `pulumi:"password" yaml:"password"`

	// Availability type: "ZONAL" or "REGIONAL" (default: "ZONAL")
	AvailabilityType *string `pulumi:"availabilityType,optional" yaml:"availability_type,omitempty"`

	// Enable automated backups (default: false)
	BackupEnabled *bool `pulumi:"backupEnabled,optional" yaml:"backup_enabled,omitempty"`

	// Enable point-in-time recovery (default: false)
	PointInTimeRecovery *bool `pulumi:"pointInTimeRecovery,optional" yaml:"point_in_time_recovery,omitempty"`

	// SSL enforcement mode (default: "ALLOW_UNENCRYPTED_AND_ENCRYPTED")
	SslMode *string `pulumi:"sslMode,optional" yaml:"ssl_mode,omitempty"`

	// Prevent accidental deletion (default: false)
	DeletionProtection *bool `pulumi:"deletionProtection,optional" yaml:"deletion_protection,omitempty"`

	// Allow burstable (micro/small) instance tiers (default: true)
	AllowBurstable *bool `pulumi:"allowBurstable,optional" yaml:"allow_burstable,omitempty"`
}

// CloudRunConfig defines GCP Cloud Run-specific configuration overrides.
type CloudRunConfig struct {
	// Ingress traffic setting (default: "INGRESS_TRAFFIC_ALL")
	Ingress *string `pulumi:"ingress,optional" yaml:"ingress,omitempty"`

	// Launch stage: "BETA" or "GA" (default: "BETA")
	LaunchStage *string `pulumi:"launchStage,optional" yaml:"launch_stage,omitempty"`

	// Maximum number of instances for autoscaling (default: uses deploy.replicas)
	MaxReplicas *int `pulumi:"maxReplicas,optional" yaml:"max_replicas,omitempty"`
}

// HealthCheckConfig defines health check parameters.
// YAML tags match Docker Compose healthcheck spec.
type HealthCheckConfig struct {
	// Health check command
	Test []string `pulumi:"test,optional" yaml:"test,omitempty"`

	// Check interval in seconds (default: 30)
	IntervalSeconds *int `pulumi:"intervalSeconds,optional" yaml:"interval,omitempty"`

	// Check timeout in seconds (default: 5)
	TimeoutSeconds *int `pulumi:"timeoutSeconds,optional" yaml:"timeout,omitempty"`

	// Number of retries before marking unhealthy (default: 3)
	Retries *int `pulumi:"retries,optional" yaml:"retries,omitempty"`

	// Grace period before health checks start in seconds (default: 0)
	StartPeriodSeconds *int `pulumi:"startPeriodSeconds,optional" yaml:"start_period,omitempty"`
}

// getPortProtocol returns the protocol, defaulting to "tcp".
func getPortProtocol(p PortConfig) string {
	if p.Protocol != "" {
		return p.Protocol
	}
	return "tcp"
}

// getAppProtocol returns the application protocol, defaulting to "http".
func getAppProtocol(p PortConfig) string {
	if p.AppProtocol != "" {
		return p.AppProtocol
	}
	return "http"
}

// getReplicas returns the number of replicas, defaulting to 1.
func getReplicas(d *DeployConfig) int {
	if d != nil && d.Replicas != nil {
		return *d.Replicas
	}
	return 1
}

// getCPUs returns the CPU reservation, defaulting to 0.25.
func getCPUs(d *DeployConfig) float64 {
	if d != nil && d.Resources != nil && d.Resources.Reservations != nil && d.Resources.Reservations.CPUs != nil {
		return *d.Resources.Reservations.CPUs
	}
	return 0.25
}

// getMemoryMiB returns the memory reservation in MiB, defaulting to 512.
func getMemoryMiB(d *DeployConfig) int {
	if d != nil && d.Resources != nil && d.Resources.Reservations != nil && d.Resources.Reservations.Memory != nil {
		return parseMemoryMiB(*d.Resources.Reservations.Memory)
	}
	return 512
}

// parseMemoryMiB parses a memory string into MiB.
// Accepts raw bytes (compose-go normalized), or suffixes: b, k, m, g, t, kb, mb, gb, tb, ki, mi, gi, ti.
func parseMemoryMiB(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 512
	}

	// Try raw number (bytes, as compose-go normalizes)
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		mib := int(n / (1024 * 1024))
		if mib <= 0 {
			return 512
		}
		return mib
	}

	// Find where the numeric part ends
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 512
	}

	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil || n <= 0 {
		return 512
	}

	suffix := strings.ToLower(strings.TrimSpace(s[i:]))
	switch suffix {
	case "b":
		return max(int(n/(1024*1024)), 1)
	case "k", "kb":
		return max(int(n/1024), 1)
	case "ki", "kib":
		return max(int(n/1024), 1)
	case "m", "mb":
		return int(n)
	case "mi", "mib":
		return int(n)
	case "g", "gb":
		return int(n * 1024)
	case "gi", "gib":
		return int(n * 1024)
	case "t", "tb":
		return int(n * 1024 * 1024)
	case "ti", "tib":
		return int(n * 1024 * 1024)
	default:
		return 512
	}
}
