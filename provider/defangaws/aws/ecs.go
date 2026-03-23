package aws

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// SharedInfra holds project-level AWS resources shared across all services.
type SharedInfra struct {
	Cluster          *ecs.Cluster
	ExecRole         *iam.Role
	LogGroup         *cloudwatch.LogGroup
	VpcID            pulumi.StringInput
	PublicSubnetIDs  pulumi.StringArrayInput
	PrivateSubnetIDs pulumi.StringArrayInput
	PrivateZoneID    pulumi.IDInput
	PrivateDomain    string
	Sg               *ec2.SecurityGroup
	HttpListener     *lb.Listener     // nil if no ALB
	Alb              *lb.LoadBalancer // nil if no ALB
	Region           string
	ImageInfra       *ImageInfra // nil if no builds needed
}

// ECSServiceArgs holds per-service arguments for CreateECSService.
type ECSServiceArgs struct {
	Infra    *SharedInfra
	ImageURI pulumi.StringInput // container image URI (built or pre-built)
}

type EcsServiceResult struct {
	Service    *ecs.Service
	Endpoint   pulumix.Output[string]
	HasIngress bool
}

// clampInt clamps v to [minimum, maximum]. Returns fallback if v is nil.
func clampInt(v *int, minimum, maximum, fallback int) int {
	if v == nil {
		return fallback
	}
	val := *v
	if val < minimum {
		return minimum
	}
	if val > maximum {
		return maximum
	}
	return val
}

// makeMinMaxCeil rounds value up to the nearest step within [minimum, maximum].
// Matches TS makeMinMaxCeil(value, min, max, step).
func makeMinMaxCeil(value, minimum, maximum, step int) int {
	if value <= minimum {
		return minimum
	}
	if value >= maximum {
		return maximum
	}
	return ((value + step - 1) / step) * step
}

// fixupFargateCPU returns the nearest valid Fargate CPU value in units.
// Matches TS: 2 ** makeMinMaxCeil(Math.log2(vCpu) + 10, 8, 14)
// Valid values: 256, 512, 1024, 2048, 4096, 8192, 16384
func fixupFargateCPU(vCpu float64) int {
	if vCpu <= 0 {
		return 256
	}
	exp := math.Log2(vCpu) + 10
	expCeil := int(math.Ceil(exp))
	if expCeil < 8 {
		expCeil = 8
	}
	if expCeil > 14 {
		expCeil = 14
	}
	return 1 << expCeil
}

// fixupFargateMemory returns the nearest valid Fargate memory in MiB for the given CPU units.
// See: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-cpu-memory-error.html
func fixupFargateMemory(cpu, memoryMiB int) int {
	switch cpu {
	case 256:
		return makeMinMaxCeil(memoryMiB, 512, 2048, 1024)
	case 512:
		return makeMinMaxCeil(memoryMiB, 1024, 4096, 1024)
	case 1024:
		return makeMinMaxCeil(memoryMiB, 2048, 8192, 1024)
	case 2048:
		return makeMinMaxCeil(memoryMiB, 4096, 16384, 1024)
	case 4096:
		return makeMinMaxCeil(memoryMiB, 8192, 30720, 1024)
	case 8192:
		return makeMinMaxCeil(memoryMiB, 16384, 61440, 4096)
	case 16384:
		return makeMinMaxCeil(memoryMiB, 32768, 122880, 4096)
	default:
		panic("Unsupported value for cpu: " + strconv.Itoa(cpu))
	}
}

// fargateResources returns valid Fargate CPU (units) and memory (MiB) as strings.
// Matches TS fixupFargateConfig: tries increasing CPU tiers until memory fits.
func fargateResources(cpus float64, memoryMiB int) (string, string) {
	cpuUnits := fixupFargateCPU(cpus)
	mem := fixupFargateMemory(cpuUnits, memoryMiB)
	// If memory exceeds this CPU tier's max, bump to next CPU tier
	for mem < memoryMiB {
		cpuUnits *= 2
		mem = fixupFargateMemory(cpuUnits, memoryMiB)
	}
	return strconv.Itoa(cpuUnits), strconv.Itoa(mem)
}

// portProtocol normalizes the transport protocol for ECS container port mappings.
// Only "tcp" and "udp" are valid; matches TS: ep.protocol === "udp" ? "udp" : "tcp"
func portProtocol(p compose.ServicePortConfig) ContainerPortProtocol {
	proto := compose.GetPortProtocol(p)
	if proto == "udp" {
		return "udp"
	}
	return "tcp"
}

const grpcProto = "grpc"

// CreateECSService creates an ECS Fargate service for a container service.
func CreateECSService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	args *ECSServiceArgs,
	deps []pulumi.Resource,
	opts ...pulumi.ResourceOption,
) (*EcsServiceResult, error) {
	// Create task role
	taskRole, err := createTaskRole(ctx, serviceName, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating task role: %w", err)
	}

	// Build container definition
	cpus := svc.GetCPUs()
	memMiB := svc.GetMemoryMiB()

	// Build port mappings (protocol normalized to tcp/udp, hostPort = containerPort)
	portMappings := make([]ContainerPortMapping, 0, len(svc.Ports))
	for _, p := range svc.Ports {
		portMappings = append(portMappings, ContainerPortMapping{
			ContainerPort: p.Target,
			// HostPort:      p.Target,
			Protocol: portProtocol(p),
		})
	}

	// Build health check with clamped values (matches TS clamp ranges)
	var healthCheck *ContainerHealthCheck
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		healthCheck = &ContainerHealthCheck{
			Command:  pulumi.ToStringArray(svc.HealthCheck.Test),
			Interval: clampInt(svc.HealthCheck.IntervalSeconds, 5, 300, 30),
			Timeout:  clampInt(svc.HealthCheck.TimeoutSeconds, 2, 60, 5),
			Retries:  clampInt(svc.HealthCheck.Retries, 1, 10, 3),
		}
		startPeriod := clampInt(svc.HealthCheck.StartPeriodSeconds, 0, 300, 0)
		if startPeriod > 0 {
			healthCheck.StartPeriod = startPeriod
		}
	}

	infra := args.Infra

	// Build environment variables (plain strings, resolved before JSON marshaling)
	type plainEnvVar struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	envVars := []plainEnvVar{
		{Name: "DEFANG_SERVICE", Value: serviceName},
	}
	if svc.DomainName != nil {
		envVars = append(envVars, plainEnvVar{Name: "DEFANG_FQDN", Value: *svc.DomainName})
	}
	for k, v := range svc.Environment {
		envVars = append(envVars, plainEnvVar{Name: k, Value: v})
	}

	// Build health check for JSON (plain types, no Pulumi outputs)
	type plainHealthCheck struct {
		Command     []string `json:"command"`
		Interval    int      `json:"interval,omitempty"`
		Timeout     int      `json:"timeout,omitempty"`
		Retries     int      `json:"retries,omitempty"`
		StartPeriod int      `json:"startPeriod,omitempty"`
	}
	var plainHC *plainHealthCheck
	if healthCheck != nil {
		plainHC = &plainHealthCheck{
			Command:     svc.HealthCheck.Test,
			Interval:    healthCheck.Interval,
			Timeout:     healthCheck.Timeout,
			Retries:     healthCheck.Retries,
			StartPeriod: healthCheck.StartPeriod,
		}
	}

	// Resolve outputs (image URI, log group name) before building the container
	// definitions JSON. The ECS ContainerDefinitions field is a plain JSON string,
	// so all values must be concrete before marshaling.
	containerDefsJSON := pulumi.All(args.ImageURI, infra.LogGroup.Name).ApplyT(
		func(resolved []interface{}) (string, error) {
			imageURI := resolved[0].(string)
			logGroupName := resolved[1].(string)

			type plainLogConfig struct {
				LogDriver string            `json:"logDriver"`
				Options   map[string]string `json:"options,omitempty"`
			}
			type plainContainerDef struct {
				Name             string                 `json:"name"`
				Image            string                 `json:"image"`
				Essential        *bool                  `json:"essential,omitempty"`
				PortMappings     []ContainerPortMapping `json:"portMappings,omitempty"`
				Environment      []plainEnvVar          `json:"environment,omitempty"`
				Command          []string               `json:"command,omitempty"`
				EntryPoint       []string               `json:"entryPoint,omitempty"`
				HealthCheck      *plainHealthCheck      `json:"healthCheck,omitempty"`
				LogConfiguration *plainLogConfig        `json:"logConfiguration,omitempty"`
			}

			essential := true
			containerDefs := []plainContainerDef{{
				Name:         serviceName,
				Essential:    &essential,
				PortMappings: portMappings,
				Environment:  envVars,
				Command:      svc.Command,
				EntryPoint:   svc.Entrypoint,
				HealthCheck:  plainHC,
				Image:        imageURI,
				LogConfiguration: &plainLogConfig{
					LogDriver: "awslogs",
					Options: map[string]string{
						"awslogs-group":         logGroupName,
						"awslogs-region":        infra.Region,
						"awslogs-stream-prefix": serviceName,
					},
				},
			}}

			jsonBytes, err := json.Marshal(containerDefs)
			if err != nil {
				return "", fmt.Errorf("marshaling container definitions: %w", err)
			}
			return string(jsonBytes), nil
		}).(pulumi.StringOutput)

	fargateCPU, fargateMemory := fargateResources(cpus, memMiB)

	cpuArch := X86_64
	if platformToArch(svc.GetPlatform()) == Arm64 {
		cpuArch = Arm64
	}

	// Create task definition
	taskDef, err := ecs.NewTaskDefinition(ctx, serviceName, &ecs.TaskDefinitionArgs{
		Family:                  pulumi.String(serviceName),
		NetworkMode:             pulumi.String("awsvpc"),
		RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
		Cpu:                     pulumi.String(fargateCPU),
		Memory:                  pulumi.String(fargateMemory),
		ExecutionRoleArn:        infra.ExecRole.Arn,
		TaskRoleArn:             taskRole.Arn,
		ContainerDefinitions:    containerDefsJSON,
		RuntimePlatform: &ecs.TaskDefinitionRuntimePlatformArgs{
			CpuArchitecture:       pulumi.String(cpuArch),
			OperatingSystemFamily: pulumi.String("LINUX"),
		},
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating task definition: %w", err)
	}

	replicas := svc.GetReplicas()

	// Create target group and listener rule if service has ingress ports
	var loadBalancers ecs.ServiceLoadBalancerArray
	var lbDependsOn []pulumi.Resource
	var endpointOutput pulumix.Output[string]

	if svc.HasIngressPorts() && infra.HttpListener != nil {
		firstIngress := true
		for _, port := range svc.Ports {
			if port.Mode != "ingress" {
				continue
			}

			// Only create TG/LR for http, http2, grpc (matches TS createTgLrPair)
			appProto := compose.GetAppProtocol(port)
			if appProto != "http" && appProto != "http2" && appProto != "grpc" {
				continue
			}

			tgName := targetGroupName(serviceName, port.Target, appProto)

			// Target group health check (matches TS createTargetGroup in lb.ts)
			defaultInterval := HealthCheckInterval.Get(ctx)
			interval := defaultInterval
			if svc.HealthCheck != nil {
				interval = clampInt(svc.HealthCheck.IntervalSeconds, 5, 300, defaultInterval)
			}
			maxTimeout := interval - 1
			if maxTimeout > 120 {
				maxTimeout = 120
			}
			timeout := 6
			if svc.HealthCheck != nil {
				timeout = clampInt(svc.HealthCheck.TimeoutSeconds, 2, maxTimeout, 6)
			}
			unhealthyThreshold := 3
			if svc.HealthCheck != nil {
				unhealthyThreshold = clampInt(svc.HealthCheck.Retries, 2, 10, 3)
			}

			// Determine matcher based on protocol (matches TS createTargetGroup)
			// With default path "/": grpc -> "0", http/http2 -> "200-399"
			matcher := "200-399"
			if appProto == grpcProto {
				matcher = "0"
			}

			tgArgs := &lb.TargetGroupArgs{
				Port:                       pulumi.Int(port.Target),
				Protocol:                   pulumi.String("HTTP"),
				TargetType:                 pulumi.String("ip"),
				VpcId:                      pulumi.StringInput(infra.VpcID),
				LoadBalancingAlgorithmType: pulumi.String("least_outstanding_requests"),
				DeregistrationDelay:        pulumi.Int(DeregistrationDelay.Get(ctx)),
				HealthCheck: &lb.TargetGroupHealthCheckArgs{
					// Port:               pulumi.String("traffic-port"),
					Path:               pulumi.String("/"),
					HealthyThreshold:   pulumi.Int(HealthCheckThreshold.Get(ctx)),
					UnhealthyThreshold: pulumi.Int(unhealthyThreshold),
					Interval:           pulumi.Int(interval),
					Timeout:            pulumi.Int(timeout),
					Matcher:            pulumi.String(matcher),
				},
				Tags: pulumi.StringMap{
					"defang:service": pulumi.String(serviceName),
				},
			}

			// Set protocol version for http2/grpc (matches TS createTargetGroup)
			switch appProto {
			case "http2":
				tgArgs.ProtocolVersion = pulumi.String("HTTP2")
			case "grpc":
				tgArgs.ProtocolVersion = pulumi.String("GRPC")
			}

			tg, tgErr := lb.NewTargetGroup(ctx, tgName, tgArgs, opts...)
			if tgErr != nil {
				return nil, fmt.Errorf("creating target group: %w", tgErr)
			}

			// Build listener rule conditions (matches TS createTgLrPair)
			conditions := lb.ListenerRuleConditionArray{}

			// Host-based routing: use DomainName for first ingress port, ALB DNS as fallback
			if firstIngress && svc.DomainName != nil {
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
						Values: pulumi.StringArray{pulumi.String(*svc.DomainName)},
					},
				})
			} else if infra.Alb != nil {
				// Fall back to ALB DNS name (matches TS fallback behavior)
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
						Values: pulumi.StringArray{infra.Alb.DnsName},
					},
				})
			} else {
				// Last resort: path-based routing
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					PathPattern: &lb.ListenerRuleConditionPathPatternArgs{
						Values: pulumi.StringArray{pulumi.String("/*")},
					},
				})
			}

			// Add gRPC content-type header matching (matches TS createTgLrPair)
			if appProto == grpcProto {
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HttpHeader: &lb.ListenerRuleConditionHttpHeaderArgs{
						HttpHeaderName: pulumi.String("content-type"),
						Values:         pulumi.StringArray{pulumi.String("application/grpc*")},
					},
				})
			}

			lr, lrErr := lb.NewListenerRule(ctx, tgName+"-rule", &lb.ListenerRuleArgs{
				ListenerArn: infra.HttpListener.Arn,
				Actions: lb.ListenerRuleActionArray{
					&lb.ListenerRuleActionArgs{
						Type:           pulumi.String("forward"),
						TargetGroupArn: tg.Arn,
					},
				},
				Conditions: conditions,
			}, append(opts, pulumi.DeleteBeforeReplace(true))...)
			if lrErr != nil {
				return nil, fmt.Errorf("creating listener rule: %w", lrErr)
			}

			loadBalancers = append(loadBalancers, &ecs.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String(serviceName),
				ContainerPort:  pulumi.Int(port.Target),
				TargetGroupArn: tg.Arn,
			})
			lbDependsOn = append(lbDependsOn, lr)
			firstIngress = false
		}
	}

	// Create ECS service with circuit breaker and managed tags (matches TS createEcsService)
	ecsServiceArgs := &ecs.ServiceArgs{
		Cluster:        infra.Cluster.Arn,
		TaskDefinition: taskDef.Arn,
		DesiredCount:   pulumi.Int(replicas),
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        pulumi.StringArrayInput(infra.PublicSubnetIDs),
			SecurityGroups: pulumi.StringArray{infra.Sg.ID()},
			AssignPublicIp: pulumi.Bool(true),
		},
		LoadBalancers: loadBalancers,
		DeploymentCircuitBreaker: &ecs.ServiceDeploymentCircuitBreakerArgs{
			Enable:   pulumi.Bool(true),
			Rollback: pulumi.Bool(true),
		},
		DeploymentMinimumHealthyPercent: pulumi.Int(MinHealthyPercent.Get(ctx)),
		EnableEcsManagedTags:            pulumi.Bool(true),
		PropagateTags:                   pulumi.String("SERVICE"),
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
		},
	}

	// Use capacity provider strategy if configured, otherwise default launch type
	capacityProvider := FargateCapacityProvider.Get(ctx)
	if capacityProvider != "" {
		ecsServiceArgs.CapacityProviderStrategies = ecs.ServiceCapacityProviderStrategyArray{
			&ecs.ServiceCapacityProviderStrategyArgs{
				CapacityProvider: pulumi.String(capacityProvider),
				Weight:           pulumi.Int(1),
			},
		}
	} else {
		ecsServiceArgs.LaunchType = pulumi.String("FARGATE")
	}

	ecsServiceOpts := append(opts, pulumi.DependsOn(lbDependsOn))
	if len(deps) > 0 {
		ecsServiceOpts = append(ecsServiceOpts, pulumi.DependsOn(deps))
	}
	ecsService, err := ecs.NewService(ctx, serviceName, ecsServiceArgs, ecsServiceOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service: %w", err)
	}

	hasIngress := svc.HasIngressPorts() && infra.HttpListener != nil
	if hasIngress {
		endpointOutput = pulumix.Val(fmt.Sprintf("service %s via ALB", serviceName))
	}

	return &EcsServiceResult{
		Service:    ecsService,
		Endpoint:   endpointOutput,
		HasIngress: hasIngress,
	}, nil
}

// BuildSharedInfra creates all shared AWS infrastructure for a standalone ECS service.
// The AWS provider must be passed via opts (pulumi.Providers on the parent component).
func BuildSharedInfra(
	ctx *pulumi.Context, serviceName string, svc compose.ServiceConfig, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption
) (*SharedInfra, error) {
	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := ResolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(net.VpcID),
		Description: pulumi.String("Security group for services"),
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := CreateExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	var imgInfra *ImageInfra
	if svc.NeedsBuild() {
		imgInfra, err = CreateImageInfra(ctx, logGroup, region.Name, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating image build infrastructure: %w", err)
		}
	}

	var httpListener *lb.Listener
	var svcALB *lb.LoadBalancer
	if svc.HasIngressPorts() {
		albRes, err := CreateALB(ctx, net.VpcID, net.PublicSubnetIDs, sg, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albRes.HttpListener
		svcALB = albRes.Alb
	}

	return &SharedInfra{
		Cluster:          cluster,
		ExecRole:         execRole,
		LogGroup:         logGroup,
		VpcID:            net.VpcID,
		PublicSubnetIDs:  net.PublicSubnetIDs,
		PrivateSubnetIDs: net.PrivateSubnetIDs,
		PrivateZoneID:    net.PrivateZoneID,
		PrivateDomain:    net.PrivateDomain,
		Sg:               sg,
		HttpListener:     httpListener,
		Alb:              svcALB,
		Region:           region.Name,
		ImageInfra:       imgInfra,
	}, nil
}

// BuildProjectInfra creates shared AWS infrastructure for a multi-service project.
func BuildProjectInfra(
	ctx *pulumi.Context, projectName string, services map[string]compose.ServiceConfig, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption
) (*SharedInfra, error) {
	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := ResolveNetworking(ctx, awsCfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, "svc-sg", &ec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(net.VpcID),
		Description: pulumi.String(fmt.Sprintf("Security group for %s services", projectName)),
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := CreateExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	var imgInfra *ImageInfra
	for _, svc := range services {
		if svc.NeedsBuild() {
			imgInfra, err = CreateImageInfra(ctx, logGroup, region.Name, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating image build infrastructure: %w", err)
			}
			break
		}
	}

	var httpListener *lb.Listener
	var alb *lb.LoadBalancer
	if common.NeedIngress(services) {
		albRes, err := CreateALB(ctx, net.VpcID, net.PublicSubnetIDs, sg, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albRes.HttpListener
		alb = albRes.Alb
	}

	return &SharedInfra{
		Cluster:          cluster,
		ExecRole:         execRole,
		LogGroup:         logGroup,
		VpcID:            net.VpcID,
		PublicSubnetIDs:  net.PublicSubnetIDs,
		PrivateSubnetIDs: net.PrivateSubnetIDs,
		PrivateZoneID:    net.PrivateZoneID,
		PrivateDomain:    net.PrivateDomain,
		Sg:               sg,
		HttpListener:     httpListener,
		Alb:              alb,
		Region:           region.Name,
		ImageInfra:       imgInfra,
	}, nil
}
