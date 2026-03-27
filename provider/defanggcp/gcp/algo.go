package gcp

import "github.com/DefangLabs/pulumi-defang/provider/compose"

// IsCloudRunService returns true if the service has exactly one ingress port
// and no other ports — the only configuration Cloud Run Services support natively.
// Services with zero ports or multiple ports, or host-mode ports, must run on Compute Engine.
func IsCloudRunService(svc compose.ServiceConfig) bool {
	return len(svc.Ports) == 1 && svc.Ports[0].Mode == portModeIngress
}
