package gcp

// Host-mode Compute Engine services: a public service's ports are open to the
// internet through its own external address; a private one accepts traffic
// only from the project subnet and reaches the internet through Cloud NAT.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

const projectSubnetCidr = "10.0.0.0/16"

type hostModeRun struct{ records []resourceRecord }

func constructHostMode(t *testing.T, inputs property.Map) *hostModeRun {
	t.Helper()
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{Urn: testutil.GcpURN("Project"), Inputs: inputs})
	require.NoError(t, err)
	return &hostModeRun{records: *records}
}

func (r *hostModeRun) named(t *testing.T, typ, name string) property.Map {
	t.Helper()
	for _, rec := range r.records {
		if rec.typ == typ && rec.name == name {
			return rec.inputs
		}
	}
	t.Fatalf("no %s named %s", typ, name)
	return property.Map{}
}

// trafficSources returns the source ranges of the service's port firewall.
func (r *hostModeRun) trafficSources(t *testing.T, service string) []string {
	t.Helper()
	var out []string
	for _, v := range r.named(t, gcpFirewallType, service).Get("sourceRanges").AsArray().All {
		out = append(out, v.AsString())
	}
	return out
}

// externalIP reports whether the service's instances get an external address.
func (r *hostModeRun) externalIP(t *testing.T, service string) bool {
	t.Helper()
	nic := r.named(t, "gcp:compute/instanceTemplate:InstanceTemplate", service).
		Get("networkInterfaces").AsArray().Get(0).AsMap()
	configs := nic.Get("accessConfigs")
	return configs.IsArray() && configs.AsArray().Len() > 0
}

func hostService(fields map[string]property.Value) property.Value {
	m := map[string]property.Value{
		"image": property.New("myapp:latest"),
		"ports": property.New(property.NewArray([]property.Value{testutil.HostPort(4000)})),
	}
	for k, v := range fields {
		m[k] = v
	}
	return property.New(property.NewMap(m))
}

func TestPublicHostModeServiceIsOpen(t *testing.T) {
	run := constructHostMode(t, testutil.ServicesMap(map[string]property.Value{
		"game": hostService(nil),
	}))
	assert.Equal(t, []string{"0.0.0.0/0"}, run.trafficSources(t, "game"))
	assert.True(t, run.externalIP(t, "game"))
}

func TestPrivateHostModeServiceAcceptsOnlyTheSubnet(t *testing.T) {
	// Not in the default network: reachable only by the project's services.
	run := constructHostMode(t, property.NewMap(map[string]property.Value{
		"services": property.New(property.NewMap(map[string]property.Value{
			"backend": hostService(map[string]property.Value{
				"networks": property.New(property.NewMap(map[string]property.Value{
					"private": property.New(property.NewMap(map[string]property.Value{})),
				})),
			}),
		})),
		"networks": nonInternalNetworks("private"),
	}))
	assert.Equal(t, []string{projectSubnetCidr}, run.trafficSources(t, "backend"))
	assert.False(t, run.externalIP(t, "backend"), "egress goes through Cloud NAT")
	assert.Equal(t, 1, countType(run.records, "gcp:compute/routerNat:RouterNat"))
}

func TestModelGatewayIsPrivate(t *testing.T) {
	// A managed model gateway serves the project's services, so it is private
	// even with a host port in the default network.
	run := constructHostMode(t, testutil.ServicesMap(map[string]property.Value{
		"chat": hostService(map[string]property.Value{
			"llm": property.New(property.NewMap(map[string]property.Value{})),
		}),
	}))
	assert.Equal(t, []string{projectSubnetCidr}, run.trafficSources(t, "chat"))
	assert.False(t, run.externalIP(t, "chat"), "egress goes through Cloud NAT")
	assert.Equal(t, 1, countType(run.records, "gcp:compute/routerNat:RouterNat"))

	// The health check firewall still admits Google's probers.
	hc := run.named(t, gcpFirewallType, "chat-hc").Get("sourceRanges").AsArray()
	assert.Equal(t, "130.211.0.0/22", hc.Get(0).AsString())
}

func TestPrivateServiceWithoutEgressKeepsItsAddress(t *testing.T) {
	// An internal network has no Cloud NAT, so the instance keeps its external
	// address for image pulls; its ports are still closed to the internet.
	run := constructHostMode(t, property.NewMap(map[string]property.Value{
		"services": property.New(property.NewMap(map[string]property.Value{
			"backend": hostService(map[string]property.Value{
				"networks": property.New(property.NewMap(map[string]property.Value{
					"private": property.New(property.NewMap(map[string]property.Value{})),
				})),
			}),
		})),
		"networks": property.New(property.NewMap(map[string]property.Value{
			"private": property.New(property.NewMap(map[string]property.Value{"internal": property.New(true)})),
		})),
	}))
	assert.Equal(t, []string{projectSubnetCidr}, run.trafficSources(t, "backend"))
	assert.True(t, run.externalIP(t, "backend"))
	assert.Equal(t, 0, countType(run.records, "gcp:compute/routerNat:RouterNat"))
}
