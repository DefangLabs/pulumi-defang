package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"google.golang.org/protobuf/proto"
)

func mustMarshal(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}
	return b
}

func TestExtractComposeYaml(t *testing.T) {
	compose := []byte("services:\n  web:\n    image: nginx\n")

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:  "compose only",
			input: mustMarshal(t, &defangv1.ProjectUpdate{Compose: compose}),
			want:  string(compose),
		},
		{
			name: "compose among other fields",
			input: mustMarshal(t, &defangv1.ProjectUpdate{
				Project:   "my-project",
				Compose:   compose,
				CdVersion: "1.0.0",
				Mode:      defangv1.DeploymentMode_DEVELOPMENT,
			}),
			want: string(compose),
		},
		{
			name:    "missing compose",
			input:   mustMarshal(t, &defangv1.ProjectUpdate{Project: "my-project"}),
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "invalid protobuf",
			input:   []byte{0xff, 0xff, 0xff},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractComposeYaml(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchPayloadBase64(t *testing.T) {
	original := []byte("hello world")
	encoded := base64.StdEncoding.EncodeToString(original)

	got, err := fetchPayload(t.Context(), encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("got %q, want %q", got, original)
	}
}

func TestFetchPayloadBase64Invalid(t *testing.T) {
	// "not-base64!!!" is not valid base64 and not a recognized URI scheme
	_, err := fetchPayload(t.Context(), "not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestFetchHTTP(t *testing.T) {
	body := []byte("response body")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)

	got, err := fetchHTTP(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	_, err := fetchHTTP(t.Context(), srv.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestFetchPayloadHTTP(t *testing.T) {
	body := []byte("http payload")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	t.Cleanup(srv.Close)

	got, err := fetchPayload(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestUpload(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	payload := map[string]string{"key": "value"}
	upload(t.Context(), srv.URL, payload)

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", receivedContentType)
	}

	var got map[string]string
	if err := json.Unmarshal(receivedBody, &got); err != nil {
		t.Fatalf("failed to unmarshal received body: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("got key=%q, want %q", got["key"], "value")
	}
}

func TestUploadEventsEmpty(t *testing.T) {
	// Should not send request when events are empty
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	t.Cleanup(srv.Close)

	saved := eventsUploadUrl
	eventsUploadUrl = srv.URL
	t.Cleanup(func() { eventsUploadUrl = saved })

	uploadEvents(t.Context(), nil)
	uploadEvents(t.Context(), []events.EngineEvent{})

	if called {
		t.Error("expected no HTTP request for empty events")
	}
}

func TestUploadEventsNoUrl(t *testing.T) {
	// Should not send request when URL is empty
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	t.Cleanup(srv.Close)

	saved := eventsUploadUrl
	eventsUploadUrl = ""
	t.Cleanup(func() { eventsUploadUrl = saved })

	uploadEvents(t.Context(), []events.EngineEvent{{}})

	if called {
		t.Error("expected no HTTP request when URL is empty")
	}
}

func TestUploadEventsSendsPayload(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	saved := eventsUploadUrl
	eventsUploadUrl = srv.URL
	t.Cleanup(func() { eventsUploadUrl = saved })

	uploadEvents(t.Context(), []events.EngineEvent{{}})

	var got map[string]any
	if err := json.Unmarshal(receivedBody, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	evts, ok := got["events"].([]any)
	if !ok || len(evts) != 1 {
		t.Errorf("expected 1 event, got %v", got)
	}
}

func TestCollectEvents(t *testing.T) {
	ch, evts := collectEvents()

	ch <- events.EngineEvent{}
	ch <- events.EngineEvent{}
	close(ch)

	// The goroutine drains the channel asynchronously; give it time to finish
	// after we close the channel.
	deadline := time.After(2 * time.Second)
	for {
		if len(*evts) == 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for events, got %d", len(*evts))
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

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
	if len(disabledList) != 1 {
		t.Errorf("expected 1 disabled provider, got %d", len(disabledList))
	}
	if disabledList[0] != "*" {
		t.Errorf("expected disabled provider '*', got %q", disabledList[0])
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
			if got := Getenv(tt.key, tt.fallback); got != tt.want {
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
			got := SplitByComma(tt.input)
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
			if got := GetenvBool(key); got != tt.want {
				t.Errorf("GetenvBool(%q) with value %q = %v, want %v", key, tt.val, got, tt.want)
			}
		})
	}
}

func TestStackConfigAWS(t *testing.T) {
	// Save and restore all globals that stackConfig reads
	savedAwsRegion := awsRegion
	savedGcpProject := gcpProject
	savedAzureSubscription := azureSubscription
	savedOrg := org
	savedPrefix := prefix
	savedMode := mode
	savedDomain := domain
	savedPrivateDomain := privateDomain
	savedDelegationSetId := delegationSetId
	savedRegistryCredsArn := registryCredsArn
	savedAwsProfile := awsProfile
	t.Cleanup(func() {
		awsRegion = savedAwsRegion
		gcpProject = savedGcpProject
		azureSubscription = savedAzureSubscription
		org = savedOrg
		prefix = savedPrefix
		mode = savedMode
		domain = savedDomain
		privateDomain = savedPrivateDomain
		delegationSetId = savedDelegationSetId
		registryCredsArn = savedRegistryCredsArn
		awsProfile = savedAwsProfile
	})

	awsRegion = "us-east-1"
	gcpProject = ""
	azureSubscription = ""
	org = "testorg"
	prefix = "TestPrefix"
	mode = "development"
	domain = "example.com"
	privateDomain = "internal.example.com"
	delegationSetId = "DELEGSET123"
	registryCredsArn = "arn:aws:secretsmanager:us-east-1:123456789:secret:creds"
	awsProfile = "myprofile"

	cfg := stackConfig()

	if cfg["defang:provider"].Value != "aws" {
		t.Errorf("expected provider aws, got %q", cfg["defang:provider"].Value)
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
	savedGcpProject := gcpProject
	savedAzureSubscription := azureSubscription
	savedOrg := org
	savedPrefix := prefix
	savedMode := mode
	savedRegion := region
	savedDomain := domain
	savedPrivateDomain := privateDomain
	savedDelegationSetId := delegationSetId
	savedRegistryCredsArn := registryCredsArn
	t.Cleanup(func() {
		awsRegion = savedAwsRegion
		gcpProject = savedGcpProject
		azureSubscription = savedAzureSubscription
		org = savedOrg
		prefix = savedPrefix
		mode = savedMode
		region = savedRegion
		domain = savedDomain
		privateDomain = savedPrivateDomain
		delegationSetId = savedDelegationSetId
		registryCredsArn = savedRegistryCredsArn
	})

	awsRegion = ""
	gcpProject = "my-gcp-project"
	azureSubscription = ""
	region = "us-central1"
	org = "testorg"
	prefix = "Test"
	mode = "production"
	domain = ""
	privateDomain = ""
	delegationSetId = ""
	registryCredsArn = ""

	cfg := stackConfig()

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
	savedGcpProject := gcpProject
	savedAzureSubscription := azureSubscription
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
		gcpProject = savedGcpProject
		azureSubscription = savedAzureSubscription
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
	gcpProject = ""
	azureSubscription = "sub-123"
	azureLocation = "westus2"
	org = "testorg"
	prefix = "Test"
	mode = "development"
	domain = ""
	privateDomain = ""
	delegationSetId = ""
	registryCredsArn = ""

	cfg := stackConfig()

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
