package aws

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alarmMocks records every MetricAlarm registration.
type alarmMocks struct {
	alarms *[]pulumi.MockResourceArgs
}

func (m alarmMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "aws:cloudwatch/metricAlarm:MetricAlarm" {
		*m.alarms = append(*m.alarms, args)
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

func runCreateDBAlarms(t *testing.T) []pulumi.MockResourceArgs {
	t.Helper()
	var created []pulumi.MockResourceArgs
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return createDBAlarms(ctx, "cache", "AWS/MemoryDB",
			pulumi.StringMap{"ClusterName": pulumi.String("cache-abc")}, testAlarms)
	}, pulumi.WithMocks("myproj", "mystack", alarmMocks{alarms: &created}))
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

	memory := created[0].Inputs
	assert.Equal(t, "cache-memory-usage", created[0].Name)
	assert.Equal(t, "AWS/MemoryDB", memory["namespace"].StringValue())
	assert.Equal(t, "DatabaseCapacityUsagePercentage", memory["metricName"].StringValue())
	assert.Equal(t, "GreaterThanThreshold", memory["comparisonOperator"].StringValue())
	assert.InDelta(t, 80, memory["threshold"].NumberValue(), 0)
	assert.Equal(t, "Maximum", memory["statistic"].StringValue())
	assert.InDelta(t, 300, memory["period"].NumberValue(), 0)
	assert.InDelta(t, 2, memory["evaluationPeriods"].NumberValue(), 0)
	assert.Equal(t, "cache-abc", memory["dimensions"].ObjectValue()["ClusterName"].StringValue())
	assert.False(t, memory.HasValue("alarmActions"), "no actions without alarm-topic-arn")

	assert.Equal(t, "cache-cpu-usage", created[1].Name)
	assert.Equal(t, "CPUUtilization", created[1].Inputs["metricName"].StringValue())
}

func TestCreateDBAlarms_TopicArn(t *testing.T) {
	arn := "arn:aws:sns:us-west-2:123456789012:slackbot-alarms"
	t.Setenv("PULUMI_CONFIG", `{"defang-aws:alarms": "true", "defang-aws:alarm-topic-arn": "`+arn+`"}`)
	created := runCreateDBAlarms(t)
	require.Len(t, created, 2)

	for _, alarm := range created {
		alarmActions := alarm.Inputs["alarmActions"].ArrayValue()
		require.Len(t, alarmActions, 1)
		assert.Equal(t, arn, alarmActions[0].StringValue())
		okActions := alarm.Inputs["okActions"].ArrayValue()
		require.Len(t, okActions, 1)
		assert.Equal(t, arn, okActions[0].StringValue())
	}
}
