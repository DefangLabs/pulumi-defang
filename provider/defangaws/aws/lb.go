// Ported from https://github.com/DefangLabs/defang-mvp/blob/main/pulumi/shared/aws/lb.ts
package aws

import (
	"fmt"

	"encoding/json"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type LoadBalancerType string

const (
	LoadBalancerTypeApplication LoadBalancerType = "application"
	LoadBalancerTypeNetwork     LoadBalancerType = "network"
)

func createLbLogsBucket(
	ctx *pulumi.Context,
	name string,
	typ LoadBalancerType,
	opt pulumi.ResourceOrInvokeOption,
) (*s3.Bucket, error) {
	lbAccountId, err := getCallerAccountId(ctx, opt)
	if err != nil {
		return nil, err
	}

	var lbPrincipal any
	if typ == LoadBalancerTypeNetwork {
		lbPrincipal = getNlbPrincipal()
	} else {
		lbRegion, err := getCallerRegion(ctx, opt)
		if err != nil {
			return nil, err
		}
		lbPrincipal = getElbPrincipal(lbRegion)
	}

	sseRules := s3.BucketServerSideEncryptionConfigurationRuleArray{
		s3.BucketServerSideEncryptionConfigurationRuleArgs{
			ApplyServerSideEncryptionByDefault: &s3.
				BucketServerSideEncryptionConfigurationRuleApplyServerSideEncryptionByDefaultArgs{
				SseAlgorithm: pulumi.String("AES256"),
			},
			BucketKeyEnabled: pulumi.Bool(true), // frequently accessed objects will use bucket keys
		},
	}
	bucket, err := createPrivateBucket(ctx, name, &s3.BucketArgs{}, sseRules, opt)
	if err != nil {
		return nil, err
	}

	// Expire logs after N days (from recipe)
	_, err = s3.NewBucketLifecycleConfiguration(ctx, name, &s3.BucketLifecycleConfigurationArgs{
		Bucket: bucket.ID(),
		Rules: s3.BucketLifecycleConfigurationRuleArray{
			s3.BucketLifecycleConfigurationRuleArgs{
				Id:     pulumi.String("expire-logs"),
				Status: pulumi.String("Enabled"),
				Expiration: &s3.BucketLifecycleConfigurationRuleExpirationArgs{
					Days: pulumi.Int(common.LogRetentionDays.Get(ctx)),
				},
			},
		},
	}, opt)
	if err != nil {
		return nil, err
	}

	// From AWS docs on access logging bucket requirements for NLB and ALB:
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/network/load-balancer-access-logs.html
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html
	policyJson := bucket.ID().ApplyT(func(bucketId string) (string, error) {
		policy := PolicyDocument{
			Version: "2012-10-17",
			Statement: []PolicyStatement{
				{
					Sid:       "AWSLogDeliveryWrite",
					Effect:    "Allow",
					Principal: lbPrincipal,
					Action:    "s3:PutObject",
					Resource:  `arn:aws:s3:::` + bucketId + `/AWSLogs/` + lbAccountId + `/*`,
				},
				{
					Sid:       "AWSLogDeliveryAclCheck",
					Effect:    "Allow",
					Principal: lbPrincipal,
					Action:    "s3:GetBucketAcl",
					Resource:  `arn:aws:s3:::` + bucketId,
				},
			},
		}
		b, err := json.Marshal(policy)
		return string(b), err
	}).(pulumi.StringOutput)

	_, err = s3.NewBucketPolicy(
		ctx,
		name,
		&s3.BucketPolicyArgs{
			Bucket: bucket.ID(),
			Policy: policyJson,
		},
		opt)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

//nolint:funlen // sequential TG+LR setup is clearer as one function
func createTgLrPair(
	ctx *pulumi.Context,
	serviceName string,
	vpcId pulumi.StringInput,
	listener *lb.Listener,
	port compose.ServicePortConfig,
	healthCheck *compose.HealthCheckConfig,
	endpoints []string,
	albDnsName pulumi.StringInput, // fallback host header when no endpoints are configured
	opt pulumi.ResourceOption,
) (*lb.TargetGroup, *lb.ListenerRule, error) {
	if !port.IsIngress() {
		return nil, nil, nil // skip
	}

	// Only create TG/LR for http, http2, grpc (matches TS createTgLrPair)
	appProto := port.GetAppProtocol()
	if appProto != "http" && appProto != "http2" && appProto != "grpc" {
		return nil, nil, nil // skip
	}

	tgName := targetGroupName(serviceName, int(port.Target), appProto)

	// Target group health check (matches TS createTargetGroup in lb.ts)
	defaultInterval := HealthCheckInterval.Get(ctx)
	interval := defaultInterval
	if healthCheck != nil {
		interval = clampInt(int(healthCheck.IntervalSeconds), 5, 300, defaultInterval)
	}
	maxTimeout := interval - 1
	if maxTimeout > 120 {
		maxTimeout = 120
	}
	// Default timeout to interval, clamped to [2, maxTimeout] (matches TS: `healthCheck?.timeout ?? interval`)
	timeout := clampInt(interval, 2, maxTimeout, interval)
	if healthCheck != nil {
		timeout = clampInt(int(healthCheck.TimeoutSeconds), 2, maxTimeout, interval)
	}
	unhealthyThreshold := (3)
	if healthCheck != nil {
		unhealthyThreshold = clampInt(int(healthCheck.Retries), 2, 10, 3)
	}

	// Determine matcher based on protocol (matches TS createTargetGroup)
	// With default path "/": grpc -> "0", http/http2 -> "200-399"
	matcher := "200-399"
	if appProto == compose.PortAppProtocolGRPC {
		matcher = "0"
	}

	tgArgs := &lb.TargetGroupArgs{
		Port:                       pulumi.Int(port.Target),
		Protocol:                   pulumi.String("HTTP"),
		TargetType:                 pulumi.String("ip"),
		VpcId:                      vpcId,
		LoadBalancingAlgorithmType: pulumi.String("least_outstanding_requests"),
		DeregistrationDelay:        pulumi.Int(DeregistrationDelay.Get(ctx)),
		HealthCheck: &lb.TargetGroupHealthCheckArgs{
			// Port:               pulumi.String("traffic-port"),
			Path:               pulumi.String("/"),
			HealthyThreshold:   pulumi.Int(HealthCheckThreshold.Get(ctx)),
			UnhealthyThreshold: pulumi.Int(unhealthyThreshold),
			Interval:           pulumi.Int(interval),
			Timeout:            pulumi.Int(timeout),
			Matcher:            pulumi.String(matcher),
		},
		Tags: pulumi.StringMap{
			"defang:service": pulumi.String(serviceName),
		},
	}

	// Set protocol version for http2/grpc (matches TS createTargetGroup)
	switch appProto {
	case compose.PortAppProtocolHTTP2:
		tgArgs.ProtocolVersion = pulumi.String("HTTP2")
	case compose.PortAppProtocolGRPC:
		tgArgs.ProtocolVersion = pulumi.String("GRPC")
	case compose.PortAppProtocolHTTP, compose.PortAppProtocolUnknown:
		// defaults to HTTP1
	}

	tg, tgErr := lb.NewTargetGroup(ctx, tgName, tgArgs, opt)
	if tgErr != nil {
		return nil, nil, fmt.Errorf("creating target group: %w", tgErr)
	}

	// Build listener rule conditions (matches TS createTgLrPair)
	conditions := lb.ListenerRuleConditionArray{}

	// Host-based routing: use endpoints if available, otherwise fall back to the ALB DNS name
	// (matches TS: `values.length || !fallback ? values : [fallback]`)
	if len(endpoints) > 0 {
		conditions = append(conditions, &lb.ListenerRuleConditionArgs{
			HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
				Values: compose.ToPulumiStringArray(endpoints),
			},
		})
	} else {
		// Note: if no endpoints are available, only the first service will be reachable
		conditions = append(conditions, &lb.ListenerRuleConditionArgs{
			HostHeader: &lb.ListenerRuleConditionHostHeaderArgs{
				Values: pulumi.StringArray{albDnsName},
			},
		})
	}

	// TODO: path-based routing
	// path := splitHostPortPath(endpoint[])
	// conditions = append(conditions, &lb.ListenerRuleConditionArgs{
	// 	PathPattern: &lb.ListenerRuleConditionPathPatternArgs{
	// 		Values: pulumi.StringArray{pulumi.String(path)},
	// 	},
	// })

	// Add gRPC content-type header matching (matches TS createTgLrPair)
	if appProto == compose.PortAppProtocolGRPC {
		conditions = append(conditions, &lb.ListenerRuleConditionArgs{
			HttpHeader: &lb.ListenerRuleConditionHttpHeaderArgs{
				HttpHeaderName: pulumi.String("content-type"),
				Values:         pulumi.StringArray{pulumi.String("application/grpc*")},
			},
		})
	}

	lr, lrErr := lb.NewListenerRule(ctx, tgName+"-rule", &lb.ListenerRuleArgs{
		ListenerArn: listener.Arn,
		Actions: lb.ListenerRuleActionArray{
			&lb.ListenerRuleActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: tg.Arn,
			},
		},
		Conditions: conditions,
	}, opt, pulumi.DeleteBeforeReplace(true))
	if lrErr != nil {
		return nil, nil, fmt.Errorf("creating listener rule: %w", lrErr)
	}

	return tg, lr, nil
}
