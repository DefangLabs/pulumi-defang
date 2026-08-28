package gcp

import (
	"errors"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const getManagedZonesToken = "gcp:dns/getManagedZones:getManagedZones"

var errDNSPermissionDenied = errors.New("googleapi: Error 403: Permission denied on resource")

// TestDelegateZoneName pins the zone name against the Defang CLI's own derivation
// in PrepareDomainDelegation ("defang-" + dns.SafeLabel(delegateDomain), defang
// src/pkg/cli/client/byoc/gcp/byoc.go). If this drifts, the CD stops recognising
// the zone the CLI created and delegated, and silently creates a duplicate.
func TestDelegateZoneName(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{"delegate domain", "myproject.tenant.defang.app", "defang-myproject-tenant-defang-app"},
		{"apex", "example.com", "defang-example-com"},
		{"uppercase is lowered", "Example.COM", "defang-example-com"},
		{"trailing dot is dropped", "example.com.", "defang-example-com"},
		{"hyphens are preserved", "my-app.tenant.defang.app", "defang-my-app-tenant-defang-app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DelegateZoneName(tt.domain))
		})
	}
}

// zoneListMocks answers the getManagedZones invoke with a fixed set of zones, or
// with an error when listErr is set.
type zoneListMocks struct {
	zones   []map[string]any
	listErr error
	calls   int
}

func (*zoneListMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (m *zoneListMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != getManagedZonesToken {
		return args.Args, nil
	}
	m.calls++
	if m.listErr != nil {
		return nil, m.listErr
	}
	zones := make([]resource.PropertyValue, 0, len(m.zones))
	for _, z := range m.zones {
		zones = append(zones, resource.NewObjectProperty(resource.NewPropertyMapFromMap(z)))
	}
	return resource.PropertyMap{
		"managedZones": resource.NewArrayProperty(zones),
	}, nil
}

func zone(name, dnsName, visibility string) map[string]any {
	return map[string]any{"name": name, "dnsName": dnsName, "visibility": visibility}
}

func TestFindDelegateZone(t *testing.T) {
	const fqdn = "myproject.tenant.defang.app"
	const cliZone = "defang-myproject-tenant-defang-app"

	tests := []struct {
		name  string
		zones []map[string]any
		want  string
	}{
		{
			name:  "no zones at all",
			zones: nil,
			want:  "",
		},
		{
			name:  "the CLI-created zone is adopted",
			zones: []map[string]any{zone(cliZone, fqdn+".", "public")},
			want:  cliZone,
		},
		{
			name: "unrelated zones are ignored",
			zones: []map[string]any{
				zone("defang-other-defang-app", "other.defang.app.", "public"),
				zone("customer-apex", "example.com.", "public"),
			},
			want: "",
		},
		{
			name: "the google.internal private zone is never adopted",
			zones: []map[string]any{
				zone("private-dns", "google.internal.", "private"),
			},
			want: "",
		},
		{
			name: "a private zone for the delegate domain is not the delegate zone",
			zones: []map[string]any{
				zone("private-mirror", fqdn+".", "private"),
			},
			want: "",
		},
		{
			name: "a zone created under another name is still adopted by dnsName",
			zones: []map[string]any{
				zone("public-dns-a1b2c3d", fqdn+".", "public"),
			},
			want: "public-dns-a1b2c3d",
		},
		{
			name: "the CLI name wins over another zone with the same dnsName",
			zones: []map[string]any{
				zone("aaa-duplicate", fqdn+".", "public"),
				zone(cliZone, fqdn+".", "public"),
			},
			want: cliZone,
		},
		{
			name: "duplicates without the CLI name resolve deterministically",
			zones: []map[string]any{
				zone("zzz-duplicate", fqdn+".", "public"),
				zone("aaa-duplicate", fqdn+".", "public"),
			},
			want: "aaa-duplicate",
		},
		{
			name: "a parent zone is not a match for the delegate domain",
			zones: []map[string]any{
				zone("defang-tenant-defang-app", "tenant.defang.app.", "public"),
			},
			want: "",
		},
		{
			name:  "a zone with no name is skipped",
			zones: []map[string]any{zone("", fqdn+".", "public")},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := &zoneListMocks{zones: tt.zones}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := findDelegateZone(ctx, fqdn, pulumi.Parent(nil))
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
				return nil
			}, pulumi.WithMocks("myproject", "mystack", mocks))
			require.NoError(t, err)
		})
	}
}

// A failure to list zones must abort the deploy. Treating it as "no zone exists"
// is what creates the duplicate zone this code exists to prevent: the CD identity
// missing roles/dns.reader is a real, observed failure mode (the CLI hits the same
// permission on PrepareDomainDelegation), and it must not look like a fresh project.
func TestFindDelegateZoneListErrorAborts(t *testing.T) {
	mocks := &zoneListMocks{listErr: errDNSPermissionDenied}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := findDelegateZone(ctx, "myproject.tenant.defang.app", pulumi.Parent(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing Cloud DNS managed zones")
		assert.Contains(t, err.Error(), "Permission denied")
		return nil
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.NoError(t, err)
}
