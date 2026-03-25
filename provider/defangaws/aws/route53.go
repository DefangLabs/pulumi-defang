package aws

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeCAA   RecordType = "CAA"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeDS    RecordType = "DS"
	RecordTypeMX    RecordType = "MX"
	RecordTypeNAPTR RecordType = "NAPTR"
	RecordTypeNS    RecordType = "NS"
	RecordTypePTR   RecordType = "PTR"
	RecordTypeSOA   RecordType = "SOA"
	RecordTypeSPF   RecordType = "SPF"
	RecordTypeSRV   RecordType = "SRV"
	RecordTypeTXT   RecordType = "TXT"
)

func NormalizeDNS(name string) string {
	return strings.ToLower(strings.TrimRight(name, "."))
}

func CreateRecord(
	ctx *pulumi.Context, name string, typ RecordType, args *route53.RecordArgs, opts ...pulumi.ResourceOption,
) (*route53.Record, error) {
	if args == nil {
		args = &route53.RecordArgs{}
	}
	// TODO: look up the zone by name
	// if args.ZoneId == nil {
	// }
	// Route 53 treats www.example.com (without a trailing dot) and www.example.com. (with a trailing dot) as identical.
	normalized := NormalizeDNS(name)
	args.Name = pulumi.String(normalized)
	args.Type = pulumi.String(string(typ))
	if args.AllowOverwrite == nil {
		args.AllowOverwrite = pulumi.BoolPtr(AllowOverwriteRecords.Get(ctx)) // allow overwrite existing records for ZDT
	}
	opts = append([]pulumi.ResourceOption{
		pulumi.DeleteBeforeReplace(true), // delete first, or we risk deleting the record that was just recreated
		pulumi.RetainOnDelete(RetainDnsOnDelete.Get(ctx)),
	}, opts...)
	return route53.NewRecord(ctx, normalized+"_"+string(typ), args, opts...)
}

type SoaRecordArgs struct {
	Expire     pulumi.IntPtrInput    // defaults to 1209600
	Minimum    pulumi.IntPtrInput    // TTL for negative responses; defaults to 86400
	NameServer pulumi.StringPtrInput // defaults to primaryNameServer of zone
	Refresh    pulumi.IntPtrInput    // defaults to 7200
	Retry      pulumi.IntPtrInput    // defaults to 900
	Rname      pulumi.StringPtrInput // defaults to "awsdns-hostmaster.amazon.com."
	Serial     pulumi.IntInput       // RFC1912 "YYYYMMDDNN" recommended; required
	Ttl        pulumi.IntPtrInput    // defaults to 3600
	// Zone       pulumi.StringPtrInput
}

type IZone interface {
	ZoneId() pulumi.StringOutput
	PrimaryNameServer() pulumi.StringOutput
}

func createSoaRecord(
	ctx *pulumi.Context, name string, zone IZone, args SoaRecordArgs, opts ...pulumi.ResourceOption,
) (*route53.Record, error) {
	if args.Expire == nil {
		args.Expire = pulumi.Int(1209600)
	}
	if args.Minimum == nil {
		args.Minimum = pulumi.Int(86400)
	}
	if args.NameServer == nil {
		args.NameServer = zone.PrimaryNameServer()
	}
	if args.Refresh == nil {
		args.Refresh = pulumi.Int(7200)
	}
	if args.Retry == nil {
		args.Retry = pulumi.Int(900)
	}
	if args.Rname == nil {
		args.Rname = pulumi.String("awsdns-hostmaster.amazon.com.")
	}
	if args.Ttl == nil {
		args.Ttl = pulumi.Int(3600)
	}
	record := pulumi.Sprintf("%s %s %d %d %d %d %d",
		args.NameServer, args.Rname, args.Serial, args.Refresh, args.Retry, args.Expire, args.Minimum)
	return CreateRecord(ctx, name, RecordTypeSOA, &route53.RecordArgs{
		AllowOverwrite: pulumi.Bool(true),
		Records:        pulumi.StringArray{record},
		Ttl:            args.Ttl,
		ZoneId:         zone.ZoneId(),
	}, append([]pulumi.ResourceOption{pulumi.RetainOnDelete(true)}, opts...)...)
}

func createCaaDnsRecord(
	ctx *pulumi.Context,
	hostname string,
	zone IZone,
	issuer []string,
	opts ...pulumi.ResourceOption,
) (*route53.Record, error) {
	// When we're asked to create a CAA record for a wildcard domain, we can
	// assume the intent is to issue a wildcard certificate.
	issueType := "issue"
	if strings.HasPrefix(hostname, "*") {
		issueType = "issuewild"
	}
	if len(issuer) == 0 {
		issuer = []string{";"} // ";" means "no CA is authorized to issue certs for this domain"
	}
	records := make(pulumi.StringArray, len(issuer))
	for i, value := range issuer {
		records[i] = pulumi.Sprintf("0 %s %q", issueType, value) // TODO: support iodef, etc.
	}
	return CreateRecord(ctx, hostname, RecordTypeCAA, &route53.RecordArgs{
		Records: records,
		Ttl:     pulumi.Int(3600), // 1 HOUR
		ZoneId:  zone.ZoneId(),
	},
		opts...,
	)
}

// https://www.rfc-editor.org/rfc/rfc6762#appendix-G and https://www.rfc-editor.org/rfc/rfc8375
var PRIVATE_ZONE_REGEX = regexp.MustCompile(`\b(intranet|internal|private|corp|home|home\.arpa|lan|local)\.?$`)

func isPrivateZone(domain string) bool {
	return PRIVATE_ZONE_REGEX.MatchString(domain)
}

func getHostedZone(ctx *pulumi.Context, domain string, opts ...pulumi.InvokeOption) IZone {
	zone := route53.LookupZoneOutput(ctx, route53.LookupZoneOutputArgs{
		Name:        pulumi.String(domain),
		PrivateZone: pulumi.Bool(isPrivateZone(domain)),
	}, opts...)
	return zone
}

func getOrCreatePublicZone(
	ctx *pulumi.Context,
	domain string,
	delegationSetId string,
	opts ...pulumi.ResourceOption,
) (IZone, error) {
	if delegationSetId == "" {
		return getHostedZone(ctx, domain), nil
	}

	zone, err := route53.NewZone(ctx,
		domain,
		&route53.ZoneArgs{
			Comment:         pulumi.String(common.DefangComment),
			DelegationSetId: pulumi.String(delegationSetId),
			ForceDestroy:    pulumi.Bool(ForceDestroyHostedzone.Get(ctx)),
			Name:            pulumi.String(domain),
		},
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating Route53 public zone: %w", err)
	}

	zoneOutput := zone.ToZoneOutput()
	_, err = createSoaRecord(
		ctx,
		domain,
		zoneOutput,
		SoaRecordArgs{
			Serial:  pulumi.Int(2025092601),
			Minimum: pulumi.Int(15),
		},
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating SOA record for public zone: %w", err)
	}

	return zoneOutput, nil
}
