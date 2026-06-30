package azure

import (
	"strings"
	"testing"

	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zoneImportMocks captures the import ID and inputs of the imported DNS zone
// resource ("domain-zone") so tests can assert EnsureDomainZone imports the
// right zone in the right resource group. Every other resource echoes its
// inputs like azureNoopMocks.
type zoneImportMocks struct {
	importID   string
	zoneInputs resource.PropertyMap
}

func (m *zoneImportMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if strings.HasSuffix(args.TypeToken, ":Zone") {
		m.importID = args.ID // the physical ID passed via pulumi.Import
		m.zoneInputs = args.Inputs
	}
	return args.Name + "_id", args.Inputs, nil
}

func (zoneImportMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// TestEnsureDomainZoneNoDomain verifies the no-op path: a project without a
// delegate domain imports nothing and returns (nil, nil), so infra.DomainZone
// stays nil and CreateCustomDomain keeps its literal-domain fallback.
func TestEnsureDomainZoneNoDomain(t *testing.T) {
	mocks := &zoneImportMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		zone, err := EnsureDomainZone(ctx, "proj", &SharedInfra{Domain: ""}, nil)
		require.NoError(t, err)
		assert.Nil(t, zone, "no delegate domain must import no zone")
		return nil
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)
	assert.Empty(t, mocks.importID, "no zone resource should be created without a domain")
}

// TestEnsureDomainZoneImports verifies that with a delegate domain set, the
// CLI-created zone is adopted into Pulumi state by importing the constructed ARM
// resource ID (no RetainOnDelete, so teardown deletes it). The assertion pins
// the import ID to domainZoneID(subscription, project RG, domain) so the test
// fails if EnsureDomainZone imports the wrong zone or resource group, or stops
// importing entirely.
func TestEnsureDomainZoneImports(t *testing.T) {
	mocks := &zoneImportMocks{}
	var wantID string
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rg, err := resources.NewResourceGroup(ctx, "rg", &resources.ResourceGroupArgs{
			ResourceGroupName: pulumi.String("rg"),
		})
		require.NoError(t, err)

		// Expected ID uses the same derivation as the implementation: the lookup
		// RG is ProjectResourceGroupName(ctx, projectName), NOT the fixture RG.
		wantID = domainZoneID(resolveSubscriptionID(ctx), ProjectResourceGroupName(ctx, "proj"), "proj.example.com")

		infra := &SharedInfra{ResourceGroup: rg, Domain: "proj.example.com"}
		zone, err := EnsureDomainZone(ctx, "proj", infra, pulumi.Parent(rg))
		require.NoError(t, err)
		require.NotNil(t, zone, "a delegate domain must import the zone")
		return nil
	}, pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)

	assert.Equal(t, wantID, mocks.importID, "zone must be imported by the constructed ARM resource ID")
	require.NotNil(t, mocks.zoneInputs, "zone resource must have been registered")
	assert.Equal(t, "proj.example.com", mocks.zoneInputs["zoneName"].StringValue(),
		"imported zone must target the delegate domain")
	assert.Equal(t, "global", mocks.zoneInputs["location"].StringValue(), "public DNS zones live at location global")
}

// TestDomainZoneID pins the ARM resource ID format, including the lowercase
// "dnszones" segment that Azure's canonical ID uses — a casing mismatch can make
// the import propose a replacement.
func TestDomainZoneID(t *testing.T) {
	got := domainZoneID("sub", "rg", "proj.example.com")
	want := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/dnszones/proj.example.com"
	assert.Equal(t, want, got)
}
