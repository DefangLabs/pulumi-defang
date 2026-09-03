package aws

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

type eventsTestMocks struct {
	mu     sync.Mutex
	inputs map[string]resource.PropertyMap // "<type>/<name>" → inputs
}

func (m *eventsTestMocks) NewResource(
	args pulumi.MockResourceArgs,
) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inputs == nil {
		m.inputs = map[string]resource.PropertyMap{}
	}
	m.inputs[args.TypeToken+"/"+args.Name] = args.Inputs

	outputs := args.Inputs.Copy()
	switch args.TypeToken {
	case "aws:ecs/cluster:Cluster":
		outputs["arn"] = resource.NewStringProperty(
			"arn:aws:ecs:us-west-2:123401343364:cluster/Defang-myproject-beta-cluster-4a858f2")
	case "aws:cloudwatch/logGroup:LogGroup":
		outputs["arn"] = resource.NewStringProperty(
			"arn:aws:logs:us-west-2:123401343364:log-group:/Defang/myproject/beta/ecs")
	}
	return args.Name + "_id", outputs, nil
}

func (m *eventsTestMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func withConfig(cfg map[string]string) pulumi.RunOption {
	return func(info *pulumi.RunInfo) { info.Config = cfg }
}

// TestCreateECSLifecycleToCWLogsNames checks the log group name the CLI derives
// from prefix/project/stack, and that the resource policy is scoped to that log
// group rather than to the account (account-scoped policies are capped at 10
// per region, so a per-stack one would break the 11th stack in an account).
func TestCreateECSLifecycleToCWLogsNames(t *testing.T) {
	mocks := &eventsTestMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{})
		require.NoError(t, err)
		// Version("") is a no-op option: the real caller passes the AWS provider.
		return createECSLifecycleToCWLogs(ctx, "myproject", cluster, pulumi.Version(""))
	},
		pulumi.WithMocks("myproject", "beta", mocks),
		withConfig(map[string]string{
			"defang:prefix":     "Defang",
			"pulumi:autonaming": `{"pattern":"Defang-${project}-${stack}-${name}-${hex(7)}"}`,
		}),
	)
	require.NoError(t, err)

	logGroup, ok := mocks.inputs["aws:cloudwatch/logGroup:LogGroup/ecs-events"]
	require.True(t, ok, "expected an ECS events log group")
	assert.Equal(t, "/Defang/myproject/beta/ecs", logGroup["name"].StringValue())

	policy, ok := mocks.inputs["aws:cloudwatch/logResourcePolicy:LogResourcePolicy/ecs-events"]
	require.True(t, ok, "expected a log resource policy")
	assert.False(t, policy.HasValue("policyName"),
		"policyName would make the policy account-scoped; exactly one of the two may be set")
	assert.Equal(t,
		"arn:aws:logs:us-west-2:123401343364:log-group:/Defang/myproject/beta/ecs",
		policy["resourceArn"].StringValue())
}
