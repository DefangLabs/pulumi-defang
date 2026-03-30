package aws

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
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
	VpcID            pulumi.StringInput      `pulumi:"vpcID"`
	PublicSubnetIDs  pulumi.StringArrayInput `pulumi:"publicSubnetIDs,optional"`
	PrivateSubnetIDs pulumi.StringArrayInput `pulumi:"privateSubnetIDs,optional"`
	PrivateZoneID    pulumi.StringPtrInput   `pulumi:"privateZoneID,optional"`
	PrivateDomain    string
	ProjectDomain    string
	ZoneId           pulumi.StringPtrInput // Route53 zone ID for public DNS records (empty if no public DNS)
	// shared "private SG" — attached to all services, no ingress rules
	PrivateSgID    pulumi.IDPtrInput
	AlbSG          *ec2.SecurityGroup // nil if no ALB
	HttpListener   *lb.Listener       // nil if no ALB
	HttpsListener  *lb.Listener       // nil if no ALB
	Alb            *lb.LoadBalancer   // nil if no ALB
	Region         string
	BuildInfra     *BuildInfra       // nil if no builds needed
	PublicEcrCache *PullThroughCache // ECR public pull-through cache
	SkipNatGW      bool
	Policies
}

// ECSServiceArgs holds per-service arguments for CreateECSService.
type ECSServiceArgs struct {
	Infra              *SharedInfra
	ImageURI           pulumi.StringInput // container image URI (built or pre-built)
	Networks           compose.Networks   // from project
	WaitForSteadyState bool               // true if another service depends on this one with condition: service_healthy
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
	if p.GetProtocol() == udpProto {
		return awsecs.TransportProtocolUdp
	}
	return awsecs.TransportProtocolTcp
}

const (
	grpcProto = "grpc"
	udpProto  = "udp"
	tcpProto  = "tcp"
)

func roleArn(r *iam.Role) pulumi.StringOutput {
	if r == nil {
		return pulumi.StringOutput{}
	}
	return r.Arn
}

// createServiceSG creates a per-service security group with port-specific ingress rules.
// Matches TS createServiceSG → createInstanceSG pattern:
//   - LB-backed (ingress) ports: allow from ALB SG (or VPC CIDR if no ALB SG)
//   - Public non-LB ports: allow from 0.0.0.0/0
//   - Private non-LB ports: allow from privateSG (infra.PrivateSgID) as source SG
//   - ICMP PMTUD rule when there are endpoints
func createServiceSG(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	isPrivate bool,
	opts ...pulumi.ResourceOption,
) (*ec2.SecurityGroup, error) {
	var ingress ec2.SecurityGroupIngressArray

	for _, port := range svc.Ports {
		proto := tcpProto
		if port.GetProtocol() == udpProto {
			proto = udpProto
		}

		rule := &ec2.SecurityGroupIngressArgs{
			Description: pulumi.String(fmt.Sprintf("port %d/%s", port.Target, proto)),
			FromPort:    pulumi.Int(port.Target),
			ToPort:      pulumi.Int(port.Target),
			Protocol:    pulumi.String(proto),
		}

		switch {
		case port.IsIngress() && infra.AlbSG != nil:
			// LB-backed port: allow from ALB SG (matches TS: VPC CIDR for ingress ports)
			rule.SecurityGroups = pulumi.StringArray{infra.AlbSG.ID()}
		case isPrivate && infra.PrivateSgID != nil:
			// Private port: allow from privateSG as source SG
			rule.SecurityGroups = pulumi.StringArray{infra.PrivateSgID.ToIDPtrOutput().Elem()}
		default:
			// Public port: allow from anywhere
			rule.CidrBlocks = pulumi.StringArray{pulumi.String("0.0.0.0/0")}
		}

		ingress = append(ingress, rule)
	}

	// ICMP Path MTU Discovery (RFC 1191) — matches TS createInstanceSG
	if len(svc.Ports) > 0 {
		icmpRule := &ec2.SecurityGroupIngressArgs{
			Description: pulumi.String("Allow ICMP Path MTU Discovery"),
			FromPort:    pulumi.Int(-1),
			ToPort:      pulumi.Int(-1),
			Protocol:    pulumi.String("icmp"),
		}
		if isPrivate && infra.PrivateSgID != nil {
			icmpRule.SecurityGroups = pulumi.StringArray{infra.PrivateSgID.ToIDPtrOutput().Elem()}
		} else {
			icmpRule.CidrBlocks = pulumi.StringArray{pulumi.String("0.0.0.0/0")}
		}
		ingress = append(ingress, icmpRule)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName+"-sg", &ec2.SecurityGroupArgs{
		VpcId:       infra.VpcID,
		Description: pulumi.String("Security group for " + serviceName),
		Ingress:     ingress,
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Description: pulumi.String("Allow all outbound traffic"),
				Protocol:    pulumi.String("-1"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
			"defang:scope":   pulumi.String(map[bool]string{true: "priv", false: "pub"}[isPrivate]),
		},
	}, common.MergeOptions(opts,
		pulumi.Timeouts(&pulumi.CustomTimeouts{Delete: "2m"}),
	)...)
	if err != nil {
		return nil, fmt.Errorf("creating service security group: %w", err)
	}

	return sg, nil
}

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
	opt pulumi.ResourceOrInvokeOption,
) (*EcsServiceResult, error) {
	infra := args.Infra
	if infra == nil {
		infra = &SharedInfra{} // allow standalone usage without shared infra
	}

	// Create task role
	taskRole, err := createTaskRole(ctx, serviceName, opt)
	if err != nil {
		return nil, fmt.Errorf("creating task role: %w", err)
	}

	// Attach route53 sidecar policy if service has host ports (private DNS)
	var lbDependsOn []pulumi.Resource
	if svc.HasHostPorts() && infra.route53SidecarPolicy != nil {
		dep, err := iam.NewRolePolicyAttachment(ctx, serviceName+"-AllowRoute53Sidecar", &iam.RolePolicyAttachmentArgs{
			Role:      taskRole.Name,
			PolicyArn: infra.route53SidecarPolicy.Arn,
		}, opt)
		if err != nil {
			return nil, fmt.Errorf("attaching route53 sidecar policy: %w", err)
		}
		lbDependsOn = append(lbDependsOn, dep)
	}

	// Attach bedrock policy if service uses LLM (x-defang-llm)
	if svc.LLM != nil && infra.bedrockPolicy != nil {
		dep, err := iam.NewRolePolicyAttachment(ctx, serviceName+"-BedrockPolicy", &iam.RolePolicyAttachmentArgs{
			Role:      taskRole.Name,
			PolicyArn: infra.bedrockPolicy.Arn,
		}, opt)
		if err != nil {
			return nil, fmt.Errorf("attaching bedrock policy: %w", err)
		}
		lbDependsOn = append(lbDependsOn, dep)
	}

	// Build container definition
	cpus := svc.GetCPUs()
	memMiB := svc.GetMemoryMiB()

	// Build port mappings (protocol normalized to tcp/udp, hostPort = containerPort)
	portMappings := make([]PortMapping, 0, len(svc.Ports))
	for _, p := range svc.Ports {
		portMappings = append(portMappings, PortMapping{
			ContainerPort: ptr.Int32(p.Target),
			HostPort:      ptr.Int32(p.Target), // awsvpc mode: AWS normalizes hostPort = containerPort
			Protocol:      portProtocol(p),
		})
	}

	// Build health check with clamped values (matches TS clamp ranges)
	var healthCheck *HealthCheck
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		healthCheck = &HealthCheck{
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

	// Build environment variables (plain strings, resolved before JSON marshaling)
	envVars := []KeyValuePair{
		{Name: ptr.String("DEFANG_SERVICE"), Value: ptr.String(serviceName)},
	}
	if svc.DomainName != "" {
		envVars = append(envVars, KeyValuePair{Name: ptr.String("DEFANG_FQDN"), Value: ptr.String(svc.DomainName)})
	}
	for k, v := range svc.Environment {
		envVars = append(envVars, KeyValuePair{Name: ptr.String(k), Value: ptr.String(v)})
	}
	// Sort environment variables for deterministic JSON output (map iteration order is random)
	slices.SortFunc(envVars, func(a, b KeyValuePair) int {
		return cmp.Compare(*a.Name, *b.Name)
	})

	// Resolve outputs (image URI, log group name) before building the container
	// definitions JSON. The ECS ContainerDefinitions field is a plain JSON string,
	// so all values must be concrete before marshaling.
	containerName := serviceName

	allInputs := []interface{}{args.ImageURI}
	hasLogGroup := infra != nil && infra.LogGroup != nil
	if hasLogGroup {
		allInputs = append(allInputs, infra.LogGroup.Name)
	}
	hasPrivateZone := infra != nil && infra.PrivateZoneID != nil
	if hasPrivateZone {
		allInputs = append(allInputs, infra.PrivateZoneID)
	}

	containerDefsJSON := pulumi.All(allInputs...).ApplyT(func(all []any) (string, error) {
		imageUri := all[0].(string)

		var logConfiguration *LogConfiguration
		if hasLogGroup {
			logGroupName := all[1].(string)
			logConfiguration = &LogConfiguration{
				LogDriver: awsecs.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         logGroupName,
					"awslogs-region":        infra.Region,
					"awslogs-stream-prefix": containerName,
				},
			}
		}

		// NOTE: dnsSearchDomains is NOT supported on Fargate with awsvpc network mode.
		// Instead, we rewrite environment variables to use FQDNs at the provider level.

		containerDefs := []ContainerDefinition{{
			Name:             &containerName,
			Essential:        ptr.Bool(true),
			PortMappings:     portMappings,
			Environment:      envVars,
			Command:          svc.Command,
			EntryPoint:       svc.Entrypoint,
			HealthCheck:      healthCheck,
			Image:            &imageUri,
			LogConfiguration: logConfiguration,
			// AWS normalizes these to [] on read; use empty slices to avoid null vs [] diffs
			MountPoints:    []MountPoint{},
			SystemControls: []SystemControl{},
			VolumesFrom:    []VolumeFrom{},
		}}

		if svc.HasHostPorts() && hasPrivateZone {
			privateZoneID := all[2].(string)                                         // FIXME: this is [1] if there's no loggroup
			privateFqdn := common.SafeLabel(serviceName) + "." + infra.PrivateDomain // route53 sidecar needs FQDN
			sidecarDef := ContainerDefinition{
				Name:      ptr.String("route53-sidecar"),
				Image:     ptr.String("public.ecr.aws/defang-io/route53-sidecar:65e431c"),
				Essential: ptr.Bool(false),
				Environment: []KeyValuePair{
					{Name: ptr.String("HOSTEDZONE"), Value: ptr.String(privateZoneID)},
					{Name: ptr.String("DNS"), Value: ptr.String(privateFqdn)},
					{Name: ptr.String("IPADDRESS"), Value: ptr.String("ecs")},
					// not (always?) set by the ECS agent; https://github.com/aws/containers-roadmap/issues/1611
					{Name: ptr.String("AWS_REGION"), Value: ptr.String(infra.Region)},
				},
				DependsOn: []ContainerDependency{{
					ContainerName: &containerName,
					Condition:     awsecs.ContainerConditionStart,
				}},
			}
			if Route53SidecarLogs.Get(ctx) && logConfiguration != nil {
				sidecarDef.LogConfiguration = &LogConfiguration{
					LogDriver: awsecs.LogDriverAwslogs,
					Options: map[string]string{
						"awslogs-group":         logConfiguration.Options["awslogs-group"],
						"awslogs-region":        infra.Region,
						"awslogs-stream-prefix": "route53-sidecar",
					},
				}
			}
			containerDefs = append(containerDefs, sidecarDef)
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
		ExecutionRoleArn:        roleArn(infra.ExecRole),
		TaskRoleArn:             taskRole.Arn,
		ContainerDefinitions:    containerDefsJSON,
		RuntimePlatform: &ecs.TaskDefinitionRuntimePlatformArgs{
			CpuArchitecture:       pulumi.String(string(cpuArch)),
			OperatingSystemFamily: pulumi.String("LINUX"),
		},
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
		},
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating task definition: %w", err)
	}

	replicas := svc.GetReplicas()

	// Create target group and listener rule if service has ingress ports
	var loadBalancers ecs.ServiceLoadBalancerArray
	var endpointOutput pulumix.Output[string]

	listener := infra.HttpListener
	if infra.HttpsListener != nil {
		// If HTTPS listener exists, use it instead of HTTP (matches TS createTgLrPair)
		listener = infra.HttpsListener
	}

	if svc.HasIngressPorts() && listener != nil {
		serviceLabel := common.SafeLabel(serviceName)
		publicFqdn := fmt.Sprintf("%s.%s", serviceLabel, infra.ProjectDomain)

		firstIngress := true
		for _, port := range svc.Ports {
			endpoint := fmt.Sprintf("%s--%d.%s", serviceLabel, port.Target, infra.ProjectDomain)

			endpoints := []string{endpoint}
			if port.IsIngress() && firstIngress {
				if svc.DomainName != "" {
					endpoints = append(endpoints, svc.DomainName)
				} else {
					// FIXME: which service should listen on the project domain?
					endpoints = append(endpoints, publicFqdn, infra.ProjectDomain)
				}
				endpoints = append(endpoints, svc.Networks[compose.DefaultNetwork].Aliases...)
				firstIngress = false
			}

			tg, lr, err := createTgLrPair(ctx, serviceName, infra.VpcID, listener, port, svc.HealthCheck, endpoints, opt)
			if err != nil {
				return nil, fmt.Errorf("creating TG/LR pair for port %d: %w", port.Target, err)
			}
			if tg == nil || lr == nil {
				continue
			}

			loadBalancers = append(loadBalancers, &ecs.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String(containerName),
				ContainerPort:  pulumi.Int(port.Target),
				TargetGroupArn: tg.Arn,
			})
			lbDependsOn = append(lbDependsOn, lr)
		}
	}

	isPrivate := !common.AcceptPublicTraffic(args.Networks, svc)
	assignPublicIp :=
		(infra.SkipNatGW && common.AllowEgress(args.Networks, svc)) || !isPrivate

	subnetIds := infra.PrivateSubnetIDs
	if assignPublicIp || !isPrivate {
		subnetIds = infra.PublicSubnetIDs
	}

	// Create per-service SG with port-specific ingress (matches TS createServiceSg)
	serviceSG, err := createServiceSG(ctx, serviceName, svc, infra, isPrivate, opt)
	if err != nil {
		return nil, err
	}

	// Create ECS service with circuit breaker and managed tags (matches TS createEcsService)
	var clusterArn pulumi.StringInput
	if infra.Cluster != nil {
		clusterArn = infra.Cluster.Arn
	}
	// Attach both per-service SG and shared privateSG (matches TS: [privateSg])
	securityGroups := pulumi.StringArray{serviceSG.ID()}
	if infra.PrivateSgID != nil {
		securityGroups = append(securityGroups, infra.PrivateSgID.ToIDPtrOutput().Elem())
	}
	ecsServiceArgs := &ecs.ServiceArgs{
		Cluster:        clusterArn,
		TaskDefinition: taskDef.Arn,
		DesiredCount:   pulumi.Int(replicas),
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        subnetIds,
			SecurityGroups: securityGroups,
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

	if args.WaitForSteadyState {
		ecsServiceArgs.WaitForSteadyState = pulumi.Bool(true)
	}

	ecsService, err := ecs.NewService(
		ctx,
		serviceName,
		ecsServiceArgs,
		opt,
		pulumi.DependsOn(lbDependsOn),
		pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("creating ECS service: %w", err)
	}

	hasIngress := svc.HasIngressPorts() && infra.HttpListener != nil
	serviceLabel := common.SafeLabel(serviceName)
	switch {
	case hasIngress && infra.ProjectDomain != "":
		endpointOutput = pulumix.Val(fmt.Sprintf("%s.%s", serviceLabel, infra.ProjectDomain))
	case svc.HasHostPorts() && infra.PrivateDomain != "":
		endpointOutput = pulumix.Val(fmt.Sprintf("%s.%s", serviceName, infra.PrivateDomain))
	default:
		endpointOutput = pulumix.Val(serviceName)
	}

	err = createCertsAndRoute53Dns(ctx, serviceName, svc, args.Infra, opt)
	if err != nil {
		return nil, fmt.Errorf("creating certs and Route53 DNS: %w", err)
	}

	return &EcsServiceResult{
		Service:    ecsService,
		Endpoint:   endpointOutput,
		HasIngress: hasIngress,
	}, nil
}
