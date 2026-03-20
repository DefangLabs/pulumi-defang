package tests

import (
	"context"

	defangaws "github.com/DefangLabs/pulumi-defang/provider/defangaws"
	defangazure "github.com/DefangLabs/pulumi-defang/provider/defangazure"
	defanggcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp"
	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
)

func makeTestServer() integration.Server {
	return mustNewServer(defangaws.Name, defangaws.Provider())
}

func makeAzureTestServer() integration.Server {
	return mustNewServer(defangazure.Name, defangazure.Provider())
}

func makeGcpTestServer() integration.Server {
	return mustNewServer(defanggcp.Name, defanggcp.Provider())
}

func mustNewServer(name string, provider p.Provider) integration.Server {
	server, err := integration.NewServer(context.Background(), name, semver.MustParse("1.0.0"),
		integration.WithProvider(provider),
	)
	if err != nil {
		panic(err)
	}
	return server
}
