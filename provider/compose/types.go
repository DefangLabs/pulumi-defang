// Package compose contains Pulumi-tagged input/output types used across all defang plugins.
// Each plugin's schema will generate its own copy of these types with cloud-specific tokens.
package compose

import (
	"regexp"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

type NetworkID string

type Services = map[string]ServiceConfig

type Networks map[NetworkID]NetworkConfig

const DefaultNetwork NetworkID = "default"

// Project defines the top-level project configuration, including services and networks.
// This is a subset of the full types.Project from compose-go, from
// https://github.com/compose-spec/compose-go/blob/main/types/project.go
type Project struct {
	Name string `pulumi:"name,optional" yaml:"name,omitempty"`

	// Services map: name -> service config
	Services Services `pulumi:"services" yaml:"services"`

	// Networks map: name -> network config (only needed if using custom networks in Compose)
	Networks Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
}

// ServiceConfig defines the configuration for a single service.
// YAML tags are aligned with Docker Compose spec where possible.
type ServiceConfig struct {
	// Build configuration
	Build *BuildConfig `pulumi:"build,optional" yaml:"build,omitempty"`

	// Container image to deploy (required if no build config)
	Image *string `pulumi:"image,optional" yaml:"image,omitempty"`

	// Target platform: "linux/amd64" or "linux/arm64"
	Platform *string `pulumi:"platform,optional" yaml:"platform,omitempty"`

	// Port configurations
	Ports []ServicePortConfig `pulumi:"ports,optional" yaml:"ports,omitempty"`

	// Deployment configuration (replicas, resources)
	Deploy *DeployConfig `pulumi:"deploy,optional" yaml:"deploy,omitempty"`

	// Environment variables
	Environment map[string]string `pulumi:"environment,optional" yaml:"environment,omitempty"`

	// Command to run
	Command []string `pulumi:"command,optional" yaml:"command,omitempty"`

	// Entrypoint override
	Entrypoint []string `pulumi:"entrypoint,optional" yaml:"entrypoint,omitempty"`

	// Managed Postgres: presence enables managed postgres. Matches x-defang-postgres extension.
	Postgres *PostgresConfig `pulumi:"postgres,optional" yaml:"x-defang-postgres,omitempty"`

	// Managed Redis: presence enables managed Redis. Matches x-defang-redis extension.
	Redis *RedisConfig `pulumi:"redis,optional" yaml:"x-defang-redis,omitempty"`

	// Health check configuration
	HealthCheck *HealthCheckConfig `pulumi:"healthCheck,optional" yaml:"healthcheck,omitempty"`

	// Custom domain name
	DomainName string `pulumi:"domainName,optional" yaml:"domainname,omitempty"`

	Networks map[NetworkID]ServiceNetworkConfig `pulumi:"networks,optional" yaml:"networks,omitempty"`

	DependsOn DependsOnConfig `pulumi:"dependsOn,optional" yaml:"depends_on,omitempty"`

	LLM *LlmConfig `pulumi:"llm,optional" yaml:"x-defang-llm,omitempty"`

	// Models map[string]*ServiceModelConfig `pulumi:"models,optional" yaml:"models,omitempty"`
}

type LlmConfig struct{}

// UnmarshalYAML allows `x-defang-llm: true` (bare boolean) in addition to an object.
func (c *LlmConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return nil // bare true/false just enables with defaults
	}
	type raw LlmConfig
	return value.Decode((*raw)(c))
}

// type ServiceModelConfig struct {}

type DependsOnConfig map[string]ServiceDependency

type NetworkConfig struct {
	Internal bool `pulumi:"internal,optional" yaml:"internal,omitempty"`
	//   IPAM *IPAMConfigInput `pulumi:"ipam,optional" yaml:"ipam,omitempty"`
}

type ServiceDependency struct {
	// Condition is one of "service_healthy" | "service_started" (default) | "service_completed_successfully"
	Condition string `pulumi:"condition,optional" yaml:"condition,omitempty"`
	Required  bool   `pulumi:"required,optional"  yaml:"required,omitempty"`
}

type ServiceNetworkConfig struct {
	Aliases []string `pulumi:"aliases,optional" yaml:"aliases,omitempty"`
}

// ServicePortConfig defines a port mapping for a service.
type ServicePortConfig struct {
	// Container port
	Target int32 `pulumi:"target" yaml:"target"`

	// Port mode: "host" or "ingress" (default: "ingress")
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
	Replicas *int32 `pulumi:"replicas,optional" yaml:"replicas,omitempty"`

	// Resource reservations and limits
	Resources *Resources `pulumi:"resources,optional" yaml:"resources,omitempty"`
}

// Resources defines resource reservations and limits.
// Mirrors Docker Compose deploy.resources spec.
type Resources struct {
	// Resource reservations (guaranteed minimums)
	Reservations *ResourceConfig `pulumi:"reservations,optional" yaml:"reservations,omitempty"`

	// Resource limits (hard caps)
	// Limits *ResourceConfig `pulumi:"limits,optional" yaml:"limits,omitempty"`
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

// BuildConfig mirrors the Docker Compose build spec.
type BuildConfig struct {
	// Build context path or URL (required). May be a computed Pulumi output (e.g. S3 URL).
	Context pulumi.StringInput `pulumi:"context"`

	// Dockerfile path relative to context (default: "Dockerfile")
	Dockerfile *string `pulumi:"dockerfile,optional" yaml:"dockerfile,omitempty"`

	// Build arguments
	Args map[string]string `pulumi:"args,optional" yaml:"args,omitempty"`

	// Shared memory size (used for build task memory sizing)
	ShmSize *string `pulumi:"shmSize,optional" yaml:"shm_size,omitempty"`

	// Multi-stage build target
	Target *string `pulumi:"target,optional" yaml:"target,omitempty"`
}

// UnmarshalYAML allows BuildConfig to be parsed from YAML, converting the
// string "context" field into a pulumi.String.
func (b *BuildConfig) UnmarshalYAML(value *yaml.Node) error {
	// Use a plain struct to avoid infinite recursion.
	var raw struct {
		Context    string            `yaml:"context"`
		Dockerfile *string           `yaml:"dockerfile,omitempty"`
		Args       map[string]string `yaml:"args,omitempty"`
		ShmSize    *string           `yaml:"shm_size,omitempty"`
		Target     *string           `yaml:"target,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	b.Context = pulumi.String(raw.Context)
	b.Dockerfile = raw.Dockerfile
	b.Args = raw.Args
	b.ShmSize = raw.ShmSize
	b.Target = raw.Target
	return nil
}

// PostgresConfig matches the x-defang-postgres Compose extension.
// Version is derived from image tag; DBName/Username/Password from env vars.
type PostgresConfig struct {
	// Allow applying changes that cause downtime (default: recipe-controlled)
	AllowDowntime *bool `pulumi:"allowDowntime,optional" yaml:"allow-downtime,omitempty"`

	// Restore from a snapshot identifier
	FromSnapshot *string `pulumi:"fromSnapshot,optional" yaml:"from-snapshot,omitempty"`
}

// UnmarshalYAML allows `x-defang-postgres: true` (bare boolean) in addition to an object.
func (c *PostgresConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return nil // bare true/false just enables with defaults
	}
	type raw PostgresConfig
	return value.Decode((*raw)(c))
}

// RedisConfig matches the x-defang-redis Compose extension.
// Version is derived from image tag; FromSnapshot allows restoring from a backup.
type RedisConfig struct {
	AllowDowntime *bool `pulumi:"allowDowntime,optional" yaml:"allow-downtime,omitempty"`

	FromSnapshot *string `pulumi:"fromSnapshot,optional" yaml:"from-snapshot,omitempty"`
}

// UnmarshalYAML allows `x-defang-redis: true` (bare boolean) in addition to an object.
func (c *RedisConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return nil // bare true/false just enables with defaults
	}
	type raw RedisConfig
	return value.Decode((*raw)(c))
}

// HealthCheckConfig defines health check parameters.
// YAML tags match Docker Compose healthcheck spec.
type HealthCheckConfig struct {
	// Health check command
	Test []string `pulumi:"test,optional" yaml:"test,omitempty"`

	// Check interval in seconds (default: 30s)
	IntervalSeconds int32 `pulumi:"intervalSeconds,optional" yaml:"interval,omitempty"`

	// Check timeout in seconds (default: 5s)
	TimeoutSeconds int32 `pulumi:"timeoutSeconds,optional" yaml:"timeout,omitempty"`

	// Number of retries before marking unhealthy (default: 3)
	Retries int32 `pulumi:"retries,optional" yaml:"retries,omitempty"`

	// Grace period before health checks start in seconds (default: 0s)
	StartPeriodSeconds int32 `pulumi:"startPeriodSeconds,optional" yaml:"start_period,omitempty"`
}

// UnmarshalYAML parses Docker Compose duration strings (e.g. "5s", "30s") into seconds.
func (h *HealthCheckConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Test        []string `yaml:"test,omitempty"`
		Interval    string   `yaml:"interval,omitempty"`
		Timeout     string   `yaml:"timeout,omitempty"`
		Retries     int32    `yaml:"retries,omitempty"`
		StartPeriod string   `yaml:"start_period,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	h.Test = raw.Test
	h.Retries = raw.Retries
	h.IntervalSeconds = int32(parseDurationSeconds(raw.Interval))
	h.TimeoutSeconds = int32(parseDurationSeconds(raw.Timeout))
	h.StartPeriodSeconds = int32(parseDurationSeconds(raw.StartPeriod))
	return nil
}

func parseDurationSeconds(s string) float64 {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d.Seconds()
}

type ConfigProvider interface {
	GetConfig(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) pulumi.StringOutput
}

// PostgresConfigArgs holds resolved managed Postgres configuration.
// Derived from PostgresConfig + image tag + environment variables.
//
// Deprecated: only used internally.
type PostgresConfigArgs struct {
	Version       pulumi.StringPtrInput // Major version (derived from image tag, e.g. postgres:16 → 16)
	DBName        pulumi.StringInput    // From POSTGRES_DB env or default "postgres"
	Username      pulumi.StringInput    // From POSTGRES_USER env or default "postgres"
	Password      pulumi.StringInput    // From POSTGRES_PASSWORD env
	AllowDowntime bool                  // From x-defang-postgres "allow-downtime"
	FromSnapshot  string                // From x-defang-postgres "from-snapshot"
}

// see https://hub.docker.com/_/postgres for default values
const DEFAULT_POSTGRES_USER = "postgres"
const DEFAULT_POSTGRES_DB = "postgres"

var rePostgresVersion = regexp.MustCompile(`^(?:[\d.-]*pg)?([\d.]+(?:-rds\.\d+)?)`)

func getPostgresVersion(tag string) string {
	// If the tag starts with a pgvector version, cut if off
	matches := rePostgresVersion.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ResolvePostgres derives PostgresConfig from the service's postgres extension, image tag, and env vars.
func (s ServiceConfig) ResolvePostgres(ctx *pulumi.Context, configProvider ConfigProvider) *PostgresConfigArgs {
	if s.Postgres == nil {
		return nil
	}

	var version pulumi.StringPtrInput
	if s.Image != nil {
		version = pulumi.StringPtr(getPostgresVersion(*s.Image))
	}

	dbName := GetConfigOrEnvValue(ctx, configProvider, s, "POSTGRES_DB", DEFAULT_POSTGRES_DB)
	username := GetConfigOrEnvValue(ctx, configProvider, s, "POSTGRES_USER", DEFAULT_POSTGRES_USER)
	password := GetConfigOrEnvValue(ctx, configProvider, s, "POSTGRES_PASSWORD", "")

	allowDowntime := false
	if s.Postgres.AllowDowntime != nil {
		allowDowntime = *s.Postgres.AllowDowntime
	}
	fromSnapshot := ""
	if s.Postgres.FromSnapshot != nil {
		fromSnapshot = *s.Postgres.FromSnapshot
	}

	return &PostgresConfigArgs{
		Version:       version,
		DBName:        dbName,
		Username:      username,
		Password:      password,
		AllowDowntime: allowDowntime,
		FromSnapshot:  fromSnapshot,
	}
}

// GetReplicas returns the replica count, defaulting to 1.
func (s ServiceConfig) GetReplicas() int32 {
	if s.Deploy != nil && s.Deploy.Replicas != nil && *s.Deploy.Replicas > 0 {
		return *s.Deploy.Replicas
	}
	return 1
}

// HasResourceReservations returns true if the service has explicit CPU or memory reservations.
func (s ServiceConfig) HasResourceReservations() bool {
	return s.Deploy != nil &&
		s.Deploy.Resources != nil &&
		s.Deploy.Resources.Reservations != nil
}

// GetCPUs returns the CPU reservation, defaulting to 0.25.
func (s ServiceConfig) GetCPUs() float64 {
	if s.Deploy != nil &&
		s.Deploy.Resources != nil &&
		s.Deploy.Resources.Reservations != nil &&
		s.Deploy.Resources.Reservations.CPUs != nil {
		return *s.Deploy.Resources.Reservations.CPUs
	}
	return 0.25
}

// GetMemoryMiB returns the memory reservation in MiB, defaulting to 512.
func (s ServiceConfig) GetMemoryMiB() int {
	if s.Deploy != nil && s.Deploy.Resources != nil &&
		s.Deploy.Resources.Reservations != nil &&
		s.Deploy.Resources.Reservations.Memory != nil {
		return ParseMemoryMiB(*s.Deploy.Resources.Reservations.Memory)
	}
	return 512
}

// NeedsBuild returns true if the service has a build config.
func (s ServiceConfig) NeedsBuild() bool {
	return s.Build != nil
}

// GetPlatform returns the platform, defaulting to "linux/amd64".
func (s ServiceConfig) GetPlatform() string {
	if s.Platform != nil {
		return *s.Platform
	}
	return "linux/amd64"
}

// DefaultNetwork returns the default network config for the service, defaulting to empty config.
func (s ServiceConfig) DefaultNetwork() ServiceNetworkConfig {
	return s.Networks[DefaultNetwork]
}

// HasIngressPorts returns true if any port has mode "ingress".
func (s ServiceConfig) HasIngressPorts() bool {
	for _, p := range s.Ports {
		if p.Mode == "ingress" || p.Mode == "" { // default to "ingress" if mode is not set
			return true
		}
	}
	return false
}

// HasHostPorts returns true if any port has mode "host".
func (s ServiceConfig) HasHostPorts() bool {
	for _, p := range s.Ports {
		if p.Mode == "host" {
			return true
		}
	}
	return false
}

// GetDockerfile returns the Dockerfile path, defaulting to "Dockerfile".
func (b BuildConfig) GetDockerfile() string {
	if b.Dockerfile != nil {
		return *b.Dockerfile
	}
	return "Dockerfile"
}

// GetTarget returns the build target, defaulting to "".
func (b BuildConfig) GetTarget() string {
	if b.Target != nil {
		return *b.Target
	}
	return ""
}

// GetShmSizeBytes returns the shared memory size in bytes, defaulting to 0.
func (b BuildConfig) GetShmSizeBytes() int {
	if b.ShmSize != nil {
		return ParseMemoryMiB(*b.ShmSize) * 1024 * 1024
	}
	return 0
}
