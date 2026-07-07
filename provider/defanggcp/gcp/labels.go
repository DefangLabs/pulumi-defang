package gcp

import "strings"

// SanitizeLabel rewrites a Compose label key/value into a form GCP accepts.
//
// GCP label keys and values must contain only lowercase letters, digits,
// underscores and hyphens (≤63 chars); keys must additionally start with a
// lowercase letter and be non-empty. Compose labels routinely use reverse-DNS
// keys with dots and mixed-case values (e.g. "com.acme.team"="Core"), which GCP
// rejects. We sanitize rather than pass through verbatim (unlike AWS) because an
// invalid label would otherwise fail the entire deploy.
//
// Sanitization is lossy and can collide (e.g. "a.b" and "a-b" both → "a_b"); the
// last value wins on collision. This is documented in docs/labels-as-tags.md.
func SanitizeLabel(k, v string) (string, string) {
	return sanitizeLabelKey(k), sanitizeLabelChars(v)
}

// sanitizeLabelChars lowercases s and replaces every character outside
// [a-z0-9_-] with an underscore, truncating to GCP's 63-char limit.
func sanitizeLabelChars(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return truncate63(b.String())
}

// sanitizeLabelKey is sanitizeLabelChars plus GCP's key rules: a key must be
// non-empty and start with a lowercase letter (digits/_/- are not allowed as
// the first character).
func sanitizeLabelKey(s string) string {
	out := sanitizeLabelChars(s)
	if out == "" {
		return "label"
	}
	if c := out[0]; c < 'a' || c > 'z' {
		out = truncate63("k_" + out)
	}
	return out
}

func truncate63(s string) string {
	if len(s) > 63 {
		return s[:63]
	}
	return s
}
