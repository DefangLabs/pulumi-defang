package aws

import (
	"encoding/json"

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
	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Sid:      "AllowBedrockModelList",
				Action:   []string{"bedrock:List*"},
				Effect:   "Allow",
				Resource: "*", // These actions do not support resource-level permissions
			},
			{
				Sid: "AllowBedrockInvoke",
				Action: []string{
					"bedrock:InvokeModel",
					"bedrock:InvokeModelWithResponseStream",
				},
				Effect:   "Allow",
				Resource: models,
			},
		},
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}
	return iam.NewPolicy(ctx, name, &iam.PolicyArgs{Policy: pulumi.String(b)}, opts...)
}

func createRoute53SidecarPolicy(
	ctx *pulumi.Context,
	name string,
	privateZone *route53.Zone,
	opts ...pulumi.ResourceOption,
) (*iam.Policy, error) {
	policyJson := privateZone.Arn.ApplyT(func(privateZoneArn string) (string, error) {
		policy := PolicyDocument{
			Version: "2012-10-17",
			Statement: []PolicyStatement{
				{
					Sid:      "AllowRoute53SidecarWrite",
					Effect:   "Allow",
					Action:   []string{"route53:ChangeResourceRecordSets"},
					Resource: privateZoneArn,
				},
				{
					Sid:      "AllowRoute53Read",
					Effect:   "Allow",
					Action:   []string{"route53:GetChange"}, // These actions do not support resource-level permissions
					Resource: "*",
				},
			},
		}
		b, err := json.Marshal(policy)
		return string(b), err
	})
	return iam.NewPolicy(ctx, name, &iam.PolicyArgs{Policy: policyJson}, opts...)
}
