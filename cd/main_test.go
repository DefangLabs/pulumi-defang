package main

import (
	"testing"
)

func TestColor(t *testing.T) {
	saved := noColor
	t.Cleanup(func() { noColor = saved })

	noColor = false
	if got := color(); got != "always" {
		t.Errorf("color() = %q, want %q", got, "always")
	}

	noColor = true
	if got := color(); got != "never" {
		t.Errorf("color() = %q, want %q", got, "never")
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

func TestGetenv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		fallback string
		want     string
	}{
		{"env set", "TEST_GETENV_SET", "fromenv", "fallback", "fromenv"},
		{"env empty", "TEST_GETENV_EMPTY", "", "fallback", "fallback"},
		{"env unset", "TEST_GETENV_UNSET_XYZ", "", "fallback", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.key, tt.envVal)
			}
			if got := getenv(tt.key, tt.fallback); got != tt.want {
				t.Errorf("Getenv(%q, %q) = %q, want %q", tt.key, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestSplitByComma(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"one,,three", []string{"one", "", "three"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitByComma(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("SplitByComma(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("SplitByComma(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitByComma(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"false", "false", false},
		{"0", "0", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_GETENVBOOL_" + tt.name
			t.Setenv(key, tt.val)
			if got := getenvBool(key); got != tt.want {
				t.Errorf("GetenvBool(%q) with value %q = %v, want %v", key, tt.val, got, tt.want)
			}
		})
	}
}

func TestStackConfigAWS(t *testing.T) {
	// Save and restore all globals that stackConfig reads
	savedAwsRegion := awsRegion
	savedGcpProjectId := gcpProjectId
	savedAzureSubscriptionId := azureSubscriptionId
	savedOrg := org
	savedPrefix := prefix
	savedMode := mode
	savedDomain := domain
	savedPrivateDomain := privateDomain
	savedDelegationSetId := delegationSetId
	savedRegistryCredsArn := registryCredsArn
	savedAwsProfile := awsProfile
	savedStateUrl := stateUrl
	t.Cleanup(func() {
		awsRegion = savedAwsRegion
		gcpProjectId = savedGcpProjectId
		azureSubscriptionId = savedAzureSubscriptionId
		org = savedOrg
		prefix = savedPrefix
		mode = savedMode
		domain = savedDomain
		privateDomain = savedPrivateDomain
		delegationSetId = savedDelegationSetId
		registryCredsArn = savedRegistryCredsArn
		awsProfile = savedAwsProfile
		stateUrl = savedStateUrl
	})

	awsRegion = "us-east-1"
	gcpProjectId = ""
	azureSubscriptionId = ""
	org = "testorg"
	prefix = "TestPrefix"
	mode = "development"
	domain = "example.com"
	privateDomain = "internal.example.com"
	delegationSetId = "DELEGSET123"
	registryCredsArn = "arn:aws:secretsmanager:us-east-1:123456789:secret:creds"
	awsProfile = "myprofile"
	stateUrl = "http://example.com/state"

	cfg, err := stackConfig()
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
	if cfg["defang:ciRegistryCredentialsArn"].Value != registryCredsArn {
		t.Errorf("expected ciRegistryCredentialsArn, got %q", cfg["defang:ciRegistryCredentialsArn"].Value)
	}
}

func TestStackConfigGCP(t *testing.T) {
	savedAwsRegion := awsRegion
	savedGcpProjectId := gcpProjectId
	savedAzureSubscriptionId := azureSubscriptionId
	savedOrg := org
	savedPrefix := prefix
	savedMode := mode
	savedGcpRegion := gcpRegion
	savedDomain := domain
	savedPrivateDomain := privateDomain
	savedDelegationSetId := delegationSetId
	savedRegistryCredsArn := registryCredsArn
	t.Cleanup(func() {
		awsRegion = savedAwsRegion
		gcpProjectId = savedGcpProjectId
		azureSubscriptionId = savedAzureSubscriptionId
		org = savedOrg
		prefix = savedPrefix
		mode = savedMode
		gcpRegion = savedGcpRegion
		domain = savedDomain
		privateDomain = savedPrivateDomain
		delegationSetId = savedDelegationSetId
		registryCredsArn = savedRegistryCredsArn
	})

	awsRegion = ""
	gcpProjectId = "my-gcp-project"
	azureSubscriptionId = ""
	gcpRegion = "us-central1"
	org = "testorg"
	prefix = "Test"
	mode = "production"
	domain = ""
	privateDomain = ""
	delegationSetId = ""
	registryCredsArn = ""

	cfg, err := stackConfig()
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

func TestStackConfigAzure(t *testing.T) {
	savedAwsRegion := awsRegion
	savedGcpProjectId := gcpProjectId
	savedAzureSubscriptionId := azureSubscriptionId
	savedAzureLocation := azureLocation
	savedOrg := org
	savedPrefix := prefix
	savedMode := mode
	savedDomain := domain
	savedPrivateDomain := privateDomain
	savedDelegationSetId := delegationSetId
	savedRegistryCredsArn := registryCredsArn
	t.Cleanup(func() {
		awsRegion = savedAwsRegion
		gcpProjectId = savedGcpProjectId
		azureSubscriptionId = savedAzureSubscriptionId
		azureLocation = savedAzureLocation
		org = savedOrg
		prefix = savedPrefix
		mode = savedMode
		domain = savedDomain
		privateDomain = savedPrivateDomain
		delegationSetId = savedDelegationSetId
		registryCredsArn = savedRegistryCredsArn
	})

	awsRegion = ""
	gcpProjectId = ""
	azureSubscriptionId = "sub-123"
	azureLocation = "westus2"
	org = "testorg"
	prefix = "Test"
	mode = "development"
	domain = ""
	privateDomain = ""
	delegationSetId = ""
	registryCredsArn = ""

	cfg, err := stackConfig()
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
