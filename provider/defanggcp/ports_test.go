package defanggcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
)

func TestFirstIngressPort(t *testing.T) {
	tests := []struct {
		name        string
		ports       []compose.ServicePortConfig
		defaultPort int32
		want        int32
	}{
		{
			name:        "no ports defaults to 5432",
			ports:       nil,
			defaultPort: 5432,
			want:        5432,
		},
		{
			name:        "empty ports defaults to 5432",
			ports:       []compose.ServicePortConfig{},
			defaultPort: 5432,
			want:        5432,
		},
		{
			name:        "zero target defaults to 5432",
			ports:       []compose.ServicePortConfig{{Target: 0}},
			defaultPort: 5432,
			want:        5432,
		},
		{
			name:        "custom port is used",
			ports:       []compose.ServicePortConfig{{Target: 5433}},
			defaultPort: 5432,
			want:        5433,
		},
		{
			name:        "first port is used when multiple are defined",
			ports:       []compose.ServicePortConfig{{Target: 5433}, {Target: 5434}},
			defaultPort: 5432,
			want:        5433,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, firstPort(tc.ports, tc.defaultPort))
		})
	}
}
