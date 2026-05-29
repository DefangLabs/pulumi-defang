package main

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

func TestColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := color(); got != "never" {
		t.Errorf("color() = %q, want %q", got, "never")
	}

	t.Setenv("NO_COLOR", "")
	if got := color(); got != "never" {
		t.Errorf("color() = %q, want %q", got, "never")
	}

	t.Setenv("NO_COLOR", "0") // value doesn't matter, just presence of the variable
	if got := color(); got != "never" {
		t.Errorf("color() = %q, want %q", got, "never")
	}

	os.Unsetenv("NO_COLOR") // reset by t.Setenv above
	if got := color(); got != "always" {
		t.Errorf("color() = %q, want %q", got, "always")
	}
}

func TestProjectConfig(t *testing.T) {
	cfg := projectConfig("TestPrefix")

	autonaming, ok := cfg["pulumi:autonaming"]
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

	disabled, ok := cfg["pulumi:disable-default-providers"]
	if !ok {
		t.Fatal("missing pulumi:disable-default-providers config")
	}
	disabledList, ok := disabled.Value.([]string)
	if !ok {
		t.Fatal("disable-default-providers value is not []string")
	}
	if len(disabledList) < 3 {
		t.Errorf("expected >= 3 disabled provider, got %d", len(disabledList))
	}
}

func TestProjectConfigEmptyPrefix(t *testing.T) {
	cfg := projectConfig("")

	autonaming := cfg["pulumi:autonaming"]
	autonamingMap := autonaming.Value.(map[string]any)
	pattern := autonamingMap["pattern"].(string)
	// With empty prefix, pattern should not have a leading prefix-
	if pattern != "${project}-${stack}-${name}-${hex(7)}" {
		t.Errorf("unexpected pattern with empty prefix: %s", pattern)
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

	cfg, err := stackConfigFromEnv()
	if err != nil {
		t.Fatalf("stackConfig() error: %v", err)
	}

	if cfg["defang:provider"].Value != "aws" {
		t.Errorf("expected provider aws, got %q", cfg["defang:provider"].Value)
	}
	if cfg["defang:stateUrl"].Value != "http://example.com/state" {
		t.Errorf("expected stateUrl http://example.com/state, got %q", cfg["defang:stateUrl"].Value)
	}
	if cfg["aws:region"].Value != "us-east-1" {
		t.Errorf("expected region us-east-1, got %q", cfg["aws:region"].Value)
	}
	if cfg["aws:profile"].Value != "myprofile" {
		t.Errorf("expected profile myprofile, got %q", cfg["aws:profile"].Value)
	}
	if cfg["defang:org"].Value != "testorg" {
		t.Errorf("expected org testorg, got %q", cfg["defang:org"].Value)
	}
	if cfg["defang:domain"].Value != "example.com" {
		t.Errorf("expected domain example.com, got %q", cfg["defang:domain"].Value)
	}
	if cfg["defang:privateDomain"].Value != "internal.example.com" {
		t.Errorf("expected privateDomain, got %q", cfg["defang:privateDomain"].Value)
	}
	if cfg["defang:delegationSetId"].Value != "DELEGSET123" {
		t.Errorf("expected delegationSetId, got %q", cfg["defang:delegationSetId"].Value)
	}
	const wantArn = "arn:aws:secretsmanager:us-east-1:123456789:secret:creds"
	if cfg["defang:ciRegistryCredentialsArn"].Value != wantArn {
		t.Errorf("expected ciRegistryCredentialsArn, got %q", cfg["defang:ciRegistryCredentialsArn"].Value)
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

	cfg, err := stackConfigFromEnv()
	if err != nil {
		t.Fatalf("stackConfig() error: %v", err)
	}

	if cfg["defang:provider"].Value != "gcp" {
		t.Errorf("expected provider gcp, got %q", cfg["defang:provider"].Value)
	}
	if cfg["gcp:project"].Value != "my-gcp-project" {
		t.Errorf("expected gcp:project, got %q", cfg["gcp:project"].Value)
	}
	if cfg["gcp:region"].Value != "us-central1" {
		t.Errorf("expected gcp:region, got %q", cfg["gcp:region"].Value)
	}
	// Optional fields should not be present when empty
	if _, ok := cfg["defang:domain"]; ok {
		t.Error("defang:domain should not be set when empty")
	}
	if _, ok := cfg["defang:privateDomain"]; ok {
		t.Error("defang:privateDomain should not be set when empty")
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

	cfg, err := stackConfigFromEnv()
	if err != nil {
		t.Fatalf("stackConfig() error: %v", err)
	}

	if cfg["defang:provider"].Value != "azure" {
		t.Errorf("expected provider azure, got %q", cfg["defang:provider"].Value)
	}
	if cfg["azure-native:location"].Value != "westus2" {
		t.Errorf("expected location westus2, got %q", cfg["azure-native:location"].Value)
	}
	if cfg["azure-native:useMsi"].Value != "true" {
		t.Errorf("expected useMsi true, got %q", cfg["azure-native:useMsi"].Value)
	}
	if cfg["azure-native:subscriptionId"].Value != "sub-123" {
		t.Errorf("expected subscriptionId, got %q", cfg["azure-native:subscriptionId"].Value)
	}
}

func TestCollectEvents(t *testing.T) {
	var engineEvents map[string][]events.EngineEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "invalid gzip", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		if err := json.NewDecoder(gz).Decode(&engineEvents); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ch, wait := collectEvents(t.Context(), srv.URL, false)
	ch <- events.EngineEvent{EngineEvent: apitype.EngineEvent{Sequence: 1}}
	ch <- events.EngineEvent{EngineEvent: apitype.EngineEvent{Sequence: 2}}
	close(ch)
	wait()

	if len(engineEvents["events"]) != 2 {
		t.Errorf("expected 2 events, got %d", len(engineEvents["events"]))
	}
}

func unsetenv(t *testing.T, keys ...string) {
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			t.Setenv(key, "") // sets up restoration and checks for parallel test interference
			os.Unsetenv(key)
		}
	}
}

func Test_stackConfigFromRecipe(t *testing.T) {
	expected := auto.ConfigMap{
		"defang:string":  auto.ConfigValue{Value: "foo"},
		"defang:string2": auto.ConfigValue{Value: `{"value":"bar"}`},
		"defang:null":    auto.ConfigValue{Value: ""},
		"defang:number":  auto.ConfigValue{Value: "42"},
		"defang:bool":    auto.ConfigValue{Value: "true"},
		"defang:array":   auto.ConfigValue{Value: `["a","b","c"]`},
		"defang:object":  auto.ConfigValue{Value: `{"nestedString":"nested"}`},
		"defang:object2": auto.ConfigValue{Value: `{"nestedObject":{"nested":"nested"}}`},
	}

	tests := []struct {
		name         string
		pulumiConfig string
	}{
		{
			name: "from Pulumi stack config YAML",
			pulumiConfig: `# Comments are ignored
config:
  defang:array: ["a", "b", "c"]
  defang:bool: true
  defang:null: null
  defang:number: 42
  defang:object:
    nestedString: "nested"
  defang:object2:
    nestedObject:
      nested: "nested"
  defang:string: "foo"
  defang:string2:
    value: "bar"`,
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
		},
		{
			name: "from Pulumi stack config JSON with value",
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configMap, err := stackConfigFromRecipe(tt.pulumiConfig)
			if err != nil {
				t.Fatalf("stackConfigFromRecipe() error: %v", err)
			}

			if !reflect.DeepEqual(expected, configMap) {
				t.Errorf("expected config %+v, got %+v", expected, configMap)
			}
		})
	}
}
