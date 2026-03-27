package common

import (
	"regexp"
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

// https://www.rfc-editor.org/rfc/rfc6762#appendix-G and https://www.rfc-editor.org/rfc/rfc8375
var privateZoneRegex = regexp.MustCompile(`(?i)\b(intranet|internal|private|corp|home|home\.arpa|lan|local)\.?$`)

func IsPrivateZone(domain string) bool {
	return privateZoneRegex.MatchString(domain)
}

func GetZoneName(hostname string) string {
	dot := strings.Index(hostname, ".")
	nextDot := strings.Index(hostname[dot+1:], ".")
	if nextDot == -1 || dot+1+nextDot == len(hostname)-1 {
		return hostname
	}
	return hostname[dot+1:]
}
