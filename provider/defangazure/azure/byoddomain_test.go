package azure

import (
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDNSZoneID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantRG  string
		wantZon string
		wantOK  bool
	}{
		{
			name:    "standard",
			id:      "/subscriptions/abc/resourceGroups/my-rg/providers/Microsoft.Network/dnszones/example.com",
			wantRG:  "my-rg",
			wantZon: "example.com",
			wantOK:  true,
		},
		{
			name:    "mixed-case segments",
			id:      "/subscriptions/abc/resourcegroups/RG1/providers/microsoft.network/dnsZones/foo.io",
			wantRG:  "RG1",
			wantZon: "foo.io",
			wantOK:  true,
		},
		{name: "empty", id: "", wantOK: false},
		{name: "no resource group", id: "/subscriptions/abc/providers/Microsoft.Network/dnszones/example.com", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg, zone, ok := parseDNSZoneID(tt.id)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && (rg != tt.wantRG || zone != tt.wantZon) {
				t.Errorf("got rg=%q zone=%q, want rg=%q zone=%q", rg, zone, tt.wantRG, tt.wantZon)
			}
		})
	}
}

// relativeRecordName is covered by TestRelativeRecordName in frontdoor_test.go,
// which owns the helper.
func TestIsApexDomain(t *testing.T) {
	tests := []struct {
		domain, zone string
		want         bool
	}{
		{"example.com", "example.com", true},
		{"Example.COM", "example.com.", true}, // case- and trailing-dot-insensitive
		{"api.example.com", "example.com", false},
		{"example.com", "other.com", false},
	}
	for _, tt := range tests {
		if got := isApexDomain(tt.domain, tt.zone); got != tt.want {
			t.Errorf("isApexDomain(%q,%q) = %v, want %v", tt.domain, tt.zone, got, tt.want)
		}
	}
}

func TestByodRecordEligible(t *testing.T) {
	const zoneID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"
	tests := []struct {
		name         string
		domain, zone string
		want         bool
	}{
		{name: "subdomain", domain: "api.example.com", zone: zoneID, want: true},
		{name: "apex", domain: "example.com", zone: zoneID, want: true},
		{name: "not in zone", domain: "api.other.com", zone: zoneID, want: false},
		{name: "no domain", domain: "", zone: zoneID, want: false},
		{name: "no zone", domain: "api.example.com", zone: "", want: false},
		{name: "unparseable zone", domain: "api.example.com", zone: "garbage", want: false},
		// Container Apps has no wildcard binding, so a wildcard hostname's cert
		// comes from Front Door instead — enqueuing an ACA cert job for it would
		// wait out the full DNS timeout for records this path never writes.
		{name: "wildcard subdomain", domain: "*.api.example.com", zone: zoneID, want: false},
		{name: "wildcard at apex", domain: "*.example.com", zone: zoneID, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByodRecordEligible(tt.domain, tt.zone); got != tt.want {
				t.Errorf("ByodRecordEligible(%q,%q) = %v, want %v", tt.domain, tt.zone, got, tt.want)
			}
		})
	}
}

// TestCreateByodDomainShortCircuits checks the branches that create nothing and
// never touch the (nil) infra/app: no custom domain, no zone for the hostname, no
// public ingress, an unparseable zone id, a hostname outside the zone it was
// mapped to, and a wildcard (Front Door's job). The middle two warn-and-skip
// rather than failing the deploy.
func TestCreateByodDomainShortCircuits(t *testing.T) {
	ingress := []compose.ServicePortConfig{{Target: 80, Mode: "ingress"}}
	const zoneID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"

	apiSvc := compose.ServiceConfig{DomainName: "api.example.com", Ports: ingress}
	tests := []struct {
		name  string
		svc   compose.ServiceConfig
		zones map[string]string
	}{
		{
			name:  "no domain",
			svc:   compose.ServiceConfig{Ports: ingress},
			zones: map[string]string{"api.example.com": zoneID},
		},
		{name: "no zone for the hostname", svc: apiSvc, zones: nil},
		{
			name:  "hostname absent from a non-empty map",
			svc:   apiSvc,
			zones: map[string]string{"other.example.com": zoneID},
		},
		{
			name:  "no ingress",
			svc:   compose.ServiceConfig{DomainName: "api.example.com"},
			zones: map[string]string{"api.example.com": zoneID},
		},
		{
			name:  "unparseable zone",
			svc:   apiSvc,
			zones: map[string]string{"api.example.com": "garbage"},
		},
		{
			name:  "hostname not in the zone it maps to",
			svc:   compose.ServiceConfig{DomainName: "api.other.com", Ports: ingress},
			zones: map[string]string{"api.other.com": zoneID},
		},
		// A wildcard hostname is Front Door's to serve: writing a CNAME at "*"
		// aimed at the Container App would shadow the record Front Door needs and
		// answer 404 on every subdomain.
		{
			name:  "wildcard domainname",
			svc:   compose.ServiceConfig{DomainName: "*.api.example.com", Ports: ingress},
			zones: map[string]string{"*.api.example.com": zoneID},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := CreateByodDomain(ctx, "svc", tt.svc, nil, nil, tt.zones)
				if err != nil {
					t.Errorf("CreateByodDomain err: %v", err)
				}
				if len(got) != 0 {
					t.Errorf("CreateByodDomain result = %+v, want none", got)
				}
				return nil
			}, pulumi.WithMocks("project", "stack", azureNoopMocks{}))
			if err != nil {
				t.Fatalf("pulumi.RunErr: %v", err)
			}
		})
	}
}

// recordTypeMocks captures the recordType of each dns RecordSet by its relative
// name, so CreateByodDomain tests can assert which records (A vs CNAME, asuid
// TXT) were created.
type recordTypeMocks struct{ byRelative map[string]string }

func (m *recordTypeMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if strings.HasSuffix(args.TypeToken, ":RecordSet") {
		rel := args.Inputs["relativeRecordSetName"].StringValue()
		m.byRelative[rel] = args.Inputs["recordType"].StringValue()
	}
	return args.Name + "_id", args.Inputs, nil
}

func (recordTypeMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// TestCreateByodDomainRecords verifies the record types created for a subdomain
// (CNAME + asuid.<sub> TXT) vs an apex domain (A @ + asuid TXT).
func TestCreateByodDomainRecords(t *testing.T) {
	ingress := []compose.ServicePortConfig{{Target: 80, Mode: "ingress"}}
	const zoneID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"

	const otherZoneID = "/subscriptions/s/resourceGroups/rg2/providers/Microsoft.Network/dnszones/other.com"

	tests := []struct {
		name      string
		domain    string
		aliases   []string
		zones     map[string]string
		wantCount int
		wantTypes map[string]string // relative name -> record type
	}{
		{
			name: "subdomain", domain: "api.example.com",
			zones:     map[string]string{"api.example.com": zoneID},
			wantCount: 1,
			wantTypes: map[string]string{"api": "CNAME", "asuid.api": "TXT"},
		},
		{
			name: "apex", domain: "example.com",
			zones:     map[string]string{"example.com": zoneID},
			wantCount: 1,
			wantTypes: map[string]string{"@": "A", "asuid": "TXT"},
		},
		{
			// The reason the map is keyed by hostname: one service's hostnames can
			// live in different zones, and each gets records in its own.
			name: "alias in a different zone", domain: "api.example.com",
			aliases:   []string{"api.other.com"},
			zones:     map[string]string{"api.example.com": zoneID, "api.other.com": otherZoneID},
			wantCount: 2,
			wantTypes: map[string]string{"api": "CNAME", "asuid.api": "TXT"},
		},
		{
			// A wildcard alias is Front Door's; the plain hostname is still recorded.
			name: "wildcard alias is skipped", domain: "api.example.com",
			aliases:   []string{"*.api.example.com"},
			zones:     map[string]string{"api.example.com": zoneID, "*.api.example.com": zoneID},
			wantCount: 1,
			wantTypes: map[string]string{"api": "CNAME", "asuid.api": "TXT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &recordTypeMocks{byRelative: map[string]string{}}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				env, err := app.NewManagedEnvironment(ctx, "env", &app.ManagedEnvironmentArgs{
					ResourceGroupName: pulumi.String("rg"),
				})
				require.NoError(t, err)
				capp, err := app.NewContainerApp(ctx, "svc", &app.ContainerAppArgs{
					ResourceGroupName:    pulumi.String("rg"),
					ManagedEnvironmentId: env.ID().ToStringOutput(),
				})
				require.NoError(t, err)

				infra := &SharedInfra{Environment: env}
				svc := compose.ServiceConfig{DomainName: tt.domain, Ports: ingress}
				if len(tt.aliases) > 0 {
					svc.Networks = aliasNetwork(tt.aliases...)
				}
				got, err := CreateByodDomain(ctx, "svc", svc, capp, infra, tt.zones)
				require.NoError(t, err)
				require.Len(t, got, tt.wantCount)
				res := got[0]
				require.NotNil(t, res.Asuid)
				if tt.domain == "example.com" {
					assert.NotNil(t, res.A, "apex must create an A record")
					assert.Nil(t, res.Cname, "apex must not create a CNAME")
				} else {
					assert.NotNil(t, res.Cname, "subdomain must create a CNAME")
					assert.Nil(t, res.A, "subdomain must not create an A record")
				}
				return nil
			}, pulumi.WithMocks("project", "stack", m))
			require.NoError(t, err)
			assert.Equal(t, tt.wantTypes, m.byRelative)
		})
	}
}
