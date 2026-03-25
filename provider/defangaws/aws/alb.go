package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AlbResult struct {
	Alb           *lb.LoadBalancer
	HttpListener  *lb.Listener
	HttpsListener *lb.Listener
}

// CreateALB creates an Application Load Balancer with an HTTP listener.
//
//nolint:funlen // sequential ALB setup is clearer as one function
func CreateALB(
	ctx *pulumi.Context,
	vpcID pulumi.StringInput,
	subnetIDs pulumi.StringArrayInput,
	serviceSG *ec2.SecurityGroup,
	certificateArn pulumi.StringPtrInput,
	opts ...pulumi.ResourceOption,
) (*AlbResult, error) {
	// Create ALB security group allowing HTTP/HTTPS ingress
	albSG, err := ec2.NewSecurityGroup(ctx, "alb-sg", &ec2.SecurityGroupArgs{
		VpcId:       vpcID,
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
		// AccessLogs: TODO,
		Internal:                 pulumi.Bool(false),
		LoadBalancerType:         pulumi.String("application"),
		SecurityGroups:           pulumi.StringArray{albSG.ID()},
		Subnets:                  subnetIDs,
		EnableDeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
		Tags: pulumi.StringMap{
			"defang:scope": pulumi.String("pub"),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ALB: %w", err)
	}

	defaultActions := lb.ListenerDefaultActionArray{
		&lb.ListenerDefaultActionArgs{
			Type: pulumi.String("fixed-response"),
			FixedResponse: &lb.ListenerDefaultActionFixedResponseArgs{
				ContentType: pulumi.String("text/html"),
				MessageBody: pulumi.String(common.GetErrorHtml(
					"404 Not Found",
					"Resource Not Found",
					"The service you are looking for doesn't exist or is pending deployment. Check its status or domain name.",
				)),
				StatusCode: pulumi.String("404"),
			},
		},
	}

	var httpsListener *lb.Listener
	if certificateArn != nil {
		// Create HTTPS listener with default 404 response
		httpsListener, err = lb.NewListener(ctx, "http-listener", &lb.ListenerArgs{
			CertificateArn:  certificateArn,
			LoadBalancerArn: alb.Arn,
			Port:            pulumi.Int(443),
			Protocol:        pulumi.String("HTTPS"),
			DefaultActions:  defaultActions,
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating HTTPS listener: %w", err)
		}

		// Make HTTP listener redirect to HTTPS
		defaultActions = lb.ListenerDefaultActionArray{
			&lb.ListenerDefaultActionArgs{
				Type: pulumi.String("redirect"),
				Redirect: &lb.ListenerDefaultActionRedirectArgs{
					Port:       pulumi.String("443"),
					Protocol:   pulumi.String("HTTPS"),
					StatusCode: pulumi.String(HttpRedirectToHttps.Get(ctx)),
				},
			},
		}
	}

	// Create HTTP listener with default 404 response
	httpListener, err := lb.NewListener(ctx, "http-listener", &lb.ListenerArgs{
		LoadBalancerArn: alb.Arn,
		Port:            pulumi.Int(80),
		Protocol:        pulumi.String("HTTP"),
		DefaultActions:  defaultActions,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP listener: %w", err)
	}

	return &AlbResult{
		Alb:           alb,
		HttpListener:  httpListener,
		HttpsListener: httpsListener,
	}, nil
}
