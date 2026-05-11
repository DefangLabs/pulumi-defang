package scaleway

import (
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

const (
	scalewayContainerType      = "scaleway:containers/container:Container"
	scalewayDomainType         = "scaleway:containers/domain:Domain"
	scalewayNamespaceType      = "scaleway:containers/namespace:Namespace"
	scalewayPrivateNetworkType = "scaleway:network/privateNetwork:PrivateNetwork"
)

type scalewayResourceRecord struct {
	typ    string
	name   string
	inputs property.Map
}

func collectScalewayResources() (*integration.MockResourceMonitor, *[]scalewayResourceRecord) {
	var mu sync.Mutex
	var records []scalewayResourceRecord
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			mu.Lock()
			records = append(records, scalewayResourceRecord{
				typ:    string(args.TypeToken),
				name:   args.Name,
				inputs: args.Inputs,
			})
			mu.Unlock()
			return args.Name, args.Inputs, nil
		},
	}
	return mock, &records
}

func findScalewayType(records []scalewayResourceRecord, typ string) *scalewayResourceRecord {
	for i := range records {
		if records[i].typ == typ {
			return &records[i]
		}
	}
	return nil
}

func countScalewayType(records []scalewayResourceRecord, typ string) int {
	n := 0
	for _, r := range records {
		if r.typ == typ {
			n++
		}
	}
	return n
}

func TestConstructScalewayServiceWithImage(t *testing.T) {
	mock, records := collectScalewayResources()
	server := testutil.MakeScalewayTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.ScalewayURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image": property.New("nginx:latest"),
			"ports": property.New(property.NewArray([]property.Value{
				testutil.IngressPort(8080),
			})),
		}),
	})

	require.NoError(t, err)
	container := findScalewayType(*records, scalewayContainerType)
	require.NotNil(t, container, "expected Serverless Container resource")
	assert.Equal(t, "nginx:latest", container.inputs.Get("registryImage").AsString())
	assert.Equal(t, 8080.0, container.inputs.Get("port").AsNumber())
	assert.Equal(t, "public", container.inputs.Get("privacy").AsString())
}

func TestConstructScalewayServiceWithDomainName(t *testing.T) {
	mock, records := collectScalewayResources()
	server := testutil.MakeScalewayTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.ScalewayURN("Service"),
		Inputs: property.NewMap(map[string]property.Value{
			"image":      property.New("myapp:latest"),
			"domainName": property.New("api.example.com"),
		}),
	})

	require.NoError(t, err)
	require.NotNil(t, findScalewayType(*records, scalewayDomainType), "expected custom domain resource")
}

func TestConstructScalewayProjectCreatesSharedInfraAndContainer(t *testing.T) {
	mock, records := collectScalewayResources()
	server := testutil.MakeScalewayTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.ScalewayURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(80)),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countScalewayType(*records, scalewayNamespaceType))
	assert.Equal(t, 1, countScalewayType(*records, scalewayPrivateNetworkType))
	container := findScalewayType(*records, scalewayContainerType)
	require.NotNil(t, container, "expected Project to create a Serverless Container")
	assert.Equal(t, "nginx:latest", container.inputs.Get("registryImage").AsString())
	assert.Equal(t, "name-private-network", container.inputs.Get("privateNetworkId").AsString())
}


func TestConstructScalewayProjectAllResourcesAreChildren(t *testing.T) {
	mock, tracker := testutil.NewParentTracker()
	server := testutil.MakeScalewayTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.ScalewayURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image":      property.New("nginx:latest"),
				"domainName": property.New("api.example.com"),
				"ports": property.New(property.NewArray([]property.Value{
					testutil.IngressPort(80),
				})),
			})),
		}),
	})

	require.NoError(t, err)
	tracker.AssertAllDescendFrom(t, testutil.ScalewayURN("Project"))
}
