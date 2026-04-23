package aws

import awsecs "github.com/aws/aws-sdk-go-v2/service/ecs/types"

// These types mirror the AWS ECS SDK types but with json struct tags for correct
// camelCase serialization. The SDK types use Smithy serialization (no json tags),
// so json.Marshal produces PascalCase field names which ECS rejects.
// We keep using SDK enums/consts (TransportProtocol, LogDriver, etc.) since those
// are just strings that serialize correctly.

// ContainerDefinition defines a container within an ECS task definition.
// See https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ContainerDefinition.html
type ContainerDefinition struct {
	Command          []string              `json:"command,omitempty"`
	DependsOn        []ContainerDependency `json:"dependsOn,omitempty"`
	EntryPoint       []string              `json:"entryPoint,omitempty"`
	Environment      []KeyValuePair        `json:"environment,omitempty"`
	Essential        *bool                 `json:"essential,omitempty"`
	HealthCheck      *HealthCheck          `json:"healthCheck,omitempty"`
	Image            string                `json:"image"`
	LogConfiguration *LogConfiguration     `json:"logConfiguration,omitempty"`
	MountPoints      []MountPoint          `json:"mountPoints"`
	Name             string                `json:"name"`
	PortMappings     []PortMapping         `json:"portMappings"`
	Secrets          []Secret              `json:"secrets,omitempty"`
	SystemControls   []SystemControl       `json:"systemControls"`
	VolumesFrom      []VolumeFrom          `json:"volumesFrom"`
}

// PortMapping defines a port mapping for a container.
type PortMapping struct {
	ContainerPort *int32                   `json:"containerPort,omitempty"`
	HostPort      *int32                   `json:"hostPort,omitempty"`
	Protocol      awsecs.TransportProtocol `json:"protocol,omitempty"`
}

// HealthCheck defines health check parameters for a container.
type HealthCheck struct {
	Command     []string `json:"command,omitempty"`
	Interval    *int32   `json:"interval,omitempty"`
	Timeout     *int32   `json:"timeout,omitempty"`
	Retries     *int32   `json:"retries,omitempty"`
	StartPeriod *int32   `json:"startPeriod,omitempty"`
}

// KeyValuePair defines an environment variable for a container.
type KeyValuePair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Secret defines a secret to inject into a container.
type Secret struct {
	Name      string `json:"name"`
	ValueFrom string `json:"valueFrom"` // AWS SM secret or SMPS parameter ARN
}

// LogOption defines a log driver option for a container.
// See https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_LogConfiguration.html
type LogOption string

const (
	// Universal options (all log drivers)
	LogOptionMode          LogOption = "mode"            // non-blocking | blocking
	LogOptionMaxBufferSize LogOption = "max-buffer-size" // Default: 10m

	// awslogs driver options
	LogOptionAwslogsGroup            LogOption = "awslogs-group"             // required
	LogOptionAwslogsRegion           LogOption = "awslogs-region"            // required
	LogOptionAwslogsStreamPrefix     LogOption = "awslogs-stream-prefix"     // required for Fargate
	LogOptionAwslogsCreateGroup      LogOption = "awslogs-create-group"      // requires logs:CreateLogGroup IAM permission
	LogOptionAwslogsDatetimeFormat   LogOption = "awslogs-datetime-format"   // mutually exclusive with multiline-pattern
	LogOptionAwslogsMultilinePattern LogOption = "awslogs-multiline-pattern" // mutually exclusive with datetime-format

	// splunk driver options
	LogOptionSplunkToken LogOption = "splunk-token" // required
	LogOptionSplunkUrl   LogOption = "splunk-url"   // required

	// awsfirelens driver options
	LogOptionFirelensBufferLimit LogOption = "log-driver-buffer-limit"
)

// LogConfiguration configures log routing for a container.
type LogConfiguration struct {
	LogDriver awsecs.LogDriver     `json:"logDriver"`
	Options   map[LogOption]string `json:"options,omitempty"`
}

// ContainerDependency defines a dependency on another container.
type ContainerDependency struct {
	ContainerName *string                   `json:"containerName,omitempty"`
	Condition     awsecs.ContainerCondition `json:"condition,omitempty"`
}

// MountPoint defines a mount point for a container.
type MountPoint struct {
	ContainerPath *string `json:"containerPath,omitempty"`
	ReadOnly      *bool   `json:"readOnly,omitempty"`
	SourceVolume  *string `json:"sourceVolume,omitempty"`
}

// SystemControl defines a kernel parameter to set in the container.
type SystemControl struct {
	Namespace *string `json:"namespace,omitempty"`
	Value     *string `json:"value,omitempty"`
}

// VolumeFrom defines volumes to mount from another container.
type VolumeFrom struct {
	ReadOnly        *bool   `json:"readOnly,omitempty"`
	SourceContainer *string `json:"sourceContainer,omitempty"`
}
