package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func TestTgHealthCheckTiming(t *testing.T) {
	tests := []struct {
		name         string
		healthCheck  *compose.HealthCheckConfig
		wantInterval int
		wantTimeout  int
	}{
		{
			name:         "no healthcheck uses defaults",
			healthCheck:  nil,
			wantInterval: 5,
			wantTimeout:  4,
		},
		{
			// Regression: fabric's healthcheck (interval 5, no timeout) must
			// not fall back to timeout == interval — AWS rejects the TG with
			// "Health check timeout '5' must be smaller than the interval '5'".
			name:         "unset timeout stays below interval",
			healthCheck:  &compose.HealthCheckConfig{IntervalSeconds: 5},
			wantInterval: 5,
			wantTimeout:  4,
		},
		{
			name:         "explicit timeout clamped below interval",
			healthCheck:  &compose.HealthCheckConfig{IntervalSeconds: 10, TimeoutSeconds: 10},
			wantInterval: 10,
			wantTimeout:  9,
		},
		{
			name:         "explicit timeout kept when valid",
			healthCheck:  &compose.HealthCheckConfig{IntervalSeconds: 30, TimeoutSeconds: 5},
			wantInterval: 30,
			wantTimeout:  5,
		},
		{
			name:         "large interval caps timeout at 120",
			healthCheck:  &compose.HealthCheckConfig{IntervalSeconds: 300},
			wantInterval: 300,
			wantTimeout:  120,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval, timeout := tgHealthCheckTiming(5, tt.healthCheck)
			if interval != tt.wantInterval || timeout != tt.wantTimeout {
				t.Errorf("got interval=%d timeout=%d, want interval=%d timeout=%d",
					interval, timeout, tt.wantInterval, tt.wantTimeout)
			}
		})
	}
}
