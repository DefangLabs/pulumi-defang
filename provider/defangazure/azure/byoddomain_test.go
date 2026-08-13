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

func TestRelativeRecordName(t *testing.T) {
	tests := []struct {
		domain, zone, want string
		wantOK             bool
	}{
		{"api.example.com", "example.com", "api", true},
		{"a.b.example.com", "example.com", "a.b", true},
		{"API.Example.com", "example.com", "API", true}, // case-insensitive suffix, original case preserved
		{"example.com", "example.com", "", false},       // apex has no relative name
		{"other.com", "example.com", "", false},         // not in zone
	}
	for _, tt := range tests {
		got, ok := relativeRecordName(tt.domain, tt.zone)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("relativeRecordName(%q,%q) = (%q,%v), want (%q,%v)", tt.domain, tt.zone, got, ok, tt.want, tt.wantOK)
		}
	}
}

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByodRecordEligible(tt.domain, tt.zone); got != tt.want {
				t.Errorf("ByodRecordEligible(%q,%q) = %v, want %v", tt.domain, tt.zone, got, tt.want)
			}
		})
	}
}

// TestCreateByodDomainShortCircuits checks the branches that return (nil, nil)
// without creating records or touching the (nil) infra/app: no custom domain,
// no zone id, no public ingress, an unparseable zone id, and a domain that isn't
// within the zone. The last two warn-and-skip (rather than failing the deploy).
func TestCreateByodDomainShortCircuits(t *testing.T) {
	ingress := []compose.ServicePortConfig{{Target: 80, Mode: "ingress"}}
	const zoneID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"

	tests := []struct {
		name   string
		svc    compose.ServiceConfig
		zoneID string
	}{
		{name: "no domain", svc: compose.ServiceConfig{Ports: ingress}, zoneID: zoneID},
		{name: "no zone id", svc: compose.ServiceConfig{DomainName: "api.example.com", Ports: ingress}, zoneID: ""},
		{name: "no ingress", svc: compose.ServiceConfig{DomainName: "api.example.com"}, zoneID: zoneID},
		{name: "unparseable zone", svc: compose.ServiceConfig{DomainName: "api.example.com", Ports: ingress}, zoneID: "garbage"},
		{name: "domain not in zone", svc: compose.ServiceConfig{DomainName: "api.other.com", Ports: ingress}, zoneID: zoneID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := CreateByodDomain(ctx, "svc", tt.svc, nil, nil, tt.zoneID)
				if err != nil {
					t.Errorf("CreateByodDomain err: %v", err)
				}
				if got != nil {
					t.Errorf("CreateByodDomain result = %+v, want nil", got)
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

	tests := []struct {
		name      string
		domain    string
		wantTypes map[string]string // relative name -> record type
	}{
		{name: "subdomain", domain: "api.example.com", wantTypes: map[string]string{"api": "CNAME", "asuid.api": "TXT"}},
		{name: "apex", domain: "example.com", wantTypes: map[string]string{"@": "A", "asuid": "TXT"}},
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
				res, err := CreateByodDomain(ctx, "svc", svc, capp, infra, zoneID)
				require.NoError(t, err)
				require.NotNil(t, res)
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
