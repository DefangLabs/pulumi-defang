package aws

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type Policies struct {
	bedrockPolicy *iam.Policy
	// codeBuildPolicy      *iam.RolePoliciesExclusive
	route53SidecarPolicy *iam.Policy
}

// SharedInfra holds project-level AWS resources shared across all services.
type SharedInfra struct {
	Cluster          *ecs.Cluster
	ExecRole         *iam.Role
	LogGroup         *cloudwatch.LogGroup
	VpcID            pulumi.StringInput
	PublicSubnetIDs  pulumi.StringArrayInput
	PrivateSubnetIDs pulumi.StringArrayInput
	PrivateZoneID    pulumi.IDPtrOutput
	PrivateDomain    string
	Sg               *ec2.SecurityGroup
	HttpListener     *lb.Listener     // nil if no ALB
	HttpsListener    *lb.Listener     // nil if no ALB
	Alb              *lb.LoadBalancer // nil if no ALB
	Region           string
	BuildInfra       *BuildInfra // nil if no builds needed
	SkipNatGW        bool
	Policies
}

// ECSServiceArgs holds per-service arguments for CreateECSService.
type ECSServiceArgs struct {
	Infra    *SharedInfra
	ImageURI pulumi.StringInput // container image URI (built or pre-built)
	Networks compose.Networks   // from project
}

type EcsServiceResult struct {
	Service    *ecs.Service
	Endpoint   pulumix.Output[string]
	HasIngress bool
}

// clampInt clamps v to [minimum, maximum]. Returns fallback if v is 0.
func clampInt[T int | int32 | int64](val T, minimum, maximum, fallback T) T {
	if val == 0 {
		return fallback
	}
	return min(max(val, minimum), maximum)
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
func portProtocol(p compose.ServicePortConfig) awsecs.TransportProtocol {
	proto := compose.GetPortProtocol(p)
	if proto == "udp" {
		return awsecs.TransportProtocolUdp
	}
	return awsecs.TransportProtocolTcp
}

const grpcProto = "grpc"

// CreateECSService creates an ECS Fargate service for a container service.
//
//nolint:funlen,maintidx
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
	portMappings := make([]awsecs.PortMapping, 0, len(svc.Ports))
	for _, p := range svc.Ports {
		portMappings = append(portMappings, awsecs.PortMapping{
			ContainerPort: ptr.Int32(p.Target),
			// HostPort:      p.Target,
			Protocol: portProtocol(p),
		})
	}

	// Build health check with clamped values (matches TS clamp ranges)
	var healthCheck *awsecs.HealthCheck
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		healthCheck = &awsecs.HealthCheck{
			Command:  svc.HealthCheck.Test,
			Interval: ptr.Int32(clampInt(svc.HealthCheck.IntervalSeconds, 5, 300, 30)),
			Timeout:  ptr.Int32(clampInt(svc.HealthCheck.TimeoutSeconds, 2, 60, 5)),
			Retries:  ptr.Int32(clampInt(svc.HealthCheck.Retries, 1, 10, 3)),
		}
		startPeriod := clampInt(svc.HealthCheck.StartPeriodSeconds, 0, 300, 0)
		if startPeriod > 0 {
			healthCheck.StartPeriod = ptr.Int32(startPeriod)
		}
	}

	infra := args.Infra

	// Build environment variables (plain strings, resolved before JSON marshaling)
	envVars := []awsecs.KeyValuePair{
		{Name: ptr.String("DEFANG_SERVICE"), Value: ptr.String(serviceName)},
	}
	if svc.DomainName != nil {
		envVars = append(envVars, awsecs.KeyValuePair{Name: ptr.String("DEFANG_FQDN"), Value: ptr.String(*svc.DomainName)})
	}
	for k, v := range svc.Environment {
		envVars = append(envVars, awsecs.KeyValuePair{Name: ptr.String(k), Value: ptr.String(v)})
	}

	// Resolve outputs (image URI, log group name) before building the container
	// definitions JSON. The ECS ContainerDefinitions field is a plain JSON string,
	// so all values must be concrete before marshaling.
	containerName := serviceName

	allInputs := []interface{}{args.ImageURI}
	hasLogGroup := infra != nil && infra.LogGroup != nil
	if hasLogGroup {
		allInputs = append(allInputs, infra.LogGroup.Name)
	}
	hasPrivateZone := infra != nil && infra.PrivateZoneID != (pulumi.IDPtrOutput{})
	if hasPrivateZone {
		allInputs = append(allInputs, infra.PrivateZoneID)
	}

	containerDefsJSON := pulumi.All(allInputs...).ApplyT(func(all []any) (string, error) {
		imageUri := all[0].(string)

		var logConfiguration *awsecs.LogConfiguration
		if hasLogGroup {
			logGroupName := all[1].(string)
			logConfiguration = &awsecs.LogConfiguration{
				LogDriver: awsecs.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         logGroupName,
					"awslogs-region":        infra.Region,
					"awslogs-stream-prefix": containerName,
				},
			}
		}

		containerDefs := []awsecs.ContainerDefinition{{
			Name:             &containerName,
			Essential:        ptr.Bool(true),
			PortMappings:     portMappings,
			Environment:      envVars,
			Command:          svc.Command,
			EntryPoint:       svc.Entrypoint,
			HealthCheck:      healthCheck,
			Image:            &imageUri,
			LogConfiguration: logConfiguration,
		}}

		if svc.HasHostPorts() && hasPrivateZone {
			privateZoneID := all[2].(*pulumi.ID)
			privateFqdn := fmt.Sprintf("%s.%s", serviceName, infra.PrivateDomain)
			containerDefs = append(containerDefs, awsecs.ContainerDefinition{
				Name:      ptr.String("route53-sidecar"),
				Image:     ptr.String("public.ecr.aws/defang-io/route53-sidecar:65e431c"),
				Essential: ptr.Bool(false),
				Environment: []awsecs.KeyValuePair{
					{Name: ptr.String("HOSTEDZONE"), Value: ptr.String(string(*privateZoneID))},
					{Name: ptr.String("DNS"), Value: ptr.String(privateFqdn)},
					{Name: ptr.String("IPADDRESS"), Value: ptr.String("ecs")},
					// not (always?) set by the ECS agent; https://github.com/aws/containers-roadmap/issues/1611
					{Name: ptr.String("AWS_REGION"), Value: ptr.String(infra.Region)},
				},
				DependsOn: []awsecs.ContainerDependency{{
					ContainerName: &containerName,
					Condition:     awsecs.ContainerConditionStart,
				}},
			})
		}
		bytes, err := json.Marshal(containerDefs)
		return string(bytes), err
	}).(pulumi.StringOutput)

	fargateCPU, fargateMemory := fargateResources(cpus, memMiB)

	cpuArch := awsecs.CPUArchitectureX8664
	if platformToArch(svc.GetPlatform()) == Arm64 {
		cpuArch = awsecs.CPUArchitectureArm64
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
			CpuArchitecture:       pulumi.String(string(cpuArch)),
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

			tgName := targetGroupName(serviceName, int(port.Target), appProto)

			// Target group health check (matches TS createTargetGroup in lb.ts)
			defaultInterval := HealthCheckInterval.Get(ctx)
			interval := defaultInterval
			if svc.HealthCheck != nil {
				interval = clampInt(int(svc.HealthCheck.IntervalSeconds), 5, 300, defaultInterval)
			}
			maxTimeout := interval - 1
			if maxTimeout > 120 {
				maxTimeout = 120
			}
			timeout := (6)
			if svc.HealthCheck != nil {
				timeout = clampInt(int(svc.HealthCheck.TimeoutSeconds), 2, maxTimeout, 6)
			}
			unhealthyThreshold := (3)
			if svc.HealthCheck != nil {
				unhealthyThreshold = clampInt(int(svc.HealthCheck.Retries), 2, 10, 3)
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
				VpcId:                      infra.VpcID,
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
			switch {
			case firstIngress && svc.DomainName != nil:
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
						Values: pulumi.StringArray{pulumi.String(*svc.DomainName)},
					},
				})
			case infra.Alb != nil:
				// Fall back to ALB DNS name (matches TS fallback behavior)
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
						Values: pulumi.StringArray{infra.Alb.DnsName},
					},
				})
			default:
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
				ContainerName:  pulumi.String(containerName),
				ContainerPort:  pulumi.Int(port.Target),
				TargetGroupArn: tg.Arn,
			})
			lbDependsOn = append(lbDependsOn, lr)
			firstIngress = false
		}
	}

	isPrivate := !common.AcceptPublicTraffic(args.Networks, svc)
	assignPublicIp :=
		(infra.SkipNatGW && common.AllowEgress(args.Networks, svc)) || !isPrivate

	subnetIds := infra.PrivateSubnetIDs
	if assignPublicIp || !isPrivate {
		subnetIds = infra.PublicSubnetIDs
	}

	// Create ECS service with circuit breaker and managed tags (matches TS createEcsService)
	ecsServiceArgs := &ecs.ServiceArgs{
		Cluster:        infra.Cluster.Arn,
		TaskDefinition: taskDef.Arn,
		DesiredCount:   pulumi.Int(replicas),
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        subnetIds,
			SecurityGroups: pulumi.StringArray{infra.Sg.ID()},
			AssignPublicIp: pulumi.Bool(assignPublicIp),
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

	ecsServiceOpts := append(append([]pulumi.ResourceOption{}, opts...), pulumi.DependsOn(lbDependsOn))
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
