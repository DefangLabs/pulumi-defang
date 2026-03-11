package tests

import (
	"context"

	defang "github.com/DefangLabs/pulumi-defang/provider"
	"github.com/blang/semver"
	"github.com/pulumi/pulumi-go-provider/integration"
)

// Create a test server.
func makeTestServer() integration.Server {
	server, err := integration.NewServer(context.Background(), defang.Name, semver.MustParse("1.0.0"),
		integration.WithProvider(defang.Provider()),
	)
	if err != nil {
		panic(err)
	}
	return server
}
