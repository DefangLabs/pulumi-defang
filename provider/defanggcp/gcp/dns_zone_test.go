package gcp

import (
	"errors"
	"strings"
	"sync"
	"testing"

	pulumigcp "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gosec // Pulumi invoke token, not a credential
const getManagedZonesToken = "gcp:dns/getManagedZones:getManagedZones"

//nolint:gosec // Pulumi resource type token, not a credential
const managedZoneToken = "gcp:dns/managedZone:ManagedZone"

var (
	errDNSPermissionDenied       = errors.New("googleapi: Error 403: Permission denied on resource")
	errServiceAccountActAsDenied = errors.New("403 iam.serviceAccounts.actAs denied")
)

// TestDelegateZoneName pins the zone name against the Defang CLI's own derivation
// in PrepareDomainDelegation ("defang-" + dns.SafeLabel(delegateDomain), defang
// src/pkg/cli/client/byoc/gcp/byoc.go). If this drifts, the CD stops recognising
// the zone the CLI created and delegated, and silently creates a duplicate.
func TestDelegateZoneName(t *testing.T) {
	const wantExampleZone = "defang-example-com"
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{"delegate domain", "myproject.tenant.defang.app", "defang-myproject-tenant-defang-app"},
		{"apex", "example.com", wantExampleZone},
		{"uppercase is lowered", "Example.COM", wantExampleZone},
		{"trailing dot is dropped", "example.com.", wantExampleZone},
		{"uppercase and trailing dot", "EXAMPLE.COM.", wantExampleZone},
		{"hyphens are preserved", "my-app.tenant.defang.app", "defang-my-app-tenant-defang-app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DelegateZoneName(tt.domain))
		})
	}
}

// zoneListMocks answers the getManagedZones invoke and records managed-zone
// registrations/reads. readErr simulates a permission failure while reading an
// external zone after discovery succeeds.
type zoneListMocks struct {
	mu              sync.Mutex
	zones           []map[string]any
	listErr         error
	readErr         error
	calls           int
	invokeProviders []string
	resources       []pulumi.MockResourceArgs
}

type delegateZoneTestParent struct {
	pulumi.ResourceState
}

func (m *zoneListMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	m.resources = append(m.resources, args)
	readErr := m.readErr
	m.mu.Unlock()

	if args.TypeToken == managedZoneToken && args.ID != "" && readErr != nil {
		return "", nil, readErr
	}

	outputs := resource.PropertyMap{}
	for key, value := range args.Inputs {
		outputs[key] = value
	}
	if args.TypeToken == managedZoneToken && args.ReadRPC != nil {
		outputs["name"] = resource.NewStringProperty(args.ID)
		outputs["dnsName"] = resource.NewStringProperty("myproject.tenant.defang.app.")
		outputs["visibility"] = resource.NewStringProperty("public")
		return args.ID, outputs, nil
	}
	return args.Name + "_id", outputs, nil
}

func (m *zoneListMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != getManagedZonesToken {
		return args.Args, nil
	}
	m.mu.Lock()
	m.calls++
	m.invokeProviders = append(m.invokeProviders, args.Provider)
	listErr := m.listErr
	zones := append([]map[string]any(nil), m.zones...)
	m.mu.Unlock()
	if listErr != nil {
		return nil, listErr
	}
	values := make([]resource.PropertyValue, 0, len(zones))
	for _, z := range zones {
		values = append(values, resource.NewObjectProperty(resource.NewPropertyMapFromMap(z)))
	}
	return resource.PropertyMap{
		"managedZones": resource.NewArrayProperty(values),
	}, nil
}

func (m *zoneListMocks) managedZoneResources() []pulumi.MockResourceArgs {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []pulumi.MockResourceArgs
	for _, r := range m.resources {
		if r.TypeToken == managedZoneToken {
			result = append(result, r)
		}
	}
	return result
}

func zone(name, dnsName, visibility, description string) map[string]any {
	return map[string]any{
		"name":          name,
		"dnsName":       dnsName,
		"visibility":    visibility,
		"description":   description,
		"id":            name,
		"managedZoneId": "1234",
		"nameServers":   []any{},
	}
}

//nolint:funlen // The table deliberately keeps all selection outcomes together for auditability.
func TestFindDelegateZone(t *testing.T) {
	const (
		projectName        = "myproject"
		fqdn               = "myproject.tenant.defang.app"
		cliZone            = "defang-myproject-tenant-defang-app"
		managedDescription = "Public DNS zone for myproject"
	)

	tests := []struct {
		name    string
		domain  string
		zones   []map[string]any
		want    delegateZoneSelection
		wantErr []string
	}{
		{
			name: "no zones at all",
			want: delegateZoneSelection{mode: delegateZoneCreate},
		},
		{
			name:   "the CLI-created zone is read externally",
			domain: "MYPROJECT.TENANT.DEFANG.APP.",
			zones:  []map[string]any{zone(cliZone, "MyProject.Tenant.Defang.App", "PUBLIC", "defang delegate domain")},
			want:   delegateZoneSelection{name: cliZone, mode: delegateZoneExternal},
		},
		{
			name: "unrelated zones are ignored",
			zones: []map[string]any{
				zone("defang-other-defang-app", "other.defang.app.", "public", "defang delegate domain"),
				zone("customer-apex", "example.com.", "public", "customer zone"),
			},
			want: delegateZoneSelection{mode: delegateZoneCreate},
		},
		{
			name:  "a private zone for the delegate domain is ignored",
			zones: []map[string]any{zone("private-mirror", fqdn+".", "private", "private zone")},
			want:  delegateZoneSelection{mode: delegateZoneCreate},
		},
		{
			name:  "one differently named public zone is unambiguous",
			zones: []map[string]any{zone("legacy-external", fqdn+".", "public", "legacy external zone")},
			want:  delegateZoneSelection{name: "legacy-external", mode: delegateZoneExternal},
		},
		{
			name: "the exact CLI name wins over unrelated duplicates",
			zones: []map[string]any{
				zone("unrelated-duplicate", fqdn+".", "public", "other owner"),
				zone(cliZone, fqdn+".", "public", "defang delegate domain"),
			},
			want: delegateZoneSelection{name: cliZone, mode: delegateZoneExternal},
		},
		{
			name: "the existing Pulumi resource stays managed",
			zones: []map[string]any{
				zone("public-dns-a1b2c3d", fqdn+".", "public", managedDescription),
			},
			want: delegateZoneSelection{name: "public-dns-a1b2c3d", mode: delegateZoneManaged},
		},
		{
			name: "a stack ownership signal wins over an unrelated duplicate",
			zones: []map[string]any{
				zone("unrelated-duplicate", fqdn+".", "public", "other owner"),
				zone("public-dns-a1b2c3d", fqdn+".", "public", managedDescription),
			},
			want: delegateZoneSelection{name: "public-dns-a1b2c3d", mode: delegateZoneManaged},
		},
		{
			name: "duplicates without an ownership signal fail",
			zones: []map[string]any{
				zone("zzz-duplicate", fqdn+".", "public", "other owner"),
				zone("aaa-duplicate", fqdn+".", "public", "another owner"),
			},
			wantErr: []string{"multiple public Cloud DNS managed zones", "aaa-duplicate", "zzz-duplicate", "refusing to choose"},
		},
		{
			name: "CLI and stack ownership signals conflict instead of deleting the managed zone",
			zones: []map[string]any{
				zone(cliZone, fqdn+".", "public", "defang delegate domain"),
				zone("public-dns-a1b2c3d", fqdn+".", "public", managedDescription),
			},
			wantErr: []string{"CLI-created zone", "stack-managed public-dns", "migrate its Pulumi state explicitly"},
		},
		{
			name: "two stack ownership signals fail",
			zones: []map[string]any{
				zone("public-dns-a1b2c3d", fqdn+".", "public", managedDescription),
				zone("public-dns-d4e5f6a", fqdn+".", "public", managedDescription),
			},
			wantErr: []string{"more than one zone", "ownership markers", "refusing to choose"},
		},
		{
			name:  "a parent zone is not a match for the delegate domain",
			zones: []map[string]any{zone("defang-tenant-defang-app", "tenant.defang.app.", "public", "parent")},
			want:  delegateZoneSelection{mode: delegateZoneCreate},
		},
		{
			name:  "a zone with no name is skipped",
			zones: []map[string]any{zone("", fqdn+".", "public", "missing name")},
			want:  delegateZoneSelection{mode: delegateZoneCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := &zoneListMocks{zones: tt.zones}
			domain := tt.domain
			if domain == "" {
				domain = fqdn
			}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := findDelegateZone(ctx, projectName, domain, pulumi.Parent(nil))
				if len(tt.wantErr) != 0 {
					require.Error(t, err)
					for _, fragment := range tt.wantErr {
						assert.Contains(t, err.Error(), fragment)
					}
					return nil
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
				return nil
			}, pulumi.WithMocks(projectName, "mystack", mocks))
			require.NoError(t, err)
		})
	}
}

func TestFindDelegateZoneListErrorAborts(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"permission denied", errDNSPermissionDenied},
		{"credential lacks permission", errServiceAccountActAsDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := &zoneListMocks{listErr: tt.err}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				_, err := findDelegateZone(ctx, "myproject", "myproject.tenant.defang.app", pulumi.Parent(nil))
				require.Error(t, err)
				assert.Contains(t, err.Error(), "dns.managedZones.list")
				assert.Contains(t, err.Error(), "roles/dns.admin")
				assert.Contains(t, err.Error(), "403")
				return nil
			}, pulumi.WithMocks("myproject", "mystack", mocks))
			require.NoError(t, err)
			assert.Equal(t, 1, mocks.calls)
			assert.Empty(t, mocks.managedZoneResources(), "lookup failure must not fall through to create")
		})
	}
}

func TestFindDelegateZoneRecognizesConfiguredAutonaming(t *testing.T) {
	const fqdn = "myproject.tenant.defang.app"
	mocks := &zoneListMocks{zones: []map[string]any{
		zone(
			"defang-myproject-mystack-public-dns-a1b2c3d",
			fqdn+".",
			"public",
			"Public DNS zone for myproject",
		),
	}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		selection, err := findDelegateZone(ctx, "myproject", fqdn, pulumi.Parent(nil))
		require.NoError(t, err)
		assert.Equal(t, delegateZoneManaged, selection.mode)
		return nil
	},
		pulumi.WithMocks("myproject", "mystack", mocks),
		withPulumiConfig(map[string]string{
			pulumiAutonamingConfigKey: `{"pattern":"Defang-${project}-${stack}-${name}-${hex(7)}"}`,
		}),
	)
	require.NoError(t, err)
}

func TestEnsureDelegateZoneReadPermissionFailureAborts(t *testing.T) {
	const (
		fqdn    = "myproject.tenant.defang.app"
		cliZone = "defang-myproject-tenant-defang-app"
	)
	mocks := &zoneListMocks{
		zones:   []map[string]any{zone(cliZone, fqdn+".", "public", "defang delegate domain")},
		readErr: errDNSPermissionDenied,
	}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := ensureDelegateZone(ctx, "myproject", fqdn, pulumi.Parent(nil), pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Permission denied")

	resources := mocks.managedZoneResources()
	require.Len(t, resources, 1)
	assert.NotNil(t, resources[0].ReadRPC)
	assert.Equal(t, externalDelegateZoneLogicalName, resources[0].Name)
	assert.NotEqual(t, "public-dns", resources[0].Name, "read failure must not fall through to managed creation")
}

func TestEnsureDelegateZonePreservesManagedRegistration(t *testing.T) {
	const fqdn = "myproject.tenant.defang.app"
	mocks := &zoneListMocks{zones: []map[string]any{
		zone("public-dns-a1b2c3d", fqdn+".", "public", "Public DNS zone for myproject"),
	}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := ensureDelegateZone(ctx, "myproject", fqdn, pulumi.Parent(nil), pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.NoError(t, err)

	resources := mocks.managedZoneResources()
	require.Len(t, resources, 1)
	managed := resources[0]
	assert.Equal(t, "public-dns", managed.Name)
	assert.Empty(t, managed.ID)
	assert.NotNil(t, managed.RegisterRPC)
	assert.Nil(t, managed.ReadRPC)
	assert.Equal(t, fqdn+".", managed.Inputs["dnsName"].StringValue())
	assert.Equal(t, "Public DNS zone for myproject", managed.Inputs["description"].StringValue())
	assert.NotContains(t, managed.Inputs, resource.PropertyKey("name"),
		"adding immutable Name would replace an existing stack-owned zone")
}

func TestEnsureDelegateZoneReadsExternalWithParentProvider(t *testing.T) {
	const (
		fqdn    = "myproject.tenant.defang.app"
		cliZone = "defang-myproject-tenant-defang-app"
	)
	mocks := &zoneListMocks{zones: []map[string]any{
		zone(cliZone, fqdn+".", "public", "defang delegate domain"),
	}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		provider, err := pulumigcp.NewProvider(ctx, "explicit-gcp", &pulumigcp.ProviderArgs{
			Project: pulumi.String("gcp-project"),
		})
		if err != nil {
			return err
		}
		parent := &delegateZoneTestParent{}
		if err := ctx.RegisterComponentResource(
			"test:index:Parent", "parent", parent, pulumi.Providers(provider),
		); err != nil {
			return err
		}
		parentOpt := pulumi.Parent(parent)
		_, err = ensureDelegateZone(ctx, "myproject", fqdn, parentOpt, parentOpt)
		return err
	}, pulumi.WithMocks("myproject", "mystack", mocks))
	require.NoError(t, err)

	resources := mocks.managedZoneResources()
	require.Len(t, resources, 1)
	external := resources[0]
	assert.Equal(t, externalDelegateZoneLogicalName, external.Name)
	assert.Equal(t, cliZone, external.ID)
	assert.Nil(t, external.RegisterRPC)
	require.NotNil(t, external.ReadRPC)
	assert.True(t,
		external.ReadRPC.GetParent() != "" &&
			strings.HasSuffix(external.ReadRPC.GetParent(), "test:index:Parent::parent"),
	)
	assert.Contains(t, external.Provider, "pulumi:providers:gcp::explicit-gcp")

	mocks.mu.Lock()
	require.Len(t, mocks.invokeProviders, 1)
	invokeProvider := mocks.invokeProviders[0]
	mocks.mu.Unlock()
	assert.Equal(t, external.Provider, invokeProvider,
		"zone list invoke and external read must inherit the same explicit provider")
}
