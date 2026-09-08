package azure

// Project is the top-level orchestration component for Azure. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (Container
// App, Postgres, etc.) lives in their own dedicated test files.

import (
	"strings"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	defangazure "github.com/DefangLabs/pulumi-defang/provider/defangazure"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAzureProject(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}

// TestConstructAzureProjectAllResourcesAreChildren asserts that every resource
// created inside a Project descends from the Project component in the Pulumi
// hierarchy. Runs a rich Construct that exercises shared infra (resource
// group, VNet, LAW, private DNS), Container Apps, managed Postgres, and
// Redis Enterprise so the assertion covers most resource-creation paths.
func TestConstructAzureProjectAllResourcesAreChildren(t *testing.T) {
	mock, tracker := testutil.NewParentTracker()
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("nginx:latest"),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
			})),
			"worker": testutil.ServiceWithImage("myapp:worker"),
			"db": property.New(property.NewMap(map[string]property.Value{
				"image":    property.New("postgres:17"),
				"postgres": property.New(property.NewMap(map[string]property.Value{})),
				"environment": property.New(property.NewMap(map[string]property.Value{
					"POSTGRES_PASSWORD": property.New("secret"),
				})),
			})),
			"cache": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("redis:7"),
				"redis": property.New(property.NewMap(map[string]property.Value{})),
			})),
		}),
	})
	require.NoError(t, err)

	tracker.AssertAllDescendFrom(t, testutil.AzureURN("Project"))
}

func TestConstructAzureProjectRejectsForeignPolicies(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	// No cross-cloud filtering: an AWS-qualified entry on an Azure deploy is
	// a validation error pointing at per-stack variable values. Entries a
	// stack leaves empty ("${EXTRA:-}") normalize away and don't trip it.
	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myapp:latest"),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
				"policies": property.New(property.NewArray([]property.Value{
					property.New("arn:aws:iam::123456789012:policy/deployer"),
					property.New(""),
				})),
			})),
		}),
	})

	require.ErrorContains(t, err, "aws identifier")
	require.ErrorContains(t, err, "targets azure")
}

func TestConstructAzureProjectEmptyPoliciesDeploy(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	// A policies list that normalizes to nothing (all entries are unset
	// "${VAR:-}" substitutions) must not trip the unsupported error.
	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myapp:latest"),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
				"policies": property.New(property.NewArray([]property.Value{
					property.New(""),
				})),
			})),
		}),
	})

	require.NoError(t, err)
}

func TestConstructAzureProjectRejectsApplicablePolicies(t *testing.T) {
	server := testutil.MakeAzureTestServer()

	// A bare name applies on the current cloud, and Azure doesn't support
	// policies yet.
	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myapp:latest"),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
				"policies": property.New(property.NewArray([]property.Value{
					property.New("Contributor"),
				})),
			})),
		}),
	})

	require.ErrorContains(t, err, "x-defang-policies is not supported on Azure")
}

// TestConstructAzureProjectBuildCarriesPluginIdentity asserts that the Build
// resource the provider registers for itself tells the engine both where to
// fetch the plugin from and which version of it to use. Registrations that go
// through a generated SDK get both for free; the ones we make with a raw
// ctx.RegisterResource do not, and omitting them strands the stack on destroy.
// See common.PluginIdentityFrom.
func TestConstructAzureProjectBuildCarriesPluginIdentity(t *testing.T) {
	// Pin a version the way the linker does for a release build, so the
	// assertion covers the version as well as the URL.
	prev := defangazure.Version
	defangazure.Version = "9.9.9"
	t.Cleanup(func() { defangazure.Version = prev })

	mock, tracker := testutil.NewPluginTracker()
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"builder": property.New(property.NewMap(map[string]property.Value{
				"build": property.New(property.NewMap(map[string]property.Value{
					"context": property.New("https://acct.blob.core.windows.net/uploads/digest.tar.gz?sig=x"),
				})),
			})),
		}),
	})
	require.NoError(t, err)

	tracker.AssertOwnCustomResourcesCarryPluginIdentity(t, common.PluginDownloadURL, "9.9.9")
}

// TestConstructAzureProjectRecipeOptOutSkipsDelegateDomain covers
// use-defang-app-subdomain=false: the delegate-domain DNS zone and the
// per-service CNAME/asuid records that publish services under it are not
// created, even though a domain is supplied. The service keeps its
// azurecontainerapps.io name, which is what its Endpoint already reports.
func TestConstructAzureProjectRecipeOptOutSkipsDelegateDomain(t *testing.T) {
	mock, records := testutil.CollectResources()
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AzureURN("Project"),
		Config: testutil.StackConfig("defang-azure:"+common.DefangAppSubdomainKey, "false"),
		Inputs: property.NewMap(map[string]property.Value{
			"domain": property.New("example.com"),
			"services": property.New(property.NewMap(map[string]property.Value{
				"app": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("nginx:latest"),
					"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(80)})),
				})),
			})),
		}),
	})

	require.NoError(t, err, "opting out must degrade to the ACA hostname, not fail the deploy")
	assert.Equal(t, 0, countAzureDNS(*records, "Zone"), "no delegate-domain zone when opted out")
	assert.Equal(t, 0, countAzureDNS(*records, "RecordSet"), "no per-service CNAME/asuid records")
}

// TestConstructAzureProjectDelegateDomainByDefault is the control for the test
// above: with the recipe at its default the same inputs do create the zone and
// the per-service records.
func TestConstructAzureProjectDelegateDomainByDefault(t *testing.T) {
	mock, records := testutil.CollectResources()
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: property.NewMap(map[string]property.Value{
			"domain": property.New("example.com"),
			"services": property.New(property.NewMap(map[string]property.Value{
				"app": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("nginx:latest"),
					"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(80)})),
				})),
			})),
		}),
	})

	require.NoError(t, err)
	assert.Positive(t, countAzureDNS(*records, "RecordSet"),
		"the delegate domain must publish the service by default")
}

// countAzureDNS counts azure-native DNS registrations of the given kind. The
// azure-native provider stamps a dated API version into the type token
// (azure-native:dns/vNNNNNNNN:RecordSet), so match on the suffix rather than
// pinning a version the SDK bump would break.
func countAzureDNS(records []testutil.ResourceRecord, kind string) int {
	n := 0
	for _, r := range records {
		if strings.HasPrefix(r.Typ, "azure-native:dns") && strings.HasSuffix(r.Typ, ":"+kind) {
			n++
		}
	}
	return n
}
