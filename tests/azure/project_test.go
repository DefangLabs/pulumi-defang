package azure

// Project is the top-level orchestration component for Azure. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (Container
// App, Postgres, etc.) lives in their own dedicated test files.

import (
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	defangazure "github.com/DefangLabs/pulumi-defang/provider/defangazure"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

type resourceRecord struct {
	typ    string
	name   string
	inputs property.Map
}

// collectResources returns a mock and a pointer to the slice it populates.
func collectResources() (*integration.MockResourceMonitor, *[]resourceRecord) {
	var mu sync.Mutex
	var records []resourceRecord
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			mu.Lock()
			records = append(records, resourceRecord{
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

// findTypeWhere returns the first record matching the given type token and predicate, or nil.
func findTypeWhere(records []resourceRecord, typ string, pred func(property.Map) bool) *resourceRecord {
	for i := range records {
		if records[i].typ == typ && pred(records[i].inputs) {
			return &records[i]
		}
	}
	return nil
}

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

// TestConstructAzureProjectServiceWithPoliciesGrantsRoles covers the
// full-role-definition-ID form of an x-defang-policies entry: a
// RoleAssignment is registered at the project's resource group scope,
// granting the service's own managed identity. The bare-name form (resolved
// by listing role definitions against live Azure — see
// resolveRoleDefinitionID) isn't exercised here: unlike GCP's ResolvePolicyRole,
// it needs a real ARM credential and can't run against the mock resource
// monitor alone (see roleNameFilter's unit test for the pure escaping logic).
func TestConstructAzureProjectServiceWithPoliciesGrantsRoles(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeAzureTestServer(integration.WithMocks(mock))

	const roleDefID = "/subscriptions/sub/providers/Microsoft.Authorization/roleDefinitions/deployer-guid"

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AzureURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"redeployer": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("defangio/cli:latest"),
				"policies": property.New(property.NewArray([]property.Value{
					property.New(roleDefID),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	found := findTypeWhere(*records, "azure-native:authorization:RoleAssignment", func(m property.Map) bool {
		return m.Get("roleDefinitionId").AsString() == roleDefID
	})
	require.NotNil(t, found, "expected a RoleAssignment for the x-defang-policies entry")
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
