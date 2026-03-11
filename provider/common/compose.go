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
	CloudRun    *CloudRunConfig
}

// ServicePortConfig defines a port configuration.
// Mirrors ServicePortConfig from compose.ts.
type ServicePortConfig struct {
	Target      int
	Mode        string // "host" or "ingress"
	Protocol    string // "tcp" or "udp"
	AppProtocol string // "http", "http2", "grpc"
}

// DeployConfig defines resolved deployment configuration.
// Mirrors DeployConfig/Resources from compose.ts.
type DeployConfig struct {
	Replicas  int
	CPUs      float64
	MemoryMiB int
}

// PostgresConfig defines resolved managed Postgres configuration.
// Mirrors x-defang-postgres from compose.ts.
type PostgresConfig struct {
	Version             int
	DBName              string
	Username            string
	Password            string
	AvailabilityType    string
	BackupEnabled       bool
	PointInTimeRecovery bool
	SslMode             string
	DeletionProtection  bool
	AllowBurstable      bool
}

// CloudRunConfig defines resolved Cloud Run configuration.
type CloudRunConfig struct {
	Ingress     string
	LaunchStage string
	MaxReplicas int // 0 means use deploy.replicas
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
	if s.Deploy != nil && s.Deploy.Replicas > 0 {
		return s.Deploy.Replicas
	}
	return 1
}

// GetCPUs returns the CPU reservation, defaulting to 0.25.
func (s ServiceConfig) GetCPUs() float64 {
	if s.Deploy != nil && s.Deploy.CPUs > 0 {
		return s.Deploy.CPUs
	}
	return 0.25
}

// GetMemoryMiB returns the memory reservation in MiB, defaulting to 512.
func (s ServiceConfig) GetMemoryMiB() int {
	if s.Deploy != nil && s.Deploy.MemoryMiB > 0 {
		return s.Deploy.MemoryMiB
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
