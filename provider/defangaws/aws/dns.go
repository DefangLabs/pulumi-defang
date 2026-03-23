package aws

import (
	"strings"

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
	ctx *pulumi.Context, name string, typ RecordType, args route53.RecordArgs, opts ...pulumi.ResourceOption,
) (*route53.Record, error) {
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
	return route53.NewRecord(ctx, normalized+"_"+string(typ), &args, opts...)
}

type SoaRecordArgs struct {
	Expire     pulumi.IntInput    // defaults to 1209600
	Minimum    pulumi.IntInput    // TTL for negative responses; defaults to 86400
	NameServer pulumi.StringInput // defaults to primaryNameServer of zone
	Refresh    pulumi.IntInput    // defaults to 7200
	Retry      pulumi.IntInput    // defaults to 900
	Rname      pulumi.StringInput // defaults to "awsdns-hostmaster.amazon.com."
	Serial     pulumi.IntInput    // RFC1912 "YYYYMMDDNN" recommended
	Ttl        pulumi.IntInput    // defaults to 3600
	// Zone       pulumi.StringInput
}

type IZone interface {
	ZoneId() pulumi.StringOutput
	PrimaryNameServer() pulumi.StringOutput
}

func CreateSoaRecord(
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
	return CreateRecord(ctx, name, RecordTypeSOA, route53.RecordArgs{
		AllowOverwrite: pulumi.Bool(true),
		Records:        pulumi.StringArray{record},
		Ttl:            args.Ttl,
		ZoneId:         zone.ZoneId(),
	}, append([]pulumi.ResourceOption{pulumi.RetainOnDelete(true)}, opts...)...)
}
