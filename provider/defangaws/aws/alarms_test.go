package aws

import (
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alarmMocks records every MetricAlarm registration by resource name.
// Registrations arrive on concurrent goroutines, so guard with a mutex and
// don't assume ordering.
type alarmMocks struct {
	mu     *sync.Mutex
	alarms map[string]resource.PropertyMap
}

func (m alarmMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "aws:cloudwatch/metricAlarm:MetricAlarm" {
		m.mu.Lock()
		m.alarms[args.Name] = args.Inputs
		m.mu.Unlock()
	}
	return args.Name + "_id", args.Inputs, nil
}

func (m alarmMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

var testAlarms = []dbAlarm{
	{
		suffix:             "memory-usage",
		metricName:         "DatabaseCapacityUsagePercentage",
		comparisonOperator: "GreaterThanThreshold",
		threshold:          80,
		statistic:          "Maximum",
		description:        "memory usage has exceeded 80%",
	},
	{
		suffix:             "cpu-usage",
		metricName:         "CPUUtilization",
		comparisonOperator: "GreaterThanThreshold",
		threshold:          80,
		statistic:          "Maximum",
		description:        "CPU usage has exceeded 80%",
	},
}

func runCreateDBAlarms(t *testing.T) map[string]resource.PropertyMap {
	t.Helper()
	created := map[string]resource.PropertyMap{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return createDBAlarms(ctx, "cache", "AWS/MemoryDB",
			pulumi.StringMap{"ClusterName": pulumi.String("cache-abc")},
			pulumi.StringMap{"defang:service": pulumi.String("cache")}, testAlarms)
	}, pulumi.WithMocks("myproj", "mystack", alarmMocks{mu: &sync.Mutex{}, alarms: created}))
	require.NoError(t, err)
	return created
}

func TestCreateDBAlarms_AffordableDefaultCreatesNone(t *testing.T) {
	created := runCreateDBAlarms(t)
	assert.Empty(t, created, "affordable default must not create alarms")
}

func TestCreateDBAlarms_Enabled(t *testing.T) {
	t.Setenv("PULUMI_CONFIG", `{"defang-aws:alarms": "true"}`)
	created := runCreateDBAlarms(t)
	require.Len(t, created, 2)

	memory, ok := created["cache-memory-usage"]
	require.True(t, ok)
	assert.Equal(t, "AWS/MemoryDB", memory["namespace"].StringValue())
	assert.Equal(t, "DatabaseCapacityUsagePercentage", memory["metricName"].StringValue())
	assert.Equal(t, "GreaterThanThreshold", memory["comparisonOperator"].StringValue())
	assert.InDelta(t, 80, memory["threshold"].NumberValue(), 0)
	assert.Equal(t, "Maximum", memory["statistic"].StringValue())
	assert.InDelta(t, 300, memory["period"].NumberValue(), 0)
	assert.InDelta(t, 2, memory["evaluationPeriods"].NumberValue(), 0)
	assert.Equal(t, "cache-abc", memory["dimensions"].ObjectValue()["ClusterName"].StringValue())
	assert.Equal(t, "cache", memory["tags"].ObjectValue()["defang:service"].StringValue())
	assert.False(t, memory.HasValue("alarmActions"), "no actions without alarm-topic-arn")

	cpu, ok := created["cache-cpu-usage"]
	require.True(t, ok)
	assert.Equal(t, "CPUUtilization", cpu["metricName"].StringValue())
}

func TestCreateDBAlarms_TopicArn(t *testing.T) {
	arn := "arn:aws:sns:us-west-2:123456789012:slackbot-alarms"
	t.Setenv("PULUMI_CONFIG", `{"defang-aws:alarms": "true", "defang-aws:alarm-topic-arn": "`+arn+`"}`)
	created := runCreateDBAlarms(t)
	require.Len(t, created, 2)

	for name, alarm := range created {
		alarmActions := alarm["alarmActions"].ArrayValue()
		require.Len(t, alarmActions, 1, name)
		assert.Equal(t, arn, alarmActions[0].StringValue(), name)
		okActions := alarm["okActions"].ArrayValue()
		require.Len(t, okActions, 1, name)
		assert.Equal(t, arn, okActions[0].StringValue(), name)
	}
}
