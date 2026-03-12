// Package shared contains Pulumi-tagged input/output types used across all defang plugins.
// Each plugin's schema will generate its own copy of these types with cloud-specific tokens.
package shared

// ServiceInput defines the configuration for a single service.
// YAML tags are aligned with Docker Compose service spec where possible.
type ServiceInput struct {
	// Build configuration (mutually exclusive with image for source of truth)
	Build *BuildInput `pulumi:"build,optional" yaml:"build,omitempty"`

	// Container image to deploy (required if no build config)
	Image *string `pulumi:"image,optional" yaml:"image,omitempty"`

	// Target platform: "linux/amd64" or "linux/arm64"
	Platform *string `pulumi:"platform,optional" yaml:"platform,omitempty"`

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

	// Managed Postgres: presence enables managed postgres. Matches x-defang-postgres extension.
	Postgres *PostgresInput `pulumi:"postgres,optional" yaml:"x-defang-postgres,omitempty"`

	// Health check configuration
	HealthCheck *HealthCheckConfig `pulumi:"healthCheck,optional" yaml:"healthcheck,omitempty"`

	// Custom domain name
	DomainName *string `pulumi:"domainName,optional" yaml:"domainname,omitempty"`
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

// BuildInput mirrors the Docker Compose build spec.
type BuildInput struct {
	// Build context path or URL (required)
	Context string `pulumi:"context" yaml:"context"`

	// Dockerfile path relative to context (default: "Dockerfile")
	Dockerfile *string `pulumi:"dockerfile,optional" yaml:"dockerfile,omitempty"`

	// Build arguments
	Args map[string]string `pulumi:"args,optional" yaml:"args,omitempty"`

	// Shared memory size (used for build task memory sizing)
	ShmSize *string `pulumi:"shmSize,optional" yaml:"shm_size,omitempty"`

	// Multi-stage build target
	Target *string `pulumi:"target,optional" yaml:"target,omitempty"`
}

// PostgresInput matches the x-defang-postgres Compose extension.
// Version is derived from image tag; DBName/Username/Password from env vars.
type PostgresInput struct {
	// Allow applying changes that cause downtime (default: recipe-controlled)
	AllowDowntime *bool `pulumi:"allowDowntime,optional" yaml:"allow-downtime,omitempty"`

	// Restore from a snapshot identifier
	FromSnapshot *string `pulumi:"fromSnapshot,optional" yaml:"from-snapshot,omitempty"`
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

// AWSConfigInput defines optional AWS-specific infrastructure configuration (not auth/region).
type AWSConfigInput struct {
	VpcID            string   `pulumi:"vpcId,optional"`
	SubnetIDs        []string `pulumi:"subnetIds,optional"`
	PrivateSubnetIDs []string `pulumi:"privateSubnetIds,optional"`
}
