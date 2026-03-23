package common

import (
	"strings"
)

func NormalizeDNS(name string) string {
	return strings.ToLower(strings.TrimRight(name, "."))
}

func SafeLabel(name string) string {
	// Technically DNS names can have underscores, but these are reserved for SRV
	// records and some systems have issues with them.
	return strings.ReplaceAll(strings.ToLower(name), ".", "-")
}
