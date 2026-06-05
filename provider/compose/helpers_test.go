package compose

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfigProvider returns pre-seeded values; missing keys fail.
type mockConfigProvider struct {
	values map[string]string
}

func (m *mockConfigProvider) GetConfigValue(
	_ *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	if v, ok := m.values[key]; ok {
		return pulumi.String(v).ToStringOutput()
	}
	return ConfigNotFoundOutput(key)
}

func (m *mockConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return "mock-secret-ref-" + key, nil
}

// testMocks is a no-op Pulumi mock runtime required by pulumi.WithMocks.
type testMocks struct{}

func (testMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (testMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestToPulumiStringArray(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		assert.Nil(t, ToPulumiStringArray(nil))
	})
	t.Run("empty slice returns nil", func(t *testing.T) {
		assert.Nil(t, ToPulumiStringArray([]string{}))
	})
	t.Run("non-empty slice returns array of same length", func(t *testing.T) {
		result := ToPulumiStringArray([]string{"a", "b", "c"})
		assert.Len(t, result, 3)
	})
}

func TestGetConfigOrEnvValue(t *testing.T) {
	tests := []struct {
		name         string
		environment  map[string]*string
		key          string
		defaultValue string
		configs      map[string]string
		expected     string
	}{
		{
			name:         "nil environment uses default",
			environment:  nil,
			key:          "MY_KEY",
			defaultValue: "fallback",
			expected:     "fallback",
		},
		{
			name:         "key absent returns default",
			environment:  map[string]*string{},
			key:          "MY_KEY",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:        "empty string value is literal empty",
			environment: map[string]*string{"MY_KEY": ptr("")},
			key:         "MY_KEY",
			expected:    "",
		},
		{
			name:        "plain string value returned as-is",
			environment: map[string]*string{"MY_KEY": ptr("hello")},
			key:         "MY_KEY",
			expected:    "hello",
		},
		{
			name:        "interpolated value resolves variables from config provider",
			environment: map[string]*string{"MY_KEY": ptr("prefix_${SECRET}_suffix")},
			key:         "MY_KEY",
			configs:     map[string]string{"SECRET": "resolved"},
			expected:    "prefix_resolved_suffix",
		},
		{
			// Compose spec: "KEY:" (no value) → resolve from config at runtime.
			name:        "nil value resolves from config provider via ${KEY}",
			environment: map[string]*string{"MY_KEY": nil},
			key:         "MY_KEY",
			configs:     map[string]string{"MY_KEY": "from-config"},
			expected:    "from-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				svc := ServiceConfig{Environment: tt.environment}
				provider := &mockConfigProvider{values: tt.configs}
				out := GetConfigOrEnvValue(ctx, provider, svc, tt.key, tt.defaultValue)

				out.ApplyT(func(got string) string {
					assert.Equal(t, tt.expected, got)
					return got
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}
}

func TestGetConfigName(t *testing.T) {
	tests := []struct {
		value   string
		wantVar string
	}{
		{"${MY_KEY}", "MY_KEY"},  // bare braced reference
		{"$OTHER", "OTHER"},      // bare unbraced reference
		{"prefix_${MY_KEY}", ""}, // text before var
		{"${MY_KEY}_suffix", ""}, // text after var
		{"${A}_${B}", ""},        // multiple vars
		{"${VAR:-default}", ""},  // has default modifier
		{"${VAR:+alt}", ""},      // has presence modifier
		{"${VAR:?err}", ""},      // has required modifier; TODO: this could be made to work #159
		{"${VAR+}", ""},          // has empty modifier
		{"literal", ""},          // no interpolation
		{"", ""},                 // empty
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := GetConfigName(tt.value)
			assert.Equal(t, tt.wantVar, v)
		})
	}
}

func TestInterpolateEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		configs  map[string]string
		expected string
	}{
		{
			name:     "plain literal",
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "single variable",
			value:    "${MY_VAR}",
			configs:  map[string]string{"MY_VAR": "secret"},
			expected: "secret",
		},
		{
			name:     "variable with prefix and suffix",
			value:    "prefix_${MY_VAR}_suffix",
			configs:  map[string]string{"MY_VAR": "value"},
			expected: "prefix_value_suffix",
		},
		{
			name:     "multiple variables",
			value:    "${VAR1}_${VAR2}",
			configs:  map[string]string{"VAR1": "hello", "VAR2": "world"},
			expected: "hello_world",
		},
		{
			name:     "unbraced variable",
			value:    "$MY_VAR",
			configs:  map[string]string{"MY_VAR": "secret"},
			expected: "secret",
		},
		{
			name:     "escaped dollar produces literal dollar",
			value:    "$${NOT_A_VAR}",
			expected: "${NOT_A_VAR}",
		},
		{
			name:     "missing variable resolves from config",
			value:    "${SECRET}",
			configs:  map[string]string{"SECRET": "found"},
			expected: "found",
		},
		{
			name:     "empty string",
			value:    "",
			expected: "",
		},
		{
			name:     "variable adjacent to text on both sides without separators",
			value:    "arn:aws:iam::${ACCOUNT_ID}:role/my-role",
			configs:  map[string]string{"ACCOUNT_ID": "123456789"},
			expected: "arn:aws:iam::123456789:role/my-role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				provider := &mockConfigProvider{values: tt.configs}
				out := InterpolateEnvironmentVariable(ctx, provider, tt.value)

				out.ApplyT(func(got string) string {
					assert.Equal(t, tt.expected, got)
					return got
				})
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}

	t.Run("nil config provider returns raw string", func(t *testing.T) {
		err := pulumi.RunErr(func(ctx *pulumi.Context) error {
			out := InterpolateEnvironmentVariable(ctx, nil, "value with ${VAR}")

			out.ApplyT(func(got string) string {
				assert.Equal(t, "value with ${VAR}", got)
				return got
			})
			return nil
		}, pulumi.WithMocks("proj", "stack", testMocks{}))
		require.NoError(t, err)
	})
}

// healthCheckPathPortCases mirrors the TS suite at
// defang-mvp/pulumi/test/healthcheck.test.ts (#convertHealthcheck) so behavior
// parity with the legacy pipeline is explicit. Additional cases beyond the TS
// suite cover Bun-style health checks and other forms we've seen in the wild.
var healthCheckPathPortCases = []struct {
	name     string
	hc       *HealthCheckConfig
	wantPath string
	wantPort int
}{
	// --- defaults / no parse ---
	{
		name:     "nil healthcheck → defaults",
		hc:       nil,
		wantPath: "/",
		wantPort: 80,
	},
	{
		name:     "empty test slice → defaults",
		hc:       &HealthCheckConfig{Test: []string{}},
		wantPath: "/",
		wantPort: 80,
	},
	{
		name:     "test with only CMD and no args → defaults",
		hc:       &HealthCheckConfig{Test: []string{"CMD"}},
		wantPath: "/",
		wantPort: 80,
	},
	{
		name:     "NONE → defaults",
		hc:       &HealthCheckConfig{Test: []string{"NONE"}},
		wantPath: "/",
		wantPort: 80,
	},
	{
		name:     "non-CMD/CMD-SHELL test (gRPC binary) → defaults",
		hc:       &HealthCheckConfig{Test: []string{"grpc_health_probe", "-addr=:5000"}},
		wantPath: "/",
		wantPort: 80,
	},
	{
		name:     "CMD with non-URL test (test -f) → defaults",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "test", "-f", "/tmp/ready"}},
		wantPath: "/",
		wantPort: 80,
	},

	// --- bare localhost / 127.0.0.1 (no scheme) ---
	{
		name:     "curl bare localhost → defaults",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "localhost"}},
		wantPath: "/",
		wantPort: 80,
	},
	{
		name:     "curl localhost:8080/foo (no scheme)",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "localhost:8080/foo"}},
		wantPath: "/foo",
		wantPort: 8080,
	},
	{
		name:     "curl with -f flag before URL (no scheme)",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "-f", "localhost:8080/foo"}},
		wantPath: "/foo",
		wantPort: 8080,
	},

	// --- http:// scheme variants ---
	{
		name:     "CMD curl http://localhost:8080/healthz",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost:8080/healthz"}},
		wantPath: "/healthz",
		wantPort: 8080,
	},
	{
		name:     "wget pattern with trailing slash",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "wget", "-q", "--spider", "http://localhost:8080/"}},
		wantPath: "/",
		wantPort: 8080,
	},
	{
		name: "URL embedded in Python urllib code → regex stops at '",
		hc: &HealthCheckConfig{Test: []string{
			"CMD", "python", "-c",
			"import urllib; urllib.urlopen('http://localhost:8080/foo')",
		}},
		wantPath: "/foo",
		wantPort: 8080,
	},
	{
		name:     "CMD-SHELL with 127.0.0.1 (no port) and shell suffix",
		hc:       &HealthCheckConfig{Test: []string{"CMD-SHELL", "curl -f 127.0.0.1/healthz || exit 1"}},
		wantPath: "/healthz",
		wantPort: 80,
	},
	{
		name:     "CMD-SHELL with 127.0.0.1 (no port) — Bun-style wget",
		hc:       &HealthCheckConfig{Test: []string{"CMD-SHELL", "wget -q -O- http://localhost:3000/health || exit 1"}},
		wantPath: "/health",
		wantPort: 3000,
	},
	{
		name:     "URL with explicit 127.0.0.1:port/path",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "-fsS", "http://127.0.0.1:5000/api/healthz"}},
		wantPath: "/api/healthz",
		wantPort: 5000,
	},

	// --- URL without explicit port or path ---
	{
		name:     "URL without explicit port → port default kept",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost/healthz"}},
		wantPath: "/healthz",
		wantPort: 80,
	},
	{
		name:     "URL without path → path default kept",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost:9000"}},
		wantPath: "/",
		wantPort: 9000,
	},

	// --- query strings and fragments ---
	{
		name:     "URL with path + query string",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost:8000/?bar=baz"}},
		wantPath: "/?bar=baz",
		wantPort: 8000,
	},
	{
		name:     "URL with query but no leading slash",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost:8000?foo"}},
		wantPath: "?foo",
		wantPort: 8000,
	},
	{
		name:     "URL fragment stripped (not part of HTTP path)",
		hc:       &HealthCheckConfig{Test: []string{"CMD", "curl", "http://localhost:8000/foo/bar#ignore"}},
		wantPath: "/foo/bar",
		wantPort: 8000,
	},
}

func TestGetHealthCheckPathAndPort(t *testing.T) {
	for _, tt := range healthCheckPathPortCases {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotPort := GetHealthCheckPathAndPort(tt.hc)
			assert.Equal(t, tt.wantPath, gotPath)
			assert.Equal(t, tt.wantPort, gotPort)
		})
	}
}
