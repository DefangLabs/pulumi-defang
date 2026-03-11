package aws

import (
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// createExecutionRole creates the shared ECS task execution role.
func createExecutionRole(ctx *pulumi.Context, projectName string, opts ...pulumi.ResourceOption) (*iam.Role, error) {
	assumeRolePolicy, _ := json.Marshal(map[string]interface{}{
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

	execRole, err := iam.NewRole(ctx, projectName+"-exec-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(assumeRolePolicy)),
		Tags: pulumi.StringMap{
			"defang:project": pulumi.String(projectName),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	// Attach the ECS task execution policy
	_, err = iam.NewRolePolicyAttachment(ctx, projectName+"-exec-policy", &iam.RolePolicyAttachmentArgs{
		Role:      execRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("attaching execution policy: %w", err)
	}

	return execRole, nil
}

// createTaskRole creates a per-service ECS task role.
func createTaskRole(ctx *pulumi.Context, projectName, serviceName string, opts ...pulumi.ResourceOption) (*iam.Role, error) {
	assumeRolePolicy, _ := json.Marshal(map[string]interface{}{
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

	taskRole, err := iam.NewRole(ctx, projectName+"-"+serviceName+"-task-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(assumeRolePolicy)),
		Tags: pulumi.StringMap{
			"defang:project": pulumi.String(projectName),
			"defang:service": pulumi.String(serviceName),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	return taskRole, nil
}
