package compose

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// Child resource kinds addressable via the x-defang-aliases extension. Each
// provider applies the kinds relevant to the resources it creates for a
// service.
const (
	AliasCluster        = "cluster"
	AliasParameterGroup = "parameter-group"
	AliasSecurityGroup  = "security-group"
	AliasSubnetGroup    = "subnet-group"
)

// AliasOptions returns pulumi.Aliases resource options for the given child
// resource kind when x-defang-aliases configures a pre-migration URN for it;
// empty otherwise. Aliases must be applied where the child is registered
// (inside the provider): resource options passed to a remote component do not
// propagate to its children.
func (s *ServiceConfig) AliasOptions(kind string) []pulumi.ResourceOption {
	urn, ok := s.Aliases[kind]
	if !ok || urn == "" {
		return nil
	}
	return []pulumi.ResourceOption{pulumi.Aliases([]pulumi.Alias{{URN: pulumi.URN(urn)}})}
}
