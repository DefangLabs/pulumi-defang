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
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CreateProjectInfra creates shared AWS infrastructure for a multi-service project.
//
//nolint:funlen,maintidx // sequential infra setup is clearer as one function
func CreateProjectInfra(
	ctx *pulumi.Context,
	projectName string,
	awsConfig *AWSConfig,
	services compose.Services,
	opt pulumi.ResourceOrInvokeOption,
) (*SharedInfra, error) {
	region, err := aws.GetRegion(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("getting AWS region: %w", err)
	}

	net, err := ResolveNetworking(ctx, projectName, awsConfig, opt)
	if err != nil {
		return nil, fmt.Errorf("resolving networking: %w", err)
	}

	privateSg, err := ec2.NewSecurityGroup(ctx, "svc-sg", &ec2.SecurityGroupArgs{
		VpcId:       net.VpcID,
		Description: pulumi.String(fmt.Sprintf("Security group for %s services", projectName)),
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Description: pulumi.String("Allow all outbound traffic"),
				Protocol:    pulumi.String("-1"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opt, pulumi.Timeouts(&pulumi.CustomTimeouts{Delete: "2m"})) // lowered, to fail quickly when SG is in use
	if err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}

	cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating ECS cluster: %w", err)
	}

	logGroup, err := cloudwatch.NewLogGroup(ctx, "logs", &cloudwatch.LogGroupArgs{
		RetentionInDays: pulumi.Int(LogRetentionDays.Get(ctx)),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating log group: %w", err)
	}

	execRole, err := CreateExecutionRole(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("creating execution role: %w", err)
	}

	var imgInfra *BuildInfra
	for _, svc := range services {
		if svc.NeedsBuild() {
			imgInfra, err = CreateBuildInfra(ctx, logGroup, region.Region, opt)
			if err != nil {
				return nil, fmt.Errorf("creating image build infrastructure: %w", err)
			}
			break
		}
	}

	// Create public ECR pull-through cache for faster image pulls (matches TS initializeStack)
	publicEcrCache, err := createEcrPullThroughCache(ctx, "ecr-public", pulumi.String("public.ecr.aws"), opt)
	if err != nil {
		return nil, fmt.Errorf("creating ECR pull-through cache: %w", err)
	}

	// Grant execution role permissions to use pull-through cache repos
	cacheRepoArn := pulumi.Sprintf("arn:aws:ecr:%s:%s:repository/%s/*",
		region.Region, publicEcrCache.Rule.RegistryId, publicEcrCache.Rule.EcrRepositoryPrefix)
	err = attachPullThroughCachePolicy(ctx, execRole, cacheRepoArn, opt)
	if err != nil {
		return nil, fmt.Errorf("attaching pull-through cache policy: %w", err)
	}

	var alarmTopicArn pulumi.StringInput
	if awsConfig != nil {
		alarmTopicArn = awsConfig.AlarmTopicArn
	}

	// Role-assuming provider for public Route53 operations in another account
	var dnsProvider pulumi.ProviderResource
	if awsConfig != nil && awsConfig.DnsRoleArn != nil {
		dnsProvider, err = aws.NewProvider(ctx, "route53", &aws.ProviderArgs{
			Region: pulumi.String(region.Region), // required for STS
			AssumeRoles: aws.ProviderAssumeRoleArray{
				&aws.ProviderAssumeRoleArgs{
					RoleArn: awsConfig.DnsRoleArn.ToStringOutput().ToStringPtrOutput(),
				},
			},
		}, opt)
		if err != nil {
			return nil, fmt.Errorf("creating Route53 provider: %w", err)
		}
	}
	dnsOpts := []pulumi.ResourceOption{opt}
	if dnsProvider != nil {
		dnsOpts = append(dnsOpts, pulumi.Provider(dnsProvider))
	}

	var projectDomain string
	var albRes *AlbResult
	if common.NeedIngress(services) {
		var certArn pulumi.StringPtrInput
		var domains []string
		var publicZoneId pulumi.StringInput

		// Create wildcard cert if a public zone is provided
		if awsConfig != nil && awsConfig.PublicZoneId != nil && awsConfig.ProjectDomain != "" {
			publicZoneId = awsConfig.PublicZoneId.ToStringPtrOutput().Elem() // TODO: look up?
			projectDomain = awsConfig.ProjectDomain

			domains = []string{"*." + projectDomain}
			if CreateApexRecord.Get(ctx) {
				domains = append(domains, projectDomain)
			}

			certArn, err = CreateCertificateDNS(ctx, domains, CertificateDnsArgs{
				CaaIssuer:       []string{"amazon.com", "letsencrypt.org"}, // FIXME: only pick CAs that we need
				ZoneId:          publicZoneId,
				Route53Provider: dnsProvider,
				Tags: pulumi.StringMap{
					"defang:scope": pulumi.String("pub"),
				},
			}, opt, pulumi.RetainOnDelete(true)) // deletion will fail if there's a listener: keep it, ACM certs are free anyway
			if err != nil {
				return nil, fmt.Errorf("creating certificate: %w", err)
			}
		} else if awsConfig != nil && awsConfig.AlbCertificateArn != nil {
			// Caller-provided default certificate for the HTTPS listener
			certArn = awsConfig.AlbCertificateArn.ToStringOutput().ToStringPtrOutput()
		}

		albRes, err = CreateALB(ctx, net.VpcID, net.PublicSubnetIDs, certArn, opt)
		if err != nil {
			return nil, fmt.Errorf("creating ALB: %w", err)
		}

		// Create ALIAS DNS records for the ALB
		aliases := route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				EvaluateTargetHealth: pulumi.Bool(true),
				Name:                 albRes.Alb.DnsName,
				ZoneId:               albRes.Alb.ZoneId,
			},
		}
		for _, hostname := range domains {
			_, err := CreateRecord(ctx, hostname, common.RecordTypeA, &route53.RecordArgs{
				Aliases: aliases,
				ZoneId:  publicZoneId,
			}, dnsOpts...)
			if err != nil {
				return nil, fmt.Errorf("creating DNS record for %s: %w", hostname, err)
			}
		}
	}

	var bedrockPolicy *iam.Policy
	if common.IsProjectUsingLLM(services) {
		bedrockPolicy, err = createBedrockPolicy(ctx, "BedrockPolicy", []string{}, opt) // all models
		if err != nil {
			return nil, fmt.Errorf("creating Bedrock policy: %w", err)
		}
	}

	route53SidecarePolicy, err := createRoute53SidecarPolicy(ctx, "AllowRoute53Sidecar", net.PrivateZone, opt)
	if err != nil {
		return nil, fmt.Errorf("creating Route53 sidecar policy: %w", err)
	}

	result := &SharedInfra{
		Policies: Policies{
			bedrockPolicy:        bedrockPolicy,
			route53SidecarPolicy: route53SidecarePolicy,
		},
		Cluster:          cluster,
		DnsProvider:      dnsProvider,
		ExecRole:         execRole,
		LogGroup:         logGroup,
		VpcID:            net.VpcID,
		PublicSubnetIDs:  net.PublicSubnetIDs,
		PrivateSubnetIDs: net.PrivateSubnetIDs,
		PrivateZoneID:    net.PrivateZone.ZoneId,
		PrivateDomain:    net.PrivateDomain,
		ProjectDomain:    projectDomain,
		PrivateSgID:      privateSg.ID(),
		AlarmTopicArn:    alarmTopicArn,
		SkipNatGW:        !net.UseNatGW,
		Region:           region.Region,
		BuildInfra:       imgInfra,
		PublicEcrCache:   publicEcrCache,
	}

	if albRes != nil {
		result.AlbSG = albRes.AlbSG
		result.HttpListener = albRes.HttpListener
		result.HttpsListener = albRes.HttpsListener
		result.Alb = albRes.Alb
	}

	return result, nil
}
