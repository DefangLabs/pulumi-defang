package common

import (
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
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
	// Technically DNS names can have underscores, so we won't touch them here.
	return strings.ReplaceAll(strings.ToLower(name), ".", "-")
}

// ServiceLabel converts a Compose service name into the DNS label used in
// generated FQDNs (e.g. "<service>.<domain>"). It mirrors the CLI's
// getServiceLabel: underscores are mapped to hyphens before SafeLabel because,
// while technically valid in DNS names, they are reserved for SRV records and
// break some systems. Use this (not SafeLabel) for anything derived from a
// service name so provider-generated hostnames match what the CLI computed.
func ServiceLabel(name string) string {
	return SafeLabel(strings.ReplaceAll(name, "_", "-"))
}

// ServiceFQDN returns the public-facing fully-qualified domain name for a
// service, following the CLI source-of-truth precedence
// (cd: domainname || publicFqdn || privateFqdn):
//   - the custom DomainName if set;
//   - else "<ServiceLabel>.<publicDomain>" when the service has ingress ports;
//   - else "<ServiceLabel>.<privateDomain>" when it has host (internal) ports.
//
// publicDomain is the project's delegate domain, privateDomain its internal
// domain; pass "" for either when the provider has no such concept (e.g. GCP and
// Azure today have no private-FQDN fallback). Returns "" when no FQDN applies,
// which callers use to decide whether to set DEFANG_FQDN.
func ServiceFQDN(serviceName string, svc compose.ServiceConfig, publicDomain, privateDomain string) string {
	if svc.DomainName != "" {
		return svc.DomainName
	}
	switch {
	case svc.HasIngressPorts() && publicDomain != "":
		return ServiceLabel(serviceName) + "." + publicDomain
	case svc.HasHostPorts() && privateDomain != "":
		return ServiceLabel(serviceName) + "." + privateDomain
	}
	return ""
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
