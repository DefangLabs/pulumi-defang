package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ECSEventsLogGroupSuffix is the last path segment of the log group the Defang
// CLI tails for service state. ByocAws.getSubscribeLogGroupInputs subscribes to
// StackDir(project, "ecs") and parseSubscribeEvent only routes an event to the
// ECS parser when the log group name ends in "/ecs", so this suffix — and the
// StackDir shape around it — is a wire contract with the CLI, not a label.
const ECSEventsLogGroupSuffix = "ecs"

// StackDir builds the "/<prefix>/<project>/<stack>/<name>" path shared with the
// CLI (ByocBaseClient.StackDir) and the old TS stack (shared/config.ts stackDir).
// An empty prefix drops its segment, as it does on both other sides.
func StackDir(prefix, projectName, stack, name string) string {
	segments := []string{""} // leading slash
	if prefix != "" {
		segments = append(segments, prefix)
	}
	return strings.Join(append(segments, projectName, stack, name), "/")
}

func stackDir(ctx *pulumi.Context, projectName, name string) string {
	return StackDir(common.Prefix.Get(ctx), projectName, ctx.Stack(), name)
}

// clusterServiceArnPrefix turns a cluster ARN into the prefix every ECS service
// ARN in that cluster starts with:
//
//	arn:aws:ecs:<region>:<account>:cluster/<name>
//	arn:aws:ecs:<region>:<account>:service/<name>/
//
// The TS original derived this by stripping a trailing "cluster" from the
// cluster name (shared/aws/common.ts clusterArnToServicePrefix), which worked
// only because its clusters were named "…-cluster"/"…-gpucluster". This
// provider autonames the cluster ("Defang-<project>-<stack>-cluster-<hex>"), so
// that regex would fall through and produce a prefix matching nothing.
func clusterServiceArnPrefix(clusterArn string) string {
	prefix, name, ok := strings.Cut(clusterArn, ":cluster/")
	if !ok {
		return clusterArn // not a cluster ARN; a prefix that matches nothing beats a wrong one
	}
	return prefix + ":service/" + name + "/"
}

// ecsEventPattern is the EventBridge pattern forwarding ECS lifecycle events for
// one cluster. It mirrors the TS createECSLifeCycleEventRule: "ECS Task State
// Change" is what carries the container names the CLI reads the etag from, and
// "ECS Deployment State Change" is the only source of DEPLOYMENT_COMPLETED.
// Deployment events carry no detail.clusterArn (aws/containers-roadmap#1371),
// so they are matched on the service-ARN prefix instead.
func ecsEventPattern(clusterArn string) (string, error) {
	pattern := map[string]any{
		"source": []string{"aws.ecs"},
		"detail-type": []string{
			"ECS Task State Change",
			"ECS Service Action",
			"ECS Deployment State Change",
		},
		"$or": []any{
			map[string]any{
				"resources": []any{
					map[string]any{"prefix": clusterServiceArnPrefix(clusterArn)},
				},
			},
			map[string]any{
				"detail": map[string]any{"clusterArn": []string{clusterArn}},
			},
		},
	}
	b, err := json.Marshal(pattern)
	return string(b), err
}

// createECSLifecycleToCWLogs forwards the cluster's ECS lifecycle events into a
// CloudWatch log group the Defang CLI subscribes to. Without it `defang compose
// up` never observes DEPLOYMENT_COMPLETED and blocks until its wait timeout,
// even though the deploy itself succeeded. Ported from the TS BYOC stack
// (pulumi/cd/aws/byoc.ts createECSLifecycleToCWLogsEventBridgeRule).
func createECSLifecycleToCWLogs(
	ctx *pulumi.Context,
	projectName string,
	cluster *ecs.Cluster,
	opt pulumi.ResourceOrInvokeOption,
) (*cloudwatch.LogGroup, error) {
	logGroup, err := cloudwatch.NewLogGroup(ctx, "ecs-events", &cloudwatch.LogGroupArgs{
		// Explicit name, not autonaming: the CLI derives this exact string.
		Name:            pulumi.String(stackDir(ctx, projectName, ECSEventsLogGroupSuffix)),
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating ECS events log group: %w", err)
	}

	// EventBridge writes to a log group through a resource policy on the group,
	// not an IAM role. See DefangLabs/defang-mvp#1514 and
	// https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-use-resource-based.html#eb-cloudwatchlogs-permissions
	policyDocument := logGroup.Arn.ApplyT(func(arn string) (string, error) {
		policy := PolicyDocument{
			Version: "2012-10-17",
			Statement: []PolicyStatement{
				{
					Sid:    "TrustEventsToStoreLogEvent",
					Effect: "Allow",
					Action: []string{"logs:CreateLogStream", "logs:PutLogEvents"},
					Principal: map[string]any{
						"Service": []string{"events.amazonaws.com", "delivery.logs.amazonaws.com"},
					},
					Resource: arn + ":*", // log stream ARN
				},
			},
		}
		b, err := json.Marshal(policy)
		return string(b), err
	}).(pulumi.StringOutput)

	// A log resource policy is account+region scoped by name, so it must not
	// collide with another project's; keep the stack-qualified log group name.
	_, err = cloudwatch.NewLogResourcePolicy(ctx, "ecs-events", &cloudwatch.LogResourcePolicyArgs{
		PolicyName:     logGroup.Name,
		PolicyDocument: policyDocument,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating ECS events log resource policy: %w", err)
	}

	eventPattern := cluster.Arn.ApplyT(ecsEventPattern).(pulumi.StringOutput)

	rule, err := cloudwatch.NewEventRule(ctx, "ecs-lifecycle", &cloudwatch.EventRuleArgs{
		Description:  pulumi.Sprintf("Capture ECS task lifecycle events of cluster %s", cluster.Arn),
		EventPattern: eventPattern,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating ECS lifecycle event rule: %w", err)
	}

	_, err = cloudwatch.NewEventTarget(ctx, "ecs-lifecycle-cw", &cloudwatch.EventTargetArgs{
		Arn:  logGroup.Arn,
		Rule: rule.Name,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating ECS lifecycle event target: %w", err)
	}

	return logGroup, nil
}
