package aws

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type albResult struct {
	alb          *lb.LoadBalancer
	httpListener *lb.Listener
}

// createALB creates an Application Load Balancer with an HTTP listener.
func createALB(
	ctx *pulumi.Context,
	vpcID pulumix.Output[string],
	subnetIDs pulumix.Output[[]string],
	serviceSG *ec2.SecurityGroup,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*albResult, error) {
	// Create ALB security group allowing HTTP/HTTPS ingress
	albSG, err := ec2.NewSecurityGroup(ctx, "alb-sg", &ec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(vpcID),
		Description: pulumi.String("ALB security group"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(80),
				ToPort:     pulumi.Int(80),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(443),
				ToPort:     pulumi.Int(443),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ALB security group: %w", err)
	}

	// Allow traffic from ALB to service security group
	_, err = ec2.NewSecurityGroupRule(ctx, "alb-to-svc", &ec2.SecurityGroupRuleArgs{
		Type:                  pulumi.String("ingress"),
		FromPort:              pulumi.Int(0),
		ToPort:                pulumi.Int(65535),
		Protocol:              pulumi.String("tcp"),
		SecurityGroupId:       serviceSG.ID(),
		SourceSecurityGroupId: albSG.ID(),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ALB-to-service SG rule: %w", err)
	}

	// Create ALB
	alb, err := lb.NewLoadBalancer(ctx, "alb", &lb.LoadBalancerArgs{
		Internal:                 pulumi.Bool(false),
		LoadBalancerType:         pulumi.String("application"),
		SecurityGroups:           pulumi.StringArray{albSG.ID()},
		Subnets:                  pulumi.StringArrayOutput(subnetIDs),
		EnableDeletionProtection: pulumi.Bool(recipe.DeletionProtection),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ALB: %w", err)
	}

	// Create HTTP listener with default 404 response
	httpListener, err := lb.NewListener(ctx, "http-listener", &lb.ListenerArgs{
		LoadBalancerArn: alb.Arn,
		Port:            pulumi.Int(80),
		Protocol:        pulumi.String("HTTP"),
		DefaultActions: lb.ListenerDefaultActionArray{
			&lb.ListenerDefaultActionArgs{
				Type: pulumi.String("fixed-response"),
				FixedResponse: &lb.ListenerDefaultActionFixedResponseArgs{
					ContentType: pulumi.String("text/plain"),
					MessageBody: pulumi.String("Not Found"),
					StatusCode:  pulumi.String("404"),
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP listener: %w", err)
	}

	return &albResult{
		alb:          alb,
		httpListener: httpListener,
	}, nil
}
