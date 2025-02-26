package provider

import (
	"context"
	"fmt"

	"github.com/DefangLabs/defang/src/pkg/cli"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
)

type IDriver interface {
	client.Provider
	client.FabricClient

	GetProvider() client.Provider
	GetFabricClient() client.FabricClient
}

type Driver struct {
	client.Provider
	client.FabricClient
}

var ActiveDriver IDriver

func NewDriver(ctx context.Context, providerID client.ProviderID) (IDriver, error) {
	if ActiveDriver != nil {
		return ActiveDriver, nil
	}

	fabric := cli.NewGrpcClient(ctx, cli.DefangFabric)
	provider, err := cli.NewProvider(ctx, providerID, fabric)
	if err != nil {
		return nil, fmt.Errorf("failed to create defang cloud provider: %w", err)
	}

	return &Driver{
		Provider:     provider,
		FabricClient: fabric,
	}, nil
}

func (d *Driver) GetProvider() client.Provider {
	return d.Provider
}

func (d *Driver) GetFabricClient() client.FabricClient {
	return d.FabricClient
}
