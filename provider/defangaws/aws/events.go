package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DefangLabs/defang/src/pkg/stackpath"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// stackDir builds the log group path the Defang CLI tails, using the CLI's own
// definition of the shape (stackpath.StackDir, which ByocBaseClient.StackDir
// also calls). Importing it rather than restating it is what keeps the two
// sides from drifting: the CLI subscribes to stackpath.LogGroupECS and only
// routes an event to its ECS parser when the log group name ends in "/ecs", so
// this path is a wire contract, not a label.
func stackDir(ctx *pulumi.Context, projectName, name string) string {
	return stackpath.StackDir(common.Prefix.Get(ctx), projectName, ctx.Stack(), name)
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
) error {
	logGroup, err := cloudwatch.NewLogGroup(ctx, "ecs-events", &cloudwatch.LogGroupArgs{
		// Explicit name, not autonaming: the CLI derives this exact string.
		Name:            pulumi.String(stackDir(ctx, projectName, stackpath.LogGroupECS)),
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opt)
	if err != nil {
		return fmt.Errorf("creating ECS events log group: %w", err)
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

	// Scope the policy to this log group (ResourceArn) rather than to the
	// account (PolicyName): exactly one of the two may be set, and CloudWatch
	// Logs caps account-scoped policies at 10 per region — which the TS stack's
	// per-project PolicyName would have hit at the 11th stack in an account.
	// Resource-scoped policies have no such cap, and one attaches per log group.
	policy, err := cloudwatch.NewLogResourcePolicy(ctx, "ecs-events", &cloudwatch.LogResourcePolicyArgs{
		ResourceArn:    logGroup.Arn,
		PolicyDocument: policyDocument,
	}, opt)
	if err != nil {
		return fmt.Errorf("creating ECS events log resource policy: %w", err)
	}

	eventPattern := cluster.Arn.ApplyT(ecsEventPattern).(pulumi.StringOutput)

	rule, err := cloudwatch.NewEventRule(ctx, "ecs-lifecycle", &cloudwatch.EventRuleArgs{
		Description:  pulumi.Sprintf("Capture ECS task lifecycle events of cluster %s", cluster.Arn),
		EventPattern: eventPattern,
	}, opt)
	if err != nil {
		return fmt.Errorf("creating ECS lifecycle event rule: %w", err)
	}

	// The target is what starts delivery, and EventBridge can only write once
	// the resource policy exists — neither the rule nor the target references
	// it, so order them explicitly rather than letting the engine run them in
	// parallel and drop the events in between.
	_, err = cloudwatch.NewEventTarget(ctx, "ecs-lifecycle-cw", &cloudwatch.EventTargetArgs{
		Arn:  logGroup.Arn,
		Rule: rule.Name,
	}, opt, pulumi.DependsOn([]pulumi.Resource{policy}))
	if err != nil {
		return fmt.Errorf("creating ECS lifecycle event target: %w", err)
	}

	return nil
}
