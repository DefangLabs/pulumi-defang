package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

/*
// BuildSharedInfra creates all shared AWS infrastructure for a standalone ECS service.
// The AWS provider must be passed via opts (pulumi.Providers on the parent component).
func BuildSharedInfra(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	awsCfg *SharedInfra,
	opts ...pulumi.ResourceOption,
) (*SharedInfra, error) {
	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}
	profile := config.New(ctx, "aws").Get("profile")

	net, err := ResolveNetworking(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       net.VpcID,
		Description: pulumi.String("Security group for services"),
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
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := CreateExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	var imgInfra *ImageInfra
	if svc.NeedsBuild() {
		imgInfra, err = CreateImageInfra(ctx, logGroup, profile, region.Region, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating image build infrastructure: %w", err)
		}
	}

	var httpListener *lb.Listener
	var svcALB *lb.LoadBalancer
	if svc.HasIngressPorts() {
		albRes, err := CreateALB(ctx, net.VpcID, net.PublicSubnetIDs, sg, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albRes.HttpListener
		svcALB = albRes.Alb
	}

	return &SharedInfra{
		Cluster:          cluster,
		ExecRole:         execRole,
		LogGroup:         logGroup,
		VpcID:            net.VpcID,
		PublicSubnetIDs:  net.PublicSubnetIDs,
		PrivateSubnetIDs: net.PrivateSubnetIDs,
		PrivateZoneID:    net.PrivateZoneID,
		PrivateDomain:    net.PrivateDomain,
		SkipNatGW:        !net.UseNatGW,
		Sg:               sg,
		HttpListener:     httpListener,
		HttpsListener:    nil, // TODO: support HTTPS listener
		Alb:              svcALB,
		Region:           region.Region,
		ImageInfra:       imgInfra,
	}, nil
}
*/

// BuildProjectInfra creates shared AWS infrastructure for a multi-service project.
//
//nolint:funlen // sequential infra setup is clearer as one function
func BuildProjectInfra(
	ctx *pulumi.Context,
	projectName string,
	services compose.Services,
	opts ...pulumi.ResourceOption,
) (*SharedInfra, error) {
	region, err := aws.GetRegion(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := ResolveNetworking(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	sg, err := ec2.NewSecurityGroup(ctx, "svc-sg", &ec2.SecurityGroupArgs{
		VpcId:       net.VpcID,
		Description: pulumi.String(fmt.Sprintf("Security group for %s services", projectName)),
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
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := CreateExecutionRole(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	profile := config.New(ctx, "aws").Get("profile")

	var imgInfra *ImageInfra
	for _, svc := range services {
		if svc.NeedsBuild() {
			imgInfra, err = CreateImageInfra(ctx, logGroup, profile, region.Region, opts...)
			if err != nil {
				return nil, fmt.Errorf("creating image build infrastructure: %w", err)
			}
			break
		}
	}

	var httpListener *lb.Listener
	var alb *lb.LoadBalancer
	if common.NeedIngress(services) {
		domain := "crew.defangio.click"

		delegationSetId := ""
		publicZone, err := getOrCreatePublicZone(ctx, domain, delegationSetId, opts...)
		if err != nil {
			return nil, fmt.Errorf("looking up Route53 zone: %w", err)
		}

		wildcardDomain := "*." + domain

		// Create a wildcard and/or apex domain certificate for all the ALBs
		// TODO: should we use a separate cert for each ALB?
		domains := []string{wildcardDomain}

		const createApexDnsRecord = true
		if createApexDnsRecord {
			domains = append(domains, domain)
		}

		// Create a wildcard and/or apex domain certificate for all the ALBs
		// TODO: should we use a separate cert for each ALB?
		certArn, err := createCertificateDNS(ctx, domains, CertificateDnsArgs{
			CaaIssuer: []string{"amazon.com", "letsencrypt.org"}, // TODO: only add letsencrypt.org if we plan to use ACME
			Zone:      publicZone,
			Tags: pulumi.StringMap{
				"defang:scope": pulumi.String("pub"),
			},
		}, common.MergeOptions(opts,
			pulumi.RetainOnDelete(true), // deletion will fail if there's a listener: keep it, ACM certs are free anyway
		)...)
		if err != nil {
			return nil, fmt.Errorf("creating certificate: %w", err)
		}

		albRes, err := CreateALB(ctx, net.VpcID, net.PublicSubnetIDs, sg, certArn, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}
		httpListener = albRes.HttpListener
		alb = albRes.Alb

		// Create ALIAS DNS records for the ALB
		aliases := route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				EvaluateTargetHealth: pulumi.Bool(true),
				Name:                 alb.DnsName,
				ZoneId:               alb.ZoneId,
			},
		}
		for _, hostname := range domains {
			_, err := CreateRecord(ctx, hostname, RecordTypeA, &route53.RecordArgs{
				Aliases: aliases,
				ZoneId:  publicZone.ZoneId(),
			}, opts...) // TODO: route53Opts
			if err != nil {
				return nil, fmt.Errorf("creating DNS record for %s: %w", hostname, err)
			}
		}
	}

	var bedrockPolicy *iam.Policy
	if common.IsProjectUsingLLM(services) {
		bedrockPolicy, err = createBedrockPolicy(ctx, "BedrockPolicy", []string{}, opts...) // all models
		if err != nil {
			return nil, fmt.Errorf("creating Bedrock policy: %w", err)
		}
	}

	route53SidecarePolicy, err := createRoute53SidecarPolicy(ctx, "AllowRoute53Sidecar", net.PrivateZone, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Route53 sidecar policy: %w", err)
	}

	return &SharedInfra{
		Policies: Policies{
			bedrockPolicy:        bedrockPolicy,
			route53SidecarPolicy: route53SidecarePolicy,
		},
		Cluster:          cluster,
		ExecRole:         execRole,
		LogGroup:         logGroup,
		VpcID:            net.VpcID,
		PublicSubnetIDs:  net.PublicSubnetIDs,
		PrivateSubnetIDs: net.PrivateSubnetIDs,
		PrivateZoneID:    net.PrivateZone.ID().ToIDPtrOutput(),
		PrivateDomain:    net.PrivateDomain,
		Sg:               sg,
		HttpListener:     httpListener,
		Alb:              alb,
		Region:           region.Region,
		ImageInfra:       imgInfra,
	}, nil
}
