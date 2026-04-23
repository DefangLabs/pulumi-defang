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
	AlbSG         *ec2.SecurityGroup
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
	certificateArn pulumi.StringPtrInput,
	opt pulumi.ResourceOrInvokeOption,
) (*AlbResult, error) {
	// Create ALB security group allowing HTTP/HTTPS ingress
	albSG, err := ec2.NewSecurityGroup(ctx, "alb-sg", &ec2.SecurityGroupArgs{
		VpcId:       vpcID,
		Description: pulumi.String("ALB security group"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("Allow incoming HTTP traffic"),
				Protocol:    pulumi.String("tcp"),
				FromPort:    pulumi.Int(80),
				ToPort:      pulumi.Int(80),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("Allow incoming HTTPS traffic"),
				Protocol:    pulumi.String("tcp"),
				FromPort:    pulumi.Int(443),
				ToPort:      pulumi.Int(443),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Description: pulumi.String("Allow all outbound traffic"),
				Protocol:    pulumi.String("-1"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opt, pulumi.Timeouts(&pulumi.CustomTimeouts{Delete: "2m"}))
	if err != nil {
		return nil, fmt.Errorf("creating ALB security group: %w", err)
	}

	// Create ALB
	const name = "alb"
	albArgs := &lb.LoadBalancerArgs{
		Internal:                 pulumi.Bool(false),
		LoadBalancerType:         pulumi.String("application"),
		SecurityGroups:           pulumi.StringArray{albSG.ID()},
		Subnets:                  subnetIDs,
		EnableDeletionProtection: pulumi.Bool(DeletionProtection.Get(ctx)),
		Tags: pulumi.StringMap{
			"defang:scope": pulumi.String("pub"),
		},
	}

	if AlbAccessLogs.Get(ctx) {
		logsBucket, logErr := createLbLogsBucket(ctx, name+"-logs", LoadBalancerTypeApplication, opt)
		if logErr != nil {
			return nil, fmt.Errorf("creating ALB logs bucket: %w", logErr)
		}
		albArgs.AccessLogs = &lb.LoadBalancerAccessLogsArgs{
			Bucket:  logsBucket.ID(),
			Enabled: pulumi.Bool(true),
		}
	}

	alb, err := lb.NewLoadBalancer(ctx, name, albArgs, opt)
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
		httpsListener, err = lb.NewListener(ctx, name+"-https", &lb.ListenerArgs{
			CertificateArn:  certificateArn,
			LoadBalancerArn: alb.Arn,
			Port:            pulumi.Int(443),
			Protocol:        pulumi.String("HTTPS"),
			DefaultActions:  defaultActions,
		}, opt)
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
	httpListener, err := lb.NewListener(ctx, name+"-http", &lb.ListenerArgs{
		LoadBalancerArn: alb.Arn,
		Port:            pulumi.Int(80),
		Protocol:        pulumi.String("HTTP"),
		DefaultActions:  defaultActions,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP listener: %w", err)
	}

	return &AlbResult{
		Alb:           alb,
		AlbSG:         albSG,
		HttpListener:  httpListener,
		HttpsListener: httpsListener,
	}, nil
}
