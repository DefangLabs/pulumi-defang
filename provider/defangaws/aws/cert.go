package aws

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrEmptyHostnames = errors.New("hostnames must not be empty")

// groupHostnamesByCert groups hostnames by their ACM DNS validation base domain.
// Hostnames that differ only by a "*." prefix (e.g. "example.com" and "*.example.com")
// produce the same validation CNAME record, so they belong on a single certificate.
// The returned map keys are the base domain (without "*." prefix); each value slice
// has the non-wildcard domain first (cert CN), followed by any wildcards (SANs).
func groupHostnamesByCert(hostnames []string) map[string][]string {
	groups := make(map[string][]string)
	for _, hostname := range hostnames {
		domain, wildcard := strings.CutPrefix(hostname, "*.")
		if wildcard {
			// Wildcard: append as SAN
			groups[domain] = append(groups[domain], hostname)
		} else {
			// Non-wildcard: prepend so it becomes CN
			groups[domain] = append([]string{hostname}, groups[domain]...)
		}
	}
	return groups
}

type CertificateDnsArgs struct {
	CaaIssuer []string
	// Route53Provider aws.Provider
	Tags              pulumi.StringMapInput
	ValidationRecords map[string]pulumi.StringOutput
	ZoneId            pulumi.StringInput
}

// CreateCertificateDNS creates a new ACM certificate with DNS validation w/ or w/o CAA records.
// This needs aws:RequestCertificate IAM permission.
func CreateCertificateDNS(
	ctx *pulumi.Context,
	hostnames []string,
	args CertificateDnsArgs,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	if len(hostnames) == 0 {
		return pulumi.StringOutput{}, ErrEmptyHostnames
	}
	domainName := hostnames[0]

	certificate, err := acm.NewCertificate(ctx, domainName, &acm.CertificateArgs{
		DomainName:              pulumi.String(domainName),
		SubjectAlternativeNames: pulumi.ToStringArray(hostnames[1:]),
		ValidationMethod:        pulumi.String("DNS"),
		Tags:                    args.Tags,
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	// Remove RetainOnDelete from opts for validation resources
	// TODO: filter out RetainOnDelete from opts
	resourceOpts := opts

	// var route53Opts []pulumi.ResourceOption
	// if args != nil && args.Route53Provider != nil {
	// 	route53Opts = append(append([]pulumi.ResourceOption{}, resourceOpts...), pulumi.Provider(args.Route53Provider))
	// } else {
	// 	route53Opts = resourceOpts
	// }

	// NOTE: creation of CAA can fail with "RRSet of type CAA with DNS name sub.example.com. is not permitted
	// because a conflicting RRSet of type CNAME with the same DNS name already exists in zone example.com"
	var caaRecords []pulumi.Resource
	if len(args.CaaIssuer) != 0 {
		for _, hostname := range hostnames {
			rec, err := createCaaDnsRecord(ctx, hostname, args.ZoneId, args.CaaIssuer, opts...)
			if err != nil {
				return pulumi.StringOutput{}, err
			}
			caaRecords = append(caaRecords, rec)
		}
	}

	// Adapted from https://www.pulumi.com/registry/packages/aws/api-docs/acm/certificatevalidation/
	validationRecordFqdns := certificate.DomainValidationOptions.ApplyT(
		func(dvos []acm.CertificateDomainValidationOption) (pulumi.StringArrayOutput, error) {
			// Dedup validation records using CNAME to prevent deployment DNS collision
			seen := map[string]bool{}
			var deduped []acm.CertificateDomainValidationOption
			for _, dvo := range dvos {
				if !seen[*dvo.ResourceRecordName] {
					seen[*dvo.ResourceRecordName] = true
					deduped = append(deduped, dvo)
				}
			}
			// Sort the deduped records by domain name to ensure consistent ordering of validationRecordFqdns
			slices.SortFunc(deduped, func(a, b acm.CertificateDomainValidationOption) int {
				return strings.Compare(*a.DomainName, *b.DomainName)
			})
			// HACK: multiple domains may result in conflicting validation records,
			// so we use the original domainName as part of the Pulumi resource name
			// and always allow overwriting existing validation records.
			// See https://github.com/DefangLabs/defang-mvp/issues/1938
			// (We still need the dedup above, or we'd risk adding the same FQDN twice.)
			fqdns := make(pulumi.StringArray, len(deduped))
			for i, dvo := range deduped {
				if existing, ok := args.ValidationRecords[*dvo.ResourceRecordName]; ok {
					fqdns[i] = existing
				} else {
					record, err := createValidationDnsRecord(ctx, dvo, args.ZoneId, opts...)
					if err != nil {
						return pulumi.StringArrayOutput{}, err
					}
					if args.ValidationRecords != nil {
						args.ValidationRecords[*dvo.ResourceRecordName] = record.Fqdn
					}
					fqdns[i] = record.Fqdn
				}
			}
			return fqdns.ToStringArrayOutput(), nil
		}).(pulumi.StringArrayOutput)

	// NOTE: creation of the CertificateValidation resource will fail with CAA_ERROR
	// if there's currently a CAA record that doesn't allow AWS
	certValidationName := domainName + "Validation"
	certificateValidation, err := acm.NewCertificateValidation(ctx, certValidationName, &acm.CertificateValidationArgs{
		CertificateArn:        certificate.Arn,
		ValidationRecordFqdns: validationRecordFqdns,
	}, common.MergeOptions(resourceOpts,
		// validation may never finish, but at least we avoid locked stacks
		pulumi.Timeouts(&pulumi.CustomTimeouts{Create: "10m", Update: "10m"}),
		pulumi.DependsOn(caaRecords), // add CAA records as a dependency to avoid CAA_ERROR
	)...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}
	// Use the CertificateValidation resource, so we can depend on it (and ensure the cert is valid)
	return certificateValidation.CertificateArn, nil
}

func createValidationDnsRecord(
	ctx *pulumi.Context,
	cdvo acm.CertificateDomainValidationOption,
	zoneId pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*route53.Record, error) {
	return CreateRecord(ctx,
		*cdvo.ResourceRecordName,
		RecordType(*cdvo.ResourceRecordType),
		&route53.RecordArgs{
			Records: pulumi.StringArray{pulumi.String(*cdvo.ResourceRecordValue)},
			Ttl:     pulumi.Int(60), // 1 MINUTE
			ZoneId:  zoneId,
		},
		common.MergeOptions(opts,
			pulumi.DeleteBeforeReplace(true), // HACK: workaround for "already exists" error
			pulumi.RetainOnDelete(false),     // we don't need to keep validation records around
		)...,
	)
}

// createCertsAndRoute53Dns creates BYOD (Bring Your Own Domain) ACM certificates,
// listener attachments, and DNS records as children of the service component.
// Hostnames that share the same DNS validation record are grouped onto one cert
// (e.g. example.com and *.example.com).
func createCertsAndRoute53Dns(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	infra *SharedInfra,
	opt pulumi.ResourceOrInvokeOption,
) error {
	if infra == nil || infra.HttpsListener == nil || infra.Alb == nil {
		return nil
	}
	if svc.DomainName == "" {
		return nil
	}

	// Collect all BYOD hostnames: domainname + network aliases
	hostnames := append([]string{svc.DomainName}, svc.DefaultNetwork().Aliases...)

	// Group hostnames that share the same validation record onto one cert
	certGroups := groupHostnamesByCert(hostnames)

	albAliases := route53.RecordAliasArray{
		&route53.RecordAliasArgs{
			EvaluateTargetHealth: pulumi.Bool(true),
			Name:                 infra.Alb.DnsName,
			ZoneId:               infra.Alb.ZoneId,
		},
	}

	// Iterate groups in sorted order for deterministic resource names
	for base, group := range certGroups {
		// Look up the Route53 hosted zone for this domain
		zone, err := GetHostedZoneForHost(ctx, base, opt)
		if err != nil {
			return fmt.Errorf("finding hosted zone for %s: %w", base, err)
		}

		certArn, err := CreateCertificateDNS(ctx, group, CertificateDnsArgs{
			CaaIssuer: []string{"amazon.com", "letsencrypt.org"},
			ZoneId:    pulumi.String(zone.Id),
			Tags: pulumi.StringMap{
				"defang:service": pulumi.String(serviceName),
			},
		}, opt, pulumi.RetainOnDelete(true))
		if err != nil {
			return fmt.Errorf("creating BYOD certificate for %s: %w", base, err)
		}

		_, err = lb.NewListenerCertificate(ctx, serviceName+"-"+base+"-cert", &lb.ListenerCertificateArgs{
			ListenerArn:    infra.HttpsListener.Arn,
			CertificateArn: certArn,
		}, opt)
		if err != nil {
			return fmt.Errorf("attaching BYOD certificate for %s: %w", base, err)
		}

		// Create ALIAS DNS records for each hostname in this group → ALB
		zoneId := pulumi.String(zone.Id)
		for _, hostname := range group {
			_, err = CreateRecord(ctx, hostname, common.RecordTypeA, &route53.RecordArgs{
				Aliases: albAliases,
				ZoneId:  zoneId,
			}, opt)
			if err != nil {
				return fmt.Errorf("creating DNS record for %s: %w", hostname, err)
			}
		}
	}
	return nil
}
