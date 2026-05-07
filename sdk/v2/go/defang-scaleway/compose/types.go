package compose

import (
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ServiceConfigArgs struct {
	Image       pulumi.StringPtrInput
	Platform    pulumi.StringPtrInput
	Build       BuildConfigInput
	Ports       ServicePortConfigArrayInput
	Deploy      DeployConfigInput
	Environment pulumi.StringMapInput
	Command     pulumi.StringArrayInput
	Entrypoint  pulumi.StringArrayInput
	Postgres    PostgresConfigInput
	Redis       RedisConfigInput
	HealthCheck HealthCheckConfigInput
	Networks    ServiceNetworkConfigMapInput
	DependsOn   ServiceDependencyMapInput
	Llm         LlmConfigInput
	DomainName  pulumi.StringPtrInput
}

func (ServiceConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*serviceConfig)(nil)).Elem() }

type serviceConfig struct {
	Image       *string                  `pulumi:"image"`
	Platform    *string                  `pulumi:"platform"`
	Build       *buildConfig             `pulumi:"build"`
	Ports       []servicePortConfig      `pulumi:"ports"`
	Deploy      *deployConfig            `pulumi:"deploy"`
	Environment map[string]string        `pulumi:"environment"`
	Command     []string                 `pulumi:"command"`
	Entrypoint  []string                 `pulumi:"entrypoint"`
	Postgres    *postgresConfig          `pulumi:"postgres"`
	Redis       *redisConfig             `pulumi:"redis"`
	HealthCheck *healthCheckConfig       `pulumi:"healthCheck"`
	Networks    map[string]serviceNetworkConfig `pulumi:"networks"`
	DependsOn   map[string]serviceDependency    `pulumi:"dependsOn"`
	Llm         *llmConfig               `pulumi:"llm"`
	DomainName  *string                  `pulumi:"domainName"`
}

type ServiceConfigMap map[string]ServiceConfigArgs
type ServiceConfigMapInput interface{ pulumi.Input; ToServiceConfigMapOutput() ServiceConfigMapOutput }
type ServiceConfigMapOutput struct{ *pulumi.OutputState }
func (ServiceConfigMap) ElementType() reflect.Type { return reflect.TypeOf(map[string]serviceConfig{}) }
func (v ServiceConfigMap) ToServiceConfigMapOutput() ServiceConfigMapOutput {
	return pulumi.ToOutput(v).(ServiceConfigMapOutput)
}
func (ServiceConfigMapOutput) ElementType() reflect.Type { return reflect.TypeOf(map[string]serviceConfig{}) }
func (v ServiceConfigMapOutput) ToServiceConfigMapOutput() ServiceConfigMapOutput { return v }

type BuildConfigArgs struct {
	Context    string
	Dockerfile pulumi.StringPtrInput
	Args       pulumi.StringMapInput
	ShmSize    pulumi.StringPtrInput
	Target     pulumi.StringPtrInput
}
type buildConfig struct {
	Context    string            `pulumi:"context"`
	Dockerfile *string           `pulumi:"dockerfile"`
	Args       map[string]string `pulumi:"args"`
	ShmSize    *string           `pulumi:"shmSize"`
	Target     *string           `pulumi:"target"`
}
type BuildConfigInput interface{ pulumi.Input }
func (BuildConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*buildConfig)(nil)).Elem() }

type ServicePortConfigArgs struct {
	Target      pulumi.IntInput
	Mode        pulumi.StringInput
	Protocol    pulumi.StringInput
	AppProtocol pulumi.StringInput
}
type servicePortConfig struct {
	Target      int    `pulumi:"target"`
	Mode        string `pulumi:"mode"`
	Protocol    string `pulumi:"protocol"`
	AppProtocol string `pulumi:"appProtocol"`
}
type ServicePortConfigArray []ServicePortConfigArgs
type ServicePortConfigArrayInput interface{ pulumi.Input }
func (ServicePortConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*servicePortConfig)(nil)).Elem() }
func (ServicePortConfigArray) ElementType() reflect.Type { return reflect.TypeOf([]servicePortConfig{}) }

type DeployConfigArgs struct {
	Replicas  pulumi.IntPtrInput
	Resources ResourcesInput
}
type deployConfig struct {
	Replicas  *int      `pulumi:"replicas"`
	Resources resources `pulumi:"resources"`
}
type DeployConfigInput interface{ pulumi.Input }
func (DeployConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*deployConfig)(nil)).Elem() }

type ResourcesArgs struct{ Reservations ResourceConfigInput }
type resources struct{ Reservations resourceConfig `pulumi:"reservations"` }
type ResourcesInput interface{ pulumi.Input }
func (ResourcesArgs) ElementType() reflect.Type { return reflect.TypeOf((*resources)(nil)).Elem() }

type ResourceConfigArgs struct {
	Cpus   pulumi.Float64PtrInput
	Memory pulumi.StringPtrInput
}
type resourceConfig struct {
	Cpus   *float64 `pulumi:"cpus"`
	Memory *string  `pulumi:"memory"`
}
type ResourceConfigInput interface{ pulumi.Input }
func (ResourceConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*resourceConfig)(nil)).Elem() }

type PostgresConfigArgs struct {
	AllowDowntime pulumi.BoolPtrInput
	FromSnapshot  pulumi.StringPtrInput
}
type postgresConfig struct {
	AllowDowntime *bool   `pulumi:"allowDowntime"`
	FromSnapshot  *string `pulumi:"fromSnapshot"`
}
type PostgresConfigInput interface{ pulumi.Input }
func (PostgresConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*postgresConfig)(nil)).Elem() }

type RedisConfigArgs struct {
	AllowDowntime pulumi.BoolPtrInput
	FromSnapshot  pulumi.StringPtrInput
}
type redisConfig struct {
	AllowDowntime *bool   `pulumi:"allowDowntime"`
	FromSnapshot  *string `pulumi:"fromSnapshot"`
}
type RedisConfigInput interface{ pulumi.Input }
func (RedisConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*redisConfig)(nil)).Elem() }

type HealthCheckConfigArgs struct {
	Test               pulumi.StringArrayInput
	IntervalSeconds    pulumi.IntPtrInput
	TimeoutSeconds     pulumi.IntPtrInput
	Retries            pulumi.IntPtrInput
	StartPeriodSeconds pulumi.IntPtrInput
}
type healthCheckConfig struct {
	Test               []string `pulumi:"test"`
	IntervalSeconds    *int     `pulumi:"intervalSeconds"`
	TimeoutSeconds     *int     `pulumi:"timeoutSeconds"`
	Retries            *int     `pulumi:"retries"`
	StartPeriodSeconds *int     `pulumi:"startPeriodSeconds"`
}
type HealthCheckConfigInput interface{ pulumi.Input }
func (HealthCheckConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*healthCheckConfig)(nil)).Elem() }

type NetworkConfigArgs struct{ Internal pulumi.BoolInput }
type networkConfig struct{ Internal bool `pulumi:"internal"` }
type NetworkConfigMap map[string]NetworkConfigArgs
type NetworkConfigMapInput interface{ pulumi.Input }
func (NetworkConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*networkConfig)(nil)).Elem() }
func (NetworkConfigMap) ElementType() reflect.Type { return reflect.TypeOf(map[string]networkConfig{}) }

type ServiceNetworkConfigArgs struct{ Aliases pulumi.StringArrayInput }
type serviceNetworkConfig struct{ Aliases []string `pulumi:"aliases"` }
type ServiceNetworkConfigMap map[string]ServiceNetworkConfigArgs
type ServiceNetworkConfigMapInput interface{ pulumi.Input }
func (ServiceNetworkConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*serviceNetworkConfig)(nil)).Elem() }
func (ServiceNetworkConfigMap) ElementType() reflect.Type { return reflect.TypeOf(map[string]serviceNetworkConfig{}) }

type ServiceDependencyArgs struct {
	Condition pulumi.StringInput
	Required  pulumi.BoolInput
}
type serviceDependency struct {
	Condition string `pulumi:"condition"`
	Required  bool   `pulumi:"required"`
}
type ServiceDependencyMap map[string]ServiceDependencyArgs
type ServiceDependencyMapInput interface{ pulumi.Input }
func (ServiceDependencyArgs) ElementType() reflect.Type { return reflect.TypeOf((*serviceDependency)(nil)).Elem() }
func (ServiceDependencyMap) ElementType() reflect.Type { return reflect.TypeOf(map[string]serviceDependency{}) }

type LlmConfigArgs struct{}
type llmConfig struct{}
type LlmConfigInput interface{ pulumi.Input }
func (LlmConfigArgs) ElementType() reflect.Type { return reflect.TypeOf((*llmConfig)(nil)).Elem() }
