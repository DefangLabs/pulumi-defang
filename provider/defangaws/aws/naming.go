package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

const autonamingSuffixLen = 5 // "-" + hex(4) from autonaming stack config
const tgMaxNameLen = 32

// tgNameHashLen is how much of a truncated service name is given over to a
// hash of the whole of it.
const tgNameHashLen = 6

// targetGroupName builds a logical Pulumi resource name for a TargetGroup.
// The physical name (with its 32-char AWS limit) is handled by autonaming config.
// We keep the logical name short enough that autonaming stays within budget.
func targetGroupName(
	service string, port int, appProtocol compose.PortAppProtocol, listener compose.PortListenerProtocol,
) string {
	suffix := fmt.Sprintf("-%d", port)
	if appProtocol != "" && appProtocol != compose.PortAppProtocolHTTP {
		suffix += string(appProtocol)
	}
	// HTTPS is the default listener, so only non-default listeners get a suffix
	// (matches TS targetGroupName's listenerProtocol handling).
	if listener != compose.PortListenerDefault && listener != compose.PortListenerHTTPS {
		suffix += "-" + string(listener)
	}

	maxService := tgMaxNameLen - autonamingSuffixLen - len(suffix)
	if len(service) > maxService {
		// Plain truncation made two long service names that share a prefix
		// collide, and two target groups with one logical name is a duplicate
		// URN that fails the whole deploy. Trade the tail for a hash of the
		// full name; short names are untouched.
		digest := sha256.Sum256([]byte(service))
		service = service[:maxService-tgNameHashLen] + hex.EncodeToString(digest[:])[:tgNameHashLen]
	}

	return service + suffix
}

// QualifiedContainerName suffixes a container name with the deployment etag,
// as the old TS stack did (defang-mvp pulumi/cd/aws/defang_service.ts
// qualifiedContainerName). This is a wire contract with the Defang CLI, not a
// label: for every "ECS Task State Change" event the CLI splits the container
// name on its last "_" and treats the tail as the deployment etag
// (clouds/aws/ecs/event.go TaskStateChangeEvent.Etag). An unsuffixed name
// yields an empty etag, which makes parseECSSubscribeEvent drop the event and
// — worse — leaves the deployment-id → etag cache empty, so the
// "ECS Deployment State Change" events that carry DEPLOYMENT_COMPLETED are
// dropped too and `defang compose up` waits forever.
//
// The etag is empty for standalone Service callers (no CD deployment), in
// which case the bare name is used.
func QualifiedContainerName(name, etag string) string {
	if etag == "" {
		return name
	}
	return name + "_" + etag
}

// ECSServiceResourceName is the logical Pulumi name of an ECS service. The
// physical name is autonamed from it as "<logical>-<hex>", and the Defang CLI
// parses the compose service back out of the service ARN by taking the text
// between the first "_" and the last "-" (byoc/aws/... serviceNameFromResources),
// so the project qualifier is what makes DEPLOYMENT_COMPLETED attributable to a
// service. Matches the TS `${projectName}_${service_name}` resource name
// ("FQN but with underscore instead of dots, used by ECS events").
//
// projectName is empty for standalone Service callers, which no CLI subscribes
// to; those keep the bare service name.
func ECSServiceResourceName(projectName, serviceName string) string {
	if projectName == "" {
		return serviceName
	}
	return projectName + "_" + serviceName
}
