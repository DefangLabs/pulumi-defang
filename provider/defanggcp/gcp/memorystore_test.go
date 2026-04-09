package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
)

func TestGetRedisVersion(t *testing.T) {
	tests := []struct {
		tag  string
		want *string
	}{
		{"7", strPtr("REDIS_7_0")},
		{"7.0", strPtr("REDIS_7_0")},
		{"7.2", strPtr("REDIS_7_2")},
		{"6", strPtr("REDIS_6_X")},
		{"6.2", strPtr("REDIS_6_X")},
		{"6.2.6", strPtr("REDIS_6_X")},
		{"", nil},
		{"latest", nil},
		{"invalid", nil},
	}

	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			got := getRedisVersion(tc.tag)
			if tc.want == nil {
				assert.Nil(t, got)
			} else if assert.NotNil(t, got) {
				assert.Equal(t, *tc.want, *got)
			}
		})
	}
}

func TestGetRedisMemoryGiB(t *testing.T) {
	tests := []struct {
		name   string
		svc    compose.ServiceConfig
		wantGb int
	}{
		{
			name:   "no deploy config defaults to 1 GiB",
			svc:    compose.ServiceConfig{},
			wantGb: 1,
		},
		{
			name: "less than 1 GiB rounds up to 1",
			svc: compose.ServiceConfig{
				Deploy: &compose.DeployConfig{
					Resources: &compose.Resources{
						Reservations: &compose.ResourceConfig{Memory: strPtr("512Mi")},
					},
				},
			},
			wantGb: 1,
		},
		{
			name: "exactly 1 GiB",
			svc: compose.ServiceConfig{
				Deploy: &compose.DeployConfig{
					Resources: &compose.Resources{
						Reservations: &compose.ResourceConfig{Memory: strPtr("1Gi")},
					},
				},
			},
			wantGb: 1,
		},
		{
			name: "2 GiB",
			svc: compose.ServiceConfig{
				Deploy: &compose.DeployConfig{
					Resources: &compose.Resources{
						Reservations: &compose.ResourceConfig{Memory: strPtr("2Gi")},
					},
				},
			},
			wantGb: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantGb, getRedisMemoryGiB(tc.svc))
		})
	}
}
