package provider

import (
	"regexp"
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
	Provider string `pulumi:"providerId" yaml:"provider"`

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
	// Container image to deploy (required if no build config)
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

// postgresVersionRe extracts version from image tags like "16", "16.3-bookworm", "pg16", "0.8.0-pg17".
var postgresVersionRe = regexp.MustCompile(`^(?:[\d.-]*pg)?([\d.]+)`)

// getPostgresVersion extracts the postgres major version from an image tag.
// Returns 0 if the tag can't be parsed (caller should default to latest).
func getPostgresVersion(tag string) int {
	m := postgresVersionRe.FindStringSubmatch(tag)
	if m == nil {
		return 0
	}
	// Take just the major version (first component before any dot)
	ver := m[1]
	if dot := strings.IndexByte(ver, '.'); dot >= 0 {
		ver = ver[:dot]
	}
	n, err := strconv.Atoi(ver)
	if err != nil {
		return 0
	}
	return n
}

// parseImageTag splits "repo:tag" and returns the tag portion (empty string if no tag).
func parseImageTag(image string) string {
	// Handle digest references like "repo@sha256:..."
	if at := strings.IndexByte(image, '@'); at >= 0 {
		return ""
	}
	if colon := strings.LastIndexByte(image, ':'); colon >= 0 {
		// Make sure we're not splitting on a port in the registry host
		afterColon := image[colon+1:]
		if !strings.Contains(afterColon, "/") {
			return afterColon
		}
	}
	return ""
}
