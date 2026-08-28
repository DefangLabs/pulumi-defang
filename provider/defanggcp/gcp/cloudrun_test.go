package gcp

import (
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// envNames extracts the static Name of each env var arg for assertions.
func envNames(envs cloudrunv2.ServiceTemplateContainerEnvArray) []string {
	names := make([]string, 0, len(envs))
	for _, e := range envs {
		args := e.(*cloudrunv2.ServiceTemplateContainerEnvArgs)
		names = append(names, string(args.Name.(pulumi.String)))
	}
	return names
}

func TestBuildEnvVarsStripsReservedPort(t *testing.T) {
	tests := []struct {
		name string
		port pulumi.StringInput
	}{
		{"matching port", pulumi.String("8080")},
		{"mismatching port", pulumi.String("9999")},
		{"non-numeric port", pulumi.String("http")},
		{"nil value", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				svc := compose.ServiceConfig{
					Environment: compose.Environment{
						"PORT": tt.port,
						"FOO":  pulumi.String("bar"),
					},
					Ports: []compose.ServicePortConfig{{Target: 8080}},
				}
				envs, secretIds := buildEnvVars(ctx, nil, "app", "etag1", "", svc)
				if len(secretIds) != 0 {
					t.Errorf("expected no secret IDs, got %v", secretIds)
				}
				names := envNames(envs)
				for _, n := range names {
					if n == "PORT" {
						t.Errorf("PORT env var should have been stripped, got %v", names)
					}
				}
				want := map[string]bool{"DEFANG_SERVICE": true, "DEFANG_ETAG": true, "FOO": true}
				for _, n := range names {
					delete(want, n)
				}
				if len(want) != 0 {
					t.Errorf("missing expected env vars %v in %v", want, names)
				}
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// replicasDeploy builds a deploy block with an explicit replica count. Taking the
// address of the parameter gives each case its own *int32.
func replicasDeploy(n int32) *compose.DeployConfig {
	return &compose.DeployConfig{Replicas: &n}
}

// wantNoFloor marks the cases where the service must stay able to scale to zero.
const wantNoFloor = -1

func TestMinInstanceScalingFromReplicas(t *testing.T) {
	tests := []struct {
		name   string
		deploy *compose.DeployConfig
		want   int // wantNoFloor when no service-level scaling block should be built
	}{
		{"no deploy block scales to zero", nil, wantNoFloor},
		{"deploy without replicas scales to zero", &compose.DeployConfig{}, wantNoFloor},
		{"replicas 0 scales to zero", replicasDeploy(0), wantNoFloor},
		{"replicas 1 keeps one instance warm", replicasDeploy(1), 1},
		{"replicas 3 keeps three instances warm", replicasDeploy(3), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minInstanceScaling(compose.ServiceConfig{Deploy: tt.deploy})
			if tt.want == wantNoFloor {
				if got != nil {
					t.Errorf("minInstanceScaling() = %+v, want nil so the service can scale to zero", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("minInstanceScaling() = nil, want MinInstanceCount %d", tt.want)
			}
			count, ok := got.MinInstanceCount.(pulumi.Int)
			if !ok {
				t.Fatalf("MinInstanceCount is %T, want pulumi.Int", got.MinInstanceCount)
			}
			if int(count) != tt.want {
				t.Errorf("MinInstanceCount = %d, want %d", int(count), tt.want)
			}
		})
	}
}

// cloudRunServiceSpy captures the inputs of the Cloud Run service resource, so the
// test asserts what actually reaches the API rather than what a builder returned.
type cloudRunServiceSpy struct {
	mu     sync.Mutex
	inputs resource.PropertyMap
}

func (m *cloudRunServiceSpy) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "gcp:cloudrunv2/service:Service" {
		m.mu.Lock()
		m.inputs = args.Inputs.Copy()
		m.mu.Unlock()
	}
	return args.Name + "_id", args.Inputs, nil
}

func (m *cloudRunServiceSpy) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// The floor has to land on the service, and the ceiling has to stay on the revision
// template. Google's guidance is to keep minimum instances at the service level, and
// the two levels interact: a revision-level maximum below the service-level minimum
// silently caps the number of warm instances.
func TestCreateCloudRunServiceScaling(t *testing.T) {
	tests := []struct {
		name    string
		deploy  *compose.DeployConfig
		wantMin int // wantNoFloor when the service must carry no scaling block
		wantMax float64
	}{
		{"replicas sets a service-level floor and keeps the revision ceiling", replicasDeploy(2), 2, 2},
		{"replicas 0 leaves the service scaling to zero", replicasDeploy(0), wantNoFloor, 1},
		{"no replicas leaves the service scaling to zero", nil, wantNoFloor, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &cloudRunServiceSpy{}
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				parent, err := compute.NewNetwork(ctx, "test-parent", &compute.NetworkArgs{})
				if err != nil {
					return err
				}
				_, err = CreateCloudRunService(ctx, nil, "app", pulumi.String("img"),
					compose.ServiceConfig{
						Deploy: tt.deploy,
						Ports:  []compose.ServicePortConfig{{Target: 8080}},
					},
					&ServiceIdentity{Email: pulumi.String("sa@example.com")},
					testInfra(ctx), nil, pulumi.Parent(parent))
				return err
			}, pulumi.WithMocks("proj", "stack", spy))
			if err != nil {
				t.Fatal(err)
			}
			if spy.inputs == nil {
				t.Fatal("no Cloud Run service was registered")
			}

			scaling := spy.inputs["scaling"]
			if tt.wantMin == wantNoFloor {
				if !scaling.IsNull() {
					t.Errorf("service scaling = %v, want none so the service can scale to zero", scaling)
				}
			} else {
				if !scaling.IsObject() {
					t.Fatalf("service scaling = %v, want an object with minInstanceCount", scaling)
				}
				got := scaling.ObjectValue()["minInstanceCount"]
				if !got.IsNumber() || got.NumberValue() != float64(tt.wantMin) {
					t.Errorf("service minInstanceCount = %v, want %d", got, tt.wantMin)
				}
			}

			// The ceiling must stay where it was, on the revision template.
			template := spy.inputs["template"]
			if !template.IsObject() {
				t.Fatalf("template = %v, want an object", template)
			}
			templateScaling := template.ObjectValue()["scaling"]
			if !templateScaling.IsObject() {
				t.Fatalf("template scaling = %v, want an object", templateScaling)
			}
			gotMax := templateScaling.ObjectValue()["maxInstanceCount"]
			if !gotMax.IsNumber() || gotMax.NumberValue() != tt.wantMax {
				t.Errorf("template maxInstanceCount = %v, want %v", gotMax, tt.wantMax)
			}
			if min := templateScaling.ObjectValue()["minInstanceCount"]; min.IsNumber() && min.NumberValue() != 0 {
				t.Errorf("template minInstanceCount = %v, want the floor on the service, not the revision", min)
			}
		})
	}
}
