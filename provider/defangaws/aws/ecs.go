package aws

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type ECSServiceArgs struct {
	Cluster         *ecs.Cluster
	ExecRole        *iam.Role
	LogGroup        *cloudwatch.LogGroup
	VpcID           pulumi.StringInput
	PublicSubnetIDs pulumi.StringArrayInput
	Sg              *ec2.SecurityGroup
	Listener        *lb.Listener
	Alb             *lb.LoadBalancer
	Region          string
	ImageURI        pulumi.StringInput // container image URI (built or pre-built)
}

type EcsServiceResult struct {
	Service    *ecs.Service
	Endpoint   pulumix.Output[string]
	HasIngress bool
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
func portProtocol(p shared.ServicePortConfig) string {
	proto := shared.GetPortProtocol(p)
	if proto == "udp" {
		return "udp"
	}
	return "tcp"
}

// CreateECSService creates an ECS Fargate service for a container service.
func CreateECSService(
	ctx *pulumi.Context,
	configProvider shared.ConfigProvider,
	serviceName string,
	svc shared.ServiceInput,
	args *ECSServiceArgs,
	recipe Recipe,
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

	// Build environment variables (matches TS getContainerDefinition)
	// Static env vars are known at plan time; dynamic ones come from interpolation as outputs.
	staticEnvVars := []ContainerEnvironment{
		{Name: "DEFANG_SERVICE", Value: serviceName},
	}
	if svc.DomainName != nil {
		staticEnvVars = append(staticEnvVars, ContainerEnvironment{
			Name:  "DEFANG_FQDN",
			Value: *svc.DomainName,
		})
	}
	type dynamicEnvVar struct {
		name   string
		output pulumi.StringOutput
	}
	var dynamicEnvVars []dynamicEnvVar
	for k, v := range svc.Environment {
		if v != nil {
			resolved := shared.InterpolateEnvironmentVariable(ctx, configProvider, *v) // resolve value from config or env
			dynamicEnvVars = append(dynamicEnvVars, dynamicEnvVar{name: k, output: resolved})
		}
	}

	// Build port mappings (protocol normalized to tcp/udp, hostPort = containerPort)
	var portMappings []ContainerPortMapping
	for _, p := range svc.Ports {
		portMappings = append(portMappings, ContainerPortMapping{
			ContainerPort: p.Target,
			HostPort:      p.Target,
			Protocol:      portProtocol(p),
		})
	}

	// Build health check with clamped values (matches TS clamp ranges)
	var healthCheck *ContainerHealthCheck
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		healthCheck = &ContainerHealthCheck{
			Command:  svc.HealthCheck.Test,
			Interval: clampInt(svc.HealthCheck.IntervalSeconds, 5, 300, 30),
			Timeout:  clampInt(svc.HealthCheck.TimeoutSeconds, 2, 60, 5),
			Retries:  clampInt(svc.HealthCheck.Retries, 1, 10, 3),
		}
		startPeriod := clampInt(svc.HealthCheck.StartPeriodSeconds, 0, 300, 0)
		if startPeriod > 0 {
			healthCheck.StartPeriod = startPeriod
		}
	}

	// Collect all outputs to resolve: logGroupName, imageURI, and dynamic env var values
	allOutputs := []interface{}{args.LogGroup.Name, args.ImageURI}
	for _, d := range dynamicEnvVars {
		allOutputs = append(allOutputs, d.output)
	}

	containerDefsJSON := pulumi.All(allOutputs...).ApplyT(func(resolved []interface{}) (string, error) {
		logGroupName := resolved[0].(string)
		imageURI := resolved[1].(string)

		// Build full env var list: static + resolved dynamic
		envVars := make([]ContainerEnvironment, len(staticEnvVars), len(staticEnvVars)+len(dynamicEnvVars))
		copy(envVars, staticEnvVars)
		for i, d := range dynamicEnvVars {
			envVars = append(envVars, ContainerEnvironment{
				Name:  d.name,
				Value: resolved[2+i].(string),
			})
		}

		essential := true
		containerDef := ContainerDefinition{
			Name:         serviceName,
			Image:        imageURI,
			Essential:    &essential,
			PortMappings: portMappings,
			Environment:  envVars,
			Command:      svc.Command,
			EntryPoint:   svc.Entrypoint,
			HealthCheck:  healthCheck,
			LogConfiguration: &ContainerLogConfiguration{
				LogDriver: "awslogs",
				Options: map[string]string{
					"awslogs-group":         logGroupName,
					"awslogs-region":        args.Region,
					"awslogs-stream-prefix": serviceName,
				},
			},
		}

		b, err := json.Marshal([]ContainerDefinition{containerDef})
		if err != nil {
			return "", fmt.Errorf("marshaling container definitions: %w", err)
		}
		return string(b), nil
	}).(pulumi.StringOutput)

	fargateCPU, fargateMemory := fargateResources(cpus, memMiB)

	cpuArch := "X86_64"
	if platformToArch(svc.GetPlatform()) == "arm64" {
		cpuArch = "ARM64"
	}

	// Create task definition
	taskDef, err := ecs.NewTaskDefinition(ctx, serviceName, &ecs.TaskDefinitionArgs{
		Family:                  pulumi.String(serviceName),
		NetworkMode:             pulumi.String("awsvpc"),
		RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
		Cpu:                     pulumi.String(fargateCPU),
		Memory:                  pulumi.String(fargateMemory),
		ExecutionRoleArn:        args.ExecRole.Arn,
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

	if svc.HasIngressPorts() && args.Listener != nil {
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
				VpcId:                      pulumi.StringInput(args.VpcID),
				LoadBalancingAlgorithmType: pulumi.String("least_outstanding_requests"),
				DeregistrationDelay:        pulumi.Int(recipe.DeregistrationDelay),
				HealthCheck: &lb.TargetGroupHealthCheckArgs{
					// Port:               pulumi.String("traffic-port"),
					Path:               pulumi.String("/"),
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
			} else if args.Alb != nil {
				// Fall back to ALB DNS name (matches TS fallback behavior)
				conditions = append(conditions, &lb.ListenerRuleConditionArgs{
					HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
						Values: pulumi.StringArray{args.Alb.DnsName},
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
				ListenerArn: args.Listener.Arn,
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
		Cluster:        args.Cluster.Arn,
		TaskDefinition: taskDef.Arn,
		DesiredCount:   pulumi.Int(replicas),
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        pulumi.StringArrayInput(args.PublicSubnetIDs),
			SecurityGroups: pulumi.StringArray{args.Sg.ID()},
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

	ecsServiceOpts := append(opts, pulumi.DependsOn(lbDependsOn))
	if len(deps) > 0 {
		ecsServiceOpts = append(ecsServiceOpts, pulumi.DependsOn(deps))
	}
	ecsService, err := ecs.NewService(ctx, serviceName, ecsServiceArgs, ecsServiceOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS service: %w", err)
	}

	hasIngress := svc.HasIngressPorts() && args.Listener != nil
	if hasIngress {
		endpointOutput = pulumix.Val(fmt.Sprintf("service %s via ALB", serviceName))
	}

	return &EcsServiceResult{
		Service:    ecsService,
		Endpoint:   endpointOutput,
		HasIngress: hasIngress,
	}, nil
}

// BuildECSArgs creates all shared AWS infrastructure for a standalone ECS service and
// returns the args to pass to NewECSServiceComponent.
// The AWS provider must be passed via opts (pulumi.Providers on the parent component).
func BuildECSArgs(ctx *pulumi.Context, serviceName string, svc shared.ServiceInput, awsCfg *common.AWSConfig, opts ...pulumi.ResourceOption) (*ECSServiceArgs, error) {
	recipe := LoadRecipe(ctx)

	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := ResolveNetworking(ctx, awsCfg, recipe, opts...)
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
		RetentionInDays: pulumi.Int(recipe.LogRetentionDays),
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
		albRes, err := CreateALB(ctx, net.VpcID, net.PublicSubnetIDs, sg, recipe, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albRes.HttpListener
		svcALB = albRes.Alb
	}

	imageURI, err := GetServiceImage(ctx, serviceName, svc, imgInfra, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving image for %s: %w", serviceName, err)
	}

	return &ECSServiceArgs{
		Cluster:         cluster,
		ExecRole:        execRole,
		LogGroup:        logGroup,
		VpcID:           net.VpcID,
		PublicSubnetIDs: net.PublicSubnetIDs,
		Sg:              sg,
		Listener:        httpListener,
		Alb:             svcALB,
		Region:          region.Name,
		ImageURI:        imageURI,
	}, nil
}
