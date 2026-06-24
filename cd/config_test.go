package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/stretchr/testify/require"
)

func Test_setDefaultStackConfig(t *testing.T) {
	config := configMap{}
	setDefaultStackConfig("TestPrefix", config)

	autonaming, ok := config["pulumi:autonaming"]
	if !ok {
		t.Fatal("missing pulumi:autonaming config")
	}
	autonamingMap, ok := autonaming.Value.(map[string]any)
	if !ok {
		t.Fatal("autonaming value is not map[string]any")
	}
	pattern, ok := autonamingMap["pattern"].(string)
	if !ok {
		t.Fatal("pattern is not a string")
	}
	if pattern != "TestPrefix-${project}-${stack}-${name}-${hex(7)}" {
		t.Errorf("unexpected pattern: %s", pattern)
	}

	// defang:prefix must be the bare prefix (no trailing "-"); consumers append
	// their own separator. A trailing hyphen here doubles it (e.g. "TestPrefix--proj").
	if got := config["defang:prefix"].Value; got != "TestPrefix" {
		t.Errorf("unexpected defang:prefix: %q, want %q", got, "TestPrefix")
	}
}

func Test_setDefaultStackConfigEmptyPrefix(t *testing.T) {
	config := configMap{}
	setDefaultStackConfig("", config)

	autonaming := config["pulumi:autonaming"]
	autonamingMap := autonaming.Value.(map[string]any)
	pattern := autonamingMap["pattern"].(string)
	// With empty prefix, pattern should not have a leading prefix-
	if pattern != "${project}-${stack}-${name}-${hex(7)}" {
		t.Errorf("unexpected pattern with empty prefix: %s", pattern)
	}
	if got := config["defang:prefix"].Value; got != "" {
		t.Errorf("unexpected defang:prefix with empty prefix: %q", got)
	}
}

func TestStackConfigFromEnvAWS(t *testing.T) {
	// Clear ambient env that could pick a second provider or override fallbacks.
	unsetenv(t, "REGION", "GCP_PROJECT", "GCLOUD_PROJECT", "AZURE_SUBSCRIPTION_ID", "PULUMI_BACKEND_URL")

	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_PROFILE", "myprofile")
	t.Setenv("DEFANG_ORG", "testorg")
	t.Setenv("DOMAIN", "example.com")
	t.Setenv("PRIVATE_DOMAIN", "internal.example.com")
	t.Setenv("DELEGATION_SET_ID", "DELEGSET123")
	t.Setenv("CI_REGISTRY_CREDENTIALS_ARN", "arn:aws:secretsmanager:us-east-1:123456789:secret:creds")
	t.Setenv("DEFANG_STATE_URL", "http://example.com/state")

	config := configMap{}
	setDefaultStackConfig("", config)
	err := addStackConfigFromEnv(config)
	if err != nil {
		t.Fatalf("stackConfig() error: %v", err)
	}

	if config["defang:provider"].Value != "aws" {
		t.Errorf("expected provider aws, got %q", config["defang:provider"].Value)
	}
	if config["defang:stateUrl"].Value != "http://example.com/state" {
		t.Errorf("expected stateUrl http://example.com/state, got %q", config["defang:stateUrl"].Value)
	}
	if config["aws:region"].Value != "us-east-1" {
		t.Errorf("expected region us-east-1, got %q", config["aws:region"].Value)
	}
	if config["aws:profile"].Value != "myprofile" {
		t.Errorf("expected profile myprofile, got %q", config["aws:profile"].Value)
	}
	if config["defang:org"].Value != "testorg" {
		t.Errorf("expected org testorg, got %q", config["defang:org"].Value)
	}
	if config["defang:domain"].Value != "example.com" {
		t.Errorf("expected domain example.com, got %q", config["defang:domain"].Value)
	}
	if config["defang-aws:privateDomain"].Value != "internal.example.com" {
		t.Errorf("expected privateDomain, got %q", config["defang-aws:privateDomain"].Value)
	}
	if config["defang-aws:delegationSetId"].Value != "DELEGSET123" {
		t.Errorf("expected delegationSetId, got %q", config["defang-aws:delegationSetId"].Value)
	}
	const wantArn = "arn:aws:secretsmanager:us-east-1:123456789:secret:creds"
	if config["defang-aws:ciRegistryCredentialsArn"].Value != wantArn {
		t.Errorf("expected ciRegistryCredentialsArn, got %q", config["defang-aws:ciRegistryCredentialsArn"].Value)
	}
}

func TestStackConfigFromEnvGCP(t *testing.T) {
	// Clear ambient env that could pick a second provider or override fallbacks.
	unsetenv(t, "REGION", "AWS_REGION", "AZURE_SUBSCRIPTION_ID", "GCLOUD_PROJECT")

	t.Setenv("GCLOUD_PROJECT", "my-gcp-project")
	t.Setenv("GCLOUD_REGION", "us-central1")
	t.Setenv("DEFANG_ORG", "testorg")
	t.Setenv("DOMAIN", "")
	t.Setenv("PRIVATE_DOMAIN", "")
	t.Setenv("DELEGATION_SET_ID", "")
	t.Setenv("CI_REGISTRY_CREDENTIALS_ARN", "")

	config := configMap{}
	setDefaultStackConfig("", config)
	err := addStackConfigFromEnv(config)
	if err != nil {
		t.Fatalf("stackConfig() error: %v", err)
	}

	if config["defang:provider"].Value != "gcp" {
		t.Errorf("expected provider gcp, got %q", config["defang:provider"].Value)
	}
	if config["gcp:project"].Value != "my-gcp-project" {
		t.Errorf("expected gcp:project, got %q", config["gcp:project"].Value)
	}
	if config["gcp:region"].Value != "us-central1" {
		t.Errorf("expected gcp:region, got %q", config["gcp:region"].Value)
	}
	// Optional fields should not be present when empty
	if _, ok := config["defang:domain"]; ok {
		t.Error("defang:domain should not be set when empty")
	}
	if _, ok := config["defang-aws:privateDomain"]; ok {
		t.Error("defang-aws:privateDomain should not be set when empty")
	}
}

func TestStackConfigFromEnvAzure(t *testing.T) {
	// Clear ambient env that could pick a second provider or override fallbacks.
	unsetenv(t, "REGION", "AWS_REGION", "GCP_PROJECT", "GCLOUD_PROJECT")

	t.Setenv("AZURE_SUBSCRIPTION_ID", "sub-123")
	t.Setenv("AZURE_LOCATION", "westus2")
	t.Setenv("DEFANG_ORG", "testorg")
	t.Setenv("DOMAIN", "")
	t.Setenv("PRIVATE_DOMAIN", "")
	t.Setenv("DELEGATION_SET_ID", "")
	t.Setenv("CI_REGISTRY_CREDENTIALS_ARN", "")

	config := configMap{}
	setDefaultStackConfig("", config)
	err := addStackConfigFromEnv(config)
	if err != nil {
		t.Fatalf("stackConfig() error: %v", err)
	}

	if config["defang:provider"].Value != "azure" {
		t.Errorf("expected provider azure, got %q", config["defang:provider"].Value)
	}
	if config["azure-native:location"].Value != "westus2" {
		t.Errorf("expected location westus2, got %q", config["azure-native:location"].Value)
	}
	if config["azure-native:useMsi"].Value != "true" {
		t.Errorf("expected useMsi true, got %q", config["azure-native:useMsi"].Value)
	}
	if config["azure-native:subscriptionId"].Value != "sub-123" {
		t.Errorf("expected subscriptionId, got %q", config["azure-native:subscriptionId"].Value)
	}
}

func Test_parseRecipePulumiConfig(t *testing.T) {
	// Pulumi config from --json export always stringifies integers and booleans
	expectedJson := configMap{
		"defang:string":  configValue{Value: "foo"},
		"defang:string2": configValue{Value: map[string]any{"value": "bar"}},
		"defang:null":    configValue{Value: ""},
		"defang:number":  configValue{Value: "42"},
		"defang:bool":    configValue{Value: "true"},
		"defang:array":   configValue{Value: []any{"a", "b", "c"}},
		"defang:object":  configValue{Value: map[string]any{"nestedString": "nested"}},
		"defang:object2": configValue{Value: map[string]any{"nestedObject": map[string]any{"nested": "nested"}}},
	}

	tests := []struct {
		name         string
		pulumiConfig string
		expected     any
	}{
		{
			name: "from Pulumi stack config YAML",
			pulumiConfig: `# Comments are ignored
config:
  defang:array: ["a", "b", "c"]
  defang:bool: true
  defang:null:
  defang:number: 42
  defang:object:
    nestedString: "nested"
  defang:object2:
    nestedObject:
      nested: "nested"
  defang:string: "foo"
  defang:string2:
    value: "bar"`,
			expected: configMap{
				"defang:string":  configValue{Value: "foo"},
				"defang:string2": configValue{Value: map[string]any{"value": "bar"}},
				"defang:null":    configValue{},
				"defang:number":  configValue{Value: 42},
				"defang:bool":    configValue{Value: true},
				"defang:array":   configValue{Value: []any{"a", "b", "c"}},
				"defang:object":  configValue{Value: map[string]any{"nestedString": "nested"}},
				"defang:object2": configValue{Value: map[string]any{"nestedObject": map[string]any{"nested": "nested"}}},
			},
		},
		{
			name: "from Pulumi stack config JSON, objectValue overrides value",
			pulumiConfig: `{
    "defang:array":{"value":"overridden","objectValue":["a","b","c"],"secret":false},
    "defang:bool":{"value":"true","secret":false},
    "defang:null":{"value":"","secret":false},
    "defang:number":{"value":"42","secret":false},
    "defang:object":{"value":"overridden","objectValue":{"nestedString":"nested"},"secret":false},
    "defang:object2":{"value":"overridden","objectValue":{"nestedObject":{"nested":"nested"}},"secret":false},
    "defang:string":{"value":"foo","secret":false},
    "defang:string2":{"value":"overridden","objectValue":{"value":"bar"},"secret":false}
}`,
			expected: expectedJson,
		},
		{
			name: "from Pulumi stack config JSON with objectValue",
			pulumiConfig: `{
    "defang:array":{"objectValue":["a","b","c"],"secret":false},
    "defang:bool":{"value":"true","secret":false},
    "defang:null":{"value":"","secret":false},
    "defang:number":{"value":"42","secret":false},
    "defang:object":{"objectValue":{"nestedString":"nested"},"secret":false},
    "defang:object2":{"objectValue":{"nestedObject":{"nested":"nested"}},"secret":false},
    "defang:string":{"value":"foo","secret":false},
    "defang:string2":{"objectValue":{"value":"bar"},"secret":false}
}`,
			expected: expectedJson,
		},
		{
			name: "from Pulumi stack config JSON with only value",
			pulumiConfig: `{
    "defang:array":{"value":"[\"a\",\"b\",\"c\"]"},
    "defang:bool":{"value":"true","secret":false},
    "defang:null":{"value":"","secret":false},
    "defang:number":{"value":"42","secret":false},
    "defang:object":{"value":"{\"nestedString\":\"nested\"}"},
    "defang:object2":{"value":"{\"nestedObject\":{\"nested\":\"nested\"}}"},
    "defang:string":{"value":"foo","secret":false},
    "defang:string2":{"value":"{\"value\":\"bar\"}"}
}`,
			expected: configMap{
				"defang:string":  configValue{Value: "foo"},
				"defang:string2": configValue{Value: `{"value":"bar"}`},
				"defang:null":    configValue{Value: ""},
				"defang:number":  configValue{Value: "42"},
				"defang:bool":    configValue{Value: "true"},
				"defang:array":   configValue{Value: `["a","b","c"]`},
				"defang:object":  configValue{Value: `{"nestedString":"nested"}`},
				"defang:object2": configValue{Value: `{"nestedObject":{"nested":"nested"}}`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := configMap{}
			err := unmarshalRecipe(tt.pulumiConfig, config)
			if err != nil {
				t.Fatalf("unmarshalRecipe() error: %v", err)
			}

			require.Equal(t, tt.expected, config)
		})
	}
}

// TestStackConfigJsonAutonaming guards against the
// "getting autonaming config: ... cannot unmarshal string into Go value of type
// autonaming.autonamingSectionJSON" error. The autonaming engine runs
// ParseAutonamingConfig at the start of every preview/up and requires
// pulumi:autonaming to be stored as a structured object (so v.ToObject()
// returns a map). `pulumi config set-all --json` only stores an object when the
// JSON carries a non-nil "objectValue"; a scalar "value" string fails to parse.
// We deliberately let Pulumi's engine do the unmarshalling so a regression in
// configValue marshaling reproduces that exact error here.
func TestStackConfigJsonAutonaming(t *testing.T) {
	// Clear ambient env that could pick a second provider or override fallbacks.
	unsetenv(t, "REGION", "GCP_PROJECT", "GCLOUD_PROJECT", "AZURE_SUBSCRIPTION_ID", "PULUMI_BACKEND_URL")
	t.Setenv("AWS_REGION", "us-west-2")

	ctx := t.Context()
	workDir := t.TempDir()
	env := map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + workDir, // local backend, no cloud login
		"PULUMI_CONFIG_PASSPHRASE": "test-passphrase",   // for the local secrets provider
	}

	// No-op program: ParseAutonamingConfig runs before any resource, so we don't
	// need to create one to exercise the engine's autonaming parse path.
	program := func(pctx *pulumi.Context) error { return nil }

	ws, err := auto.NewLocalWorkspace(ctx,
		auto.WorkDir(workDir),
		auto.Program(program),
		auto.EnvVars(env),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(TEST_PROJECT),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		}),
	)
	require.NoError(t, err)

	stack, err := auto.UpsertStack(ctx, "dev", ws)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = stack.Destroy(ctx, optdestroy.Remove()) })

	// Marshal the recipe config exactly as cd does in production...
	recipe := "config:\n  pulumi:autonaming:\n    pattern: \"lio-${name}-${hex(7)}\""
	configJson, err := stackConfigJson(recipe)
	require.NoError(t, err)
	require.NoError(t, stack.SetAllConfigJson(ctx, configJson, nil))

	// ...then let Pulumi's engine parse it. Preview triggers ParseAutonamingConfig;
	// a scalar (string) autonaming value would fail here with the production error.
	_, err = stack.Preview(ctx)
	require.NoError(t, err)
}

func Test_configMap(t *testing.T) {
	config := configMap{
		"defang:array":   configValue{Value: []string{"a", "b", "c"}},
		"defang:bool":    configValue{Value: true},
		"defang:null":    configValue{},
		"defang:number":  configValue{Value: 42},
		"defang:object":  configValue{Value: map[string]string{"nestedString": "nested"}},
		"defang:object2": configValue{Value: map[string]any{"nestedObject": map[string]any{"nested": "nested"}}},
		"defang:string":  configValue{Value: "foo"},
		"defang:string2": configValue{Value: map[string]string{"value": "bar"}},
	}

	const expected = `{
  "defang:array": {
    "objectValue": ["a", "b", "c"],
    "value": "[\"a\",\"b\",\"c\"]"
  },
  "defang:bool": {
    "value": "true"
  },
  "defang:null": {
    "objectValue": null,
    "value": "null"
  },
  "defang:number": {
    "value": "42"
  },
  "defang:object": {
    "objectValue": {
      "nestedString": "nested"
    },
    "value": "{\"nestedString\":\"nested\"}"
  },
  "defang:object2": {
    "objectValue": {
      "nestedObject": {
        "nested": "nested"
      }
    },
    "value": "{\"nestedObject\":{\"nested\":\"nested\"}}"
  },
  "defang:string": {
    "value": "foo"
  },
  "defang:string2": {
    "objectValue": {
      "value": "bar"
    },
    "value": "{\"value\":\"bar\"}"
  }
}`

	configJson, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	require.JSONEq(t, expected, string(configJson))
}

func TestPulumiConfig(t *testing.T) {
	// This test verifies that Pulumi config values can be read from a stack's config and that stack-level config
	// overrides project-level config. This is important because recipes will typically set stack-level config based on
	// the recipe's PulumiConfig, which should override any defaults set at the project level (e.g. by a shared VPC component).
	ctx := t.Context()
	workDir := t.TempDir()
	env := map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + workDir, // local backend, no cloud login
		"PULUMI_CONFIG_PASSPHRASE": "test-passphrase",   // for the local secrets provider
	}

	// Inline program: read the (merged) pulumi:autonaming pattern and export it.
	program := func(pctx *pulumi.Context) error {
		var autonaming struct {
			Pattern string `json:"pattern"`
		}
		if err := config.New(pctx, "pulumi").TryObject("autonaming", &autonaming); err != nil {
			return fmt.Errorf("reading pulumi:autonaming: %w", err)
		}
		pctx.Export("pattern", pulumi.String(autonaming.Pattern))
		return nil
	}

	ws, err := auto.NewLocalWorkspace(ctx,
		auto.WorkDir(workDir),
		auto.Program(program),
		auto.EnvVars(env),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(TEST_PROJECT),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
			// Project-level autonaming: the default a recipe would inherit.
			Config: map[string]workspace.ProjectConfigType{
				"pulumi:autonaming": {Value: map[string]any{"pattern": "PROJECT-${name}"}},
			},
		}),
	)
	require.NoError(t, err)

	const stackName = "dev"
	stack, err := auto.UpsertStack(ctx, stackName, ws)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = stack.Destroy(ctx, optdestroy.Remove()) })

	// Stack-level autonaming: should override the project-level pattern.
	config := configMap{
		"pulumi:autonaming": configValue{
			Value: map[string]any{"pattern": "STACK-${name}"},
		},
	}
	configJson, err := json.Marshal(config)
	require.NoError(t, err)

	err = stack.SetAllConfigJson(ctx, string(configJson), nil)
	require.NoError(t, err)

	res, err := stack.Up(ctx)
	require.NoError(t, err)

	require.Equal(t, "STACK-${name}", res.Outputs["pattern"].Value,
		"stack-level pulumi:autonaming should override the project-level pattern")
}

func unsetenv(t *testing.T, keys ...string) {
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			t.Setenv(key, "") // sets up restoration and checks for parallel test interference
			os.Unsetenv(key)
		}
	}
}
