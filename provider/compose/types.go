// Package compose contains Pulumi-tagged input/output types used across all defang plugins.
// Each plugin's schema will generate its own copy of these types with cloud-specific tokens.
package compose

import (
	"regexp"
	"strings"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

// ptr returns a pointer to v. Duplicates common.Ptr because common imports
// compose, so compose can't import common.
func ptr[T any](v T) *T { return &v }

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

	// Container image to deploy (required if no build config). An Input so
	// callers can pass the Output of an image build; compose files always
	// yield a literal pulumi.String (see StaticImage). yaml:"-" because the
	// yaml decoder cannot populate an interface field — see UnmarshalYAML.
	Image pulumi.StringInput `pulumi:"image,optional" yaml:"-"`

	// Target platform: "linux/amd64" or "linux/arm64"
	Platform *string `pulumi:"platform,optional" yaml:"platform,omitempty"`

	// Port configurations
	Ports []ServicePortConfig `pulumi:"ports,optional" yaml:"ports,omitempty"`

	// Deployment configuration (replicas, resources)
	Deploy *DeployConfig `pulumi:"deploy,optional" yaml:"deploy,omitempty"`

	// Environment variables
	Environment Environment `pulumi:"environment,optional" yaml:"environment,omitempty"`

	// Command to run
	Command []string `pulumi:"command,optional" yaml:"command,omitempty"`

	// Entrypoint override
	Entrypoint []string `pulumi:"entrypoint,optional" yaml:"entrypoint,omitempty"`

	// Managed Postgres: presence enables managed postgres. Matches x-defang-postgres extension.
	Postgres *PostgresConfig `pulumi:"postgres,optional" yaml:"x-defang-postgres,omitempty"`

	// Managed Redis: presence enables managed Redis. Matches x-defang-redis extension.
	Redis *RedisConfig `pulumi:"redis,optional" yaml:"x-defang-redis,omitempty"`

	// Restart policy (e.g. "no" for Cloud Run jobs)
	Restart string `pulumi:"restart,optional" yaml:"restart,omitempty"`

	// Health check configuration
	HealthCheck *HealthCheckConfig `pulumi:"healthCheck,optional" yaml:"healthcheck,omitempty"`

	// Custom domain name
	DomainName string `pulumi:"domainName,optional" yaml:"domainname,omitempty"`

	Networks map[NetworkID]ServiceNetworkConfig `pulumi:"networks,optional" yaml:"networks,omitempty"`

	DependsOn DependsOnConfig `pulumi:"dependsOn,optional" yaml:"depends_on,omitempty"`

	LLM *LlmConfig `pulumi:"llm,optional" yaml:"x-defang-llm,omitempty"`

	// Container name override (default: the service name)
	ContainerName *string `pulumi:"containerName,optional" yaml:"container_name,omitempty"`

	// Time to wait for the container to stop before killing it (Go duration, e.g. "120s")
	StopGracePeriod *string `pulumi:"stopGracePeriod,optional" yaml:"stop_grace_period,omitempty"`

	// Named volumes mounted into the container
	Volumes []ServiceVolumeConfig `pulumi:"volumes,optional" yaml:"volumes,omitempty"`

	// Mount all volumes from another service/container; entries are container
	// names with an optional ":ro" or ":rw" suffix
	VolumesFrom []string `pulumi:"volumesFrom,optional" yaml:"volumes_from,omitempty"`

	// Working directory inside the container
	WorkingDir *string `pulumi:"workingDir,optional" yaml:"working_dir,omitempty"`

	// Network mode; "service:<name>" folds this service into <name>'s task as
	// a sidecar container instead of deploying it standalone. Other values are
	// ignored.
	NetworkMode string `pulumi:"networkMode,optional" yaml:"network_mode,omitempty"`

	// Enable autoscaling. Matches the x-defang-autoscaling extension.
	Autoscaling bool `pulumi:"autoscaling,optional" yaml:"x-defang-autoscaling,omitempty"`

	// Extra IAM policies to attach to the task role created for this service;
	// each entry is a full policy ARN or a customer-managed policy name.
	// Matches the x-defang-policies extension. AWS-only: other providers
	// reject it, and it cannot be combined with a caller-supplied task role.
	Policies []string `pulumi:"policies,optional" yaml:"x-defang-policies,omitempty"`

	// Models map[string]*ServiceModelConfig `pulumi:"models,optional" yaml:"models,omitempty"`
}

// UnmarshalYAML decodes a service, converting the literal "image" field into a
// pulumi.String (the Image field is a pulumi.StringInput, which the yaml
// decoder cannot populate directly).
func (s *ServiceConfig) UnmarshalYAML(value *yaml.Node) error {
	type raw ServiceConfig // methodless alias to avoid recursion
	if err := value.Decode((*raw)(s)); err != nil {
		return err
	}
	var img struct {
		Image *string `yaml:"image"`
	}
	if err := value.Decode(&img); err != nil {
		return err
	}
	s.Image = ImageFromPtr(img.Image)
	return nil
}

// ImageFromPtr converts an optional literal image string to the Image input type.
func ImageFromPtr(p *string) pulumi.StringInput {
	if p == nil {
		return nil
	}
	return pulumi.String(*p)
}

// Environment maps env var names to values. Values are Inputs so callers can
// pass Outputs of other resources; compose files always yield literal
// pulumi.String values, or nil for "KEY:" with no value (resolve from config
// at deploy time). See StaticEnvValue.
type Environment map[string]pulumi.StringInput

// UnmarshalYAML decodes the compose map form, converting literal values into
// pulumi.String (the yaml decoder cannot populate an interface value).
func (e *Environment) UnmarshalYAML(value *yaml.Node) error {
	var m map[string]*string
	if err := value.Decode(&m); err != nil {
		return err
	}
	if m == nil {
		*e = nil
		return nil
	}
	env := make(Environment, len(m))
	for k, v := range m {
		env[k] = ImageFromPtr(v)
	}
	*e = env
	return nil
}

// StaticEnvValue returns the literal value of an environment entry and whether
// it is static. A nil input (compose "KEY:" with no value, i.e. resolve from
// config) is static with a nil pointer; Outputs of other resources are not
// static and return (nil, false).
func StaticEnvValue(v pulumi.StringInput) (*string, bool) {
	switch t := v.(type) {
	case nil:
		return nil, true
	case pulumi.String:
		s := string(t)
		return &s, true
	default:
		return nil, false
	}
}

// SidecarParent returns the service name this config attaches to as a sidecar
// (network_mode: "service:<name>"), or "" for a standalone service.
func (s ServiceConfig) SidecarParent() string {
	if parent, ok := strings.CutPrefix(s.NetworkMode, "service:"); ok {
		return parent
	}
	return ""
}

// StaticImage returns the image as a literal string when it was set from a
// compose file or plain string (pulumi.String), or nil when unset or dynamic.
func (s ServiceConfig) StaticImage() *string {
	if img, ok := s.Image.(pulumi.String); ok {
		v := string(img)
		return &v
	}
	return nil
}

// ServiceVolumeConfig defines a named-volume mount (long syntax). Bind mounts
// are not supported.
type ServiceVolumeConfig struct {
	// Volume name
	Source string `pulumi:"source" yaml:"source"`

	// Mount path inside the container
	Target string `pulumi:"target" yaml:"target"`

	ReadOnly bool `pulumi:"readOnly,optional" yaml:"read_only,omitempty"`
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

type PortMode string

const (
	PortModeHost    PortMode = "host"
	PortModeIngress PortMode = "ingress"
)

type PortProtocol string

const (
	PortProtocolAny  PortProtocol = ""
	PortProtocolTCP  PortProtocol = "tcp"
	PortProtocolUDP  PortProtocol = "udp"
	PortProtocolSCTP PortProtocol = "sctp"
)

type PortAppProtocol string

const (
	PortAppProtocolUnknown PortAppProtocol = ""
	PortAppProtocolHTTP    PortAppProtocol = "http"
	PortAppProtocolHTTP2   PortAppProtocol = "http2"
	PortAppProtocolGRPC    PortAppProtocol = "grpc"
)

// ServicePortConfig defines a port mapping for a service.
type ServicePortConfig struct {
	// Container port
	Target int32 `pulumi:"target" yaml:"target"`

	// Port mode: "host" or "ingress" (default: "ingress")
	Mode PortMode `pulumi:"mode,optional" yaml:"mode,omitempty"`

	// Transport protocol: "tcp" or "udp" (default: "tcp")
	Protocol PortProtocol `pulumi:"protocol,optional" yaml:"protocol,omitempty"`

	// Application protocol: "http", "http2", "grpc" (default: "http")
	AppProtocol PortAppProtocol `pulumi:"appProtocol,optional" yaml:"app_protocol,omitempty"`

	// Force the load-balancer listener protocol: "http" or "https" (default:
	// derived from the port). Matches the x-defang-listener extension.
	Listener PortListenerProtocol `pulumi:"listener,optional" yaml:"x-defang-listener,omitempty"`
}

type PortListenerProtocol string

const (
	PortListenerDefault PortListenerProtocol = ""
	PortListenerHTTP    PortListenerProtocol = "http"
	PortListenerHTTPS   PortListenerProtocol = "https"
)

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

	// Target platforms for multi-platform builds, e.g. ["linux/amd64", "linux/arm64"]
	Platforms []string `pulumi:"platforms,optional" yaml:"platforms,omitempty"`

	// External cache sources, passed through to `docker buildx build --cache-from`
	// (e.g. "type=registry,ref=user/app:cache")
	CacheFrom []string `pulumi:"cacheFrom,optional" yaml:"cache_from,omitempty"`

	// External cache destinations, passed through to `docker buildx build --cache-to`
	// (e.g. "type=registry,mode=max,ref=user/app:cache")
	CacheTo []string `pulumi:"cacheTo,optional" yaml:"cache_to,omitempty"`
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
		Platforms  []string          `yaml:"platforms,omitempty"`
		CacheFrom  []string          `yaml:"cache_from,omitempty"`
		CacheTo    []string          `yaml:"cache_to,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	b.Context = pulumi.String(raw.Context)
	b.Dockerfile = raw.Dockerfile
	b.Args = raw.Args
	b.ShmSize = raw.ShmSize
	b.Target = raw.Target
	b.Platforms = raw.Platforms
	b.CacheFrom = raw.CacheFrom
	b.CacheTo = raw.CacheTo
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

// PostgresConfigArgs holds resolved managed Postgres configuration.
// Derived from PostgresConfig + image tag + environment variables.
//
// Deprecated: only used internally.
type PostgresConfigArgs struct {
	Version       pulumi.StringPtrInput // Major version (derived from image tag, e.g. postgres:16 → 16)
	DBName        pulumi.StringInput    // From POSTGRES_DB env or default "postgres"
	DBNameStr     string                // Plain string for conditional resource creation (no resources inside ApplyT)
	Username      pulumi.StringInput    // From POSTGRES_USER env or default "postgres"
	Password      pulumi.StringInput    // From POSTGRES_PASSWORD env
	AllowDowntime bool                  // From x-defang-postgres "allow-downtime"
	FromSnapshot  string                // From x-defang-postgres "from-snapshot"
}

// see https://hub.docker.com/_/postgres for default values
const DEFAULT_POSTGRES_USER = "postgres"
const DEFAULT_POSTGRES_DB = "postgres"

var rePostgresVersion = regexp.MustCompile(`^(?:[\d.-]*pg)?([\d.]+(?:-rds\.\d+)?)`)

func getPostgresVersion(image string) string {
	// Strip image name prefix (e.g. "postgres:14.22" → "14.22")
	if i := strings.LastIndex(image, ":"); i >= 0 {
		image = image[i+1:]
	}
	matches := rePostgresVersion.FindStringSubmatch(image)
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
	if img := s.StaticImage(); img != nil {
		version = pulumi.StringPtr(getPostgresVersion(*img))
	}

	// POSTGRES_DB drives resource creation decisions, so it must be a static
	// string; a dynamic (Output) value falls back to the default name.
	dbNameStr, _ := StaticEnvValue(s.Environment["POSTGRES_DB"])
	if dbNameStr == nil || *dbNameStr == "" {
		dbNameStr = ptr(DEFAULT_POSTGRES_DB)
	}
	dbName := GetConfigOrEnvValue(ctx, configProvider, s, "POSTGRES_DB", DEFAULT_POSTGRES_DB)
	username := GetConfigOrEnvValue(ctx, configProvider, s, "POSTGRES_USER", DEFAULT_POSTGRES_USER)
	password := GetConfigOrEnvValue(ctx, configProvider, s, "POSTGRES_PASSWORD", "") // FIXME: should not default to ""

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
		DBNameStr:     *dbNameStr,
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

// GetContainerName returns the container name, defaulting to fallback (the service name).
func (s ServiceConfig) GetContainerName(fallback string) string {
	if s.ContainerName != nil && *s.ContainerName != "" {
		return *s.ContainerName
	}
	return fallback
}

// GetStopGracePeriodSeconds returns the stop grace period in seconds, or 0 if unset.
func (s ServiceConfig) GetStopGracePeriodSeconds() int {
	if s.StopGracePeriod == nil {
		return 0
	}
	return int(parseDurationSeconds(*s.StopGracePeriod))
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

func (s ServiceConfig) ResolvedEnvironment() map[string]string {
	env := make(map[string]string, len(s.Environment))
	for k, v := range s.Environment {
		sv, static := StaticEnvValue(v)
		switch {
		case !static:
			// dynamic values never occur on this YAML-driven path
		case sv != nil:
			env[k] = *sv
		default:
			env[k] = "${" + k + "}" // preserve undefined env vars as placeholders
		}
	}
	return env
}

// HasIngressPorts returns true if any port has mode "ingress".
func (s ServiceConfig) HasIngressPorts() bool {
	for _, p := range s.Ports {
		if p.IsIngress() {
			return true
		}
	}
	return false
}

// HasHostPorts returns true if any port has mode "host".
func (s ServiceConfig) HasHostPorts() bool {
	for _, p := range s.Ports {
		if p.IsHost() {
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
