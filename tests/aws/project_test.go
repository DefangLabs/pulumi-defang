package aws

// Project is the top-level orchestration component for AWS. These tests verify
// that the Project component correctly wires up a set of services using the
// mock resource monitor. Detailed behaviour of each sub-component (ECS service,
// Postgres, Build, etc.) lives in their own dedicated test files.
//
// Note: services that embed environment variables inside ECS container-
// definition JSON hit a mock-monitor limitation (StringOutputs cannot be
// marshaled to JSON). Those cases are covered in the per-service test files
// which can supply a richer mock.

import (
	"encoding/json"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsprov "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsProject(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})

	require.NoError(t, err)
}

// TestConstructAwsProjectFoldsSidecars verifies that a service with
// network_mode: "service:<name>" is folded into the parent's task definition
// as an additional container instead of being deployed as its own ECS service.
func TestConstructAwsProjectFoldsSidecars(t *testing.T) {
	var taskDefs []property.Map
	ecsServices := 0
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			switch args.TypeToken {
			case "aws:ecs/taskDefinition:TaskDefinition":
				taskDefs = append(taskDefs, args.Inputs)
			case "aws:ecs/service:Service":
				ecsServices++
			default:
			}
			return args.Name, args.Inputs, nil
		},
	}
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))

	resp, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app": property.New(property.NewMap(map[string]property.Value{
				"image":       property.New("myapp:latest"),
				"volumesFrom": property.New(property.NewArray([]property.Value{property.New("helper")})),
			})),
			"helper": property.New(property.NewMap(map[string]property.Value{
				"image":       property.New("helper:latest"),
				"networkMode": property.New("service:app"),
			})),
		}),
	})
	require.NoError(t, err)

	assert.Equal(t, 1, ecsServices, "sidecar must not become its own ECS service")
	require.Len(t, taskDefs, 1)
	var defs []awsprov.ContainerDefinition
	require.NoError(t, json.Unmarshal([]byte(taskDefs[0].Get("containerDefinitions").AsString()), &defs))
	require.Len(t, defs, 2, "task definition should hold the main container plus the sidecar")
	names := []string{defs[0].Name, defs[1].Name}
	assert.Contains(t, names, "app")
	assert.Contains(t, names, "helper")

	// The sidecar must not surface in the endpoints output.
	_, hasHelper := resp.State.Get("endpoints").AsMap().GetOk("helper")
	assert.False(t, hasHelper, "sidecar should not have an endpoint")
}

// TestConstructAwsProjectSidecarErrors covers invalid sidecar wiring.
func TestConstructAwsProjectSidecarErrors(t *testing.T) {
	sidecar := func(parent string) property.Value {
		return property.New(property.NewMap(map[string]property.Value{
			"image":       property.New("helper:latest"),
			"networkMode": property.New("service:" + parent),
		}))
	}

	tests := map[string]struct {
		services map[string]property.Value
		wantErr  string
	}{
		"unknown parent": {
			services: map[string]property.Value{"helper": sidecar("missing")},
			wantErr:  "sidecar parent service not found",
		},
		"parent is itself a sidecar": {
			services: map[string]property.Value{
				"app":    testutil.ServiceWithImage("myapp:latest"),
				"mid":    sidecar("app"),
				"helper": sidecar("mid"),
			},
			wantErr: "sidecar parent is itself a sidecar",
		},
		"parent is managed": {
			services: map[string]property.Value{
				"cache": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("redis:7"),
					"redis": property.New(property.NewMap(map[string]property.Value{})),
				})),
				"helper": sidecar("cache"),
			},
			wantErr: "must be a container service",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := testutil.MakeAwsTestServer()
			_, err := server.Construct(p.ConstructRequest{
				Urn:    testutil.AwsURN("Project"),
				Inputs: testutil.ServicesMap(tc.services),
			})
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// TestConstructAwsProjectInfraOutputs verifies the Project surfaces handles to
// its shared infrastructure so externally managed resources (WAF, alarms,
// resource-based policies) can attach to it.
func TestConstructAwsProjectInfraOutputs(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	resp, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"app":    testutil.ServiceWithPorts("nginx:latest", testutil.IngressPort(8080)),
			"worker": testutil.ServiceWithImage("myapp:worker"),
		}),
	})
	require.NoError(t, err)

	// The mock state keeps the raw pulumi tag for optional fields, hence
	// "loadBalancerArn,optional" (same quirk as the pre-existing
	// "loadBalancerDns,optional").
	for _, key := range []string{"clusterName", "logGroupName", "loadBalancerArn,optional"} {
		_, ok := resp.State.GetOk(key)
		assert.True(t, ok, "missing output %q", key)
	}
	serviceNames := resp.State.Get("serviceNames").AsMap()
	taskRoleArns := resp.State.Get("taskRoleArns").AsMap()
	for _, svc := range []string{"app", "worker"} {
		_, ok := serviceNames.GetOk(svc)
		assert.True(t, ok, "serviceNames missing %q", svc)
		_, ok = taskRoleArns.GetOk(svc)
		assert.True(t, ok, "taskRoleArns missing %q", svc)
	}
}

// TestConstructAwsProjectAllResourcesAreChildren asserts that every resource
// created inside a Project descends from the Project component in the Pulumi
// hierarchy. Runs a rich Construct that exercises shared infra (VPC, ALB,
// DNS), container services, build-from-source, managed Postgres, and managed
// Redis so the assertion covers most resource-creation paths.
func TestConstructAwsProjectAllResourcesAreChildren(t *testing.T) {
	mock, tracker := testutil.NewParentTracker()
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("Project"),
		Inputs: property.NewMap(map[string]property.Value{
			"domain": property.New("example.com"),
			"services": property.New(property.NewMap(map[string]property.Value{
				"app": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("nginx:latest"),
					"ports": property.New(property.NewArray([]property.Value{testutil.IngressPort(8080)})),
				})),
				"worker": testutil.ServiceWithImage("myapp:worker"),
				"builder": property.New(property.NewMap(map[string]property.Value{
					"build": property.New(property.NewMap(map[string]property.Value{
						"context": property.New("./app"),
					})),
				})),
				"db": property.New(property.NewMap(map[string]property.Value{
					"image":    property.New("postgres:17"),
					"postgres": property.New(property.NewMap(map[string]property.Value{})),
				})),
				"cache": property.New(property.NewMap(map[string]property.Value{
					"image": property.New("redis:7"),
					"redis": property.New(property.NewMap(map[string]property.Value{})),
				})),
			})),
		}),
	})
	require.NoError(t, err)

	tracker.AssertAllDescendFrom(t, testutil.AwsURN("Project"))
}
