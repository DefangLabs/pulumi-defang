package aws

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type ContainerPortProtocol string

const (
	ContainerPortProtocolTCP ContainerPortProtocol = "tcp"
	ContainerPortProtocolUDP ContainerPortProtocol = "udp"
)

// ContainerPortMapping defines a port mapping for a container.
type ContainerPortMapping struct {
	ContainerPort int                   `json:"containerPort"`
	Protocol      ContainerPortProtocol `json:"protocol,omitempty"` // "tcp" or "udp"
	Name          string                `json:"name,omitempty"`     // only for Service Connect
	// HostPort      int    `json:"hostPort,omitempty"`
}

// ContainerEnvironment defines an environment variable for a container.
type ContainerEnvironment struct {
	Name  string             `json:"name"`
	Value pulumi.StringInput `json:"value"`
}

// ContainerHealthCheck defines health check parameters for a container.
type ContainerHealthCheck struct {
	// eg. ["CMD-SHELL", "curl -f http://localhost/ || exit 1"]
	Command     pulumi.StringArrayInput `json:"command"`
	Interval    int                     `json:"interval,omitempty"`    // 5-300, default 30s
	Timeout     int                     `json:"timeout,omitempty"`     // 2-300, default 5s
	Retries     int                     `json:"retries,omitempty"`     // 1-10, default 3
	StartPeriod int                     `json:"startPeriod,omitempty"` // 0-300, default 0s
}

type ContainerEnvironmentFileType string

const (
	ContainerEnvironmentFileS3 ContainerEnvironmentFileType = "s3"
)

// ContainerEnvironmentFile references an environment file in S3.
type ContainerEnvironmentFile struct {
	Type  ContainerEnvironmentFileType `json:"type"`  // "s3"
	Value string                       `json:"value"` // S3 ARN
}

// ContainerSecret references a secret from AWS Secrets Manager or SSM Parameter Store.
type ContainerSecret struct {
	Name      string             `json:"name"`
	ValueFrom pulumi.StringInput `json:"valueFrom"` // AWS SM secret or SSM parameter ARN
}

// ContainerRepositoryCredentials references credentials for a private registry.
type ContainerRepositoryCredentials struct {
	CredentialsParameter pulumi.StringInput `json:"credentialsParameter"` // secret ARN of {"username":"…","password":"…"}
}

type ContainerResourceRequirementType string

const (
	ContainerResourceGPU                  ContainerResourceRequirementType = "GPU"
	ContainerResourceInferenceAccelerator ContainerResourceRequirementType = "InferenceAccelerator"
)

// ContainerResourceRequirement specifies a resource requirement (GPU or InferenceAccelerator).
type ContainerResourceRequirement struct {
	Type  ContainerResourceRequirementType `json:"type"`  // "GPU" or "InferenceAccelerator"
	Value string                           `json:"value"` // GPU count or InferenceAccelerator deviceName
}

// ContainerMountPoint defines a volume mount point for a container.
type ContainerMountPoint struct {
	ContainerPath string             `json:"containerPath"`
	ReadOnly      bool               `json:"readOnly,omitempty"`
	SourceVolume  pulumi.StringInput `json:"sourceVolume"`
}

// ContainerVolumeFrom mounts volumes from another container.
type ContainerVolumeFrom struct {
	ReadOnly        bool   `json:"readOnly,omitempty"`
	SourceContainer string `json:"sourceContainer"`
}

type ContainerLogConfigurationDriver string

const (
	ContainerLogConfigurationAWSLogs  ContainerLogConfigurationDriver = "awslogs"
	ContainerLogConfigurationSplunk   ContainerLogConfigurationDriver = "splunk"
	ContainerLogConfigurationFirelens ContainerLogConfigurationDriver = "awsfirelens"
)

// ContainerLogConfiguration configures log routing for a container.
type ContainerLogConfiguration struct {
	LogDriver     ContainerLogConfigurationDriver `json:"logDriver"` // "awslogs", "splunk", or "awsfirelens" for Fargate
	Options       pulumi.StringMapInput           `json:"options,omitempty"`
	SecretOptions []ContainerSecret               `json:"secretOptions,omitempty"`
}

type ContainerFirelensConfigurationType string

const (
	ContainerFirelensConfigurationFluentd   ContainerFirelensConfigurationType = "fluentd"
	ContainerFirelensConfigurationFluentBit ContainerFirelensConfigurationType = "fluentbit"
)

// ContainerFirelensConfiguration configures Firelens log routing.
type ContainerFirelensConfiguration struct {
	Type    ContainerFirelensConfigurationType    `json:"type"` // "fluentd" or "fluentbit"
	Options ContainerFirelensConfigurationOptions `json:"options,omitempty"`
}

type BoolString string

const (
	BoolStringTrue  BoolString = "true"
	BoolStringFalse BoolString = "false"
)

type FirelensConfigurationConfigFileType string

const (
	FirelensConfigurationConfigFileTypeFile FirelensConfigurationConfigFileType = "file"
	FirelensConfigurationConfigFileTypeS3   FirelensConfigurationConfigFileType = "s3" // not supported in Fargate
)

type ContainerFirelensConfigurationOptions struct {
	EnableEcsLogMetadata BoolString                          `json:"enable-ecs-log-metadata"`
	// Fargate only supports `file` configuration file type
	ConfigFileType  FirelensConfigurationConfigFileType `json:"config-file-type"`
	ConfigFileValue pulumi.StringInput                  `json:"config-file-value"` // path to config file
}

type ContainerUlimitName string

const (
	ContainerUlimitCore       ContainerUlimitName = "core"
	ContainerUlimitCpu        ContainerUlimitName = "cpu"
	ContainerUlimitData       ContainerUlimitName = "data"
	ContainerUlimitFsize      ContainerUlimitName = "fsize"
	ContainerUlimitLocks      ContainerUlimitName = "locks"
	ContainerUlimitMemlock    ContainerUlimitName = "memlock"
	ContainerUlimitMsgqueue   ContainerUlimitName = "msgqueue"
	ContainerUlimitNice       ContainerUlimitName = "nice"
	ContainerUlimitNofile     ContainerUlimitName = "nofile"
	ContainerUlimitNproc      ContainerUlimitName = "nproc"
	ContainerUlimitRss        ContainerUlimitName = "rss"
	ContainerUlimitRtprio     ContainerUlimitName = "rtprio"
	ContainerUlimitRttime     ContainerUlimitName = "rttime"
	ContainerUlimitSigpending ContainerUlimitName = "sigpending"
	ContainerUlimitStack      ContainerUlimitName = "stack"
)

// ContainerUlimit defines a ulimit override for a container.
type ContainerUlimit struct {
	HardLimit int                 `json:"hardLimit"`
	Name      ContainerUlimitName `json:"name"` // "nofile", "memlock", "core", etc.
	SoftLimit int                 `json:"softLimit"`
}

type LinuxCapability string

const (
	KernelCapability LinuxCapability = "SYS_PTRACE" // Fargate only supports SYS_PTRACE
)

// ContainerLinuxCapabilities specifies Linux kernel capabilities to add or drop.
type ContainerLinuxCapabilities struct {
	Add  []LinuxCapability `json:"add,omitempty"`
	Drop []LinuxCapability `json:"drop,omitempty"`
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

type ContainerDependencyCondition string

const (
	ContainerDependencyStart    ContainerDependencyCondition = "START"
	ContainerDependencyComplete ContainerDependencyCondition = "COMPLETE"
	ContainerDependencySuccess  ContainerDependencyCondition = "SUCCESS"
	ContainerDependencyHealthy  ContainerDependencyCondition = "HEALTHY"
)

// ContainerDependency defines a dependency on another container.
type ContainerDependency struct {
	ContainerName string                       `json:"containerName"`
	Condition     ContainerDependencyCondition `json:"condition"` // "START", "COMPLETE", "SUCCESS", or "HEALTHY"
}

// ContainerDefinition defines a container within an ECS task definition.
// See https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ContainerDefinition.html
type ContainerDefinition struct {
	Name                   string                          `json:"name"`
	Image                  pulumi.StringInput              `json:"image"`
	Memory                 int                             `json:"memory,omitempty"`   // MiB; optional for Fargate
	MemoryReservation      int                             `json:"memoryReservation,omitempty"` // MiB
	Essential              *bool                           `json:"essential,omitempty"`         // default true
	PortMappings           []ContainerPortMapping          `json:"portMappings,omitempty"`
	Environment            []ContainerEnvironment          `json:"environment,omitempty"`
	HealthCheck            *ContainerHealthCheck           `json:"healthCheck,omitempty"`
	CPU                    int                             `json:"cpu,omitempty"` // vCPU; optional for Fargate
	EntryPoint             pulumi.StringArrayInput         `json:"entryPoint,omitempty"`
	Command                pulumi.StringArrayInput         `json:"command,omitempty"`
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
	DockerLabels           pulumi.StringMapInput           `json:"dockerLabels,omitempty"`
	LinuxParameters        *ContainerLinuxParameters       `json:"linuxParameters,omitempty"`
	DependsOn              []ContainerDependency           `json:"dependsOn,omitempty"`
	StartTimeout           int                             `json:"startTimeout,omitempty"` // 0-3600, default 0s
	StopTimeout            int                             `json:"stopTimeout,omitempty"`  // 0-3600, default 0s
	Interactive            bool                            `json:"interactive,omitempty"`
	PseudoTerminal         bool                            `json:"pseudoTerminal,omitempty"`
}
