package gcp

import (
	"regexp"
	"strings"
)

// safeLabelValueRE matches every rune that is not allowed in a GCP label
// value: lowercase letters (\p{Ll}), caseless letters (\p{Lo}), digits,
// underscores, and dashes.
var safeLabelValueRE = regexp.MustCompile(`[^\p{Ll}\p{Lo}0-9_-]+`)

// SafeLabelValue normalizes a string to a valid GCP label value: lowercase,
// disallowed runes collapsed to "-", truncated to 63 characters. It MUST stay
// byte-for-byte identical to the Defang CLI's gcp.SafeLabelValue
// (src/pkg/clouds/gcp/label.go) — the CLI filters Cloud Logging queries on
// exact defang-* label values, so both sides have to normalize the same way.
func SafeLabelValue(input string) string {
	input = strings.ToLower(input)
	safe := safeLabelValueRE.ReplaceAllString(input, "-")
	if len(safe) > 63 {
		safe = safe[:63]
	}
	return safe
}
