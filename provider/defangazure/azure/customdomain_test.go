package azure

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// TestCreateCustomDomainShortCircuits checks the no-op branches that don't
// touch Pulumi: nil infra, empty Domain, and a service without ingress ports.
// Exercising these in plain unit form keeps the helper safely callable from
// CreateContainerApp on every service without conditional plumbing at the
// call site.
func TestCreateCustomDomainShortCircuits(t *testing.T) {
	ingressSvc := compose.ServiceConfig{
		Ports: []compose.ServicePortConfig{{Target: 80, Mode: "ingress"}},
	}
	internalSvc := compose.ServiceConfig{}

	tests := []struct {
		name  string
		infra *SharedInfra
		svc   compose.ServiceConfig
	}{
		{name: "nil infra", infra: nil, svc: ingressSvc},
		{name: "empty domain", infra: &SharedInfra{Domain: ""}, svc: ingressSvc},
		{name: "no ingress ports", infra: &SharedInfra{Domain: "x.example.com"}, svc: internalSvc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				got, err := CreateCustomDomain(ctx, "svc", tt.svc, nil, tt.infra)
				if err != nil {
					t.Errorf("CreateCustomDomain err: %v", err)
				}
				if got != nil {
					t.Errorf("CreateCustomDomain result = %+v, want nil", got)
				}
				return nil
			}, pulumi.WithMocks("project", "stack", &azureNoopMocks{}))
			if err != nil {
				t.Fatalf("pulumi.RunErr: %v", err)
			}
		})
	}
}
