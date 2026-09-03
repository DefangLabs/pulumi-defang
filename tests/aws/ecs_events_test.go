package aws

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/defang/src/pkg/stackpath"
	awsprov "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

// These tests pin the names and the event pattern the Defang CLI parses when it
// reports service state during `defang compose up`. Every assertion here has a
// counterpart in the CLI: see cd/ecs_events_cli_contract_test.go, which feeds
// the same strings through the CLI's own parsers.
const (
	testEventsProject = "myproject"
	testEventsStack   = "beta"
	testEventsEtag    = "kg7lqwrfnfxa"
	testClusterArn    = "arn:aws:ecs:us-west-2:123401343364:cluster/Defang-myproject-beta-cluster-4a858f2"
	testLogGroupArn   = "arn:aws:logs:us-west-2:123401343364:log-group:/Defang/myproject/beta/ecs"
)

// capturedProject holds the registered resources of one Project construct.
type capturedProject struct {
	logGroups      map[string]property.Map // logical name → inputs
	resourcePolicy property.Map
	eventRule      property.Map
	eventTarget    property.Map
	ecsServices    []string // logical names
	ecsServiceArgs []property.Map
	taskDefs       []property.Map
}

func constructProjectForEvents(t *testing.T, services map[string]property.Value) *capturedProject {
	t.Helper()

	captured := &capturedProject{logGroups: map[string]property.Map{}}
	var mu sync.Mutex
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			mu.Lock()
			defer mu.Unlock()
			outputs := args.Inputs
			switch string(args.TypeToken) {
			case "aws:ecs/cluster:Cluster":
				// The pattern is built from the cluster ARN, so it has to be
				// a realistic one rather than the mock's empty default.
				outputs = withOutput(args.Inputs, "arn", testClusterArn)
			case "aws:cloudwatch/logGroup:LogGroup":
				captured.logGroups[args.Name] = args.Inputs
				outputs = withOutput(args.Inputs, "arn", testLogGroupArn)
			case "aws:cloudwatch/logResourcePolicy:LogResourcePolicy":
				captured.resourcePolicy = args.Inputs
			case "aws:cloudwatch/eventRule:EventRule":
				captured.eventRule = args.Inputs
			case "aws:cloudwatch/eventTarget:EventTarget":
				captured.eventTarget = args.Inputs
			case "aws:ecs/service:Service":
				captured.ecsServices = append(captured.ecsServices, args.Name)
				captured.ecsServiceArgs = append(captured.ecsServiceArgs, args.Inputs)
			case "aws:ecs/taskDefinition:TaskDefinition":
				captured.taskDefs = append(captured.taskDefs, args.Inputs)
			default:
			}
			return args.Name, outputs, nil
		},
	}

	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))
	_, err := server.Construct(p.ConstructRequest{
		// The component name is the compose project name and the URN's stack
		// is the Pulumi stack; both appear in the log group path.
		Urn: resource.NewURN(
			tokens.QName(testEventsStack), "proj", "",
			tokens.Type("defang-aws:index:Project"), testEventsProject),
		Inputs: property.NewMap(map[string]property.Value{
			"etag":     property.New(testEventsEtag),
			"services": property.New(property.NewMap(services)),
		}),
	})
	require.NoError(t, err)
	return captured
}

func withOutput(inputs property.Map, key, value string) property.Map {
	m := inputs.AsMap()
	m[key] = property.New(value)
	return property.NewMap(m)
}

// TestAwsProjectCreatesECSEventsLogGroup pins the log group name the CLI
// subscribes to. ByocAws.getSubscribeLogGroupInputs asks CloudWatch for
// StackDir(project, "ecs"), and parseSubscribeEvent only routes an event to the
// ECS parser when the group name ends in "/ecs" — a group under any other name
// is silently never read.
func TestAwsProjectCreatesECSEventsLogGroup(t *testing.T) {
	captured := constructProjectForEvents(t, map[string]property.Value{
		"app": testutil.ServiceWithImage("nginx:latest"),
	})

	ecsLogGroup, ok := captured.logGroups["ecs-events"]
	require.True(t, ok, "expected an ECS events log group")
	assert.Equal(t, "/Defang/myproject/beta/ecs", ecsLogGroup.Get("name").AsString())
	// Same name the CLI derives, from the CLI's own definition of the shape.
	assert.Equal(t,
		stackpath.StackDir("Defang", testEventsProject, testEventsStack, stackpath.LogGroupECS),
		ecsLogGroup.Get("name").AsString())

	// EventBridge writes through a resource policy on the group, not a role.
	require.NotEqual(t, 0, captured.resourcePolicy.Len(), "expected a log resource policy")
	// Scoped to the log group, not the account: account-scoped policies are
	// capped at 10 per region.
	assert.Equal(t, testLogGroupArn, captured.resourcePolicy.Get("resourceArn").AsString())
	var policy struct {
		Statement []struct {
			Action    []string `json:"Action"`
			Effect    string   `json:"Effect"`
			Principal struct {
				Service []string `json:"Service"`
			} `json:"Principal"`
			Resource string `json:"Resource"`
		} `json:"Statement"`
	}
	require.NoError(t, json.Unmarshal(
		[]byte(captured.resourcePolicy.Get("policyDocument").AsString()), &policy))
	require.Len(t, policy.Statement, 1)
	assert.Equal(t, "Allow", policy.Statement[0].Effect)
	assert.ElementsMatch(t,
		[]string{"logs:CreateLogStream", "logs:PutLogEvents"}, policy.Statement[0].Action)
	assert.ElementsMatch(t,
		[]string{"events.amazonaws.com", "delivery.logs.amazonaws.com"},
		policy.Statement[0].Principal.Service)
	assert.Equal(t, testLogGroupArn+":*", policy.Statement[0].Resource)

	// And the rule has to actually target that group.
	require.NotEqual(t, 0, captured.eventTarget.Len(), "expected an event target")
	assert.Equal(t, testLogGroupArn, captured.eventTarget.Get("arn").AsString())
}

// TestAwsProjectECSEventPattern pins the EventBridge pattern. Deployment state
// change events (the only source of DEPLOYMENT_COMPLETED) carry no
// detail.clusterArn, so they match on the service-ARN prefix instead; the
// cluster of a real deploy does not end in "cluster", so that prefix cannot be
// derived the way the TS stack derived it.
func TestAwsProjectECSEventPattern(t *testing.T) {
	captured := constructProjectForEvents(t, map[string]property.Value{
		"app": testutil.ServiceWithImage("nginx:latest"),
	})

	require.NotEqual(t, 0, captured.eventRule.Len(), "expected an ECS lifecycle event rule")
	raw := captured.eventRule.Get("eventPattern").AsString()

	var pattern map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &pattern))
	assert.Equal(t, []any{"aws.ecs"}, pattern["source"])
	assert.Equal(t, []any{
		"ECS Task State Change",
		"ECS Service Action",
		"ECS Deployment State Change",
	}, pattern["detail-type"])

	servicePrefix := "arn:aws:ecs:us-west-2:123401343364:service/Defang-myproject-beta-cluster-4a858f2/"
	assert.Contains(t, raw, servicePrefix, "deployment events must match on the service-ARN prefix")
	assert.Contains(t, raw, testClusterArn, "task events must match on detail.clusterArn")
}

// TestAwsProjectECSNamesAreCLIParseable pins the two names the CLI reads a
// service and an etag out of. Both are wire format, not labels.
func TestAwsProjectECSNamesAreCLIParseable(t *testing.T) {
	captured := constructProjectForEvents(t, map[string]property.Value{
		"app": testutil.ServiceWithImage("nginx:latest"),
	})

	// serviceNameFromResources takes the text between the first "_" and the
	// last "-" of the autonamed physical name "<logical>-<hex>".
	require.Len(t, captured.ecsServices, 1)
	assert.Equal(t, "myproject_app", captured.ecsServices[0])

	// The assertion above is only about the wire format because the service is
	// autonamed: with no "name" input Pulumi derives the physical name from the
	// logical one. Setting "name" explicitly would break that derivation while
	// leaving the assertion green, so pin its absence.
	require.Len(t, captured.ecsServiceArgs, 1)
	_, hasName := captured.ecsServiceArgs[0].GetOk("name")
	assert.False(t, hasName,
		"ECS service must stay autonamed; an explicit name breaks the <logical>-<hex> shape the CLI parses")

	// TaskStateChangeEvent.Etag takes the text after the last "_" of the
	// container name; an empty etag makes the CLI drop the event.
	require.Len(t, captured.taskDefs, 1)
	var defs []awsprov.ContainerDefinition
	require.NoError(t, json.Unmarshal(
		[]byte(captured.taskDefs[0].Get("containerDefinitions").AsString()), &defs))
	require.Len(t, defs, 1)
	assert.Equal(t, "app_"+testEventsEtag, defs[0].Name)
	assert.Equal(t, testEventsEtag, defs[0].Name[strings.LastIndex(defs[0].Name, "_")+1:])
}

// TestAwsProjectSidecarContainerNamesQualified checks that intra-task
// volumes_from references still resolve once container names carry the etag.
// (depends_on on an own sidecar is covered by TestBuildDependsOnQualifiesNames;
// it cannot be driven through Project — see the TopologicalSort note in the PR.)
func TestAwsProjectSidecarContainerNamesQualified(t *testing.T) {
	captured := constructProjectForEvents(t, map[string]property.Value{
		"app": property.New(property.NewMap(map[string]property.Value{
			"image":       property.New("myapp:latest"),
			"volumesFrom": property.New(property.NewArray([]property.Value{property.New("helper")})),
		})),
		"helper": property.New(property.NewMap(map[string]property.Value{
			"image":       property.New("helper:latest"),
			"networkMode": property.New("service:app"),
		})),
	})

	require.Len(t, captured.taskDefs, 1)
	var defs []awsprov.ContainerDefinition
	require.NoError(t, json.Unmarshal(
		[]byte(captured.taskDefs[0].Get("containerDefinitions").AsString()), &defs))
	require.Len(t, defs, 2)

	names := map[string]awsprov.ContainerDefinition{}
	for _, d := range defs {
		names[d.Name] = d
	}
	require.Contains(t, names, "app_"+testEventsEtag)
	require.Contains(t, names, "helper_"+testEventsEtag)

	main := names["app_"+testEventsEtag]
	require.Len(t, main.VolumesFrom, 1)
	assert.Equal(t, "helper_"+testEventsEtag, *main.VolumesFrom[0].SourceContainer)

	// The main container must stay first: TaskStateChangeEvent reads
	// containerOverrides[0] to identify the service.
	assert.Equal(t, "app_"+testEventsEtag, defs[0].Name)
}
