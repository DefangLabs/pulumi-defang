package aws

// ContainerPortMapping defines a port mapping for a container.
type ContainerPortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"` // "tcp" or "udp"
	Name          string `json:"name,omitempty"`     // only for Service Connect
}

// ContainerEnvironment defines an environment variable for a container.
type ContainerEnvironment struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ContainerHealthCheck defines health check parameters for a container.
type ContainerHealthCheck struct {
	Command     []string `json:"command"`               // eg. ["CMD-SHELL", "curl -f http://localhost/ || exit 1"]
	Interval    int      `json:"interval,omitempty"`    // 5-300, default 30s
	Timeout     int      `json:"timeout,omitempty"`     // 2-300, default 5s
	Retries     int      `json:"retries,omitempty"`     // 1-10, default 3
	StartPeriod int      `json:"startPeriod,omitempty"` // 0-300, default 0s
}

// ContainerEnvironmentFile references an environment file in S3.
type ContainerEnvironmentFile struct {
	Type  string `json:"type"`  // "s3"
	Value string `json:"value"` // S3 ARN
}

// ContainerSecret references a secret from AWS Secrets Manager or SSM Parameter Store.
type ContainerSecret struct {
	Name      string `json:"name"`
	ValueFrom string `json:"valueFrom"` // AWS SM secret or SSM parameter ARN
}

// ContainerRepositoryCredentials references credentials for a private registry.
type ContainerRepositoryCredentials struct {
	CredentialsParameter string `json:"credentialsParameter"` // secret ARN of {"username":"…","password":"…"}
}

// ContainerResourceRequirement specifies a resource requirement (GPU or InferenceAccelerator).
type ContainerResourceRequirement struct {
	Type  string `json:"type"`  // "GPU" or "InferenceAccelerator"
	Value string `json:"value"` // GPU count or InferenceAccelerator deviceName
}

// ContainerMountPoint defines a volume mount point for a container.
type ContainerMountPoint struct {
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
	SourceVolume  string `json:"sourceVolume"`
}

// ContainerVolumeFrom mounts volumes from another container.
type ContainerVolumeFrom struct {
	ReadOnly        bool   `json:"readOnly,omitempty"`
	SourceContainer string `json:"sourceContainer"`
}

// ContainerLogConfiguration configures log routing for a container.
type ContainerLogConfiguration struct {
	LogDriver     string                  `json:"logDriver"` // "awslogs", "splunk", or "awsfirelens" for Fargate
	Options       map[string]string       `json:"options,omitempty"`
	SecretOptions []ContainerSecretOption `json:"secretOptions,omitempty"`
}

// ContainerSecretOption references a secret for log configuration.
type ContainerSecretOption struct {
	Name      string `json:"name"`
	ValueFrom string `json:"valueFrom"` // AWS SM secret or SSM parameter ARN
}

// ContainerFirelensConfiguration configures Firelens log routing.
type ContainerFirelensConfiguration struct {
	Type    string            `json:"type"` // "fluentd" or "fluentbit"
	Options map[string]string `json:"options,omitempty"`
}

// ContainerUlimit defines a ulimit override for a container.
type ContainerUlimit struct {
	HardLimit int    `json:"hardLimit"`
	Name      string `json:"name"` // "nofile", "memlock", "core", etc.
	SoftLimit int    `json:"softLimit"`
}

// ContainerLinuxCapabilities specifies Linux kernel capabilities to add or drop.
type ContainerLinuxCapabilities struct {
	Add  []string `json:"add,omitempty"`  // Fargate only supports "SYS_PTRACE"
	Drop []string `json:"drop,omitempty"`
}

// ContainerLinuxDevice specifies a host device to expose in the container.
type ContainerLinuxDevice struct {
	ContainerPath string   `json:"containerPath,omitempty"`
	HostPath      string   `json:"hostPath"`
	Permissions   []string `json:"permissions,omitempty"`
}

// ContainerLinuxParameters defines Linux-specific parameters for a container.
type ContainerLinuxParameters struct {
	Capabilities       *ContainerLinuxCapabilities `json:"capabilities,omitempty"`
	Devices            []ContainerLinuxDevice      `json:"devices,omitempty"`
	InitProcessEnabled bool                        `json:"initProcessEnabled,omitempty"`
}

// ContainerDependency defines a dependency on another container.
type ContainerDependency struct {
	ContainerName string `json:"containerName"`
	Condition     string `json:"condition"` // "START", "COMPLETE", "SUCCESS", or "HEALTHY"
}

// ContainerDefinition defines a container within an ECS task definition.
// See https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ContainerDefinition.html
type ContainerDefinition struct {
	Name                   string                          `json:"name"`
	Image                  string                          `json:"image"`
	Memory                 int                             `json:"memory,omitempty"`            // MiB; optional for Fargate
	MemoryReservation      int                             `json:"memoryReservation,omitempty"` // MiB
	Essential              *bool                           `json:"essential,omitempty"`         // default true
	PortMappings           []ContainerPortMapping          `json:"portMappings,omitempty"`
	Environment            []ContainerEnvironment          `json:"environment,omitempty"`
	HealthCheck            *ContainerHealthCheck           `json:"healthCheck,omitempty"`
	CPU                    int                             `json:"cpu,omitempty"` // vCPU; optional for Fargate
	EntryPoint             []string                        `json:"entryPoint,omitempty"`
	Command                []string                        `json:"command,omitempty"`
	WorkingDirectory       string                          `json:"workingDirectory,omitempty"`
	EnvironmentFiles       []ContainerEnvironmentFile      `json:"environmentFiles,omitempty"`
	Secrets                []ContainerSecret               `json:"secrets,omitempty"`
	ReadonlyRootFilesystem bool                            `json:"readonlyRootFilesystem,omitempty"`
	RepositoryCredentials  *ContainerRepositoryCredentials `json:"repositoryCredentials,omitempty"`
	ResourceRequirements   []ContainerResourceRequirement  `json:"resourceRequirements,omitempty"`
	MountPoints            []ContainerMountPoint           `json:"mountPoints,omitempty"`
	VolumesFrom            []ContainerVolumeFrom           `json:"volumesFrom,omitempty"`
	LogConfiguration       *ContainerLogConfiguration      `json:"logConfiguration,omitempty"`
	FirelensConfiguration  *ContainerFirelensConfiguration `json:"firelensConfiguration,omitempty"`
	Ulimits                []ContainerUlimit               `json:"ulimits,omitempty"`
	DockerLabels           map[string]string               `json:"dockerLabels,omitempty"`
	LinuxParameters        *ContainerLinuxParameters       `json:"linuxParameters,omitempty"`
	DependsOn              []ContainerDependency           `json:"dependsOn,omitempty"`
	StartTimeout           int                             `json:"startTimeout,omitempty"` // 0-3600, default 0s
	StopTimeout            int                             `json:"stopTimeout,omitempty"`  // 0-3600, default 0s
	Interactive            bool                            `json:"interactive,omitempty"`
	PseudoTerminal         bool                            `json:"pseudoTerminal,omitempty"`
}

// TaskDefinitionArgs holds the arguments for creating an ECS task definition.
type TaskDefinitionArgs struct {
	ContainerDefinitions []ContainerDefinition `json:"containerDefinitions"`
	CpuArchitecture      string                `json:"cpuArchitecture"` // "X86_64" or "ARM64"; required for Fargate
	EphemeralStorageGiB  int                   `json:"ephemeralStorageGiB,omitempty"` // 21-200, default 21; only for Fargate
	Family               string                `json:"family,omitempty"`
	MemoryMiB            int                   `json:"memoryMiB,omitempty"` // required for Fargate; defaults to sum of containers
	VCPU                 float64               `json:"vCPU,omitempty"`      // required for Fargate
	Tags                 map[string]string      `json:"tags,omitempty"`
}
