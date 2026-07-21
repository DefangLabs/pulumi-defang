package aws

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/appautoscaling"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

var (
	errSidecarImageRequired = errors.New("image is required")
	errPoliciesWithTaskRole = errors.New("policies cannot be combined with taskRoleArn")
)

type Policies struct {
	bedrockPolicy *iam.Policy
	// codeBuildPolicy      *iam.RolePoliciesExclusive
	route53SidecarPolicy *iam.Policy
}

// SharedInfra holds project-level AWS resources shared across all services.
//
// Fields come in two flavors:
//   - internal typed resource pointers (Cluster, ExecRole, LogGroup, …) set by
//     the project-scope orchestrator; untagged, so not part of the SDK schema.
//   - schema-tagged string inputs (ClusterArn, ExecutionRoleArn, …) so
//     standalone SDK users can reuse hand-provisioned infrastructure.
//
// The accessor methods below prefer the typed resource when set and fall back
// to the schema input.
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
	PrivateSgID    pulumi.StringPtrInput `pulumi:"privateSgID,optional"`
	AlbSG          *ec2.SecurityGroup    // nil if no ALB
	HttpListener   *lb.Listener          // nil if no ALB
	HttpsListener  *lb.Listener          // nil if no ALB
	Alb            *lb.LoadBalancer      // nil if no ALB
	Region         string
	BuildInfra     *BuildInfra       // nil if no builds needed
	PublicEcrCache *PullThroughCache // ECR public pull-through cache
	SkipNatGW      bool
	// DnsProvider is a role-assuming AWS provider for public Route53
	// operations when the DNS zone lives in another account; nil otherwise.
	DnsProvider pulumi.ProviderResource
	// Etag is the deployment ID supplied by the CD program; empty for
	// standalone Service callers.
	Etag string
	Policies

	// Schema-exposed handles to pre-existing infrastructure, for standalone
	// SDK callers that manage their own cluster/exec role/log group. ALB
	// attachment is deliberately not exposed: ingress-bearing services
	// belong in a Project, which owns its own ALB.
	ClusterArn       pulumi.StringInput `pulumi:"clusterArn,optional"`
	ExecutionRoleArn pulumi.StringInput `pulumi:"executionRoleArn,optional"`
	LogGroupName     pulumi.StringInput `pulumi:"logGroupName,optional"`
}

func (i *SharedInfra) clusterArn() pulumi.StringInput {
	if i.Cluster != nil {
		return i.Cluster.Arn
	}
	return i.ClusterArn
}

func (i *SharedInfra) executionRoleArn() pulumi.StringPtrInput {
	if i.ExecRole != nil {
		return i.ExecRole.Arn
	}
	if i.ExecutionRoleArn != nil {
		return i.ExecutionRoleArn.ToStringOutput()
	}
	return nil
}

func (i *SharedInfra) logGroupName() pulumi.StringInput {
	if i.LogGroup != nil {
		return i.LogGroup.Name
	}
	return i.LogGroupName
}

func (i *SharedInfra) httpListenerArn() pulumi.StringInput {
	if i.HttpListener != nil {
		return i.HttpListener.Arn
	}
	return nil
}

func (i *SharedInfra) httpsListenerArn() pulumi.StringInput {
	if i.HttpsListener != nil {
		return i.HttpsListener.Arn
	}
	return nil
}

func (i *SharedInfra) albDnsName() pulumi.StringInput {
	if i.Alb != nil {
		return i.Alb.DnsName
	}
	return nil
}

func (i *SharedInfra) albSecurityGroupId() pulumi.StringInput {
	if i.AlbSG != nil {
		return i.AlbSG.ID().ToStringOutput()
	}
	return nil
}

// ECSServiceArgs holds per-service arguments for CreateECSService.
type ECSServiceArgs struct {
	Infra              *SharedInfra
	ImageURI           pulumi.StringInput // container image URI (built or pre-built)
	Networks           compose.Networks   // from project
	WaitForSteadyState bool               // true if another service depends on this one with condition: service_healthy
	// TaskRoleArn, when set, is used as the task role instead of creating a
	// fresh per-service role. Provider-managed policy attachments (route53
	// sidecar, bedrock) are skipped — the caller owns the role's policies.
	TaskRoleArn pulumi.StringInput
	// SecurityGroupIds are extra security groups attached to the service's ENI
	// in addition to the per-service SG and the shared private SG.
	SecurityGroupIds pulumi.StringArrayInput
	// Secrets maps container environment variable names to SSM parameter or
	// Secrets Manager ARNs, injected via the ECS-native secrets mechanism.
	Secrets pulumi.StringMapInput
	// Environment holds extra env vars for the main container; values may be
	// Outputs. Bare ${VAR} values become ECS secret refs (like the
	// compose-shaped ServiceConfig.Environment); other values stay plaintext.
	Environment pulumi.StringMapInput
	// Sidecars are additional containers deployed in the same task definition.
	// Keyed by service name; volumesFrom/dependsOn on the main service may
	// reference these names.
	Sidecars map[string]compose.ServiceConfig
	// Triggers force a service redeployment when any value changes.
	Triggers pulumi.StringMapInput
}

type EcsServiceResult struct {
	Service     *ecs.Service
	Endpoint    pulumix.Output[string]
	HasIngress  bool
	TaskRoleArn pulumi.StringInput // the created or caller-supplied task role
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
	if p.GetProtocol() == compose.PortProtocolUDP {
		return awsecs.TransportProtocolUdp
	}
	return awsecs.TransportProtocolTcp
}

// parseVolumesFrom parses a compose volumes_from entry ("container[:ro|rw]" or
// "container:service:ro") into the source container name and read-only flag.
// Matches the TS parseVolumesFrom helper.
func parseVolumesFrom(ref string) (string, bool) {
	parts := strings.Split(ref, ":")
	switch len(parts) {
	case 1:
		return parts[0], false
	case 2:
		return parts[0], parts[1] == "ro"
	case 3:
		return parts[1], parts[2] == "ro"
	default:
		return ref, false
	}
}

// buildMountPoints converts compose volume entries to ECS mount points.
func buildMountPoints(volumes []compose.ServiceVolumeConfig) []MountPoint {
	mountPoints := make([]MountPoint, 0, len(volumes))
	for _, v := range volumes {
		mountPoints = append(mountPoints, MountPoint{
			ContainerPath: ptr.String(v.Target),
			SourceVolume:  ptr.String(v.Source),
			ReadOnly:      ptr.Bool(v.ReadOnly),
		})
	}
	return mountPoints
}

// buildVolumesFrom converts compose volumes_from entries to ECS volumesFrom,
// resolving service names to their container names via the sidecar map.
func buildVolumesFrom(refs []string, sidecars map[string]compose.ServiceConfig) []VolumeFrom {
	volumesFrom := make([]VolumeFrom, 0, len(refs))
	for _, ref := range refs {
		source, readOnly := parseVolumesFrom(ref)
		if sc, ok := sidecars[source]; ok {
			source = sc.GetContainerName(source)
		}
		volumesFrom = append(volumesFrom, VolumeFrom{
			SourceContainer: ptr.String(source),
			ReadOnly:        ptr.Bool(readOnly),
		})
	}
	return volumesFrom
}

// buildHealthCheck converts a compose health check to an ECS container health
// check with clamped values (matches TS clamp ranges).
func buildHealthCheck(hc *compose.HealthCheckConfig) *HealthCheck {
	if hc == nil || len(hc.Test) == 0 {
		return nil
	}
	healthCheck := &HealthCheck{
		Command:  hc.Test,
		Interval: ptr.Int32(clampInt(hc.IntervalSeconds, 5, 300, 30)),
		Timeout:  ptr.Int32(clampInt(hc.TimeoutSeconds, 2, 60, 5)),
		Retries:  ptr.Int32(clampInt(hc.Retries, 1, 10, 3)),
	}
	if startPeriod := clampInt(hc.StartPeriodSeconds, 0, 300, 0); startPeriod > 0 {
		healthCheck.StartPeriod = ptr.Int32(startPeriod)
	}
	return healthCheck
}

// buildStopTimeout converts a compose stop_grace_period to an ECS stopTimeout
// (nil if unset; ECS caps Fargate stopTimeout at 120s).
func buildStopTimeout(svc compose.ServiceConfig) *int32 {
	secs := svc.GetStopGracePeriodSeconds()
	if secs <= 0 {
		return nil
	}
	if secs > 120 {
		secs = 120
	}
	return ptr.Int32(int32(secs))
}

// composeToEcsConditions maps compose depends_on conditions to ECS container
// dependency conditions.
var composeToEcsConditions = map[string]awsecs.ContainerCondition{
	"service_started":                awsecs.ContainerConditionStart,
	"service_healthy":                awsecs.ContainerConditionHealthy,
	"service_completed_successfully": awsecs.ContainerConditionSuccess,
}

// buildDependsOn converts compose depends_on entries to ECS container
// dependencies. Only dependencies on sidecar containers in the same task are
// supported; others are dropped (matches the TS behaviour).
func buildDependsOn(
	dependsOn compose.DependsOnConfig, sidecars map[string]compose.ServiceConfig,
) []ContainerDependency {
	deps := make([]ContainerDependency, 0, len(dependsOn))
	for depName, dep := range common.Sorted(dependsOn) {
		sc, ok := sidecars[depName]
		if !ok {
			continue // ECS only supports dependsOn between containers in the same task
		}
		condition := composeToEcsConditions[dep.Condition]
		if condition == "" {
			condition = awsecs.ContainerConditionStart
		}
		deps = append(deps, ContainerDependency{
			ContainerName: ptr.String(sc.GetContainerName(depName)),
			Condition:     condition,
		})
	}
	return deps
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
		proto := ec2.ProtocolTypeTCP
		if port.GetProtocol() == compose.PortProtocolUDP {
			proto = ec2.ProtocolTypeUDP
		}

		rule := &ec2.SecurityGroupIngressArgs{
			Description: pulumi.String(fmt.Sprintf("port %d/%s", port.Target, proto)),
			FromPort:    pulumi.Int(port.Target),
			ToPort:      pulumi.Int(port.Target),
			Protocol:    pulumi.String(proto),
		}

		switch {
		case port.IsIngress() && infra.albSecurityGroupId() != nil:
			// LB-backed port: allow from ALB SG (matches TS: VPC CIDR for ingress ports)
			rule.SecurityGroups = pulumi.StringArray{infra.albSecurityGroupId()}
		case isPrivate && infra.PrivateSgID != nil:
			// Private port: allow from privateSG as source SG
			rule.SecurityGroups = pulumi.StringArray{infra.PrivateSgID.ToStringPtrOutput().Elem()}
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
			icmpRule.SecurityGroups = pulumi.StringArray{infra.PrivateSgID.ToStringPtrOutput().Elem()}
		} else {
			icmpRule.CidrBlocks = pulumi.StringArray{pulumi.String("0.0.0.0/0")}
		}
		ingress = append(ingress, icmpRule)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
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
	parentOpt pulumi.ResourceOrInvokeOption,
) (*EcsServiceResult, error) {
	infra := args.Infra
	if infra == nil {
		infra = &SharedInfra{} // allow standalone usage without shared infra
	}

	// Use the caller-supplied task role, or create one per service. Provider-
	// managed policy attachments only apply to roles we create — an external
	// role's policies are owned by the caller.
	var lbDependsOn []pulumi.Resource
	taskRoleArn := args.TaskRoleArn
	if taskRoleArn == nil {
		taskRole, err := createTaskRole(ctx, serviceName, parentOpt)
		if err != nil {
			return nil, fmt.Errorf("creating task role: %w", err)
		}
		taskRoleArn = taskRole.Arn

		// Attach route53 sidecar policy if service has host ports (private DNS)
		if svc.HasHostPorts() && infra.route53SidecarPolicy != nil {
			dep, err := iam.NewRolePolicyAttachment(ctx, serviceName+"-AllowRoute53Sidecar", &iam.RolePolicyAttachmentArgs{
				Role:      taskRole.Name,
				PolicyArn: infra.route53SidecarPolicy.Arn,
			}, parentOpt)
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
			}, parentOpt)
			if err != nil {
				return nil, fmt.Errorf("attaching bedrock policy: %w", err)
			}
			lbDependsOn = append(lbDependsOn, dep)
		}

		// Attach caller-specified policies (x-defang-policies). Entries are
		// literals by now — compose variables were interpolated before the
		// project reached the provider — so anything unresolved or qualified
		// for a different cloud is a hard error.
		policies := compose.NormalizePolicies(svc.Policies)
		if err := compose.ValidatePolicies(compose.PolicyCloudAWS, policies); err != nil {
			return nil, fmt.Errorf("service %s: %w", serviceName, err)
		}
		// Dedup repeated entries: the attachment URN embeds policyName(policy),
		// so a duplicate entry would collide on it.
		seenPolicies := make(map[string]struct{}, len(policies))
		for _, policy := range policies {
			if _, dup := seenPolicies[policy]; dup {
				continue
			}
			seenPolicies[policy] = struct{}{}
			policyArn, err := resolvePolicyArn(ctx, policy, parentOpt)
			if err != nil {
				return nil, fmt.Errorf("resolving policy %q: %w", policy, err)
			}
			attachmentName := serviceName + "-task-role-" + policyName(policy)
			dep, err := iam.NewRolePolicyAttachment(ctx, attachmentName, &iam.RolePolicyAttachmentArgs{
				Role:      taskRole.Name,
				PolicyArn: policyArn,
			}, parentOpt)
			if err != nil {
				return nil, fmt.Errorf("attaching policy %q: %w", policy, err)
			}
			lbDependsOn = append(lbDependsOn, dep)
		}
	} else if len(compose.NormalizePolicies(svc.Policies)) > 0 {
		// A caller-supplied task role's policies are owned by the caller.
		return nil, fmt.Errorf("service %s: %w", serviceName, errPoliciesWithTaskRole)
	}

	// Build container definition
	cpus := svc.GetCPUs()
	memMiB := svc.GetMemoryMiB()

	// Build port mappings (protocol normalized to tcp/udp, hostPort = containerPort).
	// Multiple compose ports can share a target port (e.g. http + grpc listeners
	// on the same port), so dedupe (matches TS dedup by containerPort/protocol).
	portMappings := make([]PortMapping, 0, len(svc.Ports))
	seenPorts := map[string]bool{}
	for _, p := range svc.Ports {
		key := fmt.Sprintf("%d/%s", p.Target, portProtocol(p))
		if seenPorts[key] {
			continue
		}
		seenPorts[key] = true
		portMappings = append(portMappings, PortMapping{
			ContainerPort: ptr.Int32(p.Target),
			HostPort:      ptr.Int32(p.Target), // awsvpc mode: AWS normalizes hostPort = containerPort
			Protocol:      portProtocol(p),
		})
	}

	healthCheck := buildHealthCheck(svc.HealthCheck)

	// Build environment variable names and resolve values via ConfigProvider interpolation
	// (matches GCP's use of compose.GetConfigOrEnvValue)
	type envEntry struct {
		name string
		idx  int // index into allInputs where the resolved value will be
	}
	staticEnvVars := []KeyValuePair{
		{Name: "DEFANG_SERVICE", Value: serviceName},
	}
	if infra != nil && infra.Etag != "" {
		staticEnvVars = append(staticEnvVars, KeyValuePair{Name: "DEFANG_ETAG", Value: infra.Etag})
	}
	// DEFANG_FQDN: custom domain, else public FQDN (ingress), else private FQDN
	// (host-port). See common.ServiceFQDN for the source-of-truth precedence.
	var publicDomain, privateDomain string
	if infra != nil {
		publicDomain, privateDomain = infra.ProjectDomain, infra.PrivateDomain
	}
	if fqdn := common.ServiceFQDN(serviceName, svc, publicDomain, privateDomain); fqdn != "" {
		staticEnvVars = append(staticEnvVars, KeyValuePair{Name: "DEFANG_FQDN", Value: fqdn})
	}

	// Resolve outputs (image URI, log group name, env vars) before building the container
	// definitions JSON. The ECS ContainerDefinitions field is a plain JSON string,
	// so all values must be concrete before marshaling. Each optional input's
	// index into allInputs is tracked explicitly.
	containerName := svc.GetContainerName(serviceName)

	allInputs := []interface{}{args.ImageURI}
	logGroupIdx := -1
	if logGroupName := infra.logGroupName(); logGroupName != nil {
		logGroupIdx = len(allInputs)
		allInputs = append(allInputs, logGroupName)
	}
	privateZoneIdx := -1
	if infra.PrivateZoneID != nil {
		privateZoneIdx = len(allInputs)
		allInputs = append(allInputs, infra.PrivateZoneID)
	}
	secretsIdx := -1
	if args.Secrets != nil {
		secretsIdx = len(allInputs)
		allInputs = append(allInputs, args.Secrets)
	}
	envInputIdx := -1
	if args.Environment != nil {
		envInputIdx = len(allInputs)
		allInputs = append(allInputs, args.Environment)
	}

	// Split env vars: bare ${VAR} references go to ECS Secrets (SSM ARN),
	// all others go to Environment (resolved plaintext).
	resolveEnv := func(svc compose.ServiceConfig) ([]envEntry, []Secret, error) {
		var entries []envEntry
		var secrets []Secret
		for k, v := range common.Sorted(svc.Environment) {
			if secretVar := compose.GetConfigName2(k, v); secretVar != "" && configProvider != nil {
				ref, err := configProvider.GetSecretRef(ctx, secretVar, parentOpt)
				if err != nil {
					return nil, nil, fmt.Errorf("getting secret ref for %q: %w", k, err)
				}
				secrets = append(secrets, Secret{Name: k, ValueFrom: ref})
			} else {
				resolved := compose.GetConfigOrEnvValue(ctx, configProvider, svc, k, "", parentOpt)
				entries = append(entries, envEntry{name: k, idx: len(allInputs)})
				allInputs = append(allInputs, resolved)
			}
		}
		return entries, secrets, nil
	}

	envEntries, secretEntries, err := resolveEnv(svc)
	if err != nil {
		return nil, err
	}

	// Resolve sidecar container env vars the same way as the main container.
	type sidecarData struct {
		name       string
		cfg        compose.ServiceConfig
		envEntries []envEntry
		secrets    []Secret
		imageIdx   int // index into allInputs where the resolved image URI will be
	}
	sidecarDatas := make([]sidecarData, 0, len(args.Sidecars))
	for scName, sc := range common.Sorted(args.Sidecars) {
		if sc.Image == nil {
			return nil, fmt.Errorf("sidecar %q: %w", scName, errSidecarImageRequired)
		}
		if img := sc.StaticImage(); img != nil && *img == "" {
			return nil, fmt.Errorf("sidecar %q: %w", scName, errSidecarImageRequired)
		}
		scEnvEntries, scSecrets, err := resolveEnv(sc)
		if err != nil {
			return nil, fmt.Errorf("sidecar %q: %w", scName, err)
		}
		imageIdx := len(allInputs)
		allInputs = append(allInputs, sc.Image)
		sidecarDatas = append(sidecarDatas, sidecarData{
			name: scName, cfg: sc, envEntries: scEnvEntries, secrets: scSecrets, imageIdx: imageIdx,
		})
	}

	containerDefsJSON := pulumi.All(allInputs...).ApplyT(func(all []any) (string, error) {
		imageUri := all[0].(string)

		// Merge ECS-native secrets from the Secrets input with the ones derived
		// from bare ${VAR} environment references.
		mainSecrets := append([]Secret{}, secretEntries...)
		if secretsIdx >= 0 {
			for name, valueFrom := range common.Sorted(all[secretsIdx].(map[string]string)) {
				mainSecrets = append(mainSecrets, Secret{Name: name, ValueFrom: valueFrom})
			}
		}

		// Build env vars from resolved outputs
		envVars := append([]KeyValuePair{}, staticEnvVars...)
		for _, e := range envEntries {
			val := all[e.idx].(string)
			envVars = append(envVars, KeyValuePair{Name: e.name, Value: val})
		}
		if envInputIdx >= 0 {
			// Same bare-${VAR} split as resolveEnv, but on already-resolved
			// values (GetSecretRef only does invokes, safe inside ApplyT).
			for k, v := range common.Sorted(all[envInputIdx].(map[string]string)) {
				if secretVar := compose.GetConfigName2(k, pulumi.String(v)); secretVar != "" && configProvider != nil {
					ref, err := configProvider.GetSecretRef(ctx, secretVar, parentOpt)
					if err != nil {
						return "", fmt.Errorf("getting secret ref for %q: %w", k, err)
					}
					mainSecrets = append(mainSecrets, Secret{Name: k, ValueFrom: ref})
					continue
				}
				envVars = append(envVars, KeyValuePair{Name: k, Value: v})
			}
		}
		slices.SortFunc(envVars, func(a, b KeyValuePair) int {
			return cmp.Compare(a.Name, b.Name)
		})

		logConfigurationFor := func(streamPrefix string) *LogConfiguration {
			if logGroupIdx < 0 {
				return nil
			}
			return &LogConfiguration{
				LogDriver: awsecs.LogDriverAwslogs,
				Options: map[LogOption]string{
					LogOptionAwslogsGroup:        all[logGroupIdx].(string),
					LogOptionAwslogsRegion:       infra.Region,
					LogOptionAwslogsStreamPrefix: streamPrefix,
				},
			}
		}
		logConfiguration := logConfigurationFor(containerName)

		// NOTE: dnsSearchDomains is NOT supported on Fargate with awsvpc network mode.
		// Instead, we rewrite environment variables to use FQDNs at the provider level.

		containerDefs := []ContainerDefinition{{
			Name:             containerName,
			Essential:        ptr.Bool(true),
			PortMappings:     portMappings,
			Environment:      envVars,
			Secrets:          mainSecrets,
			Command:          svc.Command,
			EntryPoint:       svc.Entrypoint,
			DependsOn:        buildDependsOn(svc.DependsOn, args.Sidecars),
			HealthCheck:      healthCheck,
			Image:            imageUri,
			LogConfiguration: logConfiguration,
			StopTimeout:      buildStopTimeout(svc),
			WorkingDirectory: svc.WorkingDir,
			// AWS normalizes these to [] on read; use empty slices to avoid null vs [] diffs
			MountPoints:    buildMountPoints(svc.Volumes),
			SystemControls: []SystemControl{},
			VolumesFrom:    buildVolumesFrom(svc.VolumesFrom, args.Sidecars),
		}}

		for _, sd := range sidecarDatas {
			scEnvVars := []KeyValuePair{}
			for _, e := range sd.envEntries {
				scEnvVars = append(scEnvVars, KeyValuePair{Name: e.name, Value: all[e.idx].(string)})
			}
			slices.SortFunc(scEnvVars, func(a, b KeyValuePair) int {
				return cmp.Compare(a.Name, b.Name)
			})
			scContainerName := sd.cfg.GetContainerName(sd.name)
			containerDefs = append(containerDefs, ContainerDefinition{
				Name:             scContainerName,
				Essential:        ptr.Bool(sd.cfg.Restart != "no"),
				PortMappings:     []PortMapping{},
				Environment:      scEnvVars,
				Secrets:          sd.secrets,
				Command:          sd.cfg.Command,
				EntryPoint:       sd.cfg.Entrypoint,
				DependsOn:        buildDependsOn(sd.cfg.DependsOn, args.Sidecars),
				HealthCheck:      buildHealthCheck(sd.cfg.HealthCheck),
				Image:            all[sd.imageIdx].(string),
				LogConfiguration: logConfigurationFor(scContainerName),
				StopTimeout:      buildStopTimeout(sd.cfg),
				WorkingDirectory: sd.cfg.WorkingDir,
				MountPoints:      buildMountPoints(sd.cfg.Volumes),
				SystemControls:   []SystemControl{},
				VolumesFrom:      buildVolumesFrom(sd.cfg.VolumesFrom, args.Sidecars),
			})
		}

		if svc.HasHostPorts() && privateZoneIdx >= 0 {
			privateZoneID := all[privateZoneIdx].(string)
			privateFqdn := common.ServiceLabel(serviceName) + "." + infra.PrivateDomain // route53 sidecar needs FQDN
			sidecarDef := ContainerDefinition{
				Name:      "route53-sidecar",
				Image:     "public.ecr.aws/defang-io/route53-sidecar:65e431c",
				Essential: ptr.Bool(false),
				Environment: []KeyValuePair{
					{Name: ("HOSTEDZONE"), Value: (privateZoneID)},
					{Name: ("DNS"), Value: (privateFqdn)},
					{Name: ("IPADDRESS"), Value: ("ecs")},
					// not (always?) set by the ECS agent; https://github.com/aws/containers-roadmap/issues/1611
					{Name: ("AWS_REGION"), Value: (infra.Region)},
				},
				DependsOn: []ContainerDependency{{
					ContainerName: &containerName,
					Condition:     awsecs.ContainerConditionStart,
				}},
			}
			if Route53SidecarLogs.Get(ctx) && logConfiguration != nil {
				sidecarDef.LogConfiguration = &LogConfiguration{
					LogDriver: awsecs.LogDriverAwslogs,
					Options: map[LogOption]string{
						LogOptionAwslogsGroup:        logConfiguration.Options[LogOptionAwslogsGroup],
						LogOptionAwslogsRegion:       infra.Region,
						LogOptionAwslogsStreamPrefix: "route53-sidecar",
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
		ExecutionRoleArn:        infra.executionRoleArn(),
		TaskRoleArn:             taskRoleArn.ToStringOutput(),
		ContainerDefinitions:    containerDefsJSON,
		Volumes:                 buildTaskVolumes(svc, args.Sidecars),
		RuntimePlatform: &ecs.TaskDefinitionRuntimePlatformArgs{
			CpuArchitecture:       pulumi.String(string(cpuArch)),
			OperatingSystemFamily: pulumi.String("LINUX"),
		},
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
		},
	}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating task definition: %w", err)
	}

	replicas := svc.GetReplicas()

	// Create target group and listener rule if service has ingress ports
	var loadBalancers ecs.ServiceLoadBalancerArray
	var endpointOutput pulumix.Output[string]

	// If an HTTPS listener exists, prefer it over HTTP (matches TS createTgLrPair).
	// A port may force the plain HTTP listener via x-defang-listener: http.
	defaultListenerArn := infra.httpsListenerArn()
	if defaultListenerArn == nil {
		defaultListenerArn = infra.httpListenerArn()
	}

	if svc.HasIngressPorts() && defaultListenerArn != nil {
		serviceLabel := common.ServiceLabel(serviceName)

		firstIngress := true
		var prevRule pulumi.Resource // serializes rule creation so ALB priorities follow port order
		for _, port := range svc.Ports {
			var endpoints []string

			if infra.ProjectDomain != "" {
				endpoints = append(endpoints, fmt.Sprintf("%s--%d.%s", serviceLabel, port.Target, infra.ProjectDomain))
			}
			if port.IsIngress() {
				// The service's public FQDN matches every ingress port's rule:
				// rules on the same listener are disambiguated by protocol
				// conditions (gRPC content-type) or target a different
				// listener (x-defang-listener: http).
				switch {
				case svc.DomainName != "":
					endpoints = append(endpoints, svc.DomainName)
				case infra.ProjectDomain != "":
					endpoints = append(endpoints, fmt.Sprintf("%s.%s", serviceLabel, infra.ProjectDomain))
					if firstIngress {
						// FIXME: which service should listen on the project domain?
						endpoints = append(endpoints, infra.ProjectDomain)
					}
				}
				if firstIngress {
					endpoints = append(endpoints, svc.Networks[compose.DefaultNetwork].Aliases...)
					firstIngress = false
				}
			}

			listenerArn := defaultListenerArn
			if port.Listener == compose.PortListenerHTTP {
				if httpArn := infra.httpListenerArn(); httpArn != nil {
					listenerArn = httpArn
				}
			}

			// Rules for different ports may match the same host (e.g. the
			// service's domainname), disambiguated by protocol conditions.
			// AWS assigns rule priorities in creation order, so serialize:
			// earlier ports get higher-priority (more specific) rules.
			tgLrOpt := pulumi.ResourceOption(parentOpt)
			if prevRule != nil {
				tgLrOpt = pulumi.Composite(tgLrOpt, pulumi.DependsOn([]pulumi.Resource{prevRule}))
			}
			tg, lr, err := createTgLrPair(
				ctx,
				serviceName,
				infra.VpcID,
				listenerArn,
				port,
				svc.HealthCheck,
				endpoints,
				infra.albDnsName(),
				tgLrOpt)
			if err != nil {
				return nil, fmt.Errorf("creating TG/LR pair for port %d: %w", port.Target, err)
			}
			if tg == nil || lr == nil {
				continue
			}
			prevRule = lr

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
	serviceSG, err := createServiceSG(ctx, serviceName, svc, infra, isPrivate, parentOpt)
	if err != nil {
		return nil, err
	}

	// Attach per-service SG, shared privateSG (matches TS: [privateSg]), and any
	// caller-supplied extra security groups.
	securityGroups := pulumi.StringArray{serviceSG.ID()}
	if infra.PrivateSgID != nil {
		securityGroups = append(securityGroups, infra.PrivateSgID.ToStringPtrOutput().Elem())
	}
	var serviceSecurityGroups pulumi.StringArrayInput = securityGroups
	if args.SecurityGroupIds != nil {
		serviceSecurityGroups = pulumi.All(securityGroups, args.SecurityGroupIds).
			ApplyT(func(all []any) []string {
				//nolint:forcetypeassert // pulumi.All resolves string arrays
				return append(all[0].([]string), all[1].([]string)...)
			}).(pulumi.StringArrayOutput)
	}

	// Create ECS service with circuit breaker and managed tags (matches TS createEcsService)
	ecsServiceArgs := &ecs.ServiceArgs{
		Cluster:        infra.clusterArn(),
		TaskDefinition: taskDef.Arn,
		DesiredCount:   pulumi.Int(replicas),
		Triggers:       args.Triggers,
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        subnetIds,
			SecurityGroups: serviceSecurityGroups,
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
		parentOpt,
		pulumi.DependsOn(lbDependsOn),
		pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("creating ECS service: %w", err)
	}

	hasIngress := svc.HasIngressPorts() && defaultListenerArn != nil
	serviceLabel := common.ServiceLabel(serviceName)
	switch {
	case hasIngress && infra.ProjectDomain != "":
		// HTTP listener redirects to HTTPS when a cert is bound, so advertise https://.
		endpointOutput = pulumix.Val(fmt.Sprintf("https://%s.%s", serviceLabel, infra.ProjectDomain))
	case hasIngress && infra.albDnsName() != nil:
		// No ProjectDomain: reach the service via the raw ALB DNS. createTgLrPair
		// sets the listener-rule HostHeader to the ALB DNS name when no hostnames
		// are configured, so routing works for http://<alb-dns-name>.
		endpointOutput = pulumix.Apply(
			pulumix.Output[string](infra.albDnsName().ToStringOutput()),
			func(dns string) string { return "http://" + dns },
		)
	case svc.HasHostPorts() && infra.PrivateDomain != "":
		// serviceLabel (not raw serviceName) to match the private DNS record.
		endpointOutput = pulumix.Val(fmt.Sprintf("%s.%s", serviceLabel, infra.PrivateDomain))
	default:
		endpointOutput = pulumix.Val(serviceName)
	}

	err = createCertsAndRoute53Dns(ctx, serviceName, svc, args.Infra, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("creating certs and Route53 DNS: %w", err)
	}

	// CPU-based target-tracking autoscaling (x-defang-autoscaling), replicas..2x
	// replicas (matches TS createFargateScalingTarget/PolicyCPU).
	if svc.Autoscaling {
		if err := createCPUAutoscaling(ctx, serviceName, infra, ecsService, replicas, parentOpt); err != nil {
			return nil, err
		}
	}

	return &EcsServiceResult{
		Service:     ecsService,
		Endpoint:    endpointOutput,
		HasIngress:  hasIngress,
		TaskRoleArn: taskRoleArn,
	}, nil
}

// createCPUAutoscaling registers the ECS service as an Application Auto Scaling
// target with a CPU target-tracking policy (85% average utilization).
func createCPUAutoscaling(
	ctx *pulumi.Context,
	serviceName string,
	infra *SharedInfra,
	ecsService *ecs.Service,
	replicas int32,
	parentOpt pulumi.ResourceOrInvokeOption,
) error {
	clusterName := pulumi.String("default").ToStringOutput()
	if clusterArn := infra.clusterArn(); clusterArn != nil {
		clusterName = clusterArn.ToStringOutput().ApplyT(func(arn string) string {
			// arn:aws:ecs:region:account:cluster/NAME → NAME
			return arn[strings.LastIndexByte(arn, '/')+1:]
		}).(pulumi.StringOutput)
	}
	resourceID := pulumi.Sprintf("service/%s/%s", clusterName, ecsService.Name)

	target, err := appautoscaling.NewTarget(ctx, serviceName+"-scaling", &appautoscaling.TargetArgs{
		MinCapacity:       pulumi.Int(replicas),
		MaxCapacity:       pulumi.Int(replicas * 2),
		ResourceId:        resourceID,
		ScalableDimension: pulumi.String("ecs:service:DesiredCount"),
		ServiceNamespace:  pulumi.String("ecs"),
	}, parentOpt)
	if err != nil {
		return fmt.Errorf("creating autoscaling target: %w", err)
	}

	metricSpec := &appautoscaling.PolicyTargetTrackingScalingPolicyConfigurationPredefinedMetricSpecificationArgs{
		PredefinedMetricType: pulumi.String("ECSServiceAverageCPUUtilization"),
	}
	_, err = appautoscaling.NewPolicy(ctx, serviceName+"-scaling-cpu", &appautoscaling.PolicyArgs{
		PolicyType:        pulumi.String("TargetTrackingScaling"),
		ResourceId:        target.ResourceId,
		ScalableDimension: target.ScalableDimension,
		ServiceNamespace:  target.ServiceNamespace,
		TargetTrackingScalingPolicyConfiguration: &appautoscaling.PolicyTargetTrackingScalingPolicyConfigurationArgs{
			PredefinedMetricSpecification: metricSpec,
			TargetValue:                   pulumi.Float64(85),
		},
	}, parentOpt)
	if err != nil {
		return fmt.Errorf("creating autoscaling policy: %w", err)
	}
	return nil
}

// buildTaskVolumes collects the distinct volume names referenced by the main
// service's and sidecars' compose volumes into task-definition volumes.
func buildTaskVolumes(
	svc compose.ServiceConfig, sidecars map[string]compose.ServiceConfig,
) ecs.TaskDefinitionVolumeArray {
	var volumes ecs.TaskDefinitionVolumeArray
	seen := map[string]bool{}
	add := func(vols []compose.ServiceVolumeConfig) {
		for _, v := range vols {
			if seen[v.Source] {
				continue
			}
			seen[v.Source] = true
			volumes = append(volumes, &ecs.TaskDefinitionVolumeArgs{
				Name: pulumi.String(v.Source),
			})
		}
	}
	add(svc.Volumes)
	for _, sc := range common.Sorted(sidecars) {
		add(sc.Volumes)
	}
	return volumes
}
