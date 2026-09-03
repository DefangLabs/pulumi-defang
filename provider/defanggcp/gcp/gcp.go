package gcp

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
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

var (
	errAmbiguousDelegateZones = errors.New("ambiguous public DNS managed zones")
	errInvalidDNSRecord       = errors.New("invalid DNS record in wildcard cert authorization")
)

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
	ProjectName       string                                  // compose project name (defang-project log label)
	Repos             map[string]*artifactregistry.Repository // non-empty when services reference external registries
	// Etag is the deployment ID supplied by the CD program; empty for
	// standalone Service callers.
	Etag string
}

// EnableGcpAPIs enables the GCP APIs required by the project. Project is
// deliberately left unset on each Service: it falls back to the provider's
// configured project when omitted.
func EnableGcpAPIs(ctx *pulumi.Context, opts ...pulumi.ResourceOption) error {
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

	// DisableOnDestroy false, not RetainOnDelete. Both leave the API enabled after a `down`, and
	// both drop the resource from Pulumi state: RetainOnDelete by skipping the provider's delete
	// call outright, this by telling the provider not to disable. The difference is that this is
	// the provider's own switch for exactly this intent, which keeps RetainOnDelete reserved for
	// the one case that warrants it — a recipe deliberately keeping a non-defang resource.
	for _, api := range apis {
		if _, err := projects.NewService(ctx, api, &projects.ServiceArgs{
			Service:          pulumi.String(api),
			DisableOnDestroy: pulumi.Bool(false),
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
//
// parentOpt is the same parent option as opts[0], typed so it also satisfies
// pulumi.InvokeOption: the delegate-zone lookup is an invoke, and without the parent
// it would not inherit the stack's GCP provider (see createWildcardCert).
func BuildGlobalConfig(
	ctx *pulumi.Context,
	projectName string,
	domain string,
	services map[string]compose.ServiceConfig,
	parentOpt pulumi.ResourceOrInvokeOption,
	opts ...pulumi.ResourceOption,
) (*SharedInfra, error) {
	region := GcpRegion(ctx)
	gcpProject := config.GetProject(ctx)

	// Logical names below deliberately omit projectName, same as the firewalls and
	// private-dns zone further down: Pulumi's default resource ID already prefixes
	// it with <pulumi-project>-<stack>, which includes projectName, so repeating it
	// here risked exceeding GCP's 63-char resource ID limit.
	// Deliberately NOT RetainOnDelete, even though GCP holds the subnet's IP addresses for 1-2
	// hours after the last Cloud Run service using them is deleted (Direct VPC egress; see
	// buildVpcAccess in cloudrun.go), so a `down` inside that window fails to delete the subnet
	// and, behind it, the VPC. Retaining hid that failure but dropped both resources from the
	// Pulumi state, which left the VPC orphaned with nothing left to delete it: the project then
	// ran into its NETWORKS quota (issue #183). Letting the destroy fail is what puts the CLI's
	// cleanup tool (DefangLabs/defang#2157) in front of the user.
	vpc, err := compute.NewNetwork(ctx, "vpc", &compute.NetworkArgs{
		AutoCreateSubnetworks: pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return nil, err
	}

	subnet, err := compute.NewSubnetwork(ctx, "subnet", &compute.SubnetworkArgs{
		IpCidrRange: pulumi.String("10.0.0.0/16"),
		Region:      pulumi.String(region),
		Network:     vpc.ID(),
	}, opts...)
	if err != nil {
		return nil, err
	}

	publicIP, err := compute.NewGlobalAddress(ctx, "ip", &compute.GlobalAddressArgs{
		AddressType: pulumi.String("EXTERNAL"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Allow SSH ingress to all instances in the VPC (required for GCP Console SSH).
	// Logical name deliberately omits projectName: Pulumi's default resource ID
	// already prefixes it with <pulumi-project>-<stack>, which includes
	// projectName, so repeating it here risked exceeding GCP's 63-char resource
	// ID limit (observed: "...-allow-icmp-<hash>" at 64 chars, one over, while
	// the one-shorter "...-allow-ssh-<hash>" landed at exactly 63 and happened
	// to still pass). Same fix as the wildcard-cert name below.
	if _, err := compute.NewFirewall(ctx, "allow-ssh", &compute.FirewallArgs{
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
	if _, err := compute.NewFirewall(ctx, "allow-icmp", &compute.FirewallArgs{
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

	// Logical name deliberately omits projectName, same as the firewalls above and
	// public-dns below: Pulumi's default resource ID already prefixes it with
	// <pulumi-project>-<stack>, which includes projectName, so repeating it here
	// risked exceeding GCP's 63-char resource ID limit.
	privateZone, err := dns.NewManagedZone(ctx, "private-dns", &dns.ManagedZoneArgs{
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
		ProjectName: projectName,
		VpcId:       vpc.ID().ToStringOutput(),
		SubnetId:    subnet.ID().ToStringOutput(),
		PublicIP:    publicIP,
		PrivateZone: privateZone.Name.ToStringOutput(),
	}

	if domain != "" {
		cfg.Domain = domain
		if err := createWildcardCert(ctx, projectName, domain, cfg, parentOpt, opts...); err != nil {
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
		repos, err := createRemoteRepos(ctx, projectName, externalRegistries, opts...)
		if err != nil {
			return err
		}
		cfg.Repos = repos
	}

	if needsVpcPeering(services) {
		serviceConn, err := createVPCPeeringInfra(ctx, cfg.VpcId, opts...)
		if err != nil {
			return err
		}
		cfg.ServiceConnection = serviceConn
	}
	return nil
}

// DelegateZoneName returns the Cloud DNS managed-zone name the Defang CLI gives the
// public delegate zone it pre-creates, and delegates from the parent domain, before
// the CD task runs.
//
// This must stay byte-for-byte identical to the CLI's derivation in
// defang/src/pkg/cli/client/byoc/gcp/byoc.go PrepareDomainDelegation:
// "defang-" + dns.SafeLabel(delegateDomain), where the CLI's dns.SafeLabel is the
// same lowercase-and-dots-to-hyphens function as common.SafeLabel here.
//
// Note the legacy CD derives the same name through its own safeZoneName
// (defang-mvp pulumi/cd/gcp/gcpcd/up.go:405-408), which additionally strips
// non-alphanumerics and hash-truncates to 63 characters. The CLI does neither, so
// the two disagree for long or unusual delegate domains. The CLI creates the zone,
// so the CLI wins; findDelegateZone's dnsName fallback covers the divergence.
func DelegateZoneName(domain string) string {
	return "defang-" + common.SafeLabel(common.NormalizeDNS(domain))
}

type delegateZoneMode uint8

const (
	delegateZoneCreate delegateZoneMode = iota
	delegateZoneManaged
	delegateZoneExternal
)

const externalDelegateZoneLogicalName = "external-public-dns"

type delegateZoneSelection struct {
	name string
	mode delegateZoneMode
}

// findDelegateZone decides whether the project must keep managing its existing
// public-dns resource, read a zone owned outside this stack, or create public-dns.
//
// Ported from the legacy GCP CD's adopt-or-create block
// (defang-mvp pulumi/cd/gcp/gcpcd/tenant_stack.go:180-190), with one deliberate
// change: the legacy CD calls dns.LookupManagedZone (a get-by-name) and treats *any*
// error as "no zone, create one". Listing instead means a "not found" is an empty
// result rather than an error, so there is no 404 message to pattern-match — and,
// unlike the legacy behaviour, a permission or throttling failure surfaces as an
// error instead of silently creating the duplicate zone this function exists to
// prevent. Cf. the same fail-closed reasoning in defangaws/aws/route53.go:125-161.
func findDelegateZone(
	ctx *pulumi.Context,
	projectName string,
	fqdn string,
	parentOpt pulumi.ResourceOrInvokeOption,
) (delegateZoneSelection, error) {
	zones, err := dns.GetManagedZones(ctx, &dns.GetManagedZonesArgs{}, parentOpt)
	if err != nil {
		return delegateZoneSelection{}, fmt.Errorf(
			"cannot list Cloud DNS managed zones for %s; verify the CD identity has dns.managedZones.list "+
				"(included in roles/dns.admin): %w", common.NormalizeDNS(fqdn), err,
		)
	}

	return selectDelegateZone(
		projectName,
		fqdn,
		common.AutonamingPrefix(ctx, "public-dns"),
		zones.ManagedZones,
	)
}

func selectDelegateZone(
	projectName string,
	fqdn string,
	managedNamePrefix string,
	zones []dns.GetManagedZonesManagedZone,
) (delegateZoneSelection, error) {
	fqdn = common.NormalizeDNS(fqdn)
	cliName := DelegateZoneName(fqdn)
	managedDescription := fmt.Sprintf("Public DNS zone for %v", projectName)

	var matches []dns.GetManagedZonesManagedZone
	matchNames := make(map[string]struct{})
	var cliZone *dns.GetManagedZonesManagedZone
	managedZones := make(map[string]dns.GetManagedZonesManagedZone)
	for _, zone := range zones {
		// Private zones cannot answer a DNS-01 challenge or serve public records. Normalize
		// both values because DNS names are case-insensitive and callers vary on the final dot.
		if zone.Name == nil || *zone.Name == "" ||
			!strings.EqualFold(zone.Visibility, "public") ||
			common.NormalizeDNS(zone.DnsName) != fqdn {
			continue
		}
		nameKey := strings.ToLower(*zone.Name)
		if _, seen := matchNames[nameKey]; !seen {
			matches = append(matches, zone)
			matchNames[nameKey] = struct{}{}
		}
		if strings.EqualFold(*zone.Name, cliName) {
			candidate := zone
			cliZone = &candidate
		}
		if isStackManagedDelegateZone(zone, cliName, managedNamePrefix, managedDescription) {
			managedZones[nameKey] = zone
		}
	}

	if cliZone != nil {
		// The exact CLI name is a positive ownership/delegation signal. It wins over
		// unrelated same-dnsName zones. When that same physical zone also carries the
		// stack-managed description, it is the explicitly named public-dns created by
		// this provider and must remain managed. Only a distinct managed zone conflicts.
		cliNameKey := strings.ToLower(*cliZone.Name)
		if _, managed := managedZones[cliNameKey]; managed && len(managedZones) == 1 {
			return delegateZoneSelection{name: *cliZone.Name, mode: delegateZoneManaged}, nil
		}
		if len(managedZones) != 0 {
			return delegateZoneSelection{}, delegateZoneAmbiguityError(fqdn, cliName, matches,
				"both the CLI-created zone and a stack-managed public-dns zone exist; remove the unwanted duplicate "+
					"or migrate its Pulumi state explicitly")
		}
		return delegateZoneSelection{name: *cliZone.Name, mode: delegateZoneExternal}, nil
	}

	if len(managedZones) == 1 {
		// Keep registering the original managed resource under the same logical name.
		// Returning a GetManagedZone here would change the same URN into an external read,
		// relinquishing lifecycle ownership and potentially deleting/replacing the old zone.
		for _, zone := range managedZones {
			return delegateZoneSelection{name: *zone.Name, mode: delegateZoneManaged}, nil
		}
	}
	if len(managedZones) > 1 {
		return delegateZoneSelection{}, delegateZoneAmbiguityError(fqdn, cliName, matches,
			"more than one zone has this stack's public-dns ownership markers")
	}

	switch len(matches) {
	case 0:
		return delegateZoneSelection{mode: delegateZoneCreate}, nil
	case 1:
		// A single public zone is unambiguous even if it predates the CLI naming rule.
		return delegateZoneSelection{name: *matches[0].Name, mode: delegateZoneExternal}, nil
	default:
		return delegateZoneSelection{}, delegateZoneAmbiguityError(fqdn, cliName, matches,
			"none has the exact CLI name or this stack's public-dns ownership markers")
	}
}

func isStackManagedDelegateZone(
	zone dns.GetManagedZonesManagedZone,
	cliName string,
	managedNamePrefix string,
	managedDescription string,
) bool {
	if zone.Name == nil || zone.Description != managedDescription {
		return false
	}
	name := strings.ToLower(*zone.Name)
	// Newly created zones have an explicit CLI-compatible name, which is independent
	// of Pulumi's current autonaming configuration. The stable managed description
	// distinguishes them from zones pre-created by the CLI.
	if strings.EqualFold(name, cliName) {
		return true
	}

	// Older stack-owned zones were auto-named. Preserve their managed registration
	// when their physical name still matches the active autonaming convention.
	prefix := strings.ToLower(strings.TrimSuffix(managedNamePrefix, "-"))
	return prefix != "" && (name == prefix || strings.HasPrefix(name, prefix+"-"))
}

func delegateZoneAmbiguityError(
	fqdn string,
	cliName string,
	zones []dns.GetManagedZonesManagedZone,
	reason string,
) error {
	names := make([]string, 0, len(zones))
	for _, zone := range zones {
		if zone.Name != nil {
			names = append(names, *zone.Name)
		}
	}
	sort.Strings(names)
	return fmt.Errorf(
		"%w: multiple public Cloud DNS managed zones serve %s (%s), and %s; refusing to choose one: "+
			"keep the delegated zone named %q and remove the duplicates, or migrate ownership explicitly",
		errAmbiguousDelegateZones, fqdn, strings.Join(names, ", "), reason, cliName,
	)
}

func ensureDelegateZone(
	ctx *pulumi.Context,
	projectName string,
	fqdn string,
	parentOpt pulumi.ResourceOrInvokeOption,
	opts ...pulumi.ResourceOption,
) (*dns.ManagedZone, error) {
	selection, err := findDelegateZone(ctx, projectName, fqdn, parentOpt)
	if err != nil {
		return nil, err
	}

	if selection.mode == delegateZoneExternal {
		// Use a different logical name from the managed public-dns resource. A Pulumi
		// get/read is external to this stack's lifecycle; sharing the old logical name
		// would instead convert an existing managed URN into an external one.
		zone, err := dns.GetManagedZone(
			ctx, externalDelegateZoneLogicalName, pulumi.ID(selection.name), nil, opts...,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"cannot read Cloud DNS managed zone %q for %s; verify the CD identity has dns.managedZones.get "+
					"(included in roles/dns.admin): %w", selection.name, fqdn, err,
			)
		}
		return zone, nil
	}

	zoneArgs := &dns.ManagedZoneArgs{
		Description: pulumi.String(fmt.Sprintf("Public DNS zone for %v", projectName)),
		DnsName:     pulumi.String(fqdn + "."),
	}
	if selection.mode == delegateZoneCreate {
		// The CLI only looks up this exact physical name. An auto-named zone would be
		// invisible to it and the CLI would create and delegate a duplicate next time.
		zoneArgs.Name = pulumi.String(DelegateZoneName(fqdn))
	}
	// The managed path intentionally keeps the original declaration's Name omitted.
	// Existing auto-named stacks therefore retain the same managed URN and physical
	// zone; adding the now-explicit name there would schedule an immutable replacement.
	zone, err := dns.NewManagedZone(ctx, "public-dns", zoneArgs, opts...)
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// createWildcardCert adopts or creates the public delegate DNS zone, then adds a
// wildcard DNS authorization and wildcard certificate for the given domain,
// populating cfg.PublicZoneId and cfg.WildcardCertId.
func createWildcardCert(
	ctx *pulumi.Context,
	projectName string,
	domain string,
	cfg *SharedInfra,
	parentOpt pulumi.ResourceOrInvokeOption,
	opts ...pulumi.ResourceOption,
) error {
	fqdn := common.NormalizeDNS(domain)

	zone, err := ensureDelegateZone(ctx, projectName, fqdn, parentOpt, opts...)
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
