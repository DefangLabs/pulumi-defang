package common

import (
	"strings"
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

func SafeLabel(name string) string {
	// Technically DNS names can have underscores, but these are reserved for SRV
	// records and some systems have issues with them.
	return strings.ReplaceAll(strings.ToLower(name), ".", "-")
}
