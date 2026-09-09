package azure

import (
	"reflect"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIngressSetsHttp2TransportForExplicitGrpcAndHttp2(t *testing.T) {
	tests := []struct {
		name        string
		appProtocol compose.PortAppProtocol
	}{
		{name: "grpc", appProtocol: compose.PortAppProtocolGRPC},
		{name: "http2", appProtocol: compose.PortAppProtocolHTTP2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingress := buildIngress(compose.ServiceConfig{
				Ports: []compose.ServicePortConfig{{
					Target:      3005,
					Mode:        compose.PortModeIngress,
					AppProtocol: tt.appProtocol,
				}},
			}, nil)

			require.NotNil(t, ingress)
			assert.Equal(t, "http2", stringPtrInputValue(t, ingress.Transport))
		})
	}
}

func TestBuildIngressPreservesDefaultTransport(t *testing.T) {
	tests := []struct {
		name string
		port compose.ServicePortConfig
	}{
		{
			name: "implicit http app protocol",
			port: compose.ServicePortConfig{
				Target: 3000,
				Mode:   compose.PortModeIngress,
			},
		},
		{
			name: "explicit http app protocol",
			port: compose.ServicePortConfig{
				Target:      3000,
				Mode:        compose.PortModeIngress,
				AppProtocol: compose.PortAppProtocolHTTP,
			},
		},
		{
			name: "tcp transport protocol",
			port: compose.ServicePortConfig{
				Target:   5432,
				Mode:     compose.PortModeIngress,
				Protocol: compose.PortProtocolTCP,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingress := buildIngress(compose.ServiceConfig{Ports: []compose.ServicePortConfig{tt.port}}, nil)

			require.NotNil(t, ingress)
			assert.Nil(t, ingress.Transport)
		})
	}
}

func stringPtrInputValue(t *testing.T, input any) string {
	t.Helper()
	require.NotNil(t, input)

	value := reflect.ValueOf(input)
	require.Equal(t, reflect.Ptr, value.Kind())
	require.False(t, value.IsNil())
	require.Equal(t, reflect.String, value.Elem().Kind())
	return value.Elem().String()
}
