package tests

import (
	"context"
	"errors"

	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/cli/client/byoc/aws"
	"github.com/DefangLabs/defang/src/pkg/types"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MockSubscribeServerStream struct {
	Resps []*defangv1.SubscribeResponse
	Error error
}

func (*MockSubscribeServerStream) Close() error {
	return nil
}

func (m *MockSubscribeServerStream) Receive() bool {
	return false
}

func (m *MockSubscribeServerStream) Msg() *defangv1.SubscribeResponse {
	return nil
}

func (m *MockSubscribeServerStream) Err() error {
	return connect.NewError(connect.CodeCanceled, errors.New("cancel connect error")) // cancel the connection after 5 retries to avoid infinite loop
}

type MockFollowServerStream struct {
	Resps []*defangv1.TailResponse
	Error error
}

func (*MockFollowServerStream) Close() error {
	return nil
}

func (m *MockFollowServerStream) Receive() bool {
	return false
}

func (m *MockFollowServerStream) Msg() *defangv1.TailResponse {
	return nil
}

func (m *MockFollowServerStream) Err() error {
	return connect.NewError(connect.CodeCanceled, errors.New("cancel connect error")) // cancel the connection after 5 retries to avoid infinite loop
}

type CloudProviderMock struct{}

func (c CloudProviderMock) AccountInfo(context.Context) (client.AccountInfo, error) {
	return aws.AWSAccountInfo{}, nil
}

func (c CloudProviderMock) BootstrapCommand(context.Context, client.BootstrapCommandRequest) (string, error) {
	return "", nil
}

func (c CloudProviderMock) BootstrapList(context.Context) ([]string, error) {
	return nil, nil
}

func (c CloudProviderMock) CreateUploadURL(context.Context, *defangv1.UploadURLRequest) (*defangv1.UploadURLResponse, error) {
	return nil, nil
}

func (c CloudProviderMock) PrepareDomainDelegation(context.Context, client.PrepareDomainDelegationRequest) (*client.PrepareDomainDelegationResponse, error) {
	return nil, nil
}

func (c CloudProviderMock) Delete(context.Context, *defangv1.DeleteRequest) (*defangv1.DeleteResponse, error) {
	return nil, nil
}

func (c CloudProviderMock) DeleteConfig(context.Context, *defangv1.Secrets) error {
	return nil
}

func (c CloudProviderMock) Deploy(context.Context, *defangv1.DeployRequest) (*defangv1.DeployResponse, error) {
	return &defangv1.DeployResponse{
		Etag: "abc123",
	}, nil
}

func (c CloudProviderMock) DelayBeforeRetry(context.Context) error {
	return nil
}

func (c CloudProviderMock) Destroy(context.Context, *defangv1.DestroyRequest) (types.ETag, error) {
	return types.ETag("abc123"), nil
}

func (c CloudProviderMock) Follow(context.Context, *defangv1.TailRequest) (client.ServerStream[defangv1.TailResponse], error) {
	resps := []*defangv1.TailResponse{
		{
			Service: "service1",
			Etag:    "abc123",
			Entries: []*defangv1.LogEntry{
				{
					Timestamp: timestamppb.Now(),
					Message:   "info message",
				},
			},
		},
	}
	stream := &MockFollowServerStream{Resps: resps}
	return stream, nil
}

func (c CloudProviderMock) GetService(context.Context, *defangv1.GetRequest) (*defangv1.ServiceInfo, error) {
	return nil, nil
}

func (c CloudProviderMock) GetServices(context.Context, *defangv1.GetServicesRequest) (*defangv1.GetServicesResponse, error) {
	return nil, nil
}

func (c CloudProviderMock) ListConfig(context.Context, *defangv1.ListConfigsRequest) (*defangv1.Secrets, error) {
	return &defangv1.Secrets{
		Names: []string{},
	}, nil
}

func (c CloudProviderMock) Query(context.Context, *defangv1.DebugRequest) error {
	return nil
}

func (c CloudProviderMock) Preview(context.Context, *defangv1.DeployRequest) (*defangv1.DeployResponse, error) {
	return nil, nil
}

func (c CloudProviderMock) PutConfig(context.Context, *defangv1.PutConfigRequest) error {
	return nil
}

func (c CloudProviderMock) RemoteProjectName(context.Context) (string, error) {
	return "", nil
}

func (c CloudProviderMock) ServiceDNS(string) string {
	return ""
}

func (c CloudProviderMock) SetCDImage(string) {
}

func (c CloudProviderMock) Subscribe(context.Context, *defangv1.SubscribeRequest) (client.ServerStream[defangv1.SubscribeResponse], error) {
	resps := []*defangv1.SubscribeResponse{
		{
			Name:  "service1",
			State: defangv1.ServiceState_DEPLOYMENT_COMPLETED,
		},
	}
	stream := &MockSubscribeServerStream{Resps: resps}
	return stream, nil
}

func (c CloudProviderMock) TearDown(context.Context) error {
	return nil
}
