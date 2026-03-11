package common

// ServiceConfig defines the resolved configuration for a single service.
// Mirrors the ServiceConfig type from compose.ts.
type ServiceConfig struct {
	Image       *string
	Ports       []ServicePortConfig
	Deploy      *DeployConfig
	Environment map[string]string
	Command     []string
	Entrypoint  []string
	Postgres    *PostgresConfig
	HealthCheck *HealthCheckConfig
	DomainName  *string
}

// ServicePortConfig defines a port configuration.
// Mirrors ServicePortConfig from compose.ts.
type ServicePortConfig struct {
	Target      int
	Mode        string // "host" or "ingress"
	Protocol    string // "tcp" or "udp"
	AppProtocol string // "http", "http2", "grpc"
}

// DeployConfig mirrors the Docker Compose deploy spec.
type DeployConfig struct {
	Replicas  *int
	Resources *ResourcesConfig
}

// ResourcesConfig mirrors the Docker Compose deploy.resources spec.
type ResourcesConfig struct {
	Reservations *ResourceConfig
	Limits       *ResourceConfig
}

// ResourceConfig defines CPU and memory for a resource bound.
type ResourceConfig struct {
	CPUs      *float64
	MemoryMiB *int
}

// PostgresConfig defines resolved managed Postgres configuration.
// Only contains Compose-level fields; cloud-specific tuning lives in each provider's Recipe.
type PostgresConfig struct {
	Version       int    // Major version (derived from image tag, e.g. postgres:16 → 16)
	DBName        string // From POSTGRES_DB env or default "postgres"
	Username      string // From POSTGRES_USER env or default "postgres"
	Password      string // From POSTGRES_PASSWORD env
	AllowDowntime bool   // From x-defang-postgres "allow-downtime"
	FromSnapshot  string // From x-defang-postgres "from-snapshot"
}

// HealthCheckConfig defines health check configuration.
// Mirrors HealthCheckConfig from compose.ts.
type HealthCheckConfig struct {
	Test               []string
	IntervalSeconds    *int
	TimeoutSeconds     *int
	Retries            *int
	StartPeriodSeconds *int
}

// GetImage returns the container image, defaulting to "nginx:latest".
func (s ServiceConfig) GetImage() string {
	if s.Image != nil {
		return *s.Image
	}
	return "nginx:latest"
}

// GetReplicas returns the replica count, defaulting to 1.
func (s ServiceConfig) GetReplicas() int {
	if s.Deploy != nil && s.Deploy.Replicas != nil && *s.Deploy.Replicas > 0 {
		return *s.Deploy.Replicas
	}
	return 1
}

// GetCPUs returns the CPU reservation, defaulting to 0.25.
func (s ServiceConfig) GetCPUs() float64 {
	if s.Deploy != nil && s.Deploy.Resources != nil && s.Deploy.Resources.Reservations != nil && s.Deploy.Resources.Reservations.CPUs != nil {
		return *s.Deploy.Resources.Reservations.CPUs
	}
	return 0.25
}

// GetMemoryMiB returns the memory reservation in MiB, defaulting to 512.
func (s ServiceConfig) GetMemoryMiB() int {
	if s.Deploy != nil && s.Deploy.Resources != nil && s.Deploy.Resources.Reservations != nil && s.Deploy.Resources.Reservations.MemoryMiB != nil {
		return *s.Deploy.Resources.Reservations.MemoryMiB
	}
	return 512
}

// HasIngressPorts returns true if any port has mode "ingress".
func (s ServiceConfig) HasIngressPorts() bool {
	for _, p := range s.Ports {
		if p.Mode == "ingress" {
			return true
		}
	}
	return false
}
