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

// countType returns how many records match the given type token.
func countType(records []resourceRecord, typ string) int {
	n := 0
	for _, r := range records {
		if r.typ == typ {
			n++
		}
	}
	return n
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

func TestConstructProject(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// One Cloud Run service per container service
	assert.Equal(t, 2, countType(*records, gcpCloudRunServiceType))

	// Load balancer: one NEG and one backend service for the single ingress service
	assert.Equal(t, 1, countType(*records, "gcp:compute/regionNetworkEndpointGroup:RegionNetworkEndpointGroup"))
	assert.Equal(t, 1, countType(*records, "gcp:compute/backendService:BackendService"))

	// Two URL maps: one for HTTPS routing, one for HTTP→HTTPS redirect
	assert.Equal(t, 2, countType(*records, "gcp:compute/uRLMap:URLMap"))

	// Two forwarding rules: HTTPS (443) and HTTP (80)
	assert.Equal(t, 2, countType(*records, "gcp:compute/globalForwardingRule:GlobalForwardingRule"))
}

func TestConstructProjectAlwaysCreatesVPCFirewalls(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// SSH firewall rule
	ssh := findTypeWhere(*records, "gcp:compute/firewall:Firewall", func(m property.Map) bool {
		allows := m.Get("allows").AsArray()
		return allows.Len() == 1 && allows.Get(0).AsMap().Get("protocol").AsString() == "tcp"
	})
	require.NotNil(t, ssh, "expected an SSH firewall rule")
	assert.Equal(t, "INGRESS", ssh.inputs.Get("direction").AsString())
	sshPorts := ssh.inputs.Get("allows").AsArray().Get(0).AsMap().Get("ports").AsArray()
	assert.Equal(t, 1, sshPorts.Len())
	assert.Equal(t, "22", sshPorts.Get(0).AsString())

	// ICMP firewall rule
	icmp := findTypeWhere(*records, "gcp:compute/firewall:Firewall", func(m property.Map) bool {
		allows := m.Get("allows").AsArray()
		return allows.Len() == 1 && allows.Get(0).AsMap().Get("protocol").AsString() == "icmp"
	})
	require.NotNil(t, icmp, "expected an ICMP firewall rule")
	assert.Equal(t, "INGRESS", icmp.inputs.Get("direction").AsString())
}

func TestConstructProjectAlwaysCreatesPrivateDNSZone(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	zone := findTypeWhere(*records, "gcp:dns/managedZone:ManagedZone", func(m property.Map) bool {
		return m.Get("visibility").AsString() == "private"
	})
	require.NotNil(t, zone, "expected a private ManagedZone")
	assert.Equal(t, "google.internal.", zone.inputs.Get("dnsName").AsString())
}

func TestConstructProjectWithDomainCreatesPublicDNSZone(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: property.NewMap(map[string]property.Value{
			"domain": property.New("example.com"),
			"services": property.New(property.NewMap(map[string]property.Value{
				"worker": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("myapp:worker"),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	zone := findTypeWhere(*records, "gcp:dns/managedZone:ManagedZone", func(m property.Map) bool {
		return m.Get("dnsName").AsString() == "example.com."
	})
	require.NotNil(t, zone, "expected a public ManagedZone for example.com")
	assert.Equal(t, "example.com.", zone.inputs.Get("dnsName").AsString())
}

func TestConstructProjectWithDomainCreatesCAARecord(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: property.NewMap(map[string]property.Value{
			"domain": property.New("example.com"),
			"services": property.New(property.NewMap(map[string]property.Value{
				"worker": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("myapp:worker"),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	caa := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("type").AsString() == "CAA"
	})
	require.NotNil(t, caa, "expected a CAA RecordSet")
	assert.Equal(t, "example.com.", caa.inputs.Get("name").AsString())
	rrdatas := caa.inputs.Get("rrdatas").AsArray()
	assert.Equal(t, 2, rrdatas.Len())
}

func TestConstructProjectWithoutDomainSkipsCAARecord(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	caa := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("type").AsString() == "CAA"
	})
	assert.Nil(t, caa, "expected no CAA RecordSet without a domain")
}

func TestConstructProjectWithoutIngressSkipsLoadBalancer(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, 0, countType(*records, "gcp:compute/uRLMap:URLMap"))
	assert.Equal(t, 0, countType(*records, "gcp:compute/globalForwardingRule:GlobalForwardingRule"))
	assert.Equal(t, 0, countType(*records, "gcp:compute/regionNetworkEndpointGroup:RegionNetworkEndpointGroup"))
}

func TestConstructProjectWithDomainNameSetsHostRule(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: property.NewMap(map[string]property.Value{
			"services": property.New(property.NewMap(map[string]property.Value{
				"app": property.New(property.NewMap(map[string]property.Value{
					"image":      property.New("nginx:latest"),
					"domainName": property.New("app.example.com"),
					"ports":      property.New(property.NewArray([]property.Value{testutil.IngressPort(80)})),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	// Find the main URL map (has pathMatchers), not the HTTP-redirect one (has defaultUrlRedirect)
	urlMap := findTypeWhere(*records, "gcp:compute/uRLMap:URLMap", func(m property.Map) bool {
		return !m.Get("pathMatchers").IsNull()
	})
	require.NotNil(t, urlMap, "expected a URLMap to be registered")
	hostRules := urlMap.inputs.Get("hostRules").AsArray()
	require.Equal(t, 1, hostRules.Len(), "expected one host rule for the domain name")
	hosts := hostRules.Get(0).AsMap().Get("hosts").AsArray()
	require.Equal(t, 1, hosts.Len())
	assert.Equal(t, "app.example.com", hosts.Get(0).AsString())
}

func TestConstructProjectWithDomainCreatesWildcardCert(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
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

	// Private zone always created; public zone when domain is set → 2 total
	assert.Equal(t, 2, countType(*records, "gcp:dns/managedZone:ManagedZone"))
	// DNS authorization for the wildcard cert challenge
	assert.Equal(t, 1, countType(*records, "gcp:certificatemanager/dnsAuthorization:DnsAuthorization"))
	// Wildcard certificate
	assert.Equal(t, 1, countType(*records, "gcp:certificatemanager/certificate:Certificate"))
	// Cert map entry wiring the wildcard cert into the LB cert map
	assert.Equal(t, 1, countType(*records, "gcp:certificatemanager/certificateMapEntry:CertificateMapEntry"))
}

func TestConstructProjectWithoutDomainSkipsWildcardCert(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(80)),
		}),
	})

	require.NoError(t, err)

	// Private zone is always created even without a domain
	assert.Equal(t, 1, countType(*records, "gcp:dns/managedZone:ManagedZone"))
	assert.Equal(t, 0, countType(*records, "gcp:certificatemanager/dnsAuthorization:DnsAuthorization"))
	assert.Equal(t, 0, countType(*records, "gcp:certificatemanager/certificate:Certificate"))
	assert.Equal(t, 0, countType(*records, "gcp:certificatemanager/certificateMapEntry:CertificateMapEntry"))
}

func TestConstructProjectWithBuildCreatesBuildInfra(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"build": property.New(property.NewMap(map[string]property.Value{
					"context": property.New("./app"),
				})),
			})),
		}),
	})

	require.NoError(t, err)

	// Artifact Registry repository
	assert.Equal(t, 1, countType(*records, "gcp:artifactregistry/repository:Repository"))
	// Build service account
	bsa := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		return strings.Contains(m.Get("displayName").AsString(), "build")
	})
	require.NotNil(t, bsa, "expected a build service account")
	// Registry admin IAM binding for the build SA
	assert.Equal(t, 1, countType(*records, "gcp:artifactregistry/repositoryIamBinding:RepositoryIamBinding"))
	// Logging IAM members (logWriter + bucketWriter)
	assert.Equal(t, 2, countType(*records, "gcp:projects/iAMMember:IAMMember"))
	// Build resource for the service
	assert.Equal(t, 1, countType(*records, "defang-gcp:defanggcp:Build"))
}

func TestConstructProjectWithoutBuildSkipsBuildInfra(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	assert.Equal(t, 0, countType(*records, "gcp:artifactregistry/repository:Repository"))
	assert.Equal(t, 0, countType(*records, "gcp:artifactregistry/repositoryIamBinding:RepositoryIamBinding"))
	assert.Equal(t, 0, countType(*records, "gcp:projects/iAMMember:IAMMember"))
	assert.Equal(t, 0, countType(*records, "defang-gcp:defanggcp:Build"))
}

func TestConstructProjectWithPostgresCreatesVPCPeering(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"db": property.New(property.NewMap(map[string]property.Value{
				"image":    property.New("postgres:17"),
				"postgres": property.New(property.NewMap(map[string]property.Value{})),
			})),
		}),
	})

	require.NoError(t, err)

	// Private IP range for VPC peering
	peering := findTypeWhere(*records, "gcp:compute/globalAddress:GlobalAddress", func(m property.Map) bool {
		v := m.Get("purpose")
		return !v.IsNull() && v.AsString() == gcpVPCPeeringPurpose
	})
	require.NotNil(t, peering, "expected a VPC_PEERING GlobalAddress for Cloud SQL private IP")
	assert.Equal(t, "INTERNAL", peering.inputs.Get("addressType").AsString())
	assert.InDelta(t, 16.0, peering.inputs.Get("prefixLength").AsNumber(), 0)

	// Service networking connection
	assert.Equal(t, 1, countType(*records, "gcp:servicenetworking/connection:Connection"))

	// DatabaseInstance should be present
	assert.Equal(t, 1, countType(*records, "gcp:sql/databaseInstance:DatabaseInstance"))
}

func TestConstructProjectWithoutPostgresSkipsVPCPeering(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	peering := findTypeWhere(*records, "gcp:compute/globalAddress:GlobalAddress", func(m property.Map) bool {
		v := m.Get("purpose")
		return !v.IsNull() && v.AsString() == gcpVPCPeeringPurpose
	})
	assert.Nil(t, peering, "expected no VPC_PEERING GlobalAddress without a Postgres service")
	assert.Equal(t, 0, countType(*records, "gcp:servicenetworking/connection:Connection"))
}

func TestConstructProjectWithPostgresDatabaseInstanceHasPrivateNetwork(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"db": property.New(property.NewMap(map[string]property.Value{
				"image":    property.New("postgres:17"),
				"postgres": property.New(property.NewMap(map[string]property.Value{})),
			})),
		}),
	})

	require.NoError(t, err)

	instance := findTypeWhere(*records, "gcp:sql/databaseInstance:DatabaseInstance", func(m property.Map) bool {
		return true
	})
	require.NotNil(t, instance, "expected a DatabaseInstance")
	settings := instance.inputs.Get("settings").AsMap()
	ipCfg := settings.Get("ipConfiguration").AsMap()
	assert.False(t, ipCfg.Get("privateNetwork").IsNull(), "expected privateNetwork to be set on DatabaseInstance")
	assert.True(t, ipCfg.Get("ipv4Enabled").AsBool(), "expected ipv4Enabled to be true when no private-only network")
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
