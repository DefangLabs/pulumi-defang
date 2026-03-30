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
	Name             *string               `json:"name,omitempty"`
	Image            *string               `json:"image,omitempty"`
	Essential        *bool                 `json:"essential,omitempty"`
	Command          []string              `json:"command,omitempty"`
	EntryPoint       []string              `json:"entryPoint,omitempty"`
	PortMappings     []PortMapping         `json:"portMappings"`
	Environment      []KeyValuePair        `json:"environment,omitempty"`
	HealthCheck      *HealthCheck          `json:"healthCheck,omitempty"`
	LogConfiguration *LogConfiguration     `json:"logConfiguration,omitempty"`
	DependsOn        []ContainerDependency `json:"dependsOn,omitempty"`
	MountPoints      []MountPoint          `json:"mountPoints"`
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
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

// LogConfiguration configures log routing for a container.
type LogConfiguration struct {
	LogDriver awsecs.LogDriver  `json:"logDriver"`
	Options   map[string]string `json:"options,omitempty"`
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
