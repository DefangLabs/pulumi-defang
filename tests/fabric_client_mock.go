package tests

import (
	"context"

	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/types"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/defang/src/protos/io/defang/v1/defangv1connect"
)

type FabricClientMock struct{}

func (m FabricClientMock) AgreeToS(context.Context) error {
	return nil
}

func (m FabricClientMock) CanIUse(context.Context, *defangv1.CanIUseRequest) (*defangv1.CanIUseResponse, error) {
	return &defangv1.CanIUseResponse{
		CdImage: "test-cd-image",
	}, nil
}

func (m FabricClientMock) CheckLoginAndToS(context.Context) error {
	return nil
}

func (m FabricClientMock) Debug(context.Context, *defangv1.DebugRequest) (*defangv1.DebugResponse, error) {
	return &defangv1.DebugResponse{}, nil
}

func (m FabricClientMock) DelegateSubdomainZone(
	context.Context,
	*defangv1.DelegateSubdomainZoneRequest,
) (*defangv1.DelegateSubdomainZoneResponse, error) {
	return &defangv1.DelegateSubdomainZoneResponse{
		Zone: "test-zone",
	}, nil
}

func (m FabricClientMock) DeleteSubdomainZone(context.Context) error {
	return nil
}

func (m FabricClientMock) GenerateFiles(
	context.Context,
	*defangv1.GenerateFilesRequest,
) (*defangv1.GenerateFilesResponse, error) {
	return &defangv1.GenerateFilesResponse{}, nil
}

func (m FabricClientMock) GetDelegateSubdomainZone(context.Context) (*defangv1.DelegateSubdomainZoneResponse, error) {
	return &defangv1.DelegateSubdomainZoneResponse{
		Zone: "test-zone",
	}, nil
}

func (m FabricClientMock) GetSelectedProvider(
	context.Context,
	*defangv1.GetSelectedProviderRequest,
) (*defangv1.GetSelectedProviderResponse, error) {
	return &defangv1.GetSelectedProviderResponse{}, nil
}

func (m FabricClientMock) GetTenantName() types.TenantName {
	return types.TenantName("test-tenant")
}

func (m FabricClientMock) GetController() defangv1connect.FabricControllerClient {
	return nil
}

func (m FabricClientMock) GetVersions(context.Context) (*defangv1.Version, error) {
	return &defangv1.Version{}, nil
}

func (m FabricClientMock) Publish(context.Context, *defangv1.PublishRequest) error {
	return nil
}

func (m FabricClientMock) PutDeployment(context.Context, *defangv1.PutDeploymentRequest) error {
	return nil
}

func (m FabricClientMock) ListDeployments(
	context.Context,
	*defangv1.ListDeploymentsRequest,
) (*defangv1.ListDeploymentsResponse, error) {
	return &defangv1.ListDeploymentsResponse{}, nil
}

func (m FabricClientMock) RevokeToken(context.Context) error {
	return nil
}

func (m FabricClientMock) SetSelectedProvider(context.Context, *defangv1.SetSelectedProviderRequest) error {
	return nil
}

func (m FabricClientMock) Token(context.Context, *defangv1.TokenRequest) (*defangv1.TokenResponse, error) {
	return &defangv1.TokenResponse{}, nil
}

func (m FabricClientMock) Track(string, ...client.Property) error {
	return nil
}

func (m FabricClientMock) VerifyDNSSetup(context.Context, *defangv1.VerifyDNSSetupRequest) error {
	return nil
}

func (m FabricClientMock) WhoAmI(context.Context) (*defangv1.WhoAmIResponse, error) {
	return &defangv1.WhoAmIResponse{}, nil
}
