package defanggcp

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	providergcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Project is the controller struct for the defang-gcp:index:Project component.
type Project struct{}

// ProjectInputs defines the top-level inputs for the GCP Project component.
type ProjectInputs struct {
	// Services map: name -> service config
	Services compose.Services `pulumi:"services"          yaml:"services"`
	Networks compose.Networks `pulumi:"networks,optional" yaml:"networks,omitempty"`
	// Domain is the delegate domain for the project (e.g. "example.com"). When non-empty,
	// a wildcard certificate and DNS zone are created.
	Domain string `pulumi:"domain,optional" yaml:"domain,omitempty"`
}

// ProjectOutputs holds the outputs of the Project component.
type ProjectOutputs struct {
	pulumi.ResourceState

	// Per-service endpoint URLs (service name -> URL)
	Endpoints pulumi.StringMapOutput `pulumi:"endpoints"`

	// Load balancer DNS name (unused for GCP, kept for interface compat)
	LoadBalancerDNS pulumi.StringPtrOutput `pulumi:"loadBalancerDns,optional"`
}

// Construct implements the ComponentResource interface for Project.
func (*Project) Construct(
	ctx *pulumi.Context, name, typ string, inputs ProjectInputs, opts pulumi.ResourceOption,
) (*ProjectOutputs, error) {
	comp := &ProjectOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	args := common.BuildArgs{
		Services: inputs.Services,
		Domain:   inputs.Domain,
	}

	result, err := buildProject(ctx, name, args, childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP resources: %w", err)
	}

	comp.Endpoints = result.Endpoints
	comp.LoadBalancerDNS = result.LoadBalancerDNS

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoints":       result.Endpoints,
		"loadBalancerDns": result.LoadBalancerDNS,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}

// Build creates all GCP resources for the project.
// The GCP provider must be passed via the parent chain (pulumi.Providers on the parent component).
func buildProject(
	ctx *pulumi.Context,
	projectName string,
	args common.BuildArgs,
	parentOpt pulumi.ResourceOption,
) (*common.BuildResult, error) {
	childOpts := []pulumi.ResourceOption{parentOpt}

	infra, err := providergcp.BuildGlobalConfig(ctx, projectName, args.Domain, args.Services, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build GCP infrastructure: %w", err)
	}

	// Deploy each service, wrapped in a component resource for tree organization
	endpoints := pulumi.StringMap{}
	dependencies := map[string]pulumi.Resource{} // service name → component resource for dependees
	configProvider := providergcp.NewConfigProvider(projectName)
	var lbEntries []providergcp.LBServiceEntry

	if common.IsProjectUsingLLM(args.Services) {
		// FIXME: create dependency between this NewService and the services that need this API
		_, err := projects.NewService(ctx, projectName+"-defang-llm", &projects.ServiceArgs{
			Project: pulumi.StringPtr(infra.GcpProject),
			Service: pulumi.String("aiplatform.googleapis.com"),
		}, pulumi.RetainOnDelete(true)) // Do not try disabling on compose down
		if err != nil {
			return nil, err
		}
	}

	for _, svcName := range common.TopologicalSort(args.Services) {
		svc := args.Services[svcName]

		// Collect dependency resources from services this one depends on
		var deps []pulumi.Resource
		for dep := range svc.DependsOn {
			if r, ok := dependencies[dep]; ok {
				deps = append(deps, r)
			}
		}

		endpoint, svcComp, lbEntry, err := buildService(
			ctx, projectName, configProvider, svcName, svc, infra, deps, childOpts)
		if err != nil {
			return nil, err
		}
		endpoints[svcName] = endpoint
		dependencies[svcName] = svcComp
		if lbEntry != nil {
			lbEntries = append(lbEntries, *lbEntry)
		}
	}

	if err := providergcp.CreateExternalLoadBalancer(
		ctx, projectName, infra, lbEntries, childOpts...,
	); err != nil {
		return nil, fmt.Errorf("creating external load balancer: %w", err)
	}

	return &common.BuildResult{
		Endpoints:       endpoints.ToStringMapOutput(),
		LoadBalancerDNS: pulumi.StringPtr("").ToStringPtrOutput(),
	}, nil
}

func buildService(
	ctx *pulumi.Context,
	projectName string,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	infra *providergcp.GlobalConfig,
	deps []pulumi.Resource,
	childOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, pulumi.Resource, *providergcp.LBServiceEntry, error) {
	svcComp := &struct{ pulumi.ResourceState }{}

	var endpoint pulumi.StringOutput
	var lbEntry *providergcp.LBServiceEntry

	svcChildOpts := childOpts
	if len(deps) > 0 {
		svcChildOpts = append(svcChildOpts, pulumi.DependsOn(deps))
	}

	switch {
	case svc.Postgres != nil:
		// Managed Postgres → Cloud SQL
		if err := ctx.RegisterComponentResource("defang-gcp:index:Postgres", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Cloud SQL component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		sqlResult, err := providergcp.CreateCloudSQL(ctx, configProvider, svcName, svc, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating Cloud SQL for %s: %w", svcName, err)
		}
		if err := providergcp.CreatePrivateDNSRecord(
			ctx, svcName, sqlResult.Instance.PrivateIpAddress, infra.PrivateZoneId, svcOpts...,
		); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating private DNS for %s: %w", svcName, err)
		}
		port := firstIngressPort(svc.Ports, defaultPostgresPort)
		endpoint = pulumi.Sprintf("%s:%d", sqlResult.Instance.PublicIpAddress, port)
	case svc.Redis != nil:
		// Managed Redis → Memorystore
		if err := ctx.RegisterComponentResource("defang-gcp:index:Redis", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering Memorystore component %s: %w", svcName, err)
		}
		svcOpts := []pulumi.ResourceOption{pulumi.Parent(svcComp)}

		redisResult, err := providergcp.CreateMemoryStore(ctx, svcName, svc, infra, svcOpts...)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating Memorystore for %s: %w", svcName, err)
		}
		if err := providergcp.CreatePrivateDNSRecord(
			ctx, svcName, redisResult.Instance.Host, infra.PrivateZoneId, svcOpts...,
		); err != nil {
			return pulumi.StringOutput{}, nil, nil, fmt.Errorf("creating private DNS for %s: %w", svcName, err)
		}
		endpoint = pulumi.Sprintf("%s:%d", redisResult.Instance.Host, firstIngressPort(svc.Ports, defaultRedisPort))
	default:
		var err error
		endpoint, lbEntry, err = buildContainerService(
				ctx, projectName, configProvider, svcName, svc, infra, svcComp, svcChildOpts)
		if err != nil {
			return pulumi.StringOutput{}, nil, nil, err
		}
	}

	if err := ctx.RegisterResourceOutputs(svcComp, pulumi.Map{
		"endpoint": endpoint,
	}); err != nil {
		return pulumi.StringOutput{}, nil, nil, fmt.Errorf("registering outputs for %s: %w", svcName, err)
	}

	return endpoint, svcComp, lbEntry, nil
}

func buildContainerService(
	ctx *pulumi.Context,
	projectName string,
	configProvider compose.ConfigProvider,
	svcName string,
	svc compose.ServiceConfig,
	infra *providergcp.GlobalConfig,
	svcComp pulumi.Resource,
	svcChildOpts []pulumi.ResourceOption,
) (pulumi.StringOutput, *providergcp.LBServiceEntry, error) {
	sa, err := createServiceAccount(ctx, projectName, svcName, infra, svcChildOpts)
	if err != nil {
		return pulumi.StringOutput{}, nil, err
	}

	if svc.LLM != nil {
		if err := enableLLM(ctx, svcName, &svc, sa, infra, svcChildOpts); err != nil {
			return pulumi.StringOutput{}, nil, err
		}
	}

	if providergcp.IsCloudRunService(svc) {
		// Cloud Run: single ingress port
		if err := ctx.RegisterComponentResource("defang-gcp:index:Service", svcName, svcComp, svcChildOpts...); err != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("registering Cloud Run component %s: %w", svcName, err)
		}
		crResult, err := providergcp.CreateCloudRunService(ctx, configProvider, svcName, svc, infra, pulumi.Parent(svcComp))
		if err != nil {
			return pulumi.StringOutput{}, nil, fmt.Errorf("creating Cloud Run service %s: %w", svcName, err)
		}
		lbEntry := &providergcp.LBServiceEntry{Name: svcName, Service: crResult.Service, Config: svc}
		return crResult.Service.Uri, lbEntry, nil
	}

	// Compute Engine: portless workers or services with host-mode ports
	if err := ctx.RegisterComponentResource("defang-gcp:index:Service", svcName, svcComp, svcChildOpts...); err != nil {
		return pulumi.StringOutput{}, nil, fmt.Errorf("registering Compute Engine component %s: %w", svcName, err)
	}
	ceResult, err := providergcp.CreateComputeEngine(ctx, projectName, svcName, svc, infra, pulumi.Parent(svcComp))
	if err != nil {
		return pulumi.StringOutput{}, nil, fmt.Errorf("creating Compute Engine service %s: %w", svcName, err)
	}
	var lbEntry *providergcp.LBServiceEntry
	if svc.HasIngressPorts() {
		lbEntry = &providergcp.LBServiceEntry{Name: svcName, InstanceGroup: ceResult.InstanceGroup, Config: svc}
	}
	return infra.PublicIP.Address.ToStringOutput(), lbEntry, nil
}

func createServiceAccount(
	ctx *pulumi.Context,
	projectName,
	svcName string,
	infra *providergcp.GlobalConfig,
	svcChildOpts []pulumi.ResourceOption,
) (*serviceaccount.Account, error) {
	displayName := fmt.Sprintf("%v service %v stack %v Service Account", projectName, infra.Stack, svcName)
	description := fmt.Sprintf(
		"Service Account used by run services of %v project %v service in %v stack",
		projectName,
		svcName,
		infra.Stack,
	)
	// Create a service account for the services running in cloudrun or compute engine
	sa, err := serviceaccount.NewAccount(ctx, projectName+"-"+svcName+"-service-account", &serviceaccount.AccountArgs{
		AccountId:   pulumi.String(serviceAccountId(projectName, svcName, *infra)),
		DisplayName: pulumi.String(displayName),
		Description: pulumi.String(description),
	}, svcChildOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating service account %s: %w", svcName, err)
	}
	return sa, nil
}

func enableLLM(
	ctx *pulumi.Context,
	svcName string,
	svc *compose.ServiceConfig,
	sa *serviceaccount.Account,
	infra *providergcp.GlobalConfig,
	svcChildOpts []pulumi.ResourceOption,
) error {
	// TODO: add dependency to the member resource
	_, err := projects.NewIAMMember(ctx, svcName+"-defang-llm", &projects.IAMMemberArgs{
		Project: pulumi.String(infra.GcpProject),
		// for details see https://cloud.google.com/vertex-ai/docs/general/access-control
		Role:   pulumi.String("roles/aiplatform.user"),
		Member: pulumi.Sprintf("serviceAccount:%v", sa.Email),
	}, append(svcChildOpts,
		// prevent service account does not exist error when down, will be automatically removed when sa is removed
		pulumi.DeletedWith(sa),
		// membership is not a distinct resource, so we risk deleting the membership we are trying to create
		pulumi.DeleteBeforeReplace(true),
	)...,
	)
	if err != nil {
		return fmt.Errorf("failed to grant aiplatform access to service account %v: %w", sa.Email, err)
	}

	// Inject environment variables for Vercel routing for GCP Vertex AI access
	// https://ai-sdk.dev/providers/ai-sdk-providers/google-vertex
	if val, ok := svc.Environment["GOOGLE_VERTEX_PROJECT"]; !ok || val == "" {
		svc.Environment["GOOGLE_VERTEX_PROJECT"] = infra.GcpProject
	}

	if val, ok := svc.Environment["GOOGLE_VERTEX_LOCATION"]; !ok || val == "" {
		svc.Environment["GOOGLE_VERTEX_LOCATION"] = infra.Region
	}

	// Inject environment variables for Google ADK to have access to GCP Vertex AI
	if val, ok := svc.Environment["GOOGLE_CLOUD_PROJECT"]; !ok || val == "" {
		svc.Environment["GOOGLE_CLOUD_PROJECT"] = infra.GcpProject
	}

	if val, ok := svc.Environment["GOOGLE_CLOUD_LOCATION"]; !ok || val == "" {
		svc.Environment["GOOGLE_CLOUD_LOCATION"] = infra.Region
	}
	return nil
}

const pulumiSuffixLength = 8

// Service account ID must be between 6 and 30 characters.
// Service account ID must start with a lower case letter, followed by one or
// more lower case alphanumerical characters that can be separated by hyphens.
func serviceAccountId(project, service string, config providergcp.GlobalConfig) string {
	fullName := fullDefangResourceName(project, config, service)
	fullName = replaceNonAlphaNumericOrDash(fullName)
	return hashTrim(fullName, 30-pulumiSuffixLength)
}

func fullDefangResourceName(project string, config providergcp.GlobalConfig, names ...string) string {
	var parts []string
	if config.Prefix != "" {
		parts = append(parts, config.Prefix)
	}
	parts = append(parts, project, config.Stack)
	parts = append(parts, names...)
	return strings.Join(parts, "-")
}

var nonLowerAlphaNumericOrDashRe = regexp.MustCompile(`[^a-z0-9-]`)

func replaceNonAlphaNumericOrDash(name string) string {
	name = strings.ToLower(name)
	name = nonLowerAlphaNumericOrDashRe.ReplaceAllLiteralString(name, "-")
	return strings.TrimRight(name, "-")
}

func hashTrim(name string, maxLength int) string {
	if len(name) <= maxLength {
		return name
	}

	const hashLength = 6
	prefix := name[:maxLength-hashLength]
	suffix := name[maxLength-hashLength:]
	return prefix + hashn(suffix, hashLength)
}

func hashn(str string, length int) string {
	hash := sha256.New()
	hash.Write([]byte(str))
	hashInt := binary.LittleEndian.Uint64(hash.Sum(nil)[:8])
	hashBase36 := strconv.FormatUint(hashInt, 36) // base 36 string
	// truncate if the hash is too long
	if len(hashBase36) > length {
		return hashBase36[:length]
	}
	// if the hash is too short, pad with leading zeros
	return fmt.Sprintf("%0*s", length, hashBase36)
}
