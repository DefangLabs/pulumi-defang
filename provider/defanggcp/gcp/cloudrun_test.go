package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
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
