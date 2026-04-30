package gcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/config"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/servicenetworking"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var errInvalidDNSRecord = errors.New("invalid DNS record in wildcard cert authorization")

// SharedInfra holds project-level GCP resources shared across all services.
type SharedInfra struct {
	Region            string
	GcpProject        string // GCP project ID, used for IAM bindings
	Domain            string // delegate domain (e.g. "example.com"); empty when not configured
	VpcId             pulumi.StringOutput
	SubnetId          pulumi.StringOutput
	PublicIP          *compute.GlobalAddress
	WildcardCertId    pulumi.StringInput // non-nil when a domain is configured
	PublicZoneId      pulumi.StringInput // managed zone name; non-nil when a domain is configured
	ProxySubnetId     string
	BuildInfra        *BuildInfra                             // non-nil when at least one service has a build config
	ServiceConnection *servicenetworking.Connection           // non-nil when any service uses managed Postgres or Redis
	PrivateZone       pulumi.StringOutput                     // managed zone name for the private google.internal. zone
	Prefix            string                                  // prefix for all resource names (e.g. "myproject")
	Stack             string                                  // Pulumi stack name (e.g. "dev")
	Repos             map[string]*artifactregistry.Repository // non-empty when services reference external registries
	// Etag is the deployment ID supplied by the CD program; empty for
	// standalone Service callers.
	Etag string
}

// EnableGcpAPIs enables the GCP APIs required by the project.
func EnableGcpAPIs(ctx *pulumi.Context, gcpProject string, opts ...pulumi.ResourceOption) error {
	apis := []string{
		"storage.googleapis.com",              // Cloud Storage API
		"artifactregistry.googleapis.com",     // Artifact Registry API
		"run.googleapis.com",                  // Cloud Run API
		"iam.googleapis.com",                  // IAM API
		"cloudresourcemanager.googleapis.com", // For service account and role management
		"cloudbuild.googleapis.com",           // For building images using cloud build
		"compute.googleapis.com",              // For load balancer
		"dns.googleapis.com",                  // For DNS
		"secretmanager.googleapis.com",        // For config/secrets
		"sqladmin.googleapis.com",             // For Cloud SQL
		"servicenetworking.googleapis.com",    // For VPC peering
		"redis.googleapis.com",                // For Redis
		"certificatemanager.googleapis.com",   // For SSL certs
		"firestore.googleapis.com",            // For Firestore MongoDB
	}

	opts = append(opts, pulumi.RetainOnDelete(true))
	for _, api := range apis {
		if _, err := projects.NewService(ctx, api, &projects.ServiceArgs{
			Project: pulumi.String(gcpProject),
			Service: pulumi.String(api),
		}, opts...); err != nil {
			return fmt.Errorf("failed to enable API %s: %w", api, err)
		}
	}
	return nil
}

// NewStandaloneGlobalConfig returns a minimal GlobalConfig for standalone component
// use: just Region and GcpProject read from Pulumi stack config. PublicIP is left nil
// so VPC-dependent features (Cloud Run VpcAccess, Compute Engine networking) are
// skipped or fail fast. Callers that need a full VPC/NAT/build-repo setup must use
// BuildGlobalConfig instead.
func NewStandaloneGlobalConfig(ctx *pulumi.Context) *SharedInfra {
	return &SharedInfra{
		Region:     GcpRegion(ctx),
		GcpProject: config.GetProject(ctx),
		Stack:      ctx.Stack(),
	}
}

// BuildGlobalConfig creates shared GCP infrastructure for a multi-service project.
// domain is the delegate domain for the project (e.g. "example.com"). When non-empty,
// a public DNS managed zone, a wildcard DNS authorization, and a wildcard certificate
// are created for that domain.
func BuildGlobalConfig(
	ctx *pulumi.Context,
	projectName string,
	domain string,
	services map[string]compose.ServiceConfig,
	opts ...pulumi.ResourceOption,
) (*SharedInfra, error) {
	region := GcpRegion(ctx)
	gcpProject := config.GetProject(ctx)

	vpc, err := compute.NewNetwork(ctx, projectName+"-vpc", &compute.NetworkArgs{
		AutoCreateSubnetworks: pulumi.Bool(false),
	}, append(opts, pulumi.RetainOnDelete(true))...)
	if err != nil {
		return nil, err
	}

	subnet, err := compute.NewSubnetwork(ctx, projectName+"-subnet", &compute.SubnetworkArgs{
		IpCidrRange: pulumi.String("10.0.0.0/16"),
		Region:      pulumi.String(region),
		Network:     vpc.ID(),
	}, append(opts, pulumi.RetainOnDelete(true))...)
	if err != nil {
		return nil, err
	}

	publicIP, err := compute.NewGlobalAddress(ctx, projectName+"-ip", &compute.GlobalAddressArgs{
		AddressType: pulumi.String("EXTERNAL"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Allow SSH ingress to all instances in the VPC (required for GCP Console SSH).
	if _, err := compute.NewFirewall(ctx, projectName+"-allow-ssh", &compute.FirewallArgs{
		Network:      vpc.ID(),
		SourceRanges: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		Allows: compute.FirewallAllowArray{
			&compute.FirewallAllowArgs{
				Protocol: pulumi.String("tcp"),
				Ports:    pulumi.StringArray{pulumi.String("22")},
			},
		},
		Direction: pulumi.String("INGRESS"),
	}, opts...); err != nil {
		return nil, err
	}

	// Allow ICMP ping to all instances in the VPC.
	if _, err := compute.NewFirewall(ctx, projectName+"-allow-icmp", &compute.FirewallArgs{
		Network:      vpc.ID(),
		SourceRanges: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		Allows: compute.FirewallAllowArray{
			&compute.FirewallAllowArgs{
				Protocol: pulumi.String("icmp"),
			},
		},
		Direction: pulumi.String("INGRESS"),
	}, opts...); err != nil {
		return nil, err
	}

	privateZone, err := dns.NewManagedZone(ctx, projectName+"-private-dns", &dns.ManagedZoneArgs{
		Description: pulumi.String(fmt.Sprintf("Private DNS zone for %v", projectName)),
		DnsName:     pulumi.String("google.internal."),
		Visibility:  pulumi.String("private"),
		PrivateVisibilityConfig: &dns.ManagedZonePrivateVisibilityConfigArgs{
			Networks: dns.ManagedZonePrivateVisibilityConfigNetworkArray{
				&dns.ManagedZonePrivateVisibilityConfigNetworkArgs{
					NetworkUrl: vpc.ID(),
				},
			},
		},
	}, append(opts, pulumi.ReplaceOnChanges([]string{"forwardingConfig"}), pulumi.DeleteBeforeReplace(true))...)
	if err != nil {
		return nil, err
	}

	cfg := &SharedInfra{
		Stack:       ctx.Stack(),
		Region:      region,
		GcpProject:  gcpProject,
		VpcId:       vpc.ID().ToStringOutput(),
		SubnetId:    subnet.ID().ToStringOutput(),
		PublicIP:    publicIP,
		PrivateZone: privateZone.Name.ToStringOutput(),
	}

	if domain != "" {
		cfg.Domain = domain
		if err := createWildcardCert(ctx, projectName, domain, cfg, opts...); err != nil {
			return nil, err
		}
	}

	if err := buildOptionalInfra(ctx, projectName, cfg, services, opts...); err != nil {
		return nil, err
	}

	return cfg, nil
}

// buildOptionalInfra creates build and database infrastructure when services require it.
func buildOptionalInfra(
	ctx *pulumi.Context,
	projectName string,
	cfg *SharedInfra,
	services map[string]compose.ServiceConfig,
	opts ...pulumi.ResourceOption,
) error {
	if hasBuildConfig(services) {
		buildInfra, err := createBuildInfra(ctx, projectName, opts...)
		if err != nil {
			return err
		}
		cfg.BuildInfra = buildInfra
	}

	if externalRegistries := collectExternalRegistries(services); len(externalRegistries) > 0 {
		repos, err := createRemoteRepos(ctx, externalRegistries, opts...)
		if err != nil {
			return err
		}
		cfg.Repos = repos
	}

	if needsVpcPeering(services) {
		serviceConn, err := createVPCPeeringInfra(ctx, projectName, cfg.VpcId, opts...)
		if err != nil {
			return err
		}
		cfg.ServiceConnection = serviceConn
	}
	return nil
}

// createWildcardCert creates a public DNS zone, a wildcard DNS authorization, and a
// wildcard certificate for the given domain, populating cfg.PublicZoneId and
// cfg.WildcardCertId.
func createWildcardCert(
	ctx *pulumi.Context,
	projectName string,
	domain string,
	cfg *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	fqdn := strings.TrimSuffix(domain, ".")

	zone, err := dns.NewManagedZone(ctx, "public-dns", &dns.ManagedZoneArgs{
		Description: pulumi.String(fmt.Sprintf("Public DNS zone for %v", projectName)),
		DnsName:     pulumi.String(fqdn + "."),
	}, opts...)
	if err != nil {
		return err
	}
	cfg.PublicZoneId = zone.Name

	// CAA record authorizes pki.goog (GCP Certificate Manager) and letsencrypt.org as valid CAs.
	if _, err := dns.NewRecordSet(ctx, "caa", &dns.RecordSetArgs{
		ManagedZone: zone.Name,
		Name:        pulumi.String(fqdn + "."),
		Type:        pulumi.String("CAA"),
		Ttl:         pulumi.Int(3600),
		Rrdatas: pulumi.StringArray{
			pulumi.String(`0 issue "pki.goog"`),
			pulumi.String(`0 issue "letsencrypt.org"`),
		},
	}, opts...); err != nil {
		return err
	}

	authzArgs := &certificatemanager.DnsAuthorizationArgs{
		Description: pulumi.StringPtr("Wildcard DNS authorization for " + fqdn),
		Domain:      pulumi.String(fqdn),
	}
	certAuthz, err := certificatemanager.NewDnsAuthorization(ctx, "cert-authz", authzArgs, opts...)
	if err != nil {
		return err
	}

	// Create the DNS record for the authorization challenge. The record data is only
	// available as an output, so we create it inside ApplyT (same pattern as the cd).
	type dnsRecord = certificatemanager.DnsAuthorizationDnsResourceRecord
	certAuthz.DnsResourceRecords.ApplyT(func(records []dnsRecord) ([]*dns.RecordSet, error) {
		var rs []*dns.RecordSet
		for _, record := range records {
			if record.Name == nil || record.Type == nil || record.Data == nil {
				return nil, fmt.Errorf("%w: invalid DNS record for %s: %v",
					errInvalidDNSRecord, fqdn, record)
			}
			name := *record.Name + "_" + *record.Type
			// TODO: avoid creating Pulumi resources within ApplyT
			r, err := dns.NewRecordSet(ctx, name, &dns.RecordSetArgs{
				ManagedZone: zone.Name,
				Name:        pulumi.String(*record.Name),
				Type:        pulumi.String(*record.Type),
				Ttl:         pulumi.Int(60),
				Rrdatas:     pulumi.ToStringArray([]string{*record.Data}),
			}, opts...)
			if err != nil {
				return nil, err
			}
			rs = append(rs, r)
		}
		return rs, nil
	})

	// Use a short name without the FQDN to stay within GCP's 63-char resource ID limit.
	cert, err := certificatemanager.NewCertificate(ctx, "wildcard-cert", &certificatemanager.CertificateArgs{
		Managed: &certificatemanager.CertificateManagedArgs{
			Domains:           pulumi.StringArray{pulumi.String("*." + fqdn)},
			DnsAuthorizations: pulumi.StringArray{certAuthz.ID()},
		},
	}, opts...)
	if err != nil {
		return err
	}
	cfg.WildcardCertId = cert.ID()

	return nil
}

// CreatePublicDNSRecord creates a DNS record in the public zone. The resource name
// follows the CD's convention: "{fqdn}-{type}-dns" (e.g. "app.example.com.-A-dns").
func CreatePublicDNSRecord(
	ctx *pulumi.Context,
	zoneName pulumi.StringInput,
	domain, recordType string,
	ttl pulumi.IntInput,
	value pulumi.StringArrayInput,
	opts ...pulumi.ResourceOption,
) error {
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}
	_, err := dns.NewRecordSet(ctx, domain+"_"+recordType, &dns.RecordSetArgs{
		Name:        pulumi.String(domain),
		Type:        pulumi.String(recordType),
		Ttl:         ttl,
		ManagedZone: zoneName,
		Rrdatas:     value,
	}, opts...)
	return err
}

const defaultGCPRegion = "us-central1"

// GcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func GcpRegion(ctx *pulumi.Context) string {
	if r := config.GetRegion(ctx); r != "" {
		return r
	}
	return defaultGCPRegion
}
