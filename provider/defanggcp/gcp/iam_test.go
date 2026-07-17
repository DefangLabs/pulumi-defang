package gcp

import "testing"

func TestResolvePolicyRole(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		want   string
	}{
		{"predefined role", "roles/run.developer", "roles/run.developer"},
		{"project custom role", "projects/other/roles/deployer", "projects/other/roles/deployer"},
		{"organization custom role", "organizations/123/roles/deployer", "organizations/123/roles/deployer"},
		{"bare name resolves in current project", "deployer", "projects/myproj/roles/deployer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolvePolicyRole("myproj", tt.policy); got != tt.want {
				t.Errorf("ResolvePolicyRole(%q) = %q, want %q", tt.policy, got, tt.want)
			}
		})
	}
}
