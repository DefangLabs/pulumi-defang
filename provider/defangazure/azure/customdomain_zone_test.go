package azure

import (
	"testing"

	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zoneLookupMocks answers the dns.LookupZone invoke with a concrete resource ID
// so EnsureDomainZone has something to import; every other call/resource echoes
// inputs like azureNoopMocks.
type zoneLookupMocks struct{}

func (zoneLookupMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (zoneLookupMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	out := args.Args.Copy()
	out["id"] = resource.NewStringProperty(
		"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/dnszones/proj.example.com",
	)
	return out, nil
}

// TestEnsureDomainZoneNoDomain verifies the no-op path: a project without a
// delegate domain imports nothing and returns (nil, nil), so infra.DomainZone
// stays nil and CreateCustomDomain keeps its literal-domain fallback.
func TestEnsureDomainZoneNoDomain(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		zone, err := EnsureDomainZone(ctx, "proj", &SharedInfra{Domain: ""}, nil)
		require.NoError(t, err)
		assert.Nil(t, zone, "no delegate domain must import no zone")
		return nil
	}, pulumi.WithMocks("proj", "stack", azureNoopMocks{}))
	require.NoError(t, err)
}

// TestEnsureDomainZoneImports verifies that with a delegate domain set, the
// CLI-created zone is looked up and adopted into Pulumi state (no RetainOnDelete,
// so teardown deletes it). Asserts a non-nil zone resource is returned.
func TestEnsureDomainZoneImports(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rg, err := resources.NewResourceGroup(ctx, "rg", &resources.ResourceGroupArgs{
			ResourceGroupName: pulumi.String("rg"),
		})
		require.NoError(t, err)

		infra := &SharedInfra{ResourceGroup: rg, Domain: "proj.example.com"}
		zone, err := EnsureDomainZone(ctx, "proj", infra, pulumi.Parent(rg))
		require.NoError(t, err)
		require.NotNil(t, zone, "a delegate domain must import the zone")
		return nil
	}, pulumi.WithMocks("proj", "stack", zoneLookupMocks{}))
	require.NoError(t, err)
}
