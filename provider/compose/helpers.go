package compose

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/template"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// IsIngress returns true if the port mode is "ingress", defaulting to true.
func (p ServicePortConfig) IsIngress() bool {
	return p.Mode == PortModeIngress || p.Mode == "" // default to ingress
}

// IsHost returns true if the port mode is "host", defaulting to false.
func (p ServicePortConfig) IsHost() bool {
	return p.Mode == PortModeHost
}

// GetProtocol returns the protocol, defaulting to "tcp".
func (p ServicePortConfig) GetProtocol() PortProtocol {
	if p.Protocol != "" {
		return p.Protocol
	}
	return PortProtocolTCP
}

// GetAppProtocol returns the application protocol, defaulting to "http".
func (p ServicePortConfig) GetAppProtocol() PortAppProtocol {
	if p.AppProtocol != "" {
		return p.AppProtocol
	}
	return PortAppProtocolHTTP
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

// GetConfigName returns the config variable name if value is a bare secret reference
// (exactly "${VARNAME}" or "$VARNAME" with no surrounding text or modifiers),
// signaling that it should be passed as a native secret reference (e.g. SSM
// parameter ARN) rather than interpolated.
// Returns "" for literals, multi-var interpolations, mixed text, or
// variables with default/presence modifiers like ${VAR:-default}.
func GetConfigName(value string) string {
	match := template.DefaultPattern.FindStringSubmatchIndex(value)
	// Must match the entire string, with no extra text before/after
	if match != nil && match[0] == 0 && match[1] == len(value) {
		// Check for unbraced $VAR (group 2)
		if match[4] >= 0 {
			return value[match[4]:match[5]]
		}
		// Check for braced ${VAR} (group 3), with no modifiers (group 4)
		if match[6] >= 0 && match[8] < 0 {
			return value[match[6]:match[7]]
		}
	}
	return ""
}

func GetConfigName2(key string, value *string) string {
	if value == nil {
		return key
	}
	return GetConfigName(*value)
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
		var value string
		if v == nil {
			value = "${" + key + "}"
		} else {
			value = *v
		}
		// Resolve any $VAR / ${VAR} interpolations; empty string passes through as-is.
		return InterpolateEnvironmentVariable(ctx, configProvider, value, opts...)
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
	if configProvider == nil {
		// No config provider, so we can't resolve any variables. Return the raw string.
		return pulumi.String(value).ToStringOutput()
	}
	// First pass: discover variables and set up config resolution.
	// A second pass inside ApplyT is unavoidable because Pulumi outputs are async.
	var names []string
	var outputs []interface{}
	seen := make(map[string]bool)
	escaped, err := template.SubstituteWithOptions(value, func(key string) (string, bool) {
		if !seen[key] {
			seen[key] = true
			names = append(names, key)
			outputs = append(outputs, configProvider.GetConfigValue(ctx, key, opts...))
		}
		return "", true
	}, template.WithoutLogging)

	if len(names) == 0 {
		if err != nil {
			return pulumi.String(value).ToStringOutput()
		}
		return pulumi.String(escaped).ToStringOutput()
	}

	// Wait for all resolutions, then let compose-go do the full substitution
	return pulumi.All(outputs...).ApplyT(func(resolved []interface{}) (string, error) {
		mapping := make(map[string]string, len(names))
		for i, name := range names {
			mapping[name] = resolved[i].(string)
		}
		return template.SubstituteWithOptions(value, func(key string) (string, bool) {
			v, ok := mapping[key]
			return v, ok
		}, template.WithoutLogging)
	}).(pulumi.StringOutput)
}
