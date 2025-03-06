package tests

import (
	defang "github.com/DefangLabs/pulumi-defang/provider"
	"github.com/blang/semver"
	"github.com/pulumi/pulumi-go-provider/integration"
)

// Create a test server.
func makeTestServer() integration.Server {
	defang.ActiveDriver = &defang.Driver{
		Provider:     &CloudProviderMock{},
		FabricClient: &FabricClientMock{},
	}

	return integration.NewServer(defang.Name, semver.MustParse("1.0.0"), defang.Provider())
}
