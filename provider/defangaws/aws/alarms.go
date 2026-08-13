package aws

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// dbAlarm describes one CloudWatch metric alarm attached to a managed database.
type dbAlarm struct {
	suffix             string // appended to the resource name, e.g. "memory-usage"
	metricName         string
	comparisonOperator string
	threshold          float64
	statistic          string
	description        string
}

// Shared alarm cadence matching the pre-migration fabric alarms in defang-mvp
// pulumi/ecs/alarms.ts: 2 five-minute periods.
const (
	alarmPeriod            = 300
	alarmEvaluationPeriods = 2
)

// createDBAlarms creates CloudWatch alarms for a managed database when the
// alarms recipe is enabled; a no-op under the affordable default. When
// alarmTopicArn is non-nil, each alarm notifies that SNS topic on both the
// ALARM and OK transitions.
func createDBAlarms(
	ctx *pulumi.Context,
	name string,
	namespace string,
	dimensions pulumi.StringMapInput,
	tags pulumi.StringMapInput,
	alarmTopicArn pulumi.StringInput,
	alarms []dbAlarm,
	opts ...pulumi.ResourceOption,
) error {
	if !Alarms.Get(ctx) {
		return nil
	}

	var actions pulumi.ArrayInput
	if alarmTopicArn != nil {
		actions = pulumi.Array{alarmTopicArn}
	}

	for _, alarm := range alarms {
		_, err := cloudwatch.NewMetricAlarm(ctx, name+"-"+alarm.suffix, &cloudwatch.MetricAlarmArgs{
			AlarmDescription:   pulumi.String(alarm.description),
			AlarmActions:       actions,
			OkActions:          actions,
			ComparisonOperator: pulumi.String(alarm.comparisonOperator),
			Dimensions:         dimensions,
			EvaluationPeriods:  pulumi.Int(alarmEvaluationPeriods),
			MetricName:         pulumi.String(alarm.metricName),
			Namespace:          pulumi.String(namespace),
			Period:             pulumi.Int(alarmPeriod),
			Statistic:          pulumi.String(alarm.statistic),
			Tags:               tags,
			Threshold:          pulumi.Float64(alarm.threshold),
		}, opts...)
		if err != nil {
			return fmt.Errorf("creating %s alarm: %w", alarm.suffix, err)
		}
	}
	return nil
}
