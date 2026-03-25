package testutil

import (
	"context"

	defangaws "github.com/DefangLabs/pulumi-defang/provider/defangaws"
	defangazure "github.com/DefangLabs/pulumi-defang/provider/defangazure"
	defanggcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp"
	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
)

func MakeTestServer() integration.Server {
	return MustNewServer(defangaws.Name, defangaws.Provider())
}

func MakeAzureTestServer() integration.Server {
	return MustNewServer(defangazure.Name, defangazure.Provider())
}

func MakeGcpTestServer(opts ...integration.ServerOption) integration.Server {
	return MustNewServer(defanggcp.Name, defanggcp.Provider(), opts...)
}

func MustNewServer(name string, provider p.Provider, opts ...integration.ServerOption) integration.Server {
	opts = append([]integration.ServerOption{integration.WithProvider(provider)}, opts...)
	server, err := integration.NewServer(context.Background(), name, semver.MustParse("1.0.0"), opts...)
	if err != nil {
		panic(err)
	}
	return server
}
