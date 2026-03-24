package gcp

// Project is the top-level orchestration component for GCP. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (Cloud Run
// service, Cloud SQL, etc.) lives in their own dedicated test files.

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	defanggcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructProject(t *testing.T) {
	server := testutil.MakeGcpTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}

func TestConstructProjectDependencies(t *testing.T) {
	type reg struct {
		name string
		typ  string
		deps []string
	}

	var mu sync.Mutex
	var registrations []reg

	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.RegisterRPC != nil {
				mu.Lock()
				registrations = append(registrations, reg{
					name: args.Name,
					typ:  string(args.TypeToken),
					deps: args.RegisterRPC.GetDependencies(),
				})
				mu.Unlock()
			}
			return args.Name, args.Inputs, nil
		},
	}

	server, err := integration.NewServer(
		context.Background(),
		defanggcp.Name,
		semver.MustParse("1.0.0"),
		integration.WithProvider(defanggcp.Provider()),
		integration.WithMocks(mock),
	)
	require.NoError(t, err)

	// Chain: db → cache → api → frontend
	// With 4 nodes, random map iteration has only a 1/4! ≈ 4% chance of
	// accidentally producing the correct order without topological sort.
	dependsOn := func(dep string) property.Value {
		return property.New(property.NewMap(map[string]property.Value{
			dep: property.New(property.NewMap(map[string]property.Value{})),
		}))
	}
	_, err = server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"db": property.New(property.NewMap(map[string]property.Value{
				"image":    property.New("postgres:15"),
				"postgres": property.New(property.NewMap(map[string]property.Value{})),
			})),
			"cache": property.New(property.NewMap(map[string]property.Value{
				"image":     property.New("redis:7"),
				"dependsOn": dependsOn("db"),
			})),
			"api": property.New(property.NewMap(map[string]property.Value{
				"image":     property.New("myapp:latest"),
				"dependsOn": dependsOn("cache"),
			})),
			"frontend": property.New(property.NewMap(map[string]property.Value{
				"image":     property.New("nginx:latest"),
				"dependsOn": dependsOn("api"),
			})),
		}),
	})
	require.NoError(t, err)

	// Index component registrations by name (filter by type to avoid collisions
	// with child resources that share the same name, e.g. service accounts)
	componentTypes := map[string]bool{
		"defang-gcp:index:Postgres": true,
		"defang-gcp:index:Service":  true,
	}
	idxByName := map[string]int{}
	depsByName := map[string][]string{}
	for i, r := range registrations {
		if componentTypes[r.typ] {
			idxByName[r.name] = i
			depsByName[r.name] = r.deps
		}
	}

	hasDep := func(svc, dep string) bool {
		for _, d := range depsByName[svc] {
			if strings.HasSuffix(d, "::"+dep) {
				return true
			}
		}
		return false
	}

	// Each node must be registered after its dependency
	chain := [][2]string{{"db", "cache"}, {"cache", "api"}, {"api", "frontend"}}
	for _, link := range chain {
		before, after := link[0], link[1]
		assert.Less(t, idxByName[before], idxByName[after],
			"%s should be registered before %s", before, after)
		assert.True(t, hasDep(after, before),
			"%s component should declare a dependency on %s; got deps: %v", after, before, depsByName[after])
	}
}
