package aws

import (
	"github.com/aws/smithy-go/ptr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createBedrockPolicy(
	ctx *pulumi.Context, name string, models []string, opts ...pulumi.ResourceOption,
) (*iam.Policy, error) {
	if len(models) == 0 {
		models = []string{"*"}
	}
	policy := iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.PolicyStatement{
			{
				Sid:      ptr.String("AllowBedrockModelList"),
				Action:   []string{"bedrock:List*"},
				Effect:   "Allow",
				Resource: "*", // These actions do not support resource-level permissions
			},
			{
				Sid: ptr.String("AllowBedrockInvoke"),
				Action: []string{
					"bedrock:InvokeModel",
					"bedrock:InvokeModelWithResponseStream",
				},
				Effect:   "Allow",
				Resource: models,
			},
		},
	}
	policyJson := pulumi.JSONMarshal(policy)
	return iam.NewPolicy(ctx, name, &iam.PolicyArgs{Policy: policyJson}, opts...)
}

func createRoute53SidecarPolicy(
	ctx *pulumi.Context,
	name string,
	privateZone *route53.Zone,
	opts ...pulumi.ResourceOption,
) (*iam.Policy, error) {
	policyJson := privateZone.Arn.ApplyT(func(privateZoneArn string) pulumi.StringOutput {
		policy := iam.PolicyDocument{
			Version: "2012-10-17",
			Statement: []iam.PolicyStatement{
				{
					Sid:      ptr.String("AllowRoute53SidecarWrite"),
					Action:   []string{"route53:ChangeResourceRecordSets"},
					Effect:   "Allow",
					Resource: privateZoneArn,
				},
				{
					Sid:      ptr.String("AllowRoute53Read"), // TODO: better name
					Effect:   "Allow",
					Action:   []string{"route53:GetChange"}, // These actions do not support resource-level permissions
					Resource: "*",
				},
			},
		}
		return pulumi.JSONMarshal(policy)
	})
	return iam.NewPolicy(ctx, name, &iam.PolicyArgs{Policy: policyJson}, opts...)
}
