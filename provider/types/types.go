/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/go-connections/nat"
)

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name     string   `yaml:"name,omitempty" json:"-"`
	Profiles []string `yaml:"profiles,omitempty" json:"profiles,omitempty" pulumi:"profiles,omitempty,optional"`

	Annotations  Mapping        `yaml:"annotations,omitempty" json:"annotations,omitempty" pulumi:"annotations,omitempty,optional"`
	Attach       *bool          `yaml:"attach,omitempty" json:"attach,omitempty" pulumi:"attach,omitempty,optional"`
	Build        *BuildConfig   `yaml:"build,omitempty" json:"build,omitempty" pulumi:"build,omitempty,optional"`
	Develop      *DevelopConfig `yaml:"develop,omitempty" json:"develop,omitempty" pulumi:"develop,omitempty,optional"`
	BlkioConfig  *BlkioConfig   `yaml:"blkio_config,omitempty" json:"blkio_config,omitempty" pulumi:"blkio_config,omitempty,optional"`
	CapAdd       []string       `yaml:"cap_add,omitempty" json:"cap_add,omitempty" pulumi:"cap_add,omitempty,optional"`
	CapDrop      []string       `yaml:"cap_drop,omitempty" json:"cap_drop,omitempty" pulumi:"cap_drop,omitempty,optional"`
	CgroupParent string         `yaml:"cgroup_parent,omitempty" json:"cgroup_parent,omitempty" pulumi:"cgroup_parent,omitempty,optional"`
	Cgroup       string         `yaml:"cgroup,omitempty" json:"cgroup,omitempty" pulumi:"cgroup,omitempty,optional"`
	CPUCount     int64          `yaml:"cpu_count,omitempty" json:"cpu_count,omitempty" pulumi:"cpu_count,omitempty,optional"`
	CPUPercent   float64        `yaml:"cpu_percent,omitempty" json:"cpu_percent,omitempty" pulumi:"cpu_percent,omitempty,optional"`
	CPUPeriod    int64          `yaml:"cpu_period,omitempty" json:"cpu_period,omitempty" pulumi:"cpu_period,omitempty,optional"`
	CPUQuota     int64          `yaml:"cpu_quota,omitempty" json:"cpu_quota,omitempty" pulumi:"cpu_quota,omitempty,optional"`
	CPURTPeriod  int64          `yaml:"cpu_rt_period,omitempty" json:"cpu_rt_period,omitempty" pulumi:"cpu_rt_period,omitempty,optional"`
	CPURTRuntime int64          `yaml:"cpu_rt_runtime,omitempty" json:"cpu_rt_runtime,omitempty" pulumi:"cpu_rt_runtime,omitempty,optional"`
	CPUS         float64        `yaml:"cpus,omitempty" json:"cpus,omitempty" pulumi:"cpus,omitempty,optional"`
	CPUSet       string         `yaml:"cpuset,omitempty" json:"cpuset,omitempty" pulumi:"cpuset,omitempty,optional"`
	CPUShares    int64          `yaml:"cpu_shares,omitempty" json:"cpu_shares,omitempty" pulumi:"cpu_shares,omitempty,optional"`

	// Command for the service containers.
	// If set, overrides COMMAND from the image.
	//
	// Set to `[]` or an empty string to clear the command from the image.
	Command *ShellCommand `yaml:"command,omitempty" json:"command,omitempty" pulumi:"command,omitempty,optional"` // NOTE: we can NOT omitempty for JSON! see ShellCommand type for details.

	Configs           []ServiceConfigObjConfig `yaml:"configs,omitempty" json:"configs,omitempty" pulumi:"configs,omitempty,optional"`
	ContainerName     string                   `yaml:"container_name,omitempty" json:"container_name,omitempty" pulumi:"container_name,omitempty,optional"`
	CredentialSpec    *CredentialSpecConfig    `yaml:"credential_spec,omitempty" json:"credential_spec,omitempty" pulumi:"credential_spec,omitempty,optional"`
	DependsOn         DependsOnConfig          `yaml:"depends_on,omitempty" json:"depends_on,omitempty" pulumi:"depends_on,omitempty,optional"`
	Deploy            *DeployConfig            `yaml:"deploy,omitempty" json:"deploy,omitempty" pulumi:"deploy,omitempty,optional"`
	DeviceCgroupRules []string                 `yaml:"device_cgroup_rules,omitempty" json:"device_cgroup_rules,omitempty" pulumi:"device_cgroup_rules,omitempty,optional"`
	Devices           []DeviceMapping          `yaml:"devices,omitempty" json:"devices,omitempty" pulumi:"devices,omitempty,optional"`
	DNS               StringList               `yaml:"dns,omitempty" json:"dns,omitempty" pulumi:"dns,omitempty,optional"`
	DNSOpts           []string                 `yaml:"dns_opt,omitempty" json:"dns_opt,omitempty" pulumi:"dns_opt,omitempty,optional"`
	DNSSearch         StringList               `yaml:"dns_search,omitempty" json:"dns_search,omitempty" pulumi:"dns_search,omitempty,optional"`
	Dockerfile        string                   `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty" pulumi:"dockerfile,omitempty,optional"`
	DomainName        string                   `yaml:"domainname,omitempty" json:"domainname,omitempty" pulumi:"domainname,omitempty,optional"`

	// Entrypoint for the service containers.
	// If set, overrides ENTRYPOINT from the image.
	//
	// Set to `[]` or an empty string to clear the entrypoint from the image.
	Entrypoint *ShellCommand `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty" pulumi:"entrypoint,omitempty,optional"` // NOTE: we can NOT omitempty for JSON! see ShellCommand type for details.

	Environment     MappingWithEquals                `yaml:"environment,omitempty" json:"environment,omitempty" pulumi:"environment,omitempty,optional"`
	EnvFiles        []EnvFile                        `yaml:"env_file,omitempty" json:"env_file,omitempty" pulumi:"env_file,omitempty,optional"`
	Expose          StringOrNumberList               `yaml:"expose,omitempty" json:"expose,omitempty" pulumi:"expose,omitempty,optional"`
	Extends         *ExtendsConfig                   `yaml:"extends,omitempty" json:"extends,omitempty" pulumi:"extends,omitempty,optional"`
	ExternalLinks   []string                         `yaml:"external_links,omitempty" json:"external_links,omitempty" pulumi:"external_links,omitempty,optional"`
	ExtraHosts      HostsList                        `yaml:"extra_hosts,omitempty" json:"extra_hosts,omitempty" pulumi:"extra_hosts,omitempty,optional"`
	GroupAdd        []string                         `yaml:"group_add,omitempty" json:"group_add,omitempty" pulumi:"group_add,omitempty,optional"`
	Gpus            []DeviceRequest                  `yaml:"gpus,omitempty" json:"gpus,omitempty" pulumi:"gpus,omitempty,optional"`
	Hostname        string                           `yaml:"hostname,omitempty" json:"hostname,omitempty" pulumi:"hostname,omitempty,optional"`
	HealthCheck     *HealthCheckConfig               `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty" pulumi:"healthcheck,omitempty,optional"`
	Image           string                           `yaml:"image,omitempty" json:"image,omitempty" pulumi:"image,omitempty,optional"`
	Init            *bool                            `yaml:"init,omitempty" json:"init,omitempty" pulumi:"init,omitempty,optional"`
	Ipc             string                           `yaml:"ipc,omitempty" json:"ipc,omitempty" pulumi:"ipc,omitempty,optional"`
	Isolation       string                           `yaml:"isolation,omitempty" json:"isolation,omitempty" pulumi:"isolation,omitempty,optional"`
	Labels          Labels                           `yaml:"labels,omitempty" json:"labels,omitempty" pulumi:"labels,omitempty,optional"`
	CustomLabels    Labels                           `yaml:"-" json:"-"`
	Links           []string                         `yaml:"links,omitempty" json:"links,omitempty" pulumi:"links,omitempty,optional"`
	Logging         *LoggingConfig                   `yaml:"logging,omitempty" json:"logging,omitempty" pulumi:"logging,omitempty,optional"`
	LogDriver       string                           `yaml:"log_driver,omitempty" json:"log_driver,omitempty" pulumi:"log_driver,omitempty,optional"`
	LogOpt          map[string]string                `yaml:"log_opt,omitempty" json:"log_opt,omitempty" pulumi:"log_opt,omitempty,optional"`
	MemLimit        UnitBytes                        `yaml:"mem_limit,omitempty" json:"mem_limit,omitempty" pulumi:"mem_limit,omitempty,optional"`
	MemReservation  UnitBytes                        `yaml:"mem_reservation,omitempty" json:"mem_reservation,omitempty" pulumi:"mem_reservation,omitempty,optional"`
	MemSwapLimit    UnitBytes                        `yaml:"memswap_limit,omitempty" json:"memswap_limit,omitempty" pulumi:"memswap_limit,omitempty,optional"`
	MemSwappiness   UnitBytes                        `yaml:"mem_swappiness,omitempty" json:"mem_swappiness,omitempty" pulumi:"mem_swappiness,omitempty,optional"`
	MacAddress      string                           `yaml:"mac_address,omitempty" json:"mac_address,omitempty" pulumi:"mac_address,omitempty,optional"`
	Net             string                           `yaml:"net,omitempty" json:"net,omitempty" pulumi:"net,omitempty,optional"`
	NetworkMode     string                           `yaml:"network_mode,omitempty" json:"network_mode,omitempty" pulumi:"network_mode,omitempty,optional"`
	Networks        map[string]*ServiceNetworkConfig `yaml:"networks,omitempty" json:"networks,omitempty" pulumi:"networks,omitempty,optional"`
	OomKillDisable  bool                             `yaml:"oom_kill_disable,omitempty" json:"oom_kill_disable,omitempty" pulumi:"oom_kill_disable,omitempty,optional"`
	OomScoreAdj     int64                            `yaml:"oom_score_adj,omitempty" json:"oom_score_adj,omitempty" pulumi:"oom_score_adj,omitempty,optional"`
	Pid             string                           `yaml:"pid,omitempty" json:"pid,omitempty" pulumi:"pid,omitempty,optional"`
	PidsLimit       int64                            `yaml:"pids_limit,omitempty" json:"pids_limit,omitempty" pulumi:"pids_limit,omitempty,optional"`
	Platform        string                           `yaml:"platform,omitempty" json:"platform,omitempty" pulumi:"platform,omitempty,optional"`
	Ports           []ServicePortConfig              `yaml:"ports,omitempty" json:"ports,omitempty" pulumi:"ports,omitempty,optional"`
	Privileged      bool                             `yaml:"privileged,omitempty" json:"privileged,omitempty" pulumi:"privileged,omitempty,optional"`
	PullPolicy      string                           `yaml:"pull_policy,omitempty" json:"pull_policy,omitempty" pulumi:"pull_policy,omitempty,optional"`
	ReadOnly        bool                             `yaml:"read_only,omitempty" json:"read_only,omitempty" pulumi:"read_only,omitempty,optional"`
	Restart         string                           `yaml:"restart,omitempty" json:"restart,omitempty" pulumi:"restart,omitempty,optional"`
	Runtime         string                           `yaml:"runtime,omitempty" json:"runtime,omitempty" pulumi:"runtime,omitempty,optional"`
	Scale           *int                             `yaml:"scale,omitempty" json:"scale,omitempty" pulumi:"scale,omitempty,optional"`
	Secrets         []ServiceSecretConfig            `yaml:"secrets,omitempty" json:"secrets,omitempty" pulumi:"secrets,omitempty,optional"`
	SecurityOpt     []string                         `yaml:"security_opt,omitempty" json:"security_opt,omitempty" pulumi:"security_opt,omitempty,optional"`
	ShmSize         UnitBytes                        `yaml:"shm_size,omitempty" json:"shm_size,omitempty" pulumi:"shm_size,omitempty,optional"`
	StdinOpen       bool                             `yaml:"stdin_open,omitempty" json:"stdin_open,omitempty" pulumi:"stdin_open,omitempty,optional"`
	StopGracePeriod *Duration                        `yaml:"stop_grace_period,omitempty" json:"stop_grace_period,omitempty" pulumi:"stop_grace_period,omitempty,optional"`
	StopSignal      string                           `yaml:"stop_signal,omitempty" json:"stop_signal,omitempty" pulumi:"stop_signal,omitempty,optional"`
	StorageOpt      map[string]string                `yaml:"storage_opt,omitempty" json:"storage_opt,omitempty" pulumi:"storage_opt,omitempty,optional"`
	Sysctls         Mapping                          `yaml:"sysctls,omitempty" json:"sysctls,omitempty" pulumi:"sysctls,omitempty,optional"`
	Tmpfs           StringList                       `yaml:"tmpfs,omitempty" json:"tmpfs,omitempty" pulumi:"tmpfs,omitempty,optional"`
	Tty             bool                             `yaml:"tty,omitempty" json:"tty,omitempty" pulumi:"tty,omitempty,optional"`
	Ulimits         map[string]*UlimitsConfig        `yaml:"ulimits,omitempty" json:"ulimits,omitempty" pulumi:"ulimits,omitempty,optional"`
	User            string                           `yaml:"user,omitempty" json:"user,omitempty" pulumi:"user,omitempty,optional"`
	UserNSMode      string                           `yaml:"userns_mode,omitempty" json:"userns_mode,omitempty" pulumi:"userns_mode,omitempty,optional"`
	Uts             string                           `yaml:"uts,omitempty" json:"uts,omitempty" pulumi:"uts,omitempty,optional"`
	VolumeDriver    string                           `yaml:"volume_driver,omitempty" json:"volume_driver,omitempty" pulumi:"volume_driver,omitempty,optional"`
	Volumes         []ServiceVolumeConfig            `yaml:"volumes,omitempty" json:"volumes,omitempty" pulumi:"volumes,omitempty,optional"`
	VolumesFrom     []string                         `yaml:"volumes_from,omitempty" json:"volumes_from,omitempty" pulumi:"volumes_from,omitempty,optional"`
	WorkingDir      string                           `yaml:"working_dir,omitempty" json:"working_dir,omitempty" pulumi:"working_dir,omitempty,optional"`
	PostStart       []ServiceHook                    `yaml:"post_start,omitempty" json:"post_start,omitempty" pulumi:"post_start,omitempty,optional"`
	PreStop         []ServiceHook                    `yaml:"pre_stop,omitempty" json:"pre_stop,omitempty" pulumi:"pre_stop,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`

	DefangLLM         bool `yaml:"x-defang-llm,omitempty" json:"x-defang-llm,omitempty" pulumi:"defang_llm,omitempty,optional"`
	DefangPostgres    bool `yaml:"x-defang-postgres,omitempty" json:"x-defang-postgres,omitempty" pulumi:"defang_postgres,omitempty,optional"`
	DefangRedis       bool `yaml:"x-defang-redis,omitempty" json:"x-defang-redis,omitempty" pulumi:"defang_redis,omitempty,optional"`
	DefangStaticFiles bool `yaml:"x-defang-static-files,omitempty" json:"x-defang-static-files,omitempty" pulumi:"defang_static_files,omitempty,optional"`
}

// MarshalYAML makes ServiceConfig implement yaml.Marshaller
func (s ServiceConfig) MarshalYAML() (interface{}, error) {
	type t ServiceConfig
	value := t(s)
	value.Name = "" // set during map to slice conversion, not part of the yaml representation
	return value, nil
}

// NetworksByPriority return the service networks IDs sorted according to Priority
func (s *ServiceConfig) NetworksByPriority() []string {
	type key struct {
		name     string
		priority int
	}
	var keys []key
	for k, v := range s.Networks {
		priority := 0
		if v != nil {
			priority = v.Priority
		}
		keys = append(keys, key{
			name:     k,
			priority: priority,
		})
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].priority == keys[j].priority {
			return keys[i].name < keys[j].name
		}
		return keys[i].priority > keys[j].priority
	})
	var sorted []string
	for _, k := range keys {
		sorted = append(sorted, k.name)
	}
	return sorted
}

func (s *ServiceConfig) GetScale() int {
	if s.Scale != nil {
		return *s.Scale
	}
	if s.Deploy != nil && s.Deploy.Replicas != nil {
		// this should not be required as compose-go enforce consistency between scale anr replicas
		return *s.Deploy.Replicas
	}
	return 1
}

func (s *ServiceConfig) SetScale(scale int) {
	s.Scale = &scale
	if s.Deploy != nil {
		s.Deploy.Replicas = &scale
	}
}

func (s *ServiceConfig) deepCopy() *ServiceConfig {
	if s == nil {
		return nil
	}
	n := &ServiceConfig{}
	deriveDeepCopyService(n, s)
	return n
}

const (
	// PullPolicyAlways always pull images
	PullPolicyAlways = "always"
	// PullPolicyNever never pull images
	PullPolicyNever = "never"
	// PullPolicyIfNotPresent pull missing images
	PullPolicyIfNotPresent = "if_not_present"
	// PullPolicyMissing pull missing images
	PullPolicyMissing = "missing"
	// PullPolicyBuild force building images
	PullPolicyBuild = "build"
)

const (
	// RestartPolicyAlways always restart the container if it stops
	RestartPolicyAlways = "always"
	// RestartPolicyOnFailure restart the container if it exits due to an error
	RestartPolicyOnFailure = "on-failure"
	// RestartPolicyNo do not automatically restart the container
	RestartPolicyNo = "no"
	// RestartPolicyUnlessStopped always restart the container unless the container is stopped (manually or otherwise)
	RestartPolicyUnlessStopped = "unless-stopped"
)

const (
	// ServicePrefix is the prefix for references pointing to a service
	ServicePrefix = "service:"
	// ContainerPrefix is the prefix for references pointing to a container
	ContainerPrefix = "container:"

	// NetworkModeServicePrefix is the prefix for network_mode pointing to a service
	// Deprecated prefer ServicePrefix
	NetworkModeServicePrefix = ServicePrefix
	// NetworkModeContainerPrefix is the prefix for network_mode pointing to a container
	// Deprecated prefer ContainerPrefix
	NetworkModeContainerPrefix = ContainerPrefix
)

const (
	SecretConfigXValue = "x-#value"
)

// GetDependencies retrieves all services this service depends on
func (s ServiceConfig) GetDependencies() []string {
	var dependencies []string
	for service := range s.DependsOn {
		dependencies = append(dependencies, service)
	}
	return dependencies
}

// GetDependents retrieves all services which depend on this service
func (s ServiceConfig) GetDependents(p *Project) []string {
	var dependent []string
	for _, service := range p.Services {
		for name := range service.DependsOn {
			if name == s.Name {
				dependent = append(dependent, service.Name)
			}
		}
	}
	return dependent
}

// BuildConfig is a type for build
type BuildConfig struct {
	Context            string                    `yaml:"context,omitempty" json:"context,omitempty" pulumi:"context,omitempty,optional"`
	Dockerfile         string                    `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty" pulumi:"dockerfile,omitempty,optional"`
	DockerfileInline   string                    `yaml:"dockerfile_inline,omitempty" json:"dockerfile_inline,omitempty" pulumi:"dockerfile_inline,omitempty,optional"`
	Entitlements       []string                  `yaml:"entitlements,omitempty" json:"entitlements,omitempty" pulumi:"entitlements,omitempty,optional"`
	Args               MappingWithEquals         `yaml:"args,omitempty" json:"args,omitempty" pulumi:"args,omitempty,optional"`
	SSH                SSHConfig                 `yaml:"ssh,omitempty" json:"ssh,omitempty" pulumi:"ssh,omitempty,optional"`
	Labels             Labels                    `yaml:"labels,omitempty" json:"labels,omitempty" pulumi:"labels,omitempty,optional"`
	CacheFrom          StringList                `yaml:"cache_from,omitempty" json:"cache_from,omitempty" pulumi:"cache_from,omitempty,optional"`
	CacheTo            StringList                `yaml:"cache_to,omitempty" json:"cache_to,omitempty" pulumi:"cache_to,omitempty,optional"`
	NoCache            bool                      `yaml:"no_cache,omitempty" json:"no_cache,omitempty" pulumi:"no_cache,omitempty,optional"`
	AdditionalContexts Mapping                   `yaml:"additional_contexts,omitempty" json:"additional_contexts,omitempty" pulumi:"additional_contexts,omitempty,optional"`
	Pull               bool                      `yaml:"pull,omitempty" json:"pull,omitempty" pulumi:"pull,omitempty,optional"`
	ExtraHosts         HostsList                 `yaml:"extra_hosts,omitempty" json:"extra_hosts,omitempty" pulumi:"extra_hosts,omitempty,optional"`
	Isolation          string                    `yaml:"isolation,omitempty" json:"isolation,omitempty" pulumi:"isolation,omitempty,optional"`
	Network            string                    `yaml:"network,omitempty" json:"network,omitempty" pulumi:"network,omitempty,optional"`
	Target             string                    `yaml:"target,omitempty" json:"target,omitempty" pulumi:"target,omitempty,optional"`
	Secrets            []ServiceSecretConfig     `yaml:"secrets,omitempty" json:"secrets,omitempty" pulumi:"secrets,omitempty,optional"`
	ShmSize            UnitBytes                 `yaml:"shm_size,omitempty" json:"shm_size,omitempty" pulumi:"shm_size,omitempty,optional"`
	Tags               StringList                `yaml:"tags,omitempty" json:"tags,omitempty" pulumi:"tags,omitempty,optional"`
	Ulimits            map[string]*UlimitsConfig `yaml:"ulimits,omitempty" json:"ulimits,omitempty" pulumi:"ulimits,omitempty,optional"`
	Platforms          StringList                `yaml:"platforms,omitempty" json:"platforms,omitempty" pulumi:"platforms,omitempty,optional"`
	Privileged         bool                      `yaml:"privileged,omitempty" json:"privileged,omitempty" pulumi:"privileged,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// BlkioConfig define blkio config
type BlkioConfig struct {
	Weight          int32            `yaml:"weight,omitempty" json:"weight,omitempty" pulumi:"weight,omitempty,optional"`
	WeightDevice    []WeightDevice   `yaml:"weight_device,omitempty" json:"weight_device,omitempty" pulumi:"weight_device,omitempty,optional"`
	DeviceReadBps   []ThrottleDevice `yaml:"device_read_bps,omitempty" json:"device_read_bps,omitempty" pulumi:"device_read_bps,omitempty,optional"`
	DeviceReadIOps  []ThrottleDevice `yaml:"device_read_iops,omitempty" json:"device_read_iops,omitempty" pulumi:"device_read_iops,omitempty,optional"`
	DeviceWriteBps  []ThrottleDevice `yaml:"device_write_bps,omitempty" json:"device_write_bps,omitempty" pulumi:"device_write_bps,omitempty,optional"`
	DeviceWriteIOps []ThrottleDevice `yaml:"device_write_iops,omitempty" json:"device_write_iops,omitempty" pulumi:"device_write_iops,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

type DeviceMapping struct {
	Source      string `yaml:"source,omitempty" json:"source,omitempty" pulumi:"source,omitempty,optional"`
	Target      string `yaml:"target,omitempty" json:"target,omitempty" pulumi:"target,omitempty,optional"`
	Permissions string `yaml:"permissions,omitempty" json:"permissions,omitempty" pulumi:"permissions,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// WeightDevice is a structure that holds device:weight pair
type WeightDevice struct {
	Path   string
	Weight int32

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ThrottleDevice is a structure that holds device:rate_per_second pair
type ThrottleDevice struct {
	Path string
	Rate UnitBytes

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// MappingWithColon is a mapping type that can be converted from a list of
// 'key: value' strings
type MappingWithColon map[string]string

// LoggingConfig the logging configuration for a service
type LoggingConfig struct {
	Driver  string  `yaml:"driver,omitempty" json:"driver,omitempty" pulumi:"driver,omitempty,optional"`
	Options Options `yaml:"options,omitempty" json:"options,omitempty" pulumi:"options,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// DeployConfig the deployment configuration for a service
type DeployConfig struct {
	Mode           string         `yaml:"mode,omitempty" json:"mode,omitempty" pulumi:"mode,omitempty,optional"`
	Replicas       *int           `yaml:"replicas,omitempty" json:"replicas,omitempty" pulumi:"replicas,omitempty,optional"`
	Labels         Labels         `yaml:"labels,omitempty" json:"labels,omitempty" pulumi:"labels,omitempty,optional"`
	UpdateConfig   *UpdateConfig  `yaml:"update_config,omitempty" json:"update_config,omitempty" pulumi:"update_config,omitempty,optional"`
	RollbackConfig *UpdateConfig  `yaml:"rollback_config,omitempty" json:"rollback_config,omitempty" pulumi:"rollback_config,omitempty,optional"`
	Resources      Resources      `yaml:"resources,omitempty" json:"resources,omitempty" pulumi:"resources,omitempty"`
	RestartPolicy  *RestartPolicy `yaml:"restart_policy,omitempty" json:"restart_policy,omitempty" pulumi:"restart_policy,omitempty,optional"`
	Placement      *Placement     `yaml:"placement,omitempty" json:"placement,omitempty" pulumi:"placement,omitempty,optional"`
	EndpointMode   string         `yaml:"endpoint_mode,omitempty" json:"endpoint_mode,omitempty" pulumi:"endpoint_mode,omitempty,optional"`

	Extensions Extensions `yaml:"extensions,inline,omitempty" json:"-"`
}

// UpdateConfig the service update configuration
type UpdateConfig struct {
	Parallelism     *int64   `yaml:"parallelism,omitempty" json:"parallelism,omitempty" pulumi:"parallelism,omitempty,optional"`
	Delay           Duration `yaml:"delay,omitempty" json:"delay,omitempty" pulumi:"delay,omitempty,optional"`
	FailureAction   string   `yaml:"failure_action,omitempty" json:"failure_action,omitempty" pulumi:"failure_action,omitempty,optional"`
	Monitor         Duration `yaml:"monitor,omitempty" json:"monitor,omitempty" pulumi:"monitor,omitempty,optional"`
	MaxFailureRatio float64  `yaml:"max_failure_ratio,omitempty" json:"max_failure_ratio,omitempty" pulumi:"max_failure_ratio,omitempty,optional"`
	Order           string   `yaml:"order,omitempty" json:"order,omitempty" pulumi:"order,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `yaml:"limits,omitempty" json:"limits,omitempty" pulumi:"limits,omitempty,optional"`
	Reservations *Resource `yaml:"reservations,omitempty" json:"reservations,omitempty" pulumi:"reservations,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	// TODO: types to convert from units and ratios
	NanoCPUs         NanoCPUs          `yaml:"cpus,omitempty" json:"cpus,omitempty" pulumi:"cpus,omitempty,optional"`
	MemoryBytes      UnitBytes         `yaml:"memory,omitempty" json:"memory,omitempty" pulumi:"memory,omitempty,optional"`
	Pids             int64             `yaml:"pids,omitempty" json:"pids,omitempty" pulumi:"pids,omitempty,optional"`
	Devices          []DeviceRequest   `yaml:"devices,omitempty" json:"devices,omitempty" pulumi:"devices,omitempty,optional"`
	GenericResources []GenericResource `yaml:"generic_resources,omitempty" json:"generic_resources,omitempty" pulumi:"generic_resources,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// GenericResource represents a "user defined" resource which can
// only be an integer (e.g: SSD=3) for a service
type GenericResource struct {
	DiscreteResourceSpec *DiscreteGenericResource `yaml:"discrete_resource_spec,omitempty" json:"discrete_resource_spec,omitempty" pulumi:"discrete_resource_spec,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// DiscreteGenericResource represents a "user defined" resource which is defined
// as an integer
// "Kind" is used to describe the Kind of a resource (e.g: "GPU", "FPGA", "SSD", ...)
// Value is used to count the resource (SSD=5, HDD=3, ...)
type DiscreteGenericResource struct {
	Kind  string `json:"kind" pulumi:"kind"`
	Value int64  `json:"value" pulumi:"value"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// RestartPolicy the service restart policy
type RestartPolicy struct {
	Condition   string    `yaml:"condition,omitempty" json:"condition,omitempty" pulumi:"condition,omitempty,optional"`
	Delay       *Duration `yaml:"delay,omitempty" json:"delay,omitempty" pulumi:"delay,omitempty,optional"`
	MaxAttempts *int64    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty" pulumi:"max_attempts,omitempty,optional"`
	Window      *Duration `yaml:"window,omitempty" json:"window,omitempty" pulumi:"window,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Placement constraints for the service
type Placement struct {
	Constraints []string               `yaml:"constraints,omitempty" json:"constraints,omitempty" pulumi:"constraints,omitempty,optional"`
	Preferences []PlacementPreferences `yaml:"preferences,omitempty" json:"preferences,omitempty" pulumi:"preferences,omitempty,optional"`
	MaxReplicas int64                  `yaml:"max_replicas_per_node,omitempty" json:"max_replicas_per_node,omitempty" pulumi:"max_replicas_per_node,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// PlacementPreferences is the preferences for a service placement
type PlacementPreferences struct {
	Spread string `yaml:"spread,omitempty" json:"spread,omitempty" pulumi:"spread,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServiceNetworkConfig is the network configuration for a service
type ServiceNetworkConfig struct {
	Priority     int      `yaml:"priority,omitempty" json:"priority,omitempty" pulumi:"priority,omitempty,optional"`
	Aliases      []string `yaml:"aliases,omitempty" json:"aliases,omitempty" pulumi:"aliases,omitempty,optional"`
	Ipv4Address  string   `yaml:"ipv4_address,omitempty" json:"ipv4_address,omitempty" pulumi:"ipv4_address,omitempty,optional"`
	Ipv6Address  string   `yaml:"ipv6_address,omitempty" json:"ipv6_address,omitempty" pulumi:"ipv6_address,omitempty,optional"`
	LinkLocalIPs []string `yaml:"link_local_ips,omitempty" json:"link_local_ips,omitempty" pulumi:"link_local_ips,omitempty,optional"`
	MacAddress   string   `yaml:"mac_address,omitempty" json:"mac_address,omitempty" pulumi:"mac_address,omitempty,optional"`
	DriverOpts   Options  `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty" pulumi:"driver_opts,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Name        string `yaml:"name,omitempty" json:"name,omitempty" pulumi:"name,omitempty,optional"`
	Mode        string `yaml:"mode,omitempty" json:"mode,omitempty" pulumi:"mode,omitempty,optional"`
	HostIP      string `yaml:"host_ip,omitempty" json:"host_ip,omitempty" pulumi:"host_ip,omitempty,optional"`
	Target      int32  `yaml:"target,omitempty" json:"target,omitempty" pulumi:"target,omitempty,optional"`
	Published   string `yaml:"published,omitempty" json:"published,omitempty" pulumi:"published,omitempty,optional"`
	Protocol    string `yaml:"protocol,omitempty" json:"protocol,omitempty" pulumi:"protocol,omitempty,optional"`
	AppProtocol string `yaml:"app_protocol,omitempty" json:"app_protocol,omitempty" pulumi:"app_protocol,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`

	DefangListener string `yaml:"x-defang-listener,omitempty" json:"x-defang-listener,omitempty" pulumi:"defang_listener,omitempty,optional"`
}

// ParsePortConfig parse short syntax for service port configuration
func ParsePortConfig(value string) ([]ServicePortConfig, error) {
	var portConfigs []ServicePortConfig
	ports, portBindings, err := nat.ParsePortSpecs([]string{value})
	if err != nil {
		return nil, err
	}
	// We need to sort the key of the ports to make sure it is consistent
	keys := []string{}
	for port := range ports {
		keys = append(keys, string(port))
	}
	sort.Strings(keys)

	for _, key := range keys {
		port := nat.Port(key)
		converted, err := convertPortToPortConfig(port, portBindings)
		if err != nil {
			return nil, err
		}
		portConfigs = append(portConfigs, converted...)
	}
	return portConfigs, nil
}

func convertPortToPortConfig(port nat.Port, portBindings map[nat.Port][]nat.PortBinding) ([]ServicePortConfig, error) {
	var portConfigs []ServicePortConfig
	for _, binding := range portBindings[port] {
		portConfigs = append(portConfigs, ServicePortConfig{
			HostIP:    binding.HostIP,
			Protocol:  strings.ToLower(port.Proto()),
			Target:    int32(port.Int()),
			Published: binding.HostPort,
			Mode:      "ingress",
		})
	}
	return portConfigs, nil
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type        string               `yaml:"type,omitempty" json:"type,omitempty" pulumi:"type,omitempty,optional"`
	Source      string               `yaml:"source,omitempty" json:"source,omitempty" pulumi:"source,omitempty,optional"`
	Target      string               `yaml:"target,omitempty" json:"target,omitempty" pulumi:"target,omitempty,optional"`
	ReadOnly    bool                 `yaml:"read_only,omitempty" json:"read_only,omitempty" pulumi:"read_only,omitempty,optional"`
	Consistency string               `yaml:"consistency,omitempty" json:"consistency,omitempty" pulumi:"consistency,omitempty,optional"`
	Bind        *ServiceVolumeBind   `yaml:"bind,omitempty" json:"bind,omitempty" pulumi:"bind,omitempty,optional"`
	Volume      *ServiceVolumeVolume `yaml:"volume,omitempty" json:"volume,omitempty" pulumi:"volume,omitempty,optional"`
	Tmpfs       *ServiceVolumeTmpfs  `yaml:"tmpfs,omitempty" json:"tmpfs,omitempty" pulumi:"tmpfs,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// String render ServiceVolumeConfig as a volume string, one can parse back using loader.ParseVolume
func (s ServiceVolumeConfig) String() string {
	access := "rw"
	if s.ReadOnly {
		access = "ro"
	}
	options := []string{access}
	if s.Bind != nil && s.Bind.SELinux != "" {
		options = append(options, s.Bind.SELinux)
	}
	if s.Bind != nil && s.Bind.Propagation != "" {
		options = append(options, s.Bind.Propagation)
	}
	if s.Volume != nil && s.Volume.NoCopy {
		options = append(options, "nocopy")
	}
	return fmt.Sprintf("%s:%s:%s", s.Source, s.Target, strings.Join(options, ","))
}

const (
	// VolumeTypeBind is the type for mounting host dir
	VolumeTypeBind = "bind"
	// VolumeTypeVolume is the type for remote storage volumes
	VolumeTypeVolume = "volume"
	// VolumeTypeTmpfs is the type for mounting tmpfs
	VolumeTypeTmpfs = "tmpfs"
	// VolumeTypeNamedPipe is the type for mounting Windows named pipes
	VolumeTypeNamedPipe = "npipe"
	// VolumeTypeCluster is the type for mounting container storage interface (CSI) volumes
	VolumeTypeCluster = "cluster"

	// SElinuxShared share the volume content
	SElinuxShared = "z"
	// SElinuxUnshared label content as private unshared
	SElinuxUnshared = "Z"
)

// ServiceVolumeBind are options for a service volume of type bind
type ServiceVolumeBind struct {
	SELinux        string `yaml:"selinux,omitempty" json:"selinux,omitempty" pulumi:"selinux,omitempty,optional"`
	Propagation    string `yaml:"propagation,omitempty" json:"propagation,omitempty" pulumi:"propagation,omitempty,optional"`
	CreateHostPath bool   `yaml:"create_host_path,omitempty" json:"create_host_path,omitempty" pulumi:"create_host_path,omitempty,optional"`
	Recursive      string `yaml:"recursive,omitempty" json:"recursive,omitempty" pulumi:"recursive,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// SELinux represents the SELinux re-labeling options.
const (
	// SELinuxShared option indicates that the bind mount content is shared among multiple containers
	SELinuxShared string = "z"
	// SELinuxPrivate option indicates that the bind mount content is private and unshared
	SELinuxPrivate string = "Z"
)

// Propagation represents the propagation of a mount.
const (
	// PropagationRPrivate RPRIVATE
	PropagationRPrivate string = "rprivate"
	// PropagationPrivate PRIVATE
	PropagationPrivate string = "private"
	// PropagationRShared RSHARED
	PropagationRShared string = "rshared"
	// PropagationShared SHARED
	PropagationShared string = "shared"
	// PropagationRSlave RSLAVE
	PropagationRSlave string = "rslave"
	// PropagationSlave SLAVE
	PropagationSlave string = "slave"
)

// ServiceVolumeVolume are options for a service volume of type volume
type ServiceVolumeVolume struct {
	NoCopy  bool   `yaml:"nocopy,omitempty" json:"nocopy,omitempty" pulumi:"nocopy,omitempty,optional"`
	Subpath string `yaml:"subpath,omitempty" json:"subpath,omitempty" pulumi:"subpath,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServiceVolumeTmpfs are options for a service volume of type tmpfs
type ServiceVolumeTmpfs struct {
	Size UnitBytes `yaml:"size,omitempty" json:"size,omitempty" pulumi:"size,omitempty,optional"`

	Mode int32 `yaml:"mode,omitempty" json:"mode,omitempty" pulumi:"mode,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string `yaml:"source,omitempty" json:"source,omitempty" pulumi:"source,omitempty,optional"`
	Target string `yaml:"target,omitempty" json:"target,omitempty" pulumi:"target,omitempty,optional"`
	UID    string `yaml:"uid,omitempty" json:"uid,omitempty" pulumi:"uid,omitempty,optional"`
	GID    string `yaml:"gid,omitempty" json:"gid,omitempty" pulumi:"gid,omitempty,optional"`
	Mode   *int32 `yaml:"mode,omitempty" json:"mode,omitempty" pulumi:"mode,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig

// UlimitsConfig the ulimit configuration
type UlimitsConfig struct {
	Single int `yaml:"single,omitempty" json:"single,omitempty" pulumi:"single,omitempty,optional"`
	Soft   int `yaml:"soft,omitempty" json:"soft,omitempty" pulumi:"soft,omitempty,optional"`
	Hard   int `yaml:"hard,omitempty" json:"hard,omitempty" pulumi:"hard,omitempty,optional"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

func (u *UlimitsConfig) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case *UlimitsConfig:
		// this call to DecodeMapstructure is triggered after initial value conversion as we use a map[string]*UlimitsConfig
		return nil
	case int:
		u.Single = v
		u.Soft = 0
		u.Hard = 0
	case map[string]any:
		u.Single = 0
		soft, ok := v["soft"]
		if ok {
			u.Soft = soft.(int)
		}
		hard, ok := v["hard"]
		if ok {
			u.Hard = hard.(int)
		}
	default:
		return fmt.Errorf("unexpected value type %T for ulimit", value)
	}
	return nil
}

// MarshalYAML makes UlimitsConfig implement yaml.Marshaller
func (u *UlimitsConfig) MarshalYAML() (interface{}, error) {
	if u.Single != 0 {
		return u.Single, nil
	}
	return struct {
		Soft int
		Hard int
	}{
		Soft: u.Soft,
		Hard: u.Hard,
	}, nil
}

// MarshalJSON makes UlimitsConfig implement json.Marshaller
func (u *UlimitsConfig) MarshalJSON() ([]byte, error) {
	if u.Single != 0 {
		return json.Marshal(u.Single)
	}
	// Pass as a value to avoid re-entering this method and use the default implementation
	return json.Marshal(*u)
}

// NetworkConfig for a network
type NetworkConfig struct {
	Name       string     `yaml:"name,omitempty" json:"name,omitempty" pulumi:"name,omitempty,optional"`
	Driver     string     `yaml:"driver,omitempty" json:"driver,omitempty" pulumi:"driver,omitempty,optional"`
	DriverOpts Options    `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty" pulumi:"driver_opts,omitempty,optional"`
	Ipam       IPAMConfig `yaml:"ipam,omitempty" json:"ipam,omitempty" pulumi:"ipam,omitempty"`
	External   External   `yaml:"external,omitempty" json:"external,omitempty" pulumi:"external,omitempty,optional"`
	Internal   bool       `yaml:"internal,omitempty" json:"internal,omitempty" pulumi:"internal,omitempty,optional"`
	Attachable bool       `yaml:"attachable,omitempty" json:"attachable,omitempty" pulumi:"attachable,omitempty,optional"`
	Labels     Labels     `yaml:"labels,omitempty" json:"labels,omitempty" pulumi:"labels,omitempty,optional"`
	EnableIPv6 *bool      `yaml:"enable_ipv6,omitempty" json:"enable_ipv6,omitempty" pulumi:"enable_ipv6,omitempty,optional"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// IPAMConfig for a network
type IPAMConfig struct {
	Driver     string      `yaml:"driver,omitempty" json:"driver,omitempty" pulumi:"driver,omitempty,optional"`
	Config     []*IPAMPool `yaml:"config,omitempty" json:"config,omitempty" pulumi:"config,omitempty,optional"`
	Extensions Extensions  `yaml:"#extensions,inline,omitempty" json:"-"`
}

// IPAMPool for a network
type IPAMPool struct {
	Subnet             string     `yaml:"subnet,omitempty" json:"subnet,omitempty" pulumi:"subnet,omitempty,optional"`
	Gateway            string     `yaml:"gateway,omitempty" json:"gateway,omitempty" pulumi:"gateway,omitempty,optional"`
	IPRange            string     `yaml:"ip_range,omitempty" json:"ip_range,omitempty" pulumi:"ip_range,omitempty,optional"`
	AuxiliaryAddresses Mapping    `yaml:"aux_addresses,omitempty" json:"aux_addresses,omitempty" pulumi:"aux_addresses,omitempty,optional"`
	Extensions         Extensions `yaml:",inline" json:"-"`
}

// VolumeConfig for a volume
type VolumeConfig struct {
	Name       string     `yaml:"name,omitempty" json:"name,omitempty" pulumi:"name,omitempty,optional"`
	Driver     string     `yaml:"driver,omitempty" json:"driver,omitempty" pulumi:"driver,omitempty,optional"`
	DriverOpts Options    `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty" pulumi:"driver_opts,omitempty,optional"`
	External   External   `yaml:"external,omitempty" json:"external,omitempty" pulumi:"external,omitempty,optional"`
	Labels     Labels     `yaml:"labels,omitempty" json:"labels,omitempty" pulumi:"labels,omitempty,optional"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
type External bool

// CredentialSpecConfig for credential spec on Windows
type CredentialSpecConfig struct {
	Config     string     `yaml:"config,omitempty" json:"config,omitempty" pulumi:"config,omitempty,optional"` // Config was added in API v1.40
	File       string     `yaml:"file,omitempty" json:"file,omitempty" pulumi:"file,omitempty,optional"`
	Registry   string     `yaml:"registry,omitempty" json:"registry,omitempty" pulumi:"registry,omitempty,optional"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// FileObjectConfig is a config type for a file used by a service
type FileObjectConfig struct {
	Name           string            `yaml:"name,omitempty" json:"name,omitempty" pulumi:"name,omitempty,optional"`
	File           string            `yaml:"file,omitempty" json:"file,omitempty" pulumi:"file,omitempty,optional"`
	Environment    string            `yaml:"environment,omitempty" json:"environment,omitempty" pulumi:"environment,omitempty,optional"`
	Content        string            `yaml:"content,omitempty" json:"content,omitempty" pulumi:"content,omitempty,optional"`
	External       External          `yaml:"external,omitempty" json:"external,omitempty" pulumi:"external,omitempty,optional"`
	Labels         Labels            `yaml:"labels,omitempty" json:"labels,omitempty" pulumi:"labels,omitempty,optional"`
	Driver         string            `yaml:"driver,omitempty" json:"driver,omitempty" pulumi:"driver,omitempty,optional"`
	DriverOpts     map[string]string `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty" pulumi:"driver_opts,omitempty,optional"`
	TemplateDriver string            `yaml:"template_driver,omitempty" json:"template_driver,omitempty" pulumi:"template_driver,omitempty,optional"`
	Extensions     Extensions        `yaml:"#extensions,inline,omitempty" json:"-"`
}

const (
	// ServiceConditionCompletedSuccessfully is the type for waiting until a service has completed successfully (exit code 0).
	ServiceConditionCompletedSuccessfully = "service_completed_successfully"

	// ServiceConditionHealthy is the type for waiting until a service is healthy.
	ServiceConditionHealthy = "service_healthy"

	// ServiceConditionStarted is the type for waiting until a service has started (default).
	ServiceConditionStarted = "service_started"
)

type DependsOnConfig map[string]ServiceDependency

type ServiceDependency struct {
	Condition  string     `yaml:"condition,omitempty" json:"condition,omitempty" pulumi:"condition,omitempty,optional"`
	Restart    bool       `yaml:"restart,omitempty" json:"restart,omitempty" pulumi:"restart,omitempty,optional"`
	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
	Required   bool       `yaml:"required" json:"required" pulumi:"required"`
}

type ExtendsConfig struct {
	File    string `yaml:"file,omitempty" json:"file,omitempty" pulumi:"file,omitempty,optional"`
	Service string `yaml:"service,omitempty" json:"service,omitempty" pulumi:"service,omitempty,optional"`
}

// SecretConfig for a secret
type SecretConfig FileObjectConfig

// MarshalYAML makes SecretConfig implement yaml.Marshaller
func (s SecretConfig) MarshalYAML() (interface{}, error) {
	// secret content is set while loading model. Never marshall it
	s.Content = ""
	return FileObjectConfig(s), nil
}

// MarshalJSON makes SecretConfig implement json.Marshaller
func (s SecretConfig) MarshalJSON() ([]byte, error) {
	// secret content is set while loading model. Never marshall it
	s.Content = ""
	return json.Marshal(FileObjectConfig(s))
}

// ConfigObjConfig is the config for the swarm "Config" object
type ConfigObjConfig FileObjectConfig

// MarshalYAML makes ConfigObjConfig implement yaml.Marshaller
func (s ConfigObjConfig) MarshalYAML() (interface{}, error) {
	// config content may have been set from environment while loading model. Marshall actual source
	if s.Environment != "" {
		s.Content = ""
	}
	return FileObjectConfig(s), nil
}

// MarshalJSON makes ConfigObjConfig implement json.Marshaller
func (s ConfigObjConfig) MarshalJSON() ([]byte, error) {
	// config content may have been set from environment while loading model. Marshall actual source
	if s.Environment != "" {
		s.Content = ""
	}
	return json.Marshal(FileObjectConfig(s))
}

type IncludeConfig struct {
	Path             StringList `yaml:"path,omitempty" json:"path,omitempty" pulumi:"path,omitempty,optional"`
	ProjectDirectory string     `yaml:"project_directory,omitempty" json:"project_directory,omitempty" pulumi:"project_directory,omitempty,optional"`
	EnvFile          StringList `yaml:"env_file,omitempty" json:"env_file,omitempty" pulumi:"env_file,omitempty,optional"`
}
