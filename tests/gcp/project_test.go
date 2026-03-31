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

// countTypeWhere returns how many records match the given type token and predicate.
func countTypeWhere(records []resourceRecord, typ string, pred func(property.Map) bool) int {
	n := 0
	for _, r := range records {
		if r.typ == typ && pred(r.inputs) {
			n++
		}
	}
	return n
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

	// Only the ingress-port service goes to Cloud Run; the portless worker goes to Compute Engine
	assert.Equal(t, 1, countType(*records, gcpCloudRunServiceType))
	assert.Equal(t, 1, countType(*records, "gcp:compute/regionInstanceGroupManager:RegionInstanceGroupManager"))

	// Load balancer: one NEG and one backend service for the single ingress (Cloud Run) service
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

func TestConstructProjectWithDomainCreatesPublicARecords(t *testing.T) {
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

	// Main service domain: app.example.com.
	mainA := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("name").AsString() == "app.example.com." && m.Get("type").AsString() == "A"
	})
	require.NotNil(t, mainA, "expected an A record for app.example.com.")
	assert.InDelta(t, 60.0, mainA.inputs.Get("ttl").AsNumber(), 0)

	// Port-specific domain: app--80.example.com.
	portA := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("name").AsString() == "app--80.example.com." && m.Get("type").AsString() == "A"
	})
	require.NotNil(t, portA, "expected an A record for app--80.example.com.")
	assert.InDelta(t, 60.0, portA.inputs.Get("ttl").AsNumber(), 0)
}

func TestConstructProjectWithoutDomainSkipsPublicARecords(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(80)),
		}),
	})

	require.NoError(t, err)

	// Without a domain, no public A records should be created (only private zone records for managed services)
	publicA := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("type").AsString() == "A"
	})
	assert.Nil(t, publicA, "expected no public A records without a domain")
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

	// Exactly one private DNS A record pointing to the Cloud SQL private IP
	dbDNS := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("name").AsString() == "db.google.internal." && m.Get("type").AsString() == "A"
	})
	require.NotNil(t, dbDNS, "expected a private DNS A record for Cloud SQL")
	assert.InDelta(t, 60.0, dbDNS.inputs.Get("ttl").AsNumber(), 0)
	assert.Equal(t, 1, countTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("name").AsString() == "db.google.internal." && m.Get("type").AsString() == "A"
	}), "expected exactly one private DNS A record for Cloud SQL")
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

func TestConstructProjectWithRedisCreatesVPCPeering(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"cache": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("redis:7"),
				"redis": property.New(property.NewMap(map[string]property.Value{})),
			})),
		}),
	})

	require.NoError(t, err)

	// Private IP range for VPC peering
	peering := findTypeWhere(*records, "gcp:compute/globalAddress:GlobalAddress", func(m property.Map) bool {
		v := m.Get("purpose")
		return !v.IsNull() && v.AsString() == gcpVPCPeeringPurpose
	})
	require.NotNil(t, peering, "expected a VPC_PEERING GlobalAddress for Memorystore private IP")
	assert.Equal(t, "INTERNAL", peering.inputs.Get("addressType").AsString())

	// Service networking connection
	assert.Equal(t, 1, countType(*records, "gcp:servicenetworking/connection:Connection"))

	// Memorystore instance should be present
	assert.Equal(t, 1, countType(*records, "gcp:redis/instance:Instance"))

	// Exactly one private DNS A record pointing to the Memorystore host
	redisDNS := findTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("name").AsString() == "cache.google.internal." && m.Get("type").AsString() == "A"
	})
	require.NotNil(t, redisDNS, "expected a private DNS A record for Memorystore")
	assert.InDelta(t, 60.0, redisDNS.inputs.Get("ttl").AsNumber(), 0)
	assert.Equal(t, 1, countTypeWhere(*records, "gcp:dns/recordSet:RecordSet", func(m property.Map) bool {
		return m.Get("name").AsString() == "cache.google.internal." && m.Get("type").AsString() == "A"
	}), "expected exactly one private DNS A record for Memorystore")
}

func TestConstructProjectWithRedisCreatesMemorystoreInstance(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"cache": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("redis:7"),
				"redis": property.New(property.NewMap(map[string]property.Value{})),
			})),
		}),
	})

	require.NoError(t, err)

	inst := findTypeWhere(*records, "gcp:redis/instance:Instance", func(m property.Map) bool { return true })
	require.NotNil(t, inst, "expected a Memorystore Redis instance in Project")
	assert.Equal(t, "STANDARD_HA", inst.inputs.Get("tier").AsString())
	assert.Equal(t, "PRIVATE_SERVICE_ACCESS", inst.inputs.Get("connectMode").AsString())
	assert.Equal(t, "REDIS_7_0", inst.inputs.Get("redisVersion").AsString())
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
			// Extract the resource name (everything after the last "::")
			parts := strings.Split(d, "::")
			if len(parts) == 0 {
				continue
			}
			name := parts[len(parts)-1]
			// Matches exact name (Cloud Run: service name = resource name) or
			// any child resource of a service (Compute Engine: cache-sa, cache-instance-template, etc.)
			if name == dep || strings.HasPrefix(name, dep+"-") {
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

// --- LLM support ---

// llmService returns a service property value with the llm field set and an image.
func llmService(image string) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"image": property.New(image),
		"llm":   property.New(property.NewMap(map[string]property.Value{})),
	}))
}

func TestConstructProjectWithLLMEnablesAiplatformAPI(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"ai": llmService("myai:latest"),
		}),
	})

	require.NoError(t, err)

	api := findTypeWhere(*records, "gcp:projects/service:Service", func(m property.Map) bool {
		return m.Get("service").AsString() == "aiplatform.googleapis.com"
	})
	require.NotNil(t, api, "expected aiplatform.googleapis.com API to be enabled for LLM service")
}

func TestConstructProjectWithoutLLMSkipsAiplatformAPI(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	api := findTypeWhere(*records, "gcp:projects/service:Service", func(m property.Map) bool {
		return m.Get("service").AsString() == "aiplatform.googleapis.com"
	})
	assert.Nil(t, api, "expected no aiplatform.googleapis.com API without LLM service")
}

func TestConstructProjectWithLLMGrantsAiplatformIAM(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"ai": llmService("myai:latest"),
		}),
	})

	require.NoError(t, err)

	iam := findTypeWhere(*records, "gcp:projects/iAMMember:IAMMember", func(m property.Map) bool {
		return m.Get("role").AsString() == "roles/aiplatform.user"
	})
	require.NotNil(t, iam, "expected an IAM member granting roles/aiplatform.user for LLM service")
}

func TestConstructProjectWithoutLLMSkipsAiplatformIAM(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	iam := findTypeWhere(*records, "gcp:projects/iAMMember:IAMMember", func(m property.Map) bool {
		return m.Get("role").AsString() == "roles/aiplatform.user"
	})
	assert.Nil(t, iam, "expected no roles/aiplatform.user IAM binding without LLM service")
}

func TestConstructProjectCreatesServiceAccountForContainerServices(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"web":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)

	// createServiceAccount sets a description containing "Service Account used by run services of"
	saDescription := func(m property.Map) string {
		d := m.Get("description")
		if d.IsNull() {
			return ""
		}
		return d.AsString()
	}
	isNewSA := func(m property.Map) bool {
		return strings.HasPrefix(saDescription(m), "Service Account used by run services of")
	}

	webSA := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		return isNewSA(m) && strings.Contains(saDescription(m), "web")
	})
	require.NotNil(t, webSA, "expected a project-level service account for the web service")

	workerSA := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		return isNewSA(m) && strings.Contains(saDescription(m), "worker")
	})
	require.NotNil(t, workerSA, "expected a project-level service account for the worker service")
}

func TestConstructProjectDoesNotCreateServiceAccountForManagedServices(t *testing.T) {
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

	// Managed Postgres uses Cloud SQL — no project-level service account should be created for it
	sa := findTypeWhere(*records, "gcp:serviceaccount/account:Account", func(m property.Map) bool {
		d := m.Get("description")
		return !d.IsNull() && strings.HasPrefix(d.AsString(), "Service Account used by run services of")
	})
	assert.Nil(t, sa, "expected no project-level service account for a managed Postgres service")
}

// findEnvInCloudRun scans the containers' envs array in a captured Cloud Run resource
// and returns the env entry with the given name, or nil if not found.
func findEnvInCloudRun(crInputs property.Map, name string) *property.Value {
	containers := crInputs.Get("template").AsMap().Get("containers").AsArray()
	for i := range containers.Len() {
		envs := containers.Get(i).AsMap().Get("envs").AsArray()
		for j := range envs.Len() {
			e := envs.Get(j).AsMap()
			if e.Get("name").AsString() == name {
				v := envs.Get(j)
				return &v
			}
		}
	}
	return nil
}

func TestConstructProjectWithLLMInjectsVertexEnvVarsIntoCloudRun(t *testing.T) {
	var capturedCR property.Map
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpCloudRunServiceType {
				capturedCR = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"ai": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myai:latest"),
				"llm":   property.New(property.NewMap(map[string]property.Value{})),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
			})),
		}),
	})

	require.NoError(t, err)
	require.NotEqual(t, 0, capturedCR.Len(), "expected a Cloud Run service to be registered")

	for _, envName := range []string{
		"GOOGLE_VERTEX_PROJECT",
		"GOOGLE_VERTEX_LOCATION",
		"GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_LOCATION",
	} {
		assert.NotNil(t, findEnvInCloudRun(capturedCR, envName),
			"expected %s env var to be injected into Cloud Run service", envName)
	}
}

func TestConstructProjectWithLLMInjectsVertexEnvVarsIntoComputeEngine(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"ai": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myai:latest"),
				"llm":   property.New(property.NewMap(map[string]property.Value{})),
				// no ingress port → Compute Engine
			})),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it, "expected a Compute Engine instance template")
	userData := it.inputs.Get("metadata").AsMap().Get("user-data").AsString()

	for _, envName := range []string{
		"GOOGLE_VERTEX_PROJECT",
		"GOOGLE_VERTEX_LOCATION",
		"GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_LOCATION",
	} {
		assert.Contains(t, userData, envName,
			"expected %s to appear in Compute Engine cloud-init user-data", envName)
	}
}

func TestConstructProjectWithLLMPreservesExistingEnvVars(t *testing.T) {
	var capturedCR property.Map
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			if args.TypeToken == gcpCloudRunServiceType {
				capturedCR = args.Inputs
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"ai": property.New(property.NewMap(map[string]property.Value{
				"image": property.New("myai:latest"),
				"llm":   property.New(property.NewMap(map[string]property.Value{})),
				"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
				"environment": property.New(property.NewMap(map[string]property.Value{
					"GOOGLE_VERTEX_PROJECT": property.New("my-custom-project"),
				})),
			})),
		}),
	})

	require.NoError(t, err)
	require.NotEqual(t, 0, capturedCR.Len(), "expected a Cloud Run service to be registered")

	// enableLLM should not overwrite a pre-existing non-empty value
	env := findEnvInCloudRun(capturedCR, "GOOGLE_VERTEX_PROJECT")
	require.NotNil(t, env, "expected GOOGLE_VERTEX_PROJECT env var to be present")
	assert.Equal(t, "my-custom-project", env.AsMap().Get("value").AsString(),
		"enableLLM should not overwrite a pre-existing GOOGLE_VERTEX_PROJECT value")
}
