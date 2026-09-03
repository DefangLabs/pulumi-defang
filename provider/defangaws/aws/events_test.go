package aws

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackDir(t *testing.T) {
	// Same shape as the CLI's ByocBaseClient.StackDir and the TS stackDir.
	assert.Equal(t, "/Defang/myproject/beta/ecs", StackDir("Defang", "myproject", "beta", "ecs"))
	assert.Equal(t, "/myproject/beta/ecs", StackDir("", "myproject", "beta", "ecs"))
}

func TestClusterServiceArnPrefix(t *testing.T) {
	// The autonamed cluster of a real deploy; note it does NOT end in
	// "cluster", which is why the TS regex could not be ported literally.
	assert.Equal(t,
		"arn:aws:ecs:us-west-2:123401343364:service/Defang-html-css-js-newprovideraws-cluster-4a858f2/",
		clusterServiceArnPrefix(
			"arn:aws:ecs:us-west-2:123401343364:cluster/Defang-html-css-js-newprovideraws-cluster-4a858f2"))

	// A service ARN from that cluster must start with the prefix.
	assert.Equal(t,
		"arn:aws:ecs:us-west-2:123401343364:service/ecs-dev-cluster/",
		clusterServiceArnPrefix("arn:aws:ecs:us-west-2:123401343364:cluster/ecs-dev-cluster"))

	// Not a cluster ARN: pass through rather than emit a prefix that would
	// match unrelated resources.
	assert.Equal(t, "not-an-arn", clusterServiceArnPrefix("not-an-arn"))
}

func TestECSEventPattern(t *testing.T) {
	const clusterArn = "arn:aws:ecs:us-west-2:123401343364:cluster/Defang-proj-beta-cluster-4a858f2"

	raw, err := ecsEventPattern(clusterArn)
	require.NoError(t, err)

	var pattern struct {
		Source     []string `json:"source"`
		DetailType []string `json:"detail-type"`
		Or         []struct {
			Resources []struct {
				Prefix string `json:"prefix"`
			} `json:"resources"`
			Detail struct {
				ClusterArn []string `json:"clusterArn"`
			} `json:"detail"`
		} `json:"$or"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &pattern))

	assert.Equal(t, []string{"aws.ecs"}, pattern.Source)
	// "ECS Deployment State Change" is the only source of DEPLOYMENT_COMPLETED;
	// "ECS Task State Change" is what carries the etag. Both are required.
	assert.Equal(t, []string{
		"ECS Task State Change",
		"ECS Service Action",
		"ECS Deployment State Change",
	}, pattern.DetailType)

	require.Len(t, pattern.Or, 2)
	// Deployment state change events carry no detail.clusterArn, so they are
	// matched on the service-ARN prefix.
	require.Len(t, pattern.Or[0].Resources, 1)
	assert.Equal(t,
		"arn:aws:ecs:us-west-2:123401343364:service/Defang-proj-beta-cluster-4a858f2/",
		pattern.Or[0].Resources[0].Prefix)
	assert.Equal(t, []string{clusterArn}, pattern.Or[1].Detail.ClusterArn)
}
