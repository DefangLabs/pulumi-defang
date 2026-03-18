package aws

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ecsServiceArgs struct {
	cluster   *ecs.Cluster
	execRole  *iam.Role
	logGroup  *cloudwatch.LogGroup
	vpcID     pulumi.StringOutput
	subnetIDs pulumi.StringArrayOutput
	sg        *ec2.SecurityGroup
	listener  *lb.Listener     // nil if no ALB
	alb       *lb.LoadBalancer // nil if no ALB
	region    string
	imageURI  pulumi.StringOutput // container image URI (built or pre-built)
}

type ecsServiceResult struct {
	service    *ecs.Service
	endpoint   pulumi.StringOutput
	hasIngress bool
}

// clampInt clamps v to [min, max]. Returns fallback if v is nil.
func clampInt(v *int, min, max, fallback int) int {
	if v == nil {
		return fallback
	}
	val := *v
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// makeMinMaxCeil rounds value up to the nearest step within [min, max].
// Matches TS makeMinMaxCeil(value, min, max, step).
func makeMinMaxCeil(value, min, max, step int) int {
	if value <= min {
		return min
	}
	if value >= max {
		return max
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
func fixupFargateMemory(cpuUnits, memoryMiB int) int {
	switch cpuUnits {
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
		return makeMinMaxCeil(memoryMiB, 512, 2048, 1024)
	}
}

// fargateResources returns valid Fargate CPU (units) and memory (MiB) as strings.
// Matches TS fixupFargateConfig: tries increasing CPU tiers until memory fits.
func fargateResources(cpus float64, memoryMiB int) (cpu string, memory string) {
	cpuUnits := fixupFargateCPU(cpus)
	mem := fixupFargateMemory(cpuUnits, memoryMiB)
	// If memory exceeds this CPU tier's max, bump to next CPU tier
	for mem < memoryMiB && cpuUnits < 16384 {
		cpuUnits *= 2
		mem = fixupFargateMemory(cpuUnits, memoryMiB)
	}
	return strconv.Itoa(cpuUnits), strconv.Itoa(mem)
}

// portProtocol normalizes the transport protocol for ECS container port mappings.
// Only "tcp" and "udp" are valid; matches TS: ep.protocol === "udp" ? "udp" : "tcp"
func portProtocol(p shared.PortConfig) string {
	proto := shared.GetPortProtocol(p)
	if proto == "udp" {
		return "udp"
	}
	return "tcp"
}

// createECSService creates an ECS Fargate service for a container service.
func createECSService(
	ctx *pulumi.Context,
	serviceName string,
	svc shared.ServiceInput,
	args *ecsServiceArgs,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*ecsServiceResult, error) {

	// Create task role
	taskRole, err := createTaskRole(ctx, serviceName, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating task role: %w", err)
	}

	// Build container definition
	cpus := svc.GetCPUs()
	memMiB := svc.GetMemoryMiB()

	// Build environment variables (matches TS getContainerDefinition)
	envVars := []map[string]interface{}{
		{"name": "DEFANG_SERVICE", "value": serviceName},
	}
	if svc.DomainName != nil {
		envVars = append(envVars, map[string]interface{}{
			"name":  "DEFANG_FQDN",
			"value": *svc.DomainName,
		})
	}
	for k, v := range svc.Environment {
		if v != nil {
			envVars = append(envVars, map[string]interface{}{
				"name":  k,
				"value": *v,
			})
		}
	}

	// Build port mappings (protocol normalized to tcp/udp, hostPort = containerPort)
	var portMappings []map[string]interface{}
	for _, p := range svc.Ports {
		pm := map[string]interface{}{
			"containerPort": p.Target,
			"hostPort":      p.Target,
			"protocol":      portProtocol(p),
		}
		portMappings = append(portMappings, pm)
	}

	// Build health check with clamped values (matches TS clamp ranges)
	var healthCheck map[string]interface{}
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		healthCheck = map[string]interface{}{
			"command":  svc.HealthCheck.Test,
			"interval": clampInt(svc.HealthCheck.IntervalSeconds, 5, 300, 30),
			"timeout":  clampInt(svc.HealthCheck.TimeoutSeconds, 2, 60, 5),
			"retries":  clampInt(svc.HealthCheck.Retries, 1, 10, 3),
		}
		startPeriod := clampInt(svc.HealthCheck.StartPeriodSeconds, 0, 300, 0)
		if startPeriod > 0 {
			healthCheck["startPeriod"] = startPeriod
		}
	}

	containerDefsJSON := pulumi.All(args.logGroup.Name, args.imageURI).ApplyT(func(vals []interface{}) (string, error) {
		logGroupName := vals[0].(string)
		imageURI := vals[1].(string)

		containerDef := map[string]interface{}{
			"name":         serviceName,
			"image":        imageURI,
			"essential":    true,
			"portMappings": portMappings,
			"environment":  envVars,
		}

		if len(svc.Command) > 0 {
			containerDef["command"] = svc.Command
		}
		if len(svc.Entrypoint) > 0 {
			containerDef["entryPoint"] = svc.Entrypoint
		}
		if healthCheck != nil {
			containerDef["healthCheck"] = healthCheck
		}

		containerDef["logConfiguration"] = map[string]interface{}{
			"logDriver": "awslogs",
			"options": map[string]interface{}{
				"awslogs-group":         logGroupName,
				"awslogs-region":        args.region,
				"awslogs-stream-prefix": serviceName,
			},
		}
		b, err := json.Marshal([]interface{}{containerDef})
		if err != nil {
			return "", err
		}
		return string(b), nil
	}).(pulumi.StringOutput)

	fargateCPU, fargateMemory := fargateResources(cpus, memMiB)

	// Create task definition
	taskDef, err := ecs.NewTaskDefinition(ctx, serviceName, &ecs.TaskDefinitionArgs{
		Family:                  pulumi.String(serviceName),
		NetworkMode:             pulumi.String("awsvpc"),
		RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
		Cpu:                     pulumi.String(fargateCPU),
		Memory:                  pulumi.String(fargateMemory),
		ExecutionRoleArn:        args.execRole.Arn,
		TaskRoleArn:             taskRole.Arn,
		ContainerDefinitions:    containerDefsJSON,
		RuntimePlatform: &ecs.TaskDefinitionRuntimePlatformArgs{
			CpuArchitecture:       pulumi.String("ARM64"),
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
	var endpointOutput pulumi.StringOutput

	if svc.HasIngressPorts() && args.listener != nil {
		firstIngress := true
		for _, port := range svc.Ports {
			if port.Mode != "ingress" {
				continue
			}

			// Only create TG/LR for http, http2, grpc (matches TS createTgLrPair)
			appProto := shared.GetAppProtocol(port)
			if appProto != "http" && appProto != "http2" && appProto != "grpc" {
				continue
			}

			tgName := targetGroupName(serviceName, port.Target, appProto)

			// Target group health check (matches TS createTargetGroup in lb.ts)
			defaultInterval := recipe.HealthCheckInterval
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
			if appProto == "grpc" {
				matcher = "0"
			}

			tgArgs := &lb.TargetGroupArgs{
				Port:                       pulumi.Int(port.Target),
				Protocol:                   pulumi.String("HTTP"),
				TargetType:                 pulumi.String("ip"),
				VpcId:                      args.vpcID,
				LoadBalancingAlgorithmType: pulumi.String("least_outstanding_requests"),
				DeregistrationDelay:        pulumi.Int(recipe.DeregistrationDelay),
				HealthCheck: &lb.TargetGroupHealthCheckArgs{
					Path:               pulumi.String("/"),
					Port:               pulumi.String("traffic-port"),
					HealthyThreshold:   pulumi.Int(recipe.HealthCheckThreshold),
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
			} else if args.alb != nil {
				// Fall back to ALB DNS name (matches TS fallback behavior)
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
						Values: pulumi.StringArray{args.alb.DnsName},
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
			if appProto == "grpc" {
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HttpHeader: &lb.ListenerRuleConditionHttpHeaderArgs{
						HttpHeaderName: pulumi.String("content-type"),
						Values:         pulumi.StringArray{pulumi.String("application/grpc*")},
					},
				})
			}

			lr, lrErr := lb.NewListenerRule(ctx, tgName+"-rule", &lb.ListenerRuleArgs{
				ListenerArn: args.listener.Arn,
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
		Cluster:        args.cluster.Arn,
		TaskDefinition: taskDef.Arn,
		DesiredCount:   pulumi.Int(replicas),
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        args.subnetIDs,
			SecurityGroups: pulumi.StringArray{args.sg.ID()},
			AssignPublicIp: pulumi.Bool(true),
		},
		LoadBalancers: loadBalancers,
		DeploymentCircuitBreaker: &ecs.ServiceDeploymentCircuitBreakerArgs{
			Enable:   pulumi.Bool(true),
			Rollback: pulumi.Bool(true),
		},
		DeploymentMinimumHealthyPercent: pulumi.Int(recipe.MinHealthyPercent),
		EnableEcsManagedTags:            pulumi.Bool(true),
		PropagateTags:                   pulumi.String("SERVICE"),
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
		},
	}

	// Use capacity provider strategy if configured, otherwise default launch type
	capacityProvider := recipe.FargateCapacityProvider
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

	ecsService, err := ecs.NewService(ctx, serviceName, ecsServiceArgs, append(opts, pulumi.DependsOn(lbDependsOn))...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service: %w", err)
	}

	hasIngress := svc.HasIngressPorts() && args.listener != nil
	if hasIngress {
		endpointOutput = pulumi.Sprintf("service %s via ALB", serviceName)
	}

	return &ecsServiceResult{
		service:    ecsService,
		endpoint:   endpointOutput,
		hasIngress: hasIngress,
	}, nil
}
