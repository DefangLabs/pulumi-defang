package gcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

var errInvalidDNSRecord = errors.New("invalid DNS record in wildcard cert authorization")

// GlobalConfig holds project-level GCP resources shared across all services.
type GlobalConfig struct {
	Region         string
	VpcId          pulumi.StringOutput
	SubnetId       pulumi.StringOutput
	PublicIP       *compute.GlobalAddress
	WildcardCertId pulumi.StringInput // non-nil when a domain is configured
	PublicZoneId   pulumi.StringInput // managed zone name; non-nil when a domain is configured
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

	subnet, err := compute.NewSubnetwork(ctx, projectName+"-shared-subnet", &compute.SubnetworkArgs{
		IpCidrRange: pulumi.String("10.0.0.0/16"),
		Region:      pulumi.String(region),
		Network:     vpc.ID(),
	}, append(opts, pulumi.RetainOnDelete(true))...)
	if err != nil {
		return nil, err
	}

	publicIP, err := compute.NewGlobalAddress(ctx, projectName+"-lb-ipaddress", &compute.GlobalAddressArgs{
		AddressType: pulumi.String("EXTERNAL"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	_, err = dns.NewManagedZone(ctx, projectName+"-private-dns", &dns.ManagedZoneArgs{
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
		Region:   region,
		VpcId:    vpc.ID().ToStringOutput(),
		SubnetId: subnet.ID().ToStringOutput(),
		PublicIP: publicIP,
	}

	if domain != "" {
		if err := createWildcardCert(ctx, projectName, domain, cfg, opts...); err != nil {
			return nil, err
		}
	}

	return cfg, nil
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
	if _, err := dns.NewRecordSet(ctx, fqdn+".-CAA-dns", &dns.RecordSetArgs{
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
		projectName+"-wildcardcert-dns-authz", authzArgs, opts...)
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
			name := fqdn + "-wildcardcert-dns-authz-record"
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

	certName := projectName + "-" + fqdn + "-wildcard-cert"
	cert, err := certificatemanager.NewCertificate(ctx, certName, &certificatemanager.CertificateArgs{
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

const defaultGCPRegion = "us-central1"

// GcpRegion reads the GCP region from Pulumi stack config, falling back to the default.
func GcpRegion(ctx *pulumi.Context) string {
	cfg := config.New(ctx, "gcp")
	if r := cfg.Get("region"); r != "" {
		return r
	}
	return defaultGCPRegion
}
