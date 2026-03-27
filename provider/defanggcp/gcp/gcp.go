package gcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/servicenetworking"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

var errInvalidDNSRecord = errors.New("invalid DNS record in wildcard cert authorization")

// GlobalConfig holds project-level GCP resources shared across all services.
type GlobalConfig struct {
	Region            string
	GcpProject        string                          // GCP project ID, used for IAM bindings
	Domain            string                          // delegate domain (e.g. "example.com"); empty when not configured
	VpcId             pulumi.StringOutput
	SubnetId          pulumi.StringOutput
	PublicIP          *compute.GlobalAddress
	WildcardCertId    pulumi.StringInput              // non-nil when a domain is configured
	PublicZoneId      pulumi.StringInput              // managed zone name; non-nil when a domain is configured
	BuildInfra        *BuildInfra                     // non-nil when at least one service has a build config
	ServiceConnection *servicenetworking.Connection   // non-nil when any service uses managed Postgres or Redis
	PrivateZoneId     pulumi.StringOutput             // managed zone name for the private google.internal. zone
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
) (*GlobalConfig, error) {
	region := GcpRegion(ctx)

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

	cfg := &GlobalConfig{
		Region:        region,
		GcpProject:    gcpProjectId(ctx),
		VpcId:         vpc.ID().ToStringOutput(),
		SubnetId:      subnet.ID().ToStringOutput(),
		PublicIP:      publicIP,
		PrivateZoneId: privateZone.Name.ToStringOutput(),
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
	cfg *GlobalConfig,
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

	if hasPostgresConfig(services) || hasRedisConfig(services) {
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
	cfg *GlobalConfig,
	opts ...pulumi.ResourceOption,
) error {
	fqdn := strings.TrimSuffix(domain, ".")

	zone, err := dns.NewManagedZone(ctx, projectName+"-public-dns", &dns.ManagedZoneArgs{
		Description: pulumi.String(fmt.Sprintf("Public DNS zone for %v", projectName)),
		DnsName:     pulumi.String(fqdn + "."),
	}, opts...)
	if err != nil {
		return err
	}
	cfg.PublicZoneId = zone.Name

	// CAA record authorizes pki.goog (GCP Certificate Manager) and letsencrypt.org as valid CAs.
	if _, err := dns.NewRecordSet(ctx, projectName+"-caa", &dns.RecordSetArgs{
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
	certAuthz, err := certificatemanager.NewDnsAuthorization(ctx,
		projectName+"-cert-authz", authzArgs, opts...)
	if err != nil {
		return err
	}

	// Create the DNS record for the authorization challenge. The record data is only
	// available as an output, so we create it inside ApplyT (same pattern as the cd).
	type dnsRecord = certificatemanager.DnsAuthorizationDnsResourceRecord
	certAuthz.DnsResourceRecords.ApplyT(func(records []dnsRecord) ([]*dns.RecordSet, error) {
		var rs []*dns.RecordSet
		for i, record := range records {
			if record.Name == nil || record.Type == nil || record.Data == nil {
				return nil, fmt.Errorf("%w: invalid DNS record for %s: %v",
					errInvalidDNSRecord, fqdn, record)
			}
			name := projectName + "-cert-authz-record"
			if i > 0 {
				name = fmt.Sprintf("%s-%d", name, i)
			}
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
	cert, err := certificatemanager.NewCertificate(ctx, projectName+"-wildcard-cert", &certificatemanager.CertificateArgs{
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
	_, err := dns.NewRecordSet(ctx, domain+"-"+recordType+"-dns", &dns.RecordSetArgs{
		Name:        pulumi.String(domain),
		Type:        pulumi.String(recordType),
		Ttl:         ttl,
		ManagedZone: zoneName,
		Rrdatas:     value,
	}, opts...)
	return err
}

// CreatePrivateDNSRecord creates an A record in the private google.internal. zone for a
// managed service (Cloud SQL or Memorystore), so other services can reach it by name.
func CreatePrivateDNSRecord(
	ctx *pulumi.Context,
	serviceName string,
	ip pulumi.StringInput,
	zoneId pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) error {
	_, err := dns.NewRecordSet(ctx, serviceName+"-private-dns", &dns.RecordSetArgs{
		Name:        pulumi.String(serviceName + ".google.internal."),
		Type:        pulumi.String("A"),
		Ttl:         pulumi.Int(60),
		ManagedZone: zoneId,
		Rrdatas:     pulumi.StringArray{ip},
	}, opts...)
	return err
}

const defaultGCPRegion = "us-central1"

// GcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func GcpRegion(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	if r := cfg.Get("region"); r != "" {
		return r
	}
	return defaultGCPRegion
}
