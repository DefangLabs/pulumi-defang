package compose

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// IsIngress returns true if the port mode is "ingress", defaulting to true.
func (p ServicePortConfig) IsIngress() bool {
	return p.Mode == "ingress" || p.Mode == "" // default to ingress
}

// GetProtocol returns the protocol, defaulting to "tcp".
func (p ServicePortConfig) GetProtocol() string {
	if p.Protocol != "" {
		return p.Protocol
	}
	return "tcp"
}

// GetAppProtocol returns the application protocol, defaulting to "http".
func (p ServicePortConfig) GetAppProtocol() string {
	if p.AppProtocol != "" {
		return p.AppProtocol
	}
	return "http"
}

// ParseMemoryMiB parses a memory string into MiB.
// Accepts raw bytes (compose-go normalized), or suffixes: b, k, m, g, t, kb, mb, gb, tb, ki, mi, gi, ti.
func ParseMemoryMiB(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 512
	}

	// Try raw number (bytes, as compose-go normalizes)
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		mib := int(n / (1024 * 1024))
		if mib <= 0 {
			return 512
		}
		return mib
	}

	// Find where the numeric part ends
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 512
	}

	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil || n <= 0 {
		return 512
	}

	suffix := strings.ToLower(strings.TrimSpace(s[i:]))
	switch suffix {
	case "b":
		return max(int(n/(1024*1024)), 1)
	case "k", "kb":
		return max(int(n/1024), 1)
	case "ki", "kib":
		return max(int(n/1024), 1)
	case "m", "mb":
		return int(n)
	case "mi", "mib":
		return int(n)
	case "g", "gb":
		return int(n * 1024)
	case "gi", "gib":
		return int(n * 1024)
	case "t", "tb":
		return int(n * 1024 * 1024)
	case "ti", "tib":
		return int(n * 1024 * 1024)
	default:
		return 512
	}
}

// postgresVersionRe extracts version from image tags like "16", "16.3-bookworm", "pg16", "0.8.0-pg17".
var postgresVersionRe = regexp.MustCompile(`^(?:[\d.-]*pg)?([\d.]+)`)

// GetPostgresVersion extracts the postgres major version from an image tag.
// Returns 0 if the tag can't be parsed (caller should default to latest).
func GetPostgresVersion(tag string) int {
	m := postgresVersionRe.FindStringSubmatch(tag)
	if m == nil {
		return 0
	}
	// Take just the major version (first component before any dot)
	ver := m[1]
	if dot := strings.IndexByte(ver, '.'); dot >= 0 {
		ver = ver[:dot]
	}
	n, err := strconv.Atoi(ver)
	if err != nil {
		return 0
	}
	return n
}

// ParseImageTag splits "repo:tag" and returns the tag portion (empty string if no tag).
func ParseImageTag(image string) string {
	// Handle digest references like "repo@sha256:..."
	if at := strings.IndexByte(image, '@'); at >= 0 {
		return ""
	}
	if colon := strings.LastIndexByte(image, ':'); colon >= 0 {
		// Make sure we're not splitting on a port in the registry host
		afterColon := image[colon+1:]
		if !strings.Contains(afterColon, "/") {
			return afterColon
		}
	}
	return ""
}

type Match struct {
	Literal  string
	Variable string
	IsVar    bool
}

func Literal(s string) Match  { return Match{Literal: s} }
func Variable(s string) Match { return Match{Variable: s, IsVar: true} }

// IsSecretReference returns true when value is exactly "${KEY}" where KEY matches
// the env var name. This signals that the variable should be passed as a native
// secret reference (e.g. SSM parameter ARN) instead of being interpolated to plaintext.
func IsSecretReference(key, value string) bool {
	return value == "${"+key+"}"
}

var interpolationRegex = regexp.MustCompile(`(?i)\$\{([_a-z]\w*)\}`)

func ParseInterpolatedString(s string) []Match {
	matches := interpolationRegex.FindAllStringSubmatchIndex(s, -1)
	result := make([]Match, 0, len(matches))
	lastIndex := 0

	for _, match := range matches {
		fullStart, fullEnd := match[0], match[1]
		varStart, varEnd := match[2], match[3]

		// Skip escaped: preceding char is '$'
		if fullStart > 0 && s[fullStart-1] == '$' {
			continue
		}

		if lastIndex < fullStart {
			result = append(result, Literal(s[lastIndex:fullStart]))
		}
		result = append(result, Variable(s[varStart:varEnd]))
		lastIndex = fullEnd
	}

	if lastIndex < len(s) {
		result = append(result, Literal(s[lastIndex:]))
	}
	return result
}

func GetConfigOrEnvValue(
	ctx *pulumi.Context,
	configProvider ConfigProvider,
	s ServiceConfig,
	key string,
	defaultValue string,
	opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	if v, ok := s.Environment[key]; ok {
		if v != "" {
			// Resolve any ${VAR} interpolations in the env value via the config provider.
			return InterpolateEnvironmentVariable(ctx, configProvider, v, opts...)
		}
		// Empty value in compose means "read from config store" (set via `defang config set`);
		// the config provider is expected to fail if the config is not available.
		return configProvider.GetConfigValue(ctx, key, opts...)
	}
	// Key not in environment at all: use the provided default.
	return pulumi.String(defaultValue).ToStringOutput()
}

// ToPulumiStringArray converts a plain []string to a pulumi.StringArray.
func ToPulumiStringArray(ss []string) pulumi.StringArray {
	if len(ss) == 0 {
		return nil
	}
	arr := make(pulumi.StringArray, len(ss))
	for i, s := range ss {
		arr[i] = pulumi.String(s)
	}
	return arr
}

func InterpolateEnvironmentVariable(
	ctx *pulumi.Context,
	configProvider ConfigProvider,
	value string,
	opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	parsed := ParseInterpolatedString(value)

	if len(parsed) == 0 {
		return pulumi.String("").ToStringOutput()
	}

	parts := make([]pulumi.StringOutput, len(parsed))
	for i, match := range parsed {
		if !match.IsVar {
			parts[i] = pulumi.String(match.Literal).ToStringOutput()
		} else {
			parts[i] = configProvider.GetConfigValue(ctx, match.Variable, opts...)
		}
	}

	// Fold over parts, joining with ApplyT since pulumi.Concat doesn't exist in Go
	result := parts[0]
	for _, part := range parts[1:] {
		p := part // capture loop var
		result = result.ApplyT(func(acc string) pulumi.StringOutput {
			return p.ApplyT(func(s string) string {
				return acc + s
			}).(pulumi.StringOutput)
		}).(pulumi.StringOutput)
	}
	return result
}
