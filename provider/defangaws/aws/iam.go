package aws

import (
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// PolicyDocument is like Pulumi's iam.PolicyDocument but with JSON tags.
type PolicyDocument struct {
	Id        string                    `json:"Id,omitempty"`
	Statement []PolicyStatement         `json:"Statement"`
	Version   iam.PolicyDocumentVersion `json:"Version"`
}

// PolicyStatement is like Pulumi's iam.PolicyStatement but with JSON tags.
type PolicyStatement struct {
	// Include a list of actions that the policy allows or denies. Required (either Action or NotAction)
	Action interface{} `json:"Action,omitempty"`
	// Specify the circumstances under which the policy grants permission.
	Condition map[string]interface{} `json:"Condition,omitempty"`
	// Indicate whether the policy allows or denies access.
	Effect iam.PolicyStatementEffect `json:"Effect,omitempty"`
	// Include a list of actions that are not covered by this policy. Required (either Action or NotAction)
	NotAction interface{} `json:"NotAction,omitempty"`
	// Indicate the account, user, role, or federated user to which this policy does not apply.
	NotPrincipal interface{} `json:"NotPrincipal,omitempty"`
	// A list of resources that are specifically excluded by this policy.
	NotResource interface{} `json:"NotResource,omitempty"`
	// Indicate the account, user, role, or federated user to which you would like to allow or deny access.
	// If you are creating a policy to attach to a user or role, you cannot include this element. The principal is
	// implied as that user or role.
	Principal interface{} `json:"Principal,omitempty"`
	// A list of resources to which the actions apply.
	Resource interface{} `json:"Resource,omitempty"`
	// An optional statement ID to differentiate between your statements.
	Sid string `json:"Sid,omitempty"`
}

// type PolicyPrincipal struct {
// 	AWS       interface{} `json:"AWS,omitempty"`
// 	Federated interface{} `json:"Federated,omitempty"`
// 	Service   interface{} `json:"Service,omitempty"`
// }

// CreateExecutionRole creates the shared ECS task execution role.
func CreateExecutionRole(ctx *pulumi.Context, opts ...pulumi.ResourceOption) (*iam.Role, error) {
	assumeRolePolicyBytes, err := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Action": "sts:AssumeRole",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "ecs-tasks.amazonaws.com",
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling assume role policy: %w", err)
	}

	execRole, err := iam.NewRole(ctx, "exec-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(assumeRolePolicyBytes)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	// Attach the ECS task execution policy
	_, err = iam.NewRolePolicyAttachment(ctx, "exec-policy", &iam.RolePolicyAttachmentArgs{
		Role:      execRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("attaching execution policy: %w", err)
	}

	return execRole, nil
}

// attachPullThroughCachePolicy adds an inline policy to the execution role allowing
// ECR pull-through cache operations (BatchImportUpstreamImage, CreateRepository).
// Matches TS AllowECRPassThrough in shared/ecs/initialize.ts.
func attachPullThroughCachePolicy(
	ctx *pulumi.Context,
	execRole *iam.Role,
	cacheRepoArn pulumi.StringOutput,
	opts ...pulumi.ResourceOption,
) error {
	policyJson := cacheRepoArn.ApplyT(func(arn string) (string, error) {
		policy := PolicyDocument{
			Version: "2012-10-17",
			Statement: []PolicyStatement{
				{
					Sid:    "AllowECRPassThrough",
					Effect: "Allow",
					Action: []string{
						"ecr:BatchImportUpstreamImage",
						"ecr:CreateRepository",
					},
					Resource: arn,
				},
			},
		}
		b, err := json.Marshal(policy)
		return string(b), err
	}).(pulumi.StringOutput)

	_, err := iam.NewRolePolicy(ctx, "ecr-pull-through", &iam.RolePolicyArgs{
		Role:   execRole.Name,
		Policy: policyJson,
	}, opts...)
	return err
}

// createTaskRole creates a per-service ECS task role.
func createTaskRole(ctx *pulumi.Context, serviceName string, opts ...pulumi.ResourceOption) (*iam.Role, error) {
	assumeRolePolicyBytes, err := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Action": "sts:AssumeRole",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "ecs-tasks.amazonaws.com",
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling assume role policy: %w", err)
	}
	assumeRolePolicy := string(assumeRolePolicyBytes)

	taskRole, err := iam.NewRole(ctx, serviceName, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(assumeRolePolicy),
	}, opts...)
	if err != nil {
		return nil, err
	}

	return taskRole, nil
}
