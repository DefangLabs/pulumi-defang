package main

import (
	"encoding/json"
	"testing"

	"github.com/DefangLabs/defang/src/pkg/clouds/aws/ecs"
	"github.com/DefangLabs/defang/src/pkg/stackpath"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	awsprov "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The names this provider gives AWS resources are a wire format read by the
// Defang CLI: `defang compose up` derives service state on AWS only from ECS
// lifecycle events forwarded into a CloudWatch log group, and every step of
// that path parses a name. These tests drive the provider's naming helpers
// through the CLI's own parsers, so a change on either side fails here instead
// of hanging a deploy. See DefangLabs/pulumi-defang#331.
const (
	contractPrefix     = "Defang"
	contractProject    = "myproject"
	contractStack      = "beta"
	contractService    = "app"
	contractEtag       = "kg7lqwrfnfxa"
	contractDeployment = "ecs-svc/1234567890123456789"
	contractClusterArn = "arn:aws:ecs:us-west-2:123401343364:cluster/Defang-myproject-beta-cluster-4a858f2"
)

// TestECSEventsLogGroupMatchesCLI pins the log group name the provider builds.
// Since DefangLabs/defang#2245 both sides call stackpath.StackDir — the
// provider through its own stackDir helper, the CLI through
// ByocBaseClient.StackDir — so the two can no longer disagree by construction.
// What is still worth pinning is the literal: this exact string is what
// deployed stacks already carry and what the CLI subscribes to, and only a
// group whose name ends in "/ecs" reaches parseECSSubscribeEvent.
//
// The import is cheap because stackpath is a stdlib-only leaf; the CLI's byoc
// package would have dragged its whole compose/UI tree into the cd build,
// which is why this shape used to be spelled out here instead.
func TestECSEventsLogGroupMatchesCLI(t *testing.T) {
	logGroup := stackpath.StackDir(
		contractPrefix, contractProject, contractStack, stackpath.LogGroupECS)

	assert.Equal(t, "/Defang/myproject/beta/ecs", logGroup)
	assert.True(t, stackpath.IsLogGroup(logGroup, stackpath.LogGroupECS))
}

// taskStateChangeEvent is an ECS Task State Change as EventBridge delivers it,
// trimmed to the fields the CLI reads.
func taskStateChangeEvent(t *testing.T, containerName, lastStatus string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"detail-type": "ECS Task State Change",
		"source":      "aws.ecs",
		"resources":   []string{"arn:aws:ecs:us-west-2:123401343364:task/x/y"},
		"detail": map[string]any{
			"clusterArn":    contractClusterArn,
			"lastStatus":    lastStatus,
			"desiredStatus": "RUNNING",
			"startedBy":     contractDeployment,
			"taskArn":       "arn:aws:ecs:us-west-2:123401343364:task/x/y",
			"containers":    []any{map[string]any{"name": containerName}},
			"overrides": map[string]any{
				"containerOverrides": []any{map[string]any{"name": containerName}},
			},
		},
	})
	require.NoError(t, err)
	return b
}

// deploymentStateChangeEvent is an ECS Deployment State Change. Note it carries
// no detail.clusterArn — only the service ARN in resources — which is why the
// EventBridge rule needs the service-ARN prefix clause.
func deploymentStateChangeEvent(t *testing.T, serviceArn, eventName string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"detail-type": "ECS Deployment State Change",
		"source":      "aws.ecs",
		"resources":   []string{serviceArn},
		"detail": map[string]any{
			"eventType":    "INFO",
			"eventName":    eventName,
			"deploymentId": contractDeployment,
		},
	})
	require.NoError(t, err)
	return b
}

// serviceArn is the ECS service ARN AWS derives from the provider's logical
// name: autonaming appends "-<hex>" to it.
func serviceArn(logicalName string) string {
	return "arn:aws:ecs:us-west-2:123401343364:service/" +
		"Defang-myproject-beta-cluster-4a858f2/" + logicalName + "-5336ddf"
}

// TestTaskStateChangeCarriesServiceAndEtag: the CLI reads both the service and
// the deployment etag out of the CONTAINER name. An unqualified name yields an
// empty etag, which parseECSSubscribeEvent drops.
func TestTaskStateChangeCarriesServiceAndEtag(t *testing.T) {
	containerName := awsprov.QualifiedContainerName(contractService, contractEtag)
	require.Equal(t, "app_kg7lqwrfnfxa", containerName)

	evt, err := ecs.ParseECSEvent(taskStateChangeEvent(t, containerName, "RUNNING"))
	require.NoError(t, err)

	assert.Equal(t, contractService, evt.Service())
	assert.Equal(t, contractEtag, evt.Etag())
	assert.Equal(t, defangv1.ServiceState_DEPLOYMENT_PENDING, evt.State())

	// Without the etag suffix the CLI recovers neither, and drops the event.
	bare, err := ecs.ParseECSEvent(taskStateChangeEvent(t, contractService, "RUNNING"))
	require.NoError(t, err)
	assert.Empty(t, bare.Etag(), "a bare container name is what makes the CLI drop task events")
	assert.Empty(t, bare.Service())
}

// TestDeploymentStateChangeCompletesTheService is the assertion that matters
// most: DEPLOYMENT_COMPLETED is the state `defang compose up` waits for, and it
// arrives only on this event, attributed by the ECS SERVICE name and etagged
// from the cache the task events above populate.
func TestDeploymentStateChangeCompletesTheService(t *testing.T) {
	// A task event first: it is what puts deploymentId → etag in the cache.
	containerName := awsprov.QualifiedContainerName(contractService, contractEtag)
	_, err := ecs.ParseECSEvent(taskStateChangeEvent(t, containerName, "RUNNING"))
	require.NoError(t, err)

	logicalName := awsprov.ECSServiceResourceName(contractProject, contractService)
	require.Equal(t, "myproject_app", logicalName)

	evt, err := ecs.ParseECSEvent(deploymentStateChangeEvent(
		t, serviceArn(logicalName), "SERVICE_DEPLOYMENT_COMPLETED"))
	require.NoError(t, err)

	assert.Equal(t, contractService, evt.Service(),
		"the compose service name must be recoverable from the ECS service ARN")
	assert.Equal(t, contractEtag, evt.Etag())
	assert.Equal(t, defangv1.ServiceState_DEPLOYMENT_COMPLETED, evt.State())

	// An unqualified ECS service name (what this provider produced before
	// #331) leaves the CLI with no service to attribute the event to, so
	// WaitServiceState never marks anything complete.
	bare, err := ecs.ParseECSEvent(deploymentStateChangeEvent(
		t, serviceArn(contractService), "SERVICE_DEPLOYMENT_COMPLETED"))
	require.NoError(t, err)
	assert.Empty(t, bare.Service(),
		"an unqualified ECS service name is what makes `up` hang")
}

// TestServiceNameSurvivesHyphens: both the project and the service may contain
// hyphens, and the CLI splits on the first "_" and the last "-".
func TestServiceNameSurvivesHyphens(t *testing.T) {
	logicalName := awsprov.ECSServiceResourceName("html-css-js", "s3-probe")
	evt, err := ecs.ParseECSEvent(deploymentStateChangeEvent(
		t, serviceArn(logicalName), "SERVICE_DEPLOYMENT_COMPLETED"))
	require.NoError(t, err)
	assert.Equal(t, "s3-probe", evt.Service())
}
