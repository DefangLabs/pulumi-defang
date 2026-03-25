package aws

import (
	"errors"
	"slices"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var ErrEmptyHostnames = errors.New("hostnames must not be empty")

type CertificateDnsArgs struct {
	CaaIssuer []string
	// Route53Provider pulumi.ProviderResource
	Tags pulumi.StringMapInput
	// record name=>fqdn to avoid creating conflicting validation records
	ValidationRecords map[string]pulumi.StringOutput
	Zone              IZone
}

// createCertificateDNS creates a new ACM certificate with DNS validation w/ or w/o CAA records.
// This needs aws:RequestCertificate IAM permission.
func createCertificateDNS(
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
			rec, err := createCaaDnsRecord(ctx, hostname, args.Zone, args.CaaIssuer, opts...)
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
					record, err := createValidationDnsRecord(ctx, dvo, args.Zone, opts...)
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
	zone IZone,
	opts ...pulumi.ResourceOption,
) (*route53.Record, error) {
	return CreateRecord(ctx,
		*cdvo.ResourceRecordName,
		RecordType(*cdvo.ResourceRecordType),
		&route53.RecordArgs{
			Records: pulumi.StringArray{pulumi.String(*cdvo.ResourceRecordValue)},
			Ttl:     pulumi.Int(60), // 1 MINUTE
			ZoneId:  zone.ZoneId(),
		},
		common.MergeOptions(opts,
			pulumi.DeleteBeforeReplace(true), // HACK: workaround for "already exists" error
			pulumi.RetainOnDelete(false),     // we don't need to keep validation records around
		)...,
	)
}
