package tests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/cli/client/byoc/aws"
	"github.com/DefangLabs/defang/src/pkg/types"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MockSubscribeServerStream struct {
	err      error
	msg      *defangv1.SubscribeResponse
	received bool
}

func (s *MockSubscribeServerStream) Close() error {
	return nil
}

func (s *MockSubscribeServerStream) Err() error {
	return s.err
}

func (s *MockSubscribeServerStream) Msg() *defangv1.SubscribeResponse {
	return s.msg
}

func (s *MockSubscribeServerStream) Receive() bool {
	if s.received {
		return false
	}

	s.received = true
	return s.received
}

type MockFollowServerStream struct {
	err      error
	msg      *defangv1.TailResponse
	received bool
	read     bool
	closed   bool
}

func (s *MockFollowServerStream) Close() error {
	s.closed = true
	return nil
}

func (s *MockFollowServerStream) Err() error {
	return s.err
}

func (s *MockFollowServerStream) Msg() *defangv1.TailResponse {
	if s.read {
		return nil
	}
	s.read = true

	return s.msg
}

func (s *MockFollowServerStream) Receive() bool {
	if s.received {
		return false
	}

	s.received = true
	return s.received
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

func (c CloudProviderMock) CreateUploadURL(
	context.Context,
	*defangv1.UploadURLRequest,
) (*defangv1.UploadURLResponse, error) {
	return &defangv1.UploadURLResponse{}, nil
}

func (c CloudProviderMock) PrepareDomainDelegation(
	context.Context,
	client.PrepareDomainDelegationRequest,
) (*client.PrepareDomainDelegationResponse, error) {
	return &client.PrepareDomainDelegationResponse{}, nil
}

func (c CloudProviderMock) Delete(context.Context, *defangv1.DeleteRequest) (*defangv1.DeleteResponse, error) {
	return &defangv1.DeleteResponse{}, nil
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

func (c CloudProviderMock) Follow(
	context.Context,
	*defangv1.TailRequest,
) (client.ServerStream[defangv1.TailResponse], error) {
	msg := defangv1.TailResponse{
		Service: "service1",
		Etag:    "abc123",
		Entries: []*defangv1.LogEntry{
			{
				Timestamp: timestamppb.Now(),
				Message:   "info message",
			},
		},
	}
	stream := &MockFollowServerStream{msg: &msg}
	return stream, nil
}

func (c CloudProviderMock) GetProjectUpdate(context.Context, string) (*defangv1.ProjectUpdate, error) {
	taskRole := "service1-role"
	projectOutputs := provider.V1DefangProjectOutputs{
		Services: map[string]provider.V1DefangServiceOutputs{
			"service1": {
				ID:       "service1-id",
				TaskRole: &taskRole,
			},
		},
	}

	projectOutputsJSON, err := json.Marshal(projectOutputs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal project outputs: %w", err)
	}

	return &defangv1.ProjectUpdate{
		AlbArn: "arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188",
		Services: []*defangv1.ServiceInfo{
			{
				Service: &defangv1.Service{
					Name: "service1",
				},
				Etag: "abc123",
			},
		},
		ProjectOutputsVersion: 1,
		ProjectOutputs:        projectOutputsJSON,
	}, nil
}

func (c CloudProviderMock) GetService(context.Context, *defangv1.GetRequest) (*defangv1.ServiceInfo, error) {
	return &defangv1.ServiceInfo{}, nil
}

func (c CloudProviderMock) GetServices(
	context.Context,
	*defangv1.GetServicesRequest,
) (*defangv1.GetServicesResponse, error) {
	return &defangv1.GetServicesResponse{}, nil
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
	return &defangv1.DeployResponse{}, nil
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

func (c CloudProviderMock) SetCanIUseConfig(*defangv1.CanIUseResponse) {
	// No-op
}

func (c CloudProviderMock) Subscribe(
	context.Context,
	*defangv1.SubscribeRequest,
) (client.ServerStream[defangv1.SubscribeResponse], error) {
	msg := defangv1.SubscribeResponse{
		Name:  "app",
		State: defangv1.ServiceState_DEPLOYMENT_COMPLETED,
	}
	stream := &MockSubscribeServerStream{msg: &msg}
	return stream, nil
}

func (c CloudProviderMock) TearDown(context.Context) error {
	return nil
}

func (c CloudProviderMock) QueryForDebug(context.Context, *defangv1.DebugRequest) error {
	return nil
}

func (c CloudProviderMock) QueryLogs(
	context.Context,
	*defangv1.TailRequest,
) (client.ServerStream[defangv1.TailResponse], error) {
	msg := defangv1.TailResponse{
		Service: "service1",
		Etag:    "abc123",
		Entries: []*defangv1.LogEntry{
			{
				Timestamp: timestamppb.Now(),
				Message:   "info message",
			},
		},
	}
	stream := &MockFollowServerStream{msg: &msg}
	return stream, nil
}
