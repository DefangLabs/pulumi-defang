package gcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ComputeEngineResult holds the output of CreateComputeEngine.
type ComputeEngineResult struct {
	InstanceGroup *compute.RegionInstanceGroupManager
}

// ComputeEngineArgs carries per-service extras beyond the compose config.
type ComputeEngineArgs struct {
	// SA is the identity the instances run as.
	SA *ServiceIdentity
	// Sidecars are additional containers run on the same instance, as extra
	// systemd units. The main container's volumesFrom/dependsOn may reference
	// sidecar names.
	Sidecars map[string]compose.ServiceConfig
	// Triggers force an instance template replacement (and thus a rolling
	// update) when any value changes.
	Triggers pulumi.StringMapInput
}

// CreateComputeEngine deploys a container service as a Compute Engine Managed Instance Group
// running on Container-Optimized OS with cloud-init/systemd. Used for services that cannot
// run on Cloud Run (e.g. background workers with no listening port).
func CreateComputeEngine(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	args *ComputeEngineArgs,
	gcpConfig *SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*ComputeEngineResult, error) {
	machineType := getComputeMachineType(svc)

	var iamDeps pulumi.ResourceArrayOutput
	if args.SA.Account != nil {
		iamDeps = addRolesToServiceAccount(ctx, args.SA, []string{
			"roles/artifactregistry.reader",
			"roles/logging.logWriter",
			"roles/monitoring.metricWriter",
			"roles/cloudtrace.agent",
		}, gcpConfig, parentOpt)
	}

	// Classify each container's env into inline values and native secret refs,
	// then grant the instance SA access to every referenced secret. Bare ${VAR}
	// / null "KEY:" values are boot-fetched (see secretFetchScript) rather than
	// embedded in instance metadata.
	mainPlan := classifyComputeSecretEnv(ctx, configProvider, svc.Environment)
	sidecarPlans := make(map[string]containerSecretPlan, len(args.Sidecars))
	for name, sc := range args.Sidecars {
		sidecarPlans[name] = classifyComputeSecretEnv(ctx, configProvider, sc.Environment)
	}
	secretMembers, err := grantSecretAccess(ctx, serviceName, args.SA, mainPlan, sidecarPlans, parentOpt)
	if err != nil {
		return nil, err
	}

	namedPorts, healthCheckPort, addHealthCheckSidecar := computeNamedPorts(svc)

	// DEFANG_FQDN: custom domain, else public FQDN (ingress), else the private
	// FQDN (<label>.google.internal) for internal services — the private zone is
	// provisioned in alb.go and resolves within the VPC. See common.ServiceFQDN.
	fqdn := common.ServiceFQDN(serviceName, svc, gcpConfig.Domain, "google.internal")
	cloudInit := getCloudInitConfig(
		serviceName, image, svc, gcpConfig.Region, gcpConfig.Etag, gcpConfig.ProjectName, gcpConfig.Stack, fqdn,
		gcpConfig.GcpProject, addHealthCheckSidecar, args.Sidecars, mainPlan, sidecarPlans)

	// The instance template must depend on the secret IAM bindings so instances
	// don't boot (and run the fetch script) before they can read the secrets.
	templateOpts := []pulumi.ResourceOption{parentOpt}
	if len(secretMembers) > 0 {
		templateOpts = append(templateOpts, pulumi.DependsOn(secretMembers))
	}
	instanceTemplate, err := createInstanceTemplate(
		ctx, serviceName, serviceName, machineType, getComputeBootImage(svc), cloudInit,
		args.SA, args.Triggers, gcpConfig, iamDeps, templateOpts...)
	if err != nil {
		return nil, err
	}

	autoHealing, err := createMIGAutoHealing(
		ctx, serviceName, serviceName, healthCheckPort, addHealthCheckSidecar, networkID(gcpConfig), parentOpt)
	if err != nil {
		return nil, err
	}

	// parentOpt is a ResourceOrInvokeOption so it satisfies pulumi.InvokeOption here
	// and flows the parent (and its provider) into the zones lookup — required for
	// correctness when the stack uses a non-default GCP provider.
	zones, err := compute.GetZones(ctx, &compute.GetZonesArgs{Region: pulumi.StringRef(gcpConfig.Region)}, parentOpt)
	if err != nil {
		return nil, fmt.Errorf("getting zones in region %s: %w", gcpConfig.Region, err)
	}
	// Not every machine type is offered in every zone (notably ARM t2a); pin the
	// MIG's distribution policy to the zones that actually have it.
	zoneNames, err := zonesWithMachineType(ctx, zones.Names, machineType, parentOpt)
	if err != nil {
		return nil, err
	}
	updatePolicy := buildMIGUpdatePolicy(len(zoneNames), int(svc.GetReplicas()))

	migArgs := &compute.RegionInstanceGroupManagerArgs{
		BaseInstanceName:    pulumi.String(serviceName), // FIXME: this resource does not support autonaming
		AutoHealingPolicies: autoHealing,
		Versions: compute.RegionInstanceGroupManagerVersionArray{
			&compute.RegionInstanceGroupManagerVersionArgs{
				InstanceTemplate: instanceTemplate.SelfLink,
			},
		},
		UpdatePolicy:           updatePolicy,
		Region:                 pulumi.String(gcpConfig.Region),
		TargetSize:             pulumi.Int(svc.GetReplicas()),
		NamedPorts:             namedPorts,
		WaitForInstances:       pulumi.Bool(true),
		WaitForInstancesStatus: pulumi.String("STABLE"),
	}
	if len(zoneNames) < len(zones.Names) {
		migArgs.DistributionPolicyZones = pulumi.ToStringArray(zoneNames)
	}
	instanceGroup, err := compute.NewRegionInstanceGroupManager(ctx, serviceName+"-instance-group",
		migArgs, parentOpt, pulumi.DependsOn([]pulumi.Resource{instanceTemplate}))
	if err != nil {
		return nil, fmt.Errorf("creating instance group for %s: %w", serviceName, err)
	}

	return &ComputeEngineResult{InstanceGroup: instanceGroup}, nil
}

// computeNamedPorts builds the MIG named-port array from the service's ports and
// determines the health-check port. Services with no TCP port get a dedicated
// HTTP health-check sidecar on 8080 so the MIG auto-healer can probe liveness.
func computeNamedPorts(svc compose.ServiceConfig) (
	compute.RegionInstanceGroupManagerNamedPortArray, *int, bool,
) {
	namedPorts := make(compute.RegionInstanceGroupManagerNamedPortArray, len(svc.Ports))
	var healthCheckPort *int
	for i, port := range svc.Ports {
		proto := port.GetProtocol()
		namedPorts[i] = &compute.RegionInstanceGroupManagerNamedPortArgs{
			Name: pulumi.String(fmt.Sprintf("port-%s-%d", proto, port.Target)),
			Port: pulumi.Int(port.Target),
		}
		if proto == "tcp" {
			p := int(port.Target)
			healthCheckPort = &p
		}
	}
	addHealthCheckSidecar := healthCheckPort == nil
	if addHealthCheckSidecar {
		p := 8080
		healthCheckPort = &p
	}
	return namedPorts, healthCheckPort, addHealthCheckSidecar
}

// networkID returns the network to attach instances and firewalls to: the
// project VPC when one was provisioned, else the GCP default network so
// standalone services (no shared infra) can still run on Compute Engine.
func networkID(gcpConfig *SharedInfra) pulumi.StringInput {
	if gcpConfig.PublicIP == nil {
		return pulumi.String("default")
	}
	return gcpConfig.VpcId
}

// zonesWithMachineType filters zones to those offering the given machine type.
// Falls back to all zones when the availability lookup finds none (e.g. mock
// backends in tests).
func zonesWithMachineType(
	ctx *pulumi.Context, zones []string, machineType string, opt pulumi.InvokeOption,
) ([]string, error) {
	var available []string
	for _, zone := range zones {
		mts, err := compute.GetMachineTypes(ctx, &compute.GetMachineTypesArgs{
			Filter: pulumi.StringRef(fmt.Sprintf("name = %q", machineType)),
			Zone:   pulumi.StringRef(zone),
		}, opt)
		if err != nil {
			return nil, fmt.Errorf("listing machine types in zone %s: %w", zone, err)
		}
		if len(mts.MachineTypes) > 0 {
			available = append(available, zone)
		}
	}
	if len(available) == 0 {
		return zones, nil
	}
	return available, nil
}

func createInstanceTemplate(
	ctx *pulumi.Context,
	serviceName, instanceTag, machineType, bootImage string,
	cloudInit pulumi.StringInput,
	sa *ServiceIdentity,
	triggers pulumi.StringMapInput,
	gcpConfig *SharedInfra,
	iamDeps pulumi.ResourceArrayOutput,
	opts ...pulumi.ResourceOption,
) (*compute.InstanceTemplate, error) {
	network := &compute.InstanceTemplateNetworkInterfaceArgs{
		Subnetwork: gcpConfig.SubnetId,
		AccessConfigs: compute.InstanceTemplateNetworkInterfaceAccessConfigArray{
			&compute.InstanceTemplateNetworkInterfaceAccessConfigArgs{},
		},
	}
	if gcpConfig.PublicIP == nil {
		// standalone: no project VPC — run on the default network
		network.Subnetwork = nil
		network.Network = pulumi.String("default")
	}

	// Trigger values land in instance metadata: instance templates are immutable,
	// so any change replaces the template and rolls the MIG.
	var metadata pulumi.StringMapInput = pulumi.StringMap{
		"user-data":              cloudInit,
		"google-logging-enabled": pulumi.String("true"),
	}
	if triggers != nil {
		metadata = pulumi.All(cloudInit, triggers).ApplyT(func(vs []any) map[string]string {
			m := map[string]string{
				"user-data":              vs[0].(string),
				"google-logging-enabled": "true",
			}
			for k, v := range vs[1].(map[string]string) {
				m["defang-trigger-"+k] = v
			}
			return m
		}).(pulumi.StringMapOutput)
	}

	templateOpts := make([]pulumi.ResourceOption, 0, len(opts)+2)
	templateOpts = append(templateOpts, opts...)
	templateOpts = append(templateOpts, pulumi.RetainOnDelete(true))
	if sa.Account != nil {
		templateOpts = append(templateOpts, pulumi.DependsOnInputs(iamDeps))
	}
	tmpl, err := compute.NewInstanceTemplate(ctx, serviceName+"-instance-template",
		&compute.InstanceTemplateArgs{
			MachineType: pulumi.String(machineType),
			Scheduling: &compute.InstanceTemplateSchedulingArgs{
				OnHostMaintenance: pulumi.String("MIGRATE"),
			},
			Disks: compute.InstanceTemplateDiskArray{
				&compute.InstanceTemplateDiskArgs{
					Boot:        pulumi.Bool(true),
					SourceImage: pulumi.String(bootImage),
					DiskSizeGb:  pulumi.Int(21),
				},
			},
			NetworkInterfaces: compute.InstanceTemplateNetworkInterfaceArray{network},
			Metadata:          metadata,
			ServiceAccount: &compute.InstanceTemplateServiceAccountArgs{
				Email:  sa.Email,
				Scopes: pulumi.ToStringArray([]string{"cloud-platform"}),
			},
			Tags: pulumi.StringArray{pulumi.String(instanceTag)},
		}, templateOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating instance template for %s: %w", serviceName, err)
	}
	return tmpl, nil
}

func createMIGAutoHealing(
	ctx *pulumi.Context,
	serviceName, instanceTag string,
	healthCheckPort *int,
	addSidecar bool,
	network pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (compute.RegionInstanceGroupManagerAutoHealingPoliciesPtrInput, error) {
	if healthCheckPort == nil {
		return nil, nil //nolint:nilnil
	}

	hcArgs := &compute.HealthCheckArgs{
		CheckIntervalSec:   pulumi.Int(30),
		TimeoutSec:         pulumi.Int(30),
		UnhealthyThreshold: pulumi.Int(5),
		HealthyThreshold:   pulumi.Int(2),
	}
	if addSidecar {
		hcArgs.HttpHealthCheck = &compute.HealthCheckHttpHealthCheckArgs{
			Port: pulumi.Int(*healthCheckPort), RequestPath: pulumi.String("/"),
		}
	} else {
		hcArgs.TcpHealthCheck = &compute.HealthCheckTcpHealthCheckArgs{
			Port: pulumi.Int(*healthCheckPort),
		}
	}

	hcName := serviceName + "-" + strconv.Itoa(*healthCheckPort) + "-mig-hc"
	healthCheck, err := compute.NewHealthCheck(ctx, hcName, hcArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating health check for %s: %w", serviceName, err)
	}

	portStr := strconv.Itoa(*healthCheckPort)
	if _, err := compute.NewFirewall(ctx, serviceName+"-mig-hc-fw", &compute.FirewallArgs{
		Network: network,
		// https://cloud.google.com/load-balancing/docs/health-checks#firewall_rules
		SourceRanges: pulumi.StringArray{
			pulumi.String("130.211.0.0/22"),
			pulumi.String("35.191.0.0/16"),
		},
		Allows: compute.FirewallAllowArray{
			&compute.FirewallAllowArgs{
				Protocol: pulumi.String("tcp"),
				Ports:    pulumi.StringArray{pulumi.String(portStr)},
			},
		},
		TargetTags: pulumi.StringArray{pulumi.String(instanceTag)},
		Direction:  pulumi.String("INGRESS"),
	}, opts...); err != nil {
		return nil, fmt.Errorf("creating health check firewall for %s: %w", serviceName, err)
	}

	return compute.RegionInstanceGroupManagerAutoHealingPoliciesArgs{
		HealthCheck:     healthCheck.SelfLink,
		InitialDelaySec: pulumi.Int(300),
	}, nil
}

// buildMIGUpdatePolicy returns update policy args for a regional MIG.
//
// GCP constraint: MaxSurgeFixed and MaxUnavailableFixed must each be either 0
// or >= the number of zones in the region. We satisfy this by clamping the
// batch size to at least numZones. numZones should be the number of zones in
// the deployment region (from compute.GetZones); pass 1 if unknown.
func buildMIGUpdatePolicy(numZones, targetSize int) *compute.RegionInstanceGroupManagerUpdatePolicyArgs {
	policy := &compute.RegionInstanceGroupManagerUpdatePolicyArgs{
		Type: pulumi.String("PROACTIVE"),
		// REPLACE, not RESTART: user-data metadata changes applied via RESTART
		// don't re-run cloud-init, leaving stale systemd units on the instance.
		MinimalAction: pulumi.String("REPLACE"),
	}
	if targetSize > 10 {
		policy.MaxSurgePercent = pulumi.Int(25)
		policy.MaxUnavailablePercent = pulumi.Int(25)
		return policy
	}
	if numZones < 1 {
		numZones = 1
	}
	batchSize := max(targetSize/2, numZones)
	policy.MaxSurgeFixed = pulumi.Int(batchSize)
	policy.MaxUnavailableFixed = pulumi.Int(batchSize)
	return policy
}

// grantSecretAccess grants the instance service account
// roles/secretmanager.secretAccessor on every Secret Manager secret referenced
// (as a bare ${VAR} or null "KEY:") by the main container or any sidecar, so the
// boot-time fetch script can read them. Secret IDs are deduped across
// containers. Returns the created members for use as instance-template
// dependencies.
func grantSecretAccess(
	ctx *pulumi.Context,
	serviceName string,
	sa *ServiceIdentity,
	mainPlan containerSecretPlan,
	sidecarPlans map[string]containerSecretPlan,
	parentOpt pulumi.ResourceOrInvokeOption,
) ([]pulumi.Resource, error) {
	seen := make(map[string]struct{})
	var ids []string
	collect := func(plan containerSecretPlan) {
		for _, r := range plan.secretRefs {
			if _, ok := seen[r.secretID]; !ok {
				seen[r.secretID] = struct{}{}
				ids = append(ids, r.secretID)
			}
		}
	}
	collect(mainPlan)
	for _, plan := range common.Sorted(sidecarPlans) {
		collect(plan)
	}

	var members []pulumi.Resource
	for _, sid := range ids {
		opts := append([]pulumi.ResourceOption{parentOpt}, sa.deleteOpts()...)
		member, err := secretmanager.NewSecretIamMember(ctx, serviceName+"-secret-"+sid,
			&secretmanager.SecretIamMemberArgs{
				SecretId: pulumi.String(sid),
				Role:     pulumi.String("roles/secretmanager.secretAccessor"),
				Member:   pulumi.Sprintf("serviceAccount:%v", sa.Email),
			}, opts...)
		if err != nil {
			return nil, fmt.Errorf("granting secret access for %s: %w", sid, err)
		}
		members = append(members, member)
	}
	return members, nil
}

// addRolesToServiceAccount grants IAM roles to a service account at the project level.
// Returns a resource array output that can be used as a dependency.
func addRolesToServiceAccount(
	ctx *pulumi.Context,
	sa *ServiceIdentity,
	roles []string,
	gcpConfig *SharedInfra,
	opts ...pulumi.ResourceOption,
) pulumi.ResourceArrayOutput {
	return sa.Email.ToStringOutput().ApplyT(func(email string) ([]pulumi.Resource, error) {
		var members []pulumi.Resource
		for _, role := range roles {
			member, err := projects.NewIAMMember(ctx, email+"-"+role,
				&projects.IAMMemberArgs{
					Project: pulumi.String(gcpConfig.GcpProject),
					Role:    pulumi.String(role),
					Member:  pulumi.Sprintf("serviceAccount:%v", email),
				},
				append(opts, sa.deleteOpts()...)...,
			)
			if err != nil {
				return nil, err
			}
			members = append(members, member)
		}
		return members, nil
	}).(pulumi.ResourceArrayOutput)
}

// machineTypeEntry describes a GCP machine type with its CPU and memory capacity.
type machineTypeEntry struct {
	name string
	cpu  float32 // vCPUs
	mem  uint64  // MiB
}

// e2MachineTypes lists E2 machine types in ascending order of resources.
// Source: https://cloud.google.com/compute/docs/general-purpose-machines#e2_machine_types
var e2MachineTypes = []machineTypeEntry{
	{"e2-micro", 0.25, 1 * 1024},
	{"e2-small", 0.5, 2 * 1024},
	{"e2-medium", 1, 4 * 1024},
	{"e2-standard-2", 2, 8 * 1024},
	{"e2-standard-4", 4, 16 * 1024},
	{"e2-standard-8", 8, 32 * 1024},
	{"e2-standard-16", 16, 64 * 1024},
	{"e2-standard-32", 32, 128 * 1024},
	{"e2-highmem-2", 2, 16 * 1024},
	{"e2-highmem-4", 4, 32 * 1024},
	{"e2-highmem-8", 8, 64 * 1024},
	{"e2-highmem-16", 16, 128 * 1024},
	{"e2-highcpu-2", 2, 2 * 1024},
	{"e2-highcpu-4", 4, 4 * 1024},
	{"e2-highcpu-8", 8, 8 * 1024},
	{"e2-highcpu-16", 16, 16 * 1024},
	{"e2-highcpu-32", 32, 32 * 1024},
}

// t2aMachineTypes lists T2A (Ampere ARM) machine types in ascending order.
// Source: https://cloud.google.com/compute/docs/general-purpose-machines#t2a_machines
var t2aMachineTypes = []machineTypeEntry{
	{"t2a-standard-1", 1, 4 * 1024},
	{"t2a-standard-2", 2, 8 * 1024},
	{"t2a-standard-4", 4, 16 * 1024},
	{"t2a-standard-8", 8, 32 * 1024},
	{"t2a-standard-16", 16, 64 * 1024},
	{"t2a-standard-32", 32, 128 * 1024},
	{"t2a-standard-48", 48, 192 * 1024},
}

// getComputeBootImage returns the Container-Optimized OS boot image family
// matching the service's platform: ARM machine types (T2A) can only boot the
// arm64 COS family; everything else uses the x86 family.
func getComputeBootImage(svc compose.ServiceConfig) string {
	if strings.Contains(svc.GetPlatform(), "arm64") {
		return "projects/cos-cloud/global/images/family/cos-arm64-stable"
	}
	return "projects/cos-cloud/global/images/family/cos-stable"
}

// getComputeMachineType selects the smallest machine type that satisfies the
// service's CPU and memory reservations. ARM platforms get T2A (Ampere),
// everything else E2.
func getComputeMachineType(svc compose.ServiceConfig) string {
	cpus := float32(svc.GetCPUs())
	memMiB := uint64(svc.GetMemoryMiB()) //nolint:gosec // GetMemoryMiB() always returns >= 512

	table := e2MachineTypes
	fallback := "e2-standard-2"
	if strings.Contains(svc.GetPlatform(), "arm64") {
		table = t2aMachineTypes
		fallback = "t2a-standard-2"
	}
	for _, mt := range table {
		if mt.cpu >= cpus && mt.mem >= memMiB {
			return mt.name
		}
	}
	return fallback
}

// dockerRunFlags builds `docker run` flags shared by main and sidecar
// containers: entrypoint, ports, mounts (-v), volumes-from, working dir.
// The returned command is the argv appended after the image. sidecars is
// used to resolve volumesFrom service names to container names.
func dockerRunFlags(
	svc compose.ServiceConfig, sidecars map[string]compose.ServiceConfig,
) ([]string, []string) {
	params := make([]string, 0, 2+2*len(svc.Ports)+2*len(svc.Volumes)+len(svc.VolumesFrom)+2)
	var command []string
	if len(svc.Entrypoint) > 0 {
		params = append(params, "--entrypoint", svc.Entrypoint[0])
		if len(svc.Entrypoint) > 1 {
			command = append(command, svc.Entrypoint[1:]...)
		}
	}
	command = append(command, svc.Command...)

	for _, port := range svc.Ports {
		portMapping := fmt.Sprintf("%d:%d", port.Target, port.Target)
		if port.Protocol != "" {
			portMapping = fmt.Sprintf("%s/%s", portMapping, port.Protocol)
		}
		params = append(params, "-p", portMapping)
	}

	for _, vol := range svc.Volumes {
		mount := vol.Source + ":" + vol.Target
		if vol.ReadOnly {
			mount += ":ro"
		}
		params = append(params, "-v", mount)
	}

	for _, ref := range svc.VolumesFrom {
		name, suffix, _ := strings.Cut(ref, ":")
		if sc, ok := sidecars[name]; ok {
			name = sc.GetContainerName(name)
		}
		if suffix != "" {
			name += ":" + suffix
		}
		params = append(params, "--volumes-from", name)
	}

	if svc.WorkingDir != nil && *svc.WorkingDir != "" {
		params = append(params, "-w", *svc.WorkingDir)
	}

	return params, command
}

// computeSecretEnv maps a container env var to the Secret Manager secret ID
// whose latest version supplies its value. The value is fetched at container
// start (see secretFetchScript) instead of being embedded in instance metadata,
// which is readable unauthenticated from the instance.
type computeSecretEnv struct {
	envKey   string
	secretID string
}

// containerSecretPlan is the per-container result of classifyComputeSecretEnv:
// the env values inlined into `docker run -e` and the ones boot-fetched from
// Secret Manager.
type containerSecretPlan struct {
	inline     compose.Environment
	secretRefs []computeSecretEnv
}

// classifyComputeSecretEnv splits a container's environment into values inlined
// into the `docker run` command (static literals and dynamic Outputs) and
// native secret references (bare ${VAR} or null "KEY:") that are boot-fetched
// from Secret Manager. Interpolated values (mixed literal + ${VAR}) stay on the
// inline path for now; resolving them without leaking plaintext into instance
// metadata needs deploy-time derived configs — see issue 293. With no config
// provider (e.g. standalone with no secrets), everything is inlined.
func classifyComputeSecretEnv(
	ctx *pulumi.Context, cp compose.ConfigProvider, env compose.Environment,
) containerSecretPlan {
	if cp == nil {
		return containerSecretPlan{inline: env}
	}
	plan := containerSecretPlan{inline: make(compose.Environment, len(env))}
	// Iterate sorted so the generated fetch script and IAM bindings are stable.
	for k, v := range common.Sorted(env) {
		if configKey := compose.GetConfigName2(k, v); configKey != "" {
			if secretID, err := cp.GetSecretRef(ctx, configKey); err == nil && secretID != "" {
				plan.secretRefs = append(plan.secretRefs, computeSecretEnv{envKey: k, secretID: secretID})
				continue
			}
			// Couldn't resolve a ref: fall through and inline (matches the
			// pre-secrets behavior rather than dropping the variable).
		}
		plan.inline[k] = v
	}
	return plan
}

// secretFetchScript returns the cloud-init write_files block for the boot-time
// secret fetch script, the systemd ExecStartPre line that runs it, and the
// `docker run --env-file` flag that injects the fetched values. All three are
// empty when there are no secret refs.
//
// The script reads the instance service account's OAuth token from the metadata
// server and fetches each secret's latest version from the Secret Manager REST
// API — no gcloud (absent on Container-Optimized OS) required. Values land in a
// tmpfs file (0600, /run) that never touches disk or instance metadata.
//
// NOTE: docker --env-file is line-oriented, so a secret whose value contains a
// newline (e.g. a PEM key) cannot be injected this way. Such values need a
// per-secret file mount; tracked as a follow-up.
// Returns (write_files block, ExecStartPre line, docker --env-file flag).
func secretFetchScript(gcpProject, unit string, refs []computeSecretEnv) (string, string, string) {
	if len(refs) == 0 {
		return "", "", ""
	}
	scriptPath := "/opt/defang/" + unit + "-secrets.sh"
	envFile := "/run/defang/" + unit + ".env"

	var s strings.Builder
	s.WriteString("#!/bin/bash\n")
	s.WriteString("set -uo pipefail\n")
	s.WriteString("umask 077\n")
	s.WriteString("mkdir -p /run/defang\n")
	s.WriteString(`tok=$(curl -s -H "Metadata-Flavor: Google" ` +
		`http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token ` +
		`| grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)` + "\n")
	fmt.Fprintf(&s, `sm() { curl -s -H "Authorization: Bearer $tok" `+
		`"https://secretmanager.googleapis.com/v1/projects/%s/secrets/$1/versions/latest:access" `+
		`| grep -o '"data":"[^"]*"' | cut -d'"' -f4 | base64 -d; }`+"\n", gcpProject)
	s.WriteString("{\n")
	for _, r := range refs {
		fmt.Fprintf(&s, "printf '%%s=%%s\\n' '%s' \"$(sm '%s')\"\n", r.envKey, r.secretID)
	}
	fmt.Fprintf(&s, "} > %s\n", envFile)

	// Indent the script body under the write_files `content: |` block (6 spaces).
	var wf strings.Builder
	fmt.Fprintf(&wf, "\n  - path: %s\n    permissions: \"0700\"\n    owner: root\n    content: |\n", scriptPath)
	for _, line := range strings.Split(strings.TrimRight(s.String(), "\n"), "\n") {
		wf.WriteString("      ")
		wf.WriteString(line)
		wf.WriteString("\n")
	}
	return wf.String(), "ExecStartPre=" + scriptPath, "--env-file " + envFile
}

// flattenEnvFlags renders `-e "K=V"` docker flags for the given static env
// pairs followed by the service's compose environment. Dynamic (Output)
// values are threaded through a Sprintf so they resolve at apply time.
func flattenEnvFlags(staticEnv [][2]string, env compose.Environment) pulumi.StringInput {
	var format strings.Builder
	var args []any
	for _, kv := range staticEnv {
		format.WriteString(escapePercent(fmt.Sprintf("-e %q ", kv[0]+"="+kv[1])))
	}
	for k, v := range common.Sorted(env) {
		if sv, static := compose.StaticEnvValue(v); static {
			// Compute/VM deploys embed concrete values into the `docker run -e`
			// cmd string — no runtime config-provider resolution is available
			// here, so a nil value (YAML "KEY:" with no value) flattens to an
			// empty value.
			var val string
			if sv != nil {
				val = *sv
			}
			// FIXME: provide all environment (and secrets) as env instead of flattening into the command line.
			format.WriteString(escapePercent(fmt.Sprintf("-e %q ", k+"="+val)))
		} else {
			fmt.Fprintf(&format, "-e \"%s=%%s\" ", escapePercent(k))
			args = append(args, v)
		}
	}
	// Always go through Sprintf: the static text above was %%-escaped for it.
	return pulumi.Sprintf(format.String(), args...)
}

// stopGraceSeconds returns the docker stop timeout for a service, defaulting
// to 30s (matches `docker stop`'s conservative default used previously).
func stopGraceSeconds(svc compose.ServiceConfig) int {
	if s := svc.GetStopGracePeriodSeconds(); s > 0 {
		return s
	}
	return 30
}

// buildUnitDependencies renders the [Unit] ordering directives for the main
// service: it always waits on gcr-online + docker, and additionally requires and
// orders After each same-instance sidecar named in depends_on (cross-instance
// dependencies aren't enforceable from a single unit).
func buildUnitDependencies(
	serviceName string, svc compose.ServiceConfig, sidecars map[string]compose.ServiceConfig,
) string {
	var dependencies strings.Builder
	dependencies.WriteString("Wants=gcr-online.target docker.socket\n      After=gcr-online.target docker.socket")
	for name := range common.Sorted(svc.DependsOn) {
		if _, ok := sidecars[name]; !ok {
			continue // only same-instance sidecar dependencies are enforceable here
		}
		unit := sidecarUnitName(serviceName, name)
		fmt.Fprintf(&dependencies, "\n      Requires=%s.service\n      After=%s.service", unit, unit)
	}
	return dependencies.String()
}

// getCloudInitConfig generates a cloud-init YAML string for running a container on
// Container-Optimized OS using systemd. For portless services it adds an HTTP health
// check sidecar so the MIG auto-healer can probe container liveness. User-defined
// sidecars each get their own systemd unit, started before the main service so
// `volumes_from` references resolve.
func getCloudInitConfig(
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	region, etag, projectName, stack, fqdn string,
	gcpProject string,
	addHealthCheckSidecar bool,
	sidecars map[string]compose.ServiceConfig,
	mainPlan containerSecretPlan,
	sidecarPlans map[string]containerSecretPlan,
) pulumi.StringOutput {
	var buf strings.Builder
	buf.WriteString("#cloud-config\n\nwrite_files:")

	containerName := svc.GetContainerName(serviceName)
	params, command := dockerRunFlags(svc, sidecars)

	// Secret env vars are boot-fetched into a tmpfs env-file rather than embedded
	// in instance metadata. The fetch script is written first so it precedes the
	// service unit in write_files.
	secretWriteFile, secretExecPre, secretEnvFileFlag := secretFetchScript(gcpProject, serviceName, mainPlan.secretRefs)
	if secretEnvFileFlag != "" {
		params = append(params, secretEnvFileFlag)
	}
	buf.WriteString(escapePercent(secretWriteFile))

	// Defang-injected runtime vars, mirroring the Cloud Run path (buildEnvVars).
	staticEnv := [][2]string{{"DEFANG_SERVICE", serviceName}}
	if etag != "" {
		staticEnv = append(staticEnv, [2]string{"DEFANG_ETAG", etag})
	}
	if fqdn != "" {
		staticEnv = append(staticEnv, [2]string{"DEFANG_FQDN", fqdn})
	}
	envFlags := flattenEnvFlags(staticEnv, mainPlan.inline)

	dependencies := buildUnitDependencies(serviceName, svc, sidecars)

	runcmds := make([]string, 0, 5+4+2*len(sidecars)+2)
	runcmds = append(runcmds,
		`echo 'DOCKER_OPTS="--registry-mirror=https://mirror.gcr.io"' | tee /etc/default/docker`,
		fluentBitLabelsCmd(serviceName, etag, projectName, stack),
		"systemctl daemon-reload",
		"systemctl restart fluent-bit",
		"systemctl restart docker",
	)

	var extraUnitsFmt strings.Builder
	extraUnitsArgs := make([]any, 0, len(sidecars)+1)
	if addHealthCheckSidecar {
		var hcUnits string
		hcUnits, runcmds = buildHealthCheckUnits(serviceName, containerName, runcmds)
		extraUnitsFmt.WriteString("%s")
		extraUnitsArgs = append(extraUnitsArgs, hcUnits)
	}
	for name, sc := range common.Sorted(sidecars) {
		var unitText pulumi.StringInput
		unitText, runcmds = buildSidecarUnit(serviceName, name, region, gcpProject, sc, sidecars, sidecarPlans[name], runcmds)
		extraUnitsFmt.WriteString("%s")
		extraUnitsArgs = append(extraUnitsArgs, unitText)
	}
	extraUnits := pulumi.Sprintf(extraUnitsFmt.String(), extraUnitsArgs...)

	runcmds = append(runcmds,
		fmt.Sprintf("systemctl enable %s.service", serviceName),
		fmt.Sprintf("systemctl start %s.service", serviceName),
	)

	fmt.Fprintf(&buf, `
  - path: /etc/systemd/system/%[1]s.service
    permissions: "0644"
    owner: root
    content: |
      [Unit]
      Description=%[1]s service
      %[2]s

      [Service]
      Restart=always
      RestartSec=30
      Environment="HOME=/home/container-user"
      %[9]s
      ExecStartPre=/usr/bin/docker-credential-gcr configure-docker --registries %[5]s-docker.pkg.dev
      ExecStart=/usr/bin/docker run --pull=always --rm --name=%[7]s %[3]s %%s%%s %[4]s
      ExecStop=/usr/bin/docker stop -t %[6]d %[7]s
      StandardOutput=journal+console
      StandardError=journal+console

      [Install]
      WantedBy=multi-user.target
%%s
runcmd:
  - %[8]s
`,
		serviceName,
		escapePercent(dependencies),
		escapePercent(strings.Join(params, " ")),
		escapePercent(strings.Join(command, " ")),
		region,
		stopGraceSeconds(svc),
		containerName,
		escapePercent(strings.Join(runcmds, "\n  - ")),
		escapePercent(secretExecPre))

	// buf now contains exactly three %s placeholders (env flags, image, and
	// extra units — the Inputs that may resolve at apply time); all other
	// dynamic text had its '%' doubled.
	return pulumi.Sprintf(buf.String(), envFlags, image, extraUnits)
}

// fluentBitLabelsCmd returns the runcmd that stamps the defang-* LogEntry
// labels the Defang CLI (and Fabric) filter their Cloud Logging tail queries
// on. COS's fluent-bit ships container logs to Cloud Logging (logName
// cos_containers); the 4-space indent lands the `labels` key inside the
// stackdriver [OUTPUT] section of its config. Empty values are omitted so the
// matching query clause can be omitted too.
// TODO: find a more reliable way to add labels
func fluentBitLabelsCmd(serviceName, etag, projectName, stack string) string {
	labels := make([]string, 0, 4)
	if etag != "" {
		labels = append(labels, "defang-etag="+SafeLabelValue(etag))
	}
	if projectName != "" {
		labels = append(labels, "defang-project="+SafeLabelValue(projectName))
	}
	labels = append(labels, "defang-service="+SafeLabelValue(serviceName))
	if stack != "" {
		labels = append(labels, "defang-stack="+SafeLabelValue(stack))
	}
	return fmt.Sprintf(`echo "    labels %s" >> /etc/fluent-bit/fluent-bit.conf`, strings.Join(labels, ","))
}

// escapePercent doubles '%' so text survives the final pulumi.Sprintf pass
// (which only substitutes the env flags, main image, and extra units via %s).
func escapePercent(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}

func sidecarUnitName(serviceName, sidecarName string) string {
	return serviceName + "-" + sidecarName
}

// buildSidecarUnit returns the cloud-init write_files entry and runcmd lines for a
// user-defined sidecar container. Sidecars run without --rm so the main container's
// --volumes-from keeps working across restarts; a run-once sidecar (restart: "no")
// becomes a oneshot unit the main service can order After=.
func buildSidecarUnit(
	serviceName, sidecarName, region, gcpProject string,
	sc compose.ServiceConfig,
	sidecars map[string]compose.ServiceConfig,
	plan containerSecretPlan,
	runcmds []string,
) (pulumi.StringInput, []string) {
	unit := sidecarUnitName(serviceName, sidecarName)
	containerName := sc.GetContainerName(sidecarName)
	params, command := dockerRunFlags(sc, sidecars)

	// Boot-fetch this sidecar's secret env into its own tmpfs env-file.
	secretWriteFile, secretExecPre, secretEnvFileFlag := secretFetchScript(gcpProject, unit, plan.secretRefs)
	if secretEnvFileFlag != "" {
		params = append(params, secretEnvFileFlag)
	}
	envFlags := flattenEnvFlags(nil, plan.inline)

	var serviceSection string
	if sc.Restart == "no" {
		serviceSection = "Type=oneshot\n      RemainAfterExit=yes"
	} else {
		serviceSection = fmt.Sprintf("Restart=always\n      RestartSec=30\n      ExecStop=/usr/bin/docker stop -t %d %s",
			stopGraceSeconds(sc), containerName)
	}

	// %[11]s (secret fetch script write_files entry) and %[12]s (its
	// ExecStartPre) are passed as Sprintf argument values, so any '%' in the
	// script survives literally without escaping.
	units := pulumi.Sprintf(`%[11]s
  - path: /etc/systemd/system/%[1]s.service
    permissions: "0644"
    owner: root
    content: |
      [Unit]
      Description=%[2]s sidecar for %[3]s
      Wants=gcr-online.target docker.socket
      After=gcr-online.target docker.socket
      Before=%[3]s.service

      [Service]
      %[4]s
      Environment="HOME=/home/container-user"
      %[12]s
      ExecStartPre=/usr/bin/docker-credential-gcr configure-docker --registries %[5]s-docker.pkg.dev
      ExecStartPre=-/usr/bin/docker rm -f %[6]s
      ExecStart=/usr/bin/docker run --pull=always --name=%[6]s %[7]s %[8]s%[9]s %[10]s
      StandardOutput=journal+console
      StandardError=journal+console

      [Install]
      WantedBy=multi-user.target
`,
		unit,
		sidecarName,
		serviceName,
		serviceSection,
		region,
		containerName,
		strings.Join(params, " "),
		envFlags,
		sc.Image, // validated non-nil in createService; may resolve at apply time
		strings.Join(command, " "),
		secretWriteFile,
		secretExecPre)

	runcmds = append(runcmds,
		fmt.Sprintf("systemctl enable %s.service", unit),
		fmt.Sprintf("systemctl start %s.service", unit),
	)
	return units, runcmds
}

// buildHealthCheckUnits returns the cloud-init write_files entries and runcmd lines for
// the HTTP health check sidecar: a systemd socket that returns 200 when the container is
// running and 500 otherwise, allowing the MIG auto-healer to probe portless services.
func buildHealthCheckUnits(serviceName, containerName string, runcmds []string) (string, []string) {
	units := fmt.Sprintf(`
  - path: /etc/systemd/system/%[1]s-health-firewall.service
    permissions: "0644"
    owner: root
    content: |
      [Unit]
      Description=%[1]s health check firewall rule
      Before=%[1]s-health.socket

      [Service]
      Type=oneshot
      ExecStart=/sbin/iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

      [Install]
      WantedBy=multi-user.target

  - path: /etc/systemd/system/%[1]s-health.socket
    permissions: "0644"
    owner: root
    content: |
      [Unit]
      Description=%[1]s health check socket

      [Socket]
      ListenStream=8080
      Accept=yes
      FreeBind=yes

      [Install]
      WantedBy=sockets.target

  - path: /etc/systemd/system/%[1]s-health@.service
    permissions: "0644"
    owner: root
    content: |
      [Unit]
      Description=%[1]s health check service
      After=%[1]s-health.socket
      Requires=%[1]s-health.socket

      [Service]
      ExecStart=/bin/bash -c '\
        if docker inspect --format "{{.State.Status}}" %[2]s 2>/dev/null | grep -q running; then \
          echo -e "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"; \
        else \
          echo -e "HTTP/1.1 500 FAIL\r\nContent-Length: 4\r\n\r\nFAIL"; \
        fi;'
      StandardInput=socket
      StandardOutput=socket
`, serviceName, containerName)

	runcmds = append(runcmds,
		fmt.Sprintf("systemctl enable %s-health-firewall.service", serviceName),
		fmt.Sprintf("systemctl start %s-health-firewall.service", serviceName),
		fmt.Sprintf("systemctl enable %s-health.socket", serviceName),
		fmt.Sprintf("systemctl start %s-health.socket", serviceName),
	)
	return units, runcmds
}
