package scaleway

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	scalewayconfig "github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/config"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway/containers"
)

var (
	ErrContainerImageMissing     = errors.New("Scaleway Serverless Containers require a pre-built image")
	ErrContainerNamespaceMissing = errors.New("Scaleway Serverless Containers require a namespace")
	ErrContainerUnsupported      = errors.New("unsupported Scaleway Serverless Container configuration")
)

type ContainerResult struct {
	Container *containers.Container
	Domain    *containers.Domain
	Endpoint  pulumi.StringOutput
}

func NewStandaloneInfra(ctx *pulumi.Context, projectName string) *SharedInfra {
	return &SharedInfra{
		Region:    scalewayconfig.GetRegion(ctx),
		Zone:      scalewayconfig.GetZone(ctx),
		ProjectID: scalewayconfig.GetProjectId(ctx),
	}
}

func CreateContainerNamespace(
	ctx *pulumi.Context,
	projectName string,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*containers.Namespace, error) {
	if infra == nil {
		infra = NewStandaloneInfra(ctx, projectName)
	}
	args := &containers.NamespaceArgs{
		Name: pulumi.StringPtr(projectName),
		Tags: pulumi.StringArray{
			pulumi.String("defang"),
			pulumi.String(projectName),
		},
	}
	if infra.Region != "" {
		args.Region = pulumi.StringPtr(infra.Region)
	}
	if infra.ProjectID != "" {
		args.ProjectId = pulumi.StringPtr(infra.ProjectID)
	}
	namespace, err := containers.NewNamespace(ctx, projectName+"-namespace", args, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway Serverless Containers namespace: %w", err)
	}
	return namespace, nil
}

func containerCPULimit(cpus float64) int {
	if cpus <= 0 {
		return 140
	}
	return int(math.Ceil(cpus * 1000))
}

func containerMemoryLimit(memMiB int) int {
	if memMiB <= 0 {
		return 256
	}
	return max(memMiB, 128)
}

func validateContainerResources(svc compose.ServiceConfig) error {
	cpu := containerCPULimit(svc.GetCPUs())
	if cpu < 70 || cpu > 6000 {
		return fmt.Errorf("%w: CPU limit %d mvCPU is outside Scaleway's documented 70-6000 mvCPU range", ErrContainerUnsupported, cpu)
	}
	mem := containerMemoryLimit(svc.GetMemoryMiB())
	if mem < 128 || mem > 12228 {
		return fmt.Errorf("%w: memory limit %d MB is outside Scaleway's documented 128-12228 MB range", ErrContainerUnsupported, mem)
	}
	replicas := svc.GetReplicas()
	if replicas < 1 || replicas > 50 {
		return fmt.Errorf("%w: max scale %d is outside Scaleway's documented 1-50 instance range", ErrContainerUnsupported, replicas)
	}
	return nil
}

func validateContainerPorts(svc compose.ServiceConfig) error {
	ingressPorts := 0
	for _, p := range svc.Ports {
		if p.IsHost() {
			return fmt.Errorf("%w: host-mode ports are not supported by Scaleway Serverless Containers", ErrContainerUnsupported)
		}
		switch p.Target {
		case 8008, 8012, 8013, 8022, 9090, 9091:
			return fmt.Errorf("%w: port %d is reserved by Scaleway Serverless Containers", ErrContainerUnsupported, p.Target)
		}
		if p.IsIngress() {
			ingressPorts++
		}
	}
	if ingressPorts > 1 {
		return fmt.Errorf("%w: Scaleway Serverless Containers expose exactly one public port", ErrContainerUnsupported)
	}
	return nil
}

func validateContainerEnvironment(svc compose.ServiceConfig) error {
	for k := range svc.Environment {
		if k == "PORT" || strings.HasPrefix(k, "SCW_") {
			return fmt.Errorf("%w: environment variable %q is reserved by Scaleway Serverless Containers", ErrContainerUnsupported, k)
		}
	}
	return nil
}

func validateContainerService(svc compose.ServiceConfig) error {
	if err := validateContainerResources(svc); err != nil {
		return err
	}
	if err := validateContainerPorts(svc); err != nil {
		return err
	}
	if err := validateContainerEnvironment(svc); err != nil {
		return err
	}
	if svc.LLM != nil {
		return fmt.Errorf("%w: LLM services are not implemented for Scaleway", ErrContainerUnsupported)
	}
	if platform := svc.GetPlatform(); platform != "linux/amd64" {
		return fmt.Errorf("%w: Scaleway Serverless Containers require linux/amd64 images, got %q", ErrContainerUnsupported, platform)
	}
	return nil
}

func containerProtocol(svc compose.ServiceConfig) string {
	if len(svc.Ports) == 0 {
		return "http1"
	}
	switch svc.Ports[0].GetAppProtocol() {
	case compose.PortAppProtocolHTTP2, compose.PortAppProtocolGRPC:
		return "h2c"
	default:
		return "http1"
	}
}

const defaultHealthShimPort = 8080

func containerPort(svc compose.ServiceConfig) pulumi.IntPtrInput {
	for _, p := range svc.Ports {
		if p.Target > 0 {
			return pulumi.IntPtr(int(p.Target))
		}
	}
	return nil
}

// needsHealthShim returns true when the service has no ports defined.
// Scaleway Serverless Containers require a listening HTTP port; background
// workers without ports need a small HTTP health responder injected.
func needsHealthShim(svc compose.ServiceConfig) bool {
	return containerPort(svc) == nil
}

// healthShimScript returns a shell script that starts a minimal HTTP health
// responder in the background on $PORT (defaulting to 8080), then execs the
// original command. The responder tries multiple runtimes in order of
// preference: node, python3, python. If none are available it falls back to
// a busybox-compatible shell+nc loop.
func healthShimScript(svc compose.ServiceConfig) string {
	// Build the original command to exec into. If the service has an
	// explicit entrypoint+command we use that; otherwise we fall back to
	// "exec" with whatever command was specified (the image ENTRYPOINT
	// handles the rest).
	var original string
	if len(svc.Entrypoint) > 0 {
		original = shellJoin(append(svc.Entrypoint, svc.Command...))
	} else if len(svc.Command) > 0 {
		original = shellJoin(svc.Command)
	} else {
		// No explicit command — we cannot wrap something we don't know.
		// Return empty to signal the caller should not inject the shim.
		return ""
	}

	// The health server candidates. Each is a self-contained one-liner
	// that listens on $PORT and responds 200 OK to any request.
	// We chain them with || so the first available runtime wins.
	return `(` +
		`node -e "require('http').createServer((_,r)=>{r.writeHead(200);r.end('OK')}).listen(process.env.PORT||8080)" 2>/dev/null || ` +
		`python3 -c "from http.server import HTTPServer,BaseHTTPRequestHandler;import os;HTTPServer(('0.0.0.0',int(os.environ.get('PORT',8080))),BaseHTTPRequestHandler).serve_forever()" 2>/dev/null || ` +
		`python -c "from http.server import HTTPServer,BaseHTTPRequestHandler;import os;HTTPServer(('0.0.0.0',int(os.environ.get('PORT',8080))),BaseHTTPRequestHandler).serve_forever()" 2>/dev/null || ` +
		`while true; do printf 'HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK' | nc -l -p ${PORT:-8080} -q 0 2>/dev/null || break; done` +
		`) & exec ` + original
}

// shellJoin quotes each argument for safe shell evaluation.
func shellJoin(args []string) string {
	if len(args) == 0 {
		return ""
	}
	var b strings.Builder
	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		// If the arg is safe (no spaces/special chars), use it as-is
		if !strings.ContainsAny(a, " \t\n\"'\\$;|&(){}[]<>?*~`!#") {
			b.WriteString(a)
		} else {
			b.WriteByte('\'')
			b.WriteString(strings.ReplaceAll(a, "'", "'\"'\"'"))
			b.WriteByte('\'')
		}
	}
	return b.String()
}

func containerPrivacy(svc compose.ServiceConfig) string {
	if svc.HasIngressPorts() {
		return "public"
	}
	return "private"
}

func containerMinScale(svc compose.ServiceConfig) pulumi.IntPtrInput {
	if svc.GetReplicas() <= 1 {
		return pulumi.IntPtr(0)
	}
	return pulumi.IntPtr(1)
}

func containerMaxScale(svc compose.ServiceConfig) pulumi.IntPtrInput {
	return pulumi.IntPtr(int(svc.GetReplicas()))
}

func containerHealthChecks(svc compose.ServiceConfig) containers.ContainerHealthCheckArrayInput {
	if svc.HealthCheck == nil || len(svc.HealthCheck.Test) == 0 {
		return nil
	}
	retries := svc.HealthCheck.Retries
	if retries <= 0 {
		retries = 3 // Scaleway requires failure_threshold; default to 3
	}
	interval := svc.HealthCheck.IntervalSeconds
	if interval <= 0 {
		interval = 10 // Scaleway requires interval; default to 10s
	}
	check := &containers.ContainerHealthCheckArgs{
		Https: containers.ContainerHealthCheckHttpArray{
			&containers.ContainerHealthCheckHttpArgs{
				Path: pulumi.String("/"),
			},
		},
		FailureThreshold: pulumi.Int(retries),
		Interval:         pulumi.String((time.Duration(interval) * time.Second).String()),
	}
	return containers.ContainerHealthCheckArray{check}
}

func containerEnvironment(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.InvokeOption,
) (pulumi.StringMap, pulumi.StringMap) {
	env := pulumi.StringMap{
		"DEFANG_SERVICE": pulumi.String(serviceName),
	}
	if infra != nil && infra.Etag != "" {
		env["DEFANG_ETAG"] = pulumi.String(infra.Etag)
	}
	secrets := pulumi.StringMap{}
	for k, v := range common.Sorted(svc.Environment) {
		if secretVar := compose.GetConfigName2(k, v); secretVar != "" && configProvider != nil {
			secrets[k] = configProvider.GetConfigValue(ctx, secretVar, opts...)
			continue
		}
		raw := ""
		if v != nil {
			raw = *v
		}
		// Replace env values that reference managed services:
		// - Values matching a managed service name (e.g., POSTGRES_HOST=database)
		//   are replaced with the actual hostname.
		// - POSTGRES_USER and POSTGRES_DB set to "postgres" are remapped to "defang"
		//   because Scaleway reserves the "postgres" name.
		if infra != nil && infra.ManagedHosts != nil {
			if managedHost, ok := infra.ManagedHosts[raw]; ok {
				env[k] = managedHost
				continue
			}
			if (k == "POSTGRES_USER" || k == "POSTGRES_DB") && strings.EqualFold(raw, "postgres") && len(infra.ManagedHosts) > 0 {
				env[k] = pulumi.String(defaultScalewayPostgresUser)
				continue
			}
			// Check if the value is a URL containing a managed service name as the hostname.
			if infra.ManagedConnectionURLs != nil {
				if svcName, ok := findManagedHostInURL(raw, infra.ManagedHosts); ok {
					if connURL, ok := infra.ManagedConnectionURLs[svcName]; ok {
						// Preserve query parameters from the original URL.
						if queryStart := strings.IndexByte(raw, '?'); queryStart >= 0 {
							extraParams := raw[queryStart+1:]
							env[k] = connURL.ApplyT(func(u string) string {
								if strings.Contains(u, "?") {
									return u + "&" + extraParams
								}
								return u + "?" + extraParams
							}).(pulumi.StringOutput)
						} else {
							env[k] = connURL
						}
						continue
					}
				}
			}
		}
		env[k] = compose.InterpolateEnvironmentVariable(ctx, configProvider, raw, opts...)
	}
	return env, secrets
}

// findManagedHostInURL checks whether a raw string looks like a URL whose
// hostname matches one of the managed service names.
func findManagedHostInURL(raw string, managedHosts map[string]pulumi.StringOutput) (string, bool) {
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd < 0 {
		return "", false
	}
	hostPart := raw[schemeEnd+3:]
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}
	if colon := strings.IndexByte(hostPart, ':'); colon >= 0 {
		hostPart = hostPart[:colon]
	} else if slash := strings.IndexByte(hostPart, '/'); slash >= 0 {
		hostPart = hostPart[:slash]
	}
	if _, ok := managedHosts[hostPart]; ok {
		return hostPart, true
	}
	return "", false
}

func CreateContainerService(
	ctx *pulumi.Context,
	configProvider compose.ConfigProvider,
	serviceName string,
	image pulumi.StringInput,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*ContainerResult, error) {
	if image == nil {
		return nil, ErrContainerImageMissing
	}
	if infra == nil || infra.Namespace == nil {
		return nil, ErrContainerNamespaceMissing
	}
	if err := validateContainerService(svc); err != nil {
		return nil, err
	}
	env, secrets := containerEnvironment(ctx, configProvider, serviceName, svc, infra)
	privacy := containerPrivacy(svc)

	port := containerPort(svc)
	commands := compose.ToPulumiStringArray(svc.Entrypoint)
	cmdArgs := compose.ToPulumiStringArray(svc.Command)

	// Scaleway Serverless Containers require a listening HTTP port. For
	// background workers with no ports we inject a minimal health shim
	// that starts a tiny HTTP responder on $PORT then execs the real command.
	if needsHealthShim(svc) {
		if script := healthShimScript(svc); script != "" {
			port = pulumi.IntPtr(defaultHealthShimPort)
			commands = pulumi.StringArray{pulumi.String("/bin/sh"), pulumi.String("-c")}
			cmdArgs = pulumi.StringArray{pulumi.String(script)}
		}
	}

	args := &containers.ContainerArgs{
		Name:                       pulumi.StringPtr(serviceName),
		NamespaceId:                infra.Namespace.ID(),
		RegistryImage:              image.ToStringOutput().ToStringPtrOutput(),
		Port:                       port,
		CpuLimit:                   pulumi.IntPtr(containerCPULimit(svc.GetCPUs())),
		MemoryLimit:                pulumi.IntPtr(containerMemoryLimit(svc.GetMemoryMiB())),
		MinScale:                   containerMinScale(svc),
		MaxScale:                   containerMaxScale(svc),
		Privacy:                    pulumi.StringPtr(privacy),
		Protocol:                   pulumi.StringPtr(containerProtocol(svc)),
		Deploy:                     pulumi.BoolPtr(true),
		Commands:                   commands,
		Args:                       cmdArgs,
		HealthChecks:               containerHealthChecks(svc),
		EnvironmentVariables:       env,
		SecretEnvironmentVariables: secrets,
	}
	if infra.Region != "" {
		args.Region = pulumi.StringPtr(infra.Region)
	}
	if infra.PrivateNetwork != nil {
		args.PrivateNetworkId = infra.PrivateNetwork.ID().ToStringOutput().ToStringPtrOutput()
	}

	container, err := containers.NewContainer(ctx, serviceName, args, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway Serverless Container: %w", err)
	}

	var domain *containers.Domain
	if svc.DomainName != "" {
		domain, err = containers.NewDomain(ctx, serviceName+"-domain", &containers.DomainArgs{
			ContainerId: container.ID(),
			Hostname:    pulumi.String(svc.DomainName),
			Region:      args.Region,
		}, append(opts, pulumi.Parent(container))...)
		if err != nil {
			return nil, fmt.Errorf("creating Scaleway Serverless Container domain: %w", err)
		}
	}

	// All container endpoints use the public HTTPS URL. Scaleway private networks
	// are egress-only for Serverless Containers: containers can reach databases/Redis
	// on the PN, but inbound private traffic between containers is not supported.
	// Container-to-container communication must use public endpoints.
	endpoint := pulumi.Sprintf("https://%s", container.DomainName)

	return &ContainerResult{Container: container, Domain: domain, Endpoint: endpoint}, nil
}
