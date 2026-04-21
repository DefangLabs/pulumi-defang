package gcp

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ComputeEngineResult holds the output of CreateComputeEngine.
type ComputeEngineResult struct {
	InstanceGroup *compute.RegionInstanceGroupManager
}

// CreateComputeEngine deploys a container service as a Compute Engine Managed Instance Group
// running on Container-Optimized OS with cloud-init/systemd. Used for services that cannot
// run on Cloud Run (e.g. background workers with no listening port).
func CreateComputeEngine(
	ctx *pulumi.Context,
	projectName string,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	sa *serviceaccount.Account,
	gcpConfig *SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
) (*ComputeEngineResult, error) {
	machineType := getComputeMachineType(svc)

	iamDeps := addRolesToServiceAccount(ctx, sa, []string{
		"roles/artifactregistry.reader",
		"roles/logging.logWriter",
		"roles/monitoring.metricWriter",
		"roles/cloudtrace.agent",
	}, gcpConfig, parentOpt)

	var namedPorts compute.RegionInstanceGroupManagerNamedPortArray
	var healthCheckPort *int
	for _, port := range svc.Ports {
		proto := port.GetProtocol()
		namedPorts = append(namedPorts, &compute.RegionInstanceGroupManagerNamedPortArgs{
			Name: pulumi.String(fmt.Sprintf("port-%s-%d", proto, port.Target)),
			Port: pulumi.Int(port.Target),
		})
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

	cloudInit := getCloudInitConfig(serviceName, image, svc, gcpConfig.Region, addHealthCheckSidecar)

	instanceTemplate, err := createInstanceTemplate(
		ctx, serviceName, serviceName, machineType, cloudInit, sa, gcpConfig, iamDeps, parentOpt)
	if err != nil {
		return nil, err
	}

	autoHealing, err := createMIGAutoHealing(
		ctx, serviceName, serviceName, healthCheckPort, addHealthCheckSidecar, gcpConfig.VpcId, parentOpt)
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
	updatePolicy := buildMIGUpdatePolicy(len(zones.Names), int(svc.GetReplicas()))

	instanceGroup, err := compute.NewRegionInstanceGroupManager(ctx, serviceName+"-instance-group",
		&compute.RegionInstanceGroupManagerArgs{
			BaseInstanceName:    pulumi.String(serviceName),
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
		}, parentOpt, pulumi.DependsOn([]pulumi.Resource{instanceTemplate}))
	if err != nil {
		return nil, fmt.Errorf("creating instance group for %s: %w", serviceName, err)
	}

	return &ComputeEngineResult{InstanceGroup: instanceGroup}, nil
}

func createInstanceTemplate(
	ctx *pulumi.Context,
	serviceName, instanceTag, machineType string,
	cloudInit pulumi.StringInput,
	sa *serviceaccount.Account,
	gcpConfig *SharedInfra,
	iamDeps pulumi.ResourceArrayOutput,
	opts ...pulumi.ResourceOption,
) (*compute.InstanceTemplate, error) {
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
			NetworkInterfaces: compute.InstanceTemplateNetworkInterfaceArray{
				&compute.InstanceTemplateNetworkInterfaceArgs{
					Subnetwork: gcpConfig.SubnetId,
					AccessConfigs: compute.InstanceTemplateNetworkInterfaceAccessConfigArray{
						&compute.InstanceTemplateNetworkInterfaceAccessConfigArgs{},
					},
				},
			},
			Metadata: pulumi.StringMap{
				"user-data":              cloudInit,
				"google-logging-enabled": pulumi.String("true"),
			},
			ServiceAccount: &compute.InstanceTemplateServiceAccountArgs{
				Email:  sa.Email,
				Scopes: pulumi.ToStringArray([]string{"cloud-platform"}),
			},
			Tags: pulumi.StringArray{pulumi.String(instanceTag)},
		}, append(opts,
			pulumi.DependsOnInputs(iamDeps),
			pulumi.RetainOnDelete(true),
		)...)
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
	vpcId pulumi.StringOutput,
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
		Network: vpcId,
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
		Type:          pulumi.String("PROACTIVE"),
		MinimalAction: pulumi.String("RESTART"),
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
	sa *serviceaccount.Account,
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
				append(opts,
					pulumi.DeletedWith(sa),
					pulumi.DeleteBeforeReplace(true),
				)...,
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

// getComputeMachineType selects the smallest E2 machine type that satisfies the
// service's CPU and memory reservations.
func getComputeMachineType(svc compose.ServiceConfig) string {
	cpus := float32(svc.GetCPUs())
	memMiB := uint64(svc.GetMemoryMiB()) //nolint:gosec // GetMemoryMiB() always returns >= 512

	for _, mt := range e2MachineTypes {
		if mt.cpu >= cpus && mt.mem >= memMiB {
			return mt.name
		}
	}
	return "e2-standard-2" // fallback
}

// getCloudInitConfig generates a cloud-init YAML string for running a container on
// Container-Optimized OS using systemd. For portless services it adds an HTTP health
// check sidecar so the MIG auto-healer can probe container liveness.
func getCloudInitConfig(
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	region string,
	addHealthCheckSidecar bool,
) pulumi.StringOutput {
	var buf strings.Builder
	buf.WriteString("#cloud-config\n\nwrite_files:")

	params := make([]string, 0, len(svc.Entrypoint)+len(svc.Ports)*2)
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

	var envFlags strings.Builder
	keys := slices.Sorted(maps.Keys(svc.Environment))
	for _, k := range keys {
		envFlags.WriteString(fmt.Sprintf("-e %q ", fmt.Sprintf("%s=%s", k, svc.Environment[k])))
	}

	dependencies := "Wants=gcr-online.target docker.socket\n      After=gcr-online.target docker.socket"

	runcmds := []string{
		`echo 'DOCKER_OPTS="--registry-mirror=https://mirror.gcr.io"' | tee /etc/default/docker`,
		"systemctl daemon-reload",
		"systemctl restart docker",
	}

	sidecars := ""
	if addHealthCheckSidecar {
		sidecars, runcmds = buildSidecarUnits(serviceName, runcmds)
	}

	runcmds = append(runcmds,
		fmt.Sprintf("systemctl enable %s.service", serviceName),
		fmt.Sprintf("systemctl start %s.service", serviceName),
	)

	buf.WriteString(fmt.Sprintf(`
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
      ExecStart=/usr/bin/docker run --pull=always --rm --name=%[1]s %[3]s %[6]s%%s %[4]s
      ExecStop=/usr/bin/docker stop -t 30 %[1]s
      StandardOutput=journal+console
      StandardError=journal+console

      [Install]
      WantedBy=multi-user.target
%[7]s
runcmd:
  - %[8]s
`,
		serviceName,
		dependencies,
		strings.Join(params, " "),
		strings.Join(command, " "),
		region,
		envFlags.String(),
		sidecars,
		strings.Join(runcmds, "\n  - "),
	))

	return pulumi.Sprintf(buf.String(), image)
}

// buildSidecarUnits returns the cloud-init write_files entries and runcmd lines for the
// HTTP health check sidecar: a systemd socket that returns 200 when the container is
// running and 500 otherwise, allowing the MIG auto-healer to probe portless services.
func buildSidecarUnits(serviceName string, runcmds []string) (string, []string) {
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
        if docker inspect --format "{{.State.Status}}" %[1]s 2>/dev/null | grep -q running; then \
          echo -e "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"; \
        else \
          echo -e "HTTP/1.1 500 FAIL\r\nContent-Length: 4\r\n\r\nFAIL"; \
        fi;'
      StandardInput=socket
      StandardOutput=socket
`, serviceName)

	runcmds = append(runcmds,
		fmt.Sprintf("systemctl enable %s-health-firewall.service", serviceName),
		fmt.Sprintf("systemctl start %s-health-firewall.service", serviceName),
		fmt.Sprintf("systemctl enable %s-health.socket", serviceName),
		fmt.Sprintf("systemctl start %s-health.socket", serviceName),
	)
	return units, runcmds
}
