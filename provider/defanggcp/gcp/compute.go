package gcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
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

	// DEFANG_FQDN: custom domain, else public FQDN (ingress), else the private
	// FQDN (<label>.google.internal) for internal services — the private zone is
	// provisioned in alb.go and resolves within the VPC. See common.ServiceFQDN.
	fqdn := common.ServiceFQDN(serviceName, svc, gcpConfig.Domain, "google.internal")
	cloudInit := getCloudInitConfig(
		serviceName, image, svc, gcpConfig.Region, gcpConfig.Etag, fqdn, addHealthCheckSidecar, args.Sidecars)

	instanceTemplate, err := createInstanceTemplate(
		ctx, serviceName, serviceName, machineType, cloudInit, args.SA, args.Triggers, gcpConfig, iamDeps, parentOpt)
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
	serviceName, instanceTag, machineType string,
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
					SourceImage: pulumi.String("projects/cos-cloud/global/images/family/cos-stable"),
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
	var params, command []string
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

// flattenEnvFlags renders `-e "K=V"` docker flags for the given static env
// pairs followed by the service's compose environment.
func flattenEnvFlags(staticEnv [][2]string, env map[string]*string) string {
	var envFlags strings.Builder
	for _, kv := range staticEnv {
		fmt.Fprintf(&envFlags, "-e %q ", fmt.Sprintf("%s=%s", kv[0], kv[1]))
	}
	for k, v := range common.Sorted(env) {
		// Compute/VM deploys embed concrete values into the `docker run -e` cmd
		// string — no runtime config-provider resolution is available here, so
		// a nil *string (YAML "KEY:" with no value) flattens to an empty value.
		var val string
		if v != nil {
			val = *v
		}
		// FIXME: provide all environment (and secrets) as env instead of flattening into the command line.
		fmt.Fprintf(&envFlags, "-e %q ", fmt.Sprintf("%s=%s", k, val))
	}
	return envFlags.String()
}

// stopGraceSeconds returns the docker stop timeout for a service, defaulting
// to 30s (matches `docker stop`'s conservative default used previously).
func stopGraceSeconds(svc compose.ServiceConfig) int {
	if s := svc.GetStopGracePeriodSeconds(); s > 0 {
		return s
	}
	return 30
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
	region, etag, fqdn string,
	addHealthCheckSidecar bool,
	sidecars map[string]compose.ServiceConfig,
) pulumi.StringOutput {
	var buf strings.Builder
	buf.WriteString("#cloud-config\n\nwrite_files:")

	containerName := svc.GetContainerName(serviceName)
	params, command := dockerRunFlags(svc, sidecars)

	// Defang-injected runtime vars, mirroring the Cloud Run path (buildEnvVars).
	staticEnv := [][2]string{{"DEFANG_SERVICE", serviceName}}
	if etag != "" {
		staticEnv = append(staticEnv, [2]string{"DEFANG_ETAG", etag})
	}
	if fqdn != "" {
		staticEnv = append(staticEnv, [2]string{"DEFANG_FQDN", fqdn})
	}
	envFlags := flattenEnvFlags(staticEnv, svc.Environment)

	var dependencies strings.Builder
	dependencies.WriteString("Wants=gcr-online.target docker.socket\n      After=gcr-online.target docker.socket")
	for name := range common.Sorted(svc.DependsOn) {
		if _, ok := sidecars[name]; !ok {
			continue // only same-instance sidecar dependencies are enforceable here
		}
		unit := sidecarUnitName(serviceName, name)
		fmt.Fprintf(&dependencies, "\n      Requires=%s.service\n      After=%s.service", unit, unit)
	}

	runcmds := make([]string, 0, 3+4+2*len(sidecars)+2)
	runcmds = append(runcmds,
		`echo 'DOCKER_OPTS="--registry-mirror=https://mirror.gcr.io"' | tee /etc/default/docker`,
		"systemctl daemon-reload",
		"systemctl restart docker",
	)

	var extraUnits strings.Builder
	if addHealthCheckSidecar {
		var hcUnits string
		hcUnits, runcmds = buildHealthCheckUnits(serviceName, containerName, runcmds)
		extraUnits.WriteString(hcUnits)
	}
	for name, sc := range common.Sorted(sidecars) {
		var unitText string
		unitText, runcmds = buildSidecarUnit(serviceName, name, region, sc, sidecars, runcmds)
		extraUnits.WriteString(unitText)
	}

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
      ExecStartPre=/usr/bin/docker-credential-gcr configure-docker --registries %[5]s-docker.pkg.dev
      ExecStart=/usr/bin/docker run --pull=always --rm --name=%[9]s %[3]s %[6]s%%s %[4]s
      ExecStop=/usr/bin/docker stop -t %[8]d %[9]s
      StandardOutput=journal+console
      StandardError=journal+console

      [Install]
      WantedBy=multi-user.target
%[7]s
runcmd:
  - %[10]s
`,
		serviceName,
		escapePercent(dependencies.String()),
		escapePercent(strings.Join(params, " ")),
		escapePercent(strings.Join(command, " ")),
		region,
		escapePercent(envFlags),
		escapePercent(extraUnits.String()),
		stopGraceSeconds(svc),
		containerName,
		escapePercent(strings.Join(runcmds, "\n  - ")))

	// buf now contains exactly one %s (the escaped image placeholder from the
	// format string above); all other dynamic text had its '%' doubled.
	return pulumi.Sprintf(buf.String(), image)
}

// escapePercent doubles '%' so text survives the final pulumi.Sprintf pass
// (which only substitutes the main image via %s).
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
	serviceName, sidecarName, region string,
	sc compose.ServiceConfig,
	sidecars map[string]compose.ServiceConfig,
	runcmds []string,
) (string, []string) {
	unit := sidecarUnitName(serviceName, sidecarName)
	containerName := sc.GetContainerName(sidecarName)
	params, command := dockerRunFlags(sc, sidecars)
	envFlags := flattenEnvFlags(nil, sc.Environment)
	sidecarImage := "" // validated static & non-empty in createService
	if img := sc.StaticImage(); img != nil {
		sidecarImage = *img
	}

	var serviceSection string
	if sc.Restart == "no" {
		serviceSection = "Type=oneshot\n      RemainAfterExit=yes"
	} else {
		serviceSection = fmt.Sprintf("Restart=always\n      RestartSec=30\n      ExecStop=/usr/bin/docker stop -t %d %s",
			stopGraceSeconds(sc), containerName)
	}

	units := fmt.Sprintf(`
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
		sidecarImage,
		strings.Join(command, " "))

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
