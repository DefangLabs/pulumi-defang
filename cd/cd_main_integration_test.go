//go:build integration

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestRunPreviewAzure(t *testing.T) {
	const azureLocation = "westus"
	azureSubscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if azureSubscriptionId == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set; skipping Azure preview integration test")
	}

	t.Run("config from env", func(t *testing.T) {
		t.Setenv("AZURE_LOCATION", azureLocation)
		unsetenv(t, "AWS_REGION", "GCLOUD_PROJECT")

		testProviderPreview(t, "azure", azureSubscriptionId, nil)
	})

	t.Run("config from JSON", func(t *testing.T) {
		config := auto.ConfigMap{
			"defang:cdImage":              auto.ConfigValue{},
			"defang:org":                  auto.ConfigValue{Value: "defang"},
			"defang:stateUrl":             auto.ConfigValue{},
			"defang:version":              auto.ConfigValue{Value: "development"},
			"defang:etag":                 auto.ConfigValue{Value: "defang"},
			"defang:provider":             auto.ConfigValue{Value: "azure"},
			"azure-native:subscriptionId": auto.ConfigValue{Value: azureSubscriptionId},
			"azure-native:location":       auto.ConfigValue{Value: azureLocation},
			"azure-native:useMsi":         auto.ConfigValue{Value: "true"},
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		testProviderPreview(t, "azure", azureSubscriptionId, configBytes)
	})
}

func TestRunPreviewAWS(t *testing.T) {
	const awsRegion = "us-west-2"
	awsProfile := os.Getenv("AWS_PROFILE")
	if awsProfile == "" {
		t.Skip("AWS_PROFILE not set; skipping AWS preview integration test")
	}

	t.Run("config from env", func(t *testing.T) {
		t.Setenv("AWS_REGION", awsRegion)
		unsetenv(t, "AZURE_SUBSCRIPTION_ID", "GCLOUD_PROJECT")

		testProviderPreview(t, "aws", "", nil) // account ID doesn't matter
	})

	t.Run("config from JSON", func(t *testing.T) {
		config := auto.ConfigMap{
			"defang:cdImage":  auto.ConfigValue{},
			"defang:org":      auto.ConfigValue{Value: "defang"},
			"defang:stateUrl": auto.ConfigValue{},
			"defang:version":  auto.ConfigValue{Value: "development"},
			"defang:etag":     auto.ConfigValue{Value: "defang"},
			"defang:provider": auto.ConfigValue{Value: "aws"},
			"aws:profile":     auto.ConfigValue{Value: awsProfile},
			"aws:region":      auto.ConfigValue{Value: awsRegion},
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		testProviderPreview(t, "aws", "", configBytes)
	})
}

func TestRunPreviewGCP(t *testing.T) {
	const gloudRegion = "us-central1"
	gcloudProject := os.Getenv("GCLOUD_PROJECT")
	if gcloudProject == "" {
		t.Skip("GCLOUD_PROJECT not set; skipping GCP preview integration test")
	}

	t.Run("config from env", func(t *testing.T) {
		t.Setenv("GCLOUD_REGION", gloudRegion)
		unsetenv(t, "AWS_REGION", "AZURE_SUBSCRIPTION_ID")

		testProviderPreview(t, "gcp", gcloudProject, nil)
	})

	t.Run("config from JSON", func(t *testing.T) {
		config := auto.ConfigMap{
			"defang:cdImage":  auto.ConfigValue{},
			"defang:org":      auto.ConfigValue{Value: "defang"},
			"defang:stateUrl": auto.ConfigValue{},
			"defang:version":  auto.ConfigValue{Value: "development"},
			"defang:etag":     auto.ConfigValue{Value: "defang"},
			"defang:provider": auto.ConfigValue{Value: "gcp"},
			"gcp:project":     auto.ConfigValue{Value: gcloudProject},
			"gcp:region":      auto.ConfigValue{Value: gloudRegion},
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}
		testProviderPreview(t, "gcp", gcloudProject, configBytes)
	})
}

func testProviderPreview(t *testing.T, provider, accountId string, config []byte) {
	t.Run("install defang-"+provider, func(t *testing.T) {
		makeCmd := exec.CommandContext(t.Context(), "make", "-C", "..", "install_defang-"+provider)
		makeCmd.Stderr = t.Output()
		if err := makeCmd.Run(); err != nil {
			t.Fatalf("failed to install defang-%s provider: %v", provider, err)
		}
	})

	eventsFile := t.TempDir() + "/events.json.gz"
	t.Setenv("DEFANG_EVENTS_UPLOAD_URL", "file://"+eventsFile)
	statesFile := t.TempDir() + "/states.json.gz"
	t.Setenv("DEFANG_STATES_UPLOAD_URL", "file://"+statesFile)
	t.Setenv("DEFANG_PULUMI_DIFF", "true")
	t.Setenv("DEFANG_PULUMI_DEBUG", "false")
	t.Setenv("NO_COLOR", "")
	t.Setenv("PROJECT", "cd-test")
	t.Setenv("STACK", provider)

	projectUpdate := defangv1.ProjectUpdate{
		PulumiConfig: config,
		Compose: []byte(`services:
  nginx:
    image: nginx
    ports:
      - target: 80
        published: "80"`),
	}
	projectPb, err := proto.Marshal(&projectUpdate)
	if err != nil {
		t.Fatalf("failed to marshal project update: %v", err)
	}

	if err := cdMain(t.Context(), "cd", "preview", base64.StdEncoding.EncodeToString(projectPb)); err != nil {
		t.Fatalf("preview failed: %v", err)
	}

	// Ensure events were uploaded
	uploaded, err := readFile(eventsFile)
	if err != nil {
		t.Fatalf("failed to load events: %v", err)
	}

	// Normalize generated random prefixes. The 4-7 hex suffix from autonaming
	// (`${name}-${hex(N)}`) appears in two contexts: at the end of a JSON
	// string (followed by `"`) and embedded in GCP service-account emails
	// (followed by `@`). Match both so the suffix gets erased in either spot.
	uploaded = regexp.MustCompile(`-[0-9a-f]{4,7}(["@])`).ReplaceAll(uploaded, []byte(`-***$1`))
	// Remove references to $HOME (diagnostic messages)
	if home := os.Getenv("HOME"); home != "" {
		uploaded = bytes.ReplaceAll(uploaded, []byte(home), []byte("${HOME}"))
	}
	if accountId != "" {
		// Remove references to cloud account ID
		uploaded = bytes.ReplaceAll(uploaded, []byte(accountId), []byte("***"))
	}

	var eventJson struct {
		Events []events.EngineEvent `json:"events"`
	}
	if err := json.Unmarshal(uploaded, &eventJson); err != nil {
		t.Fatalf("failed to unmarshal events: %v", err)
	}

	// Drop debug-severity diagnostics: these are Pulumi-engine RPC firehose
	// logs ("RegisterResource RPC prepared/finished: ...") emitted in
	// concurrent order with no URN to sort by — pure non-determinism noise
	// for a golden snapshot.
	eventJson.Events = slices.DeleteFunc(eventJson.Events, func(e events.EngineEvent) bool {
		return e.DiagnosticEvent != nil && e.DiagnosticEvent.Severity == "debug"
	})

	// Normalize timestamps, sequence numbers, duration
	for i, e := range eventJson.Events {
		eventJson.Events[i].Timestamp = 42
		eventJson.Events[i].Sequence = 42
		if e.SummaryEvent != nil {
			e.SummaryEvent.DurationSeconds = 42
		}
	}
	// Stable sort by URN across all resource-event variants. Non-resource
	// events (Summary, Prelude, Diagnostic, ...) get an empty URN and stay
	// in their original relative order thanks to SortStable.
	urn := func(e events.EngineEvent) string {
		switch {
		case e.ResourcePreEvent != nil:
			return e.ResourcePreEvent.Metadata.URN
		case e.ResOutputsEvent != nil:
			return e.ResOutputsEvent.Metadata.URN
		case e.ResOpFailedEvent != nil:
			return e.ResOpFailedEvent.Metadata.URN
		}
		return ""
	}
	slices.SortStableFunc(eventJson.Events, func(a, b events.EngineEvent) int {
		return strings.Compare(urn(a), urn(b))
	})

	raw, err := json.MarshalIndent(eventJson, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal events: %v", err)
	}

	goldenFile := "testdata/preview-events-" + provider + ".json"
	golden, err := readFile(goldenFile)
	if os.IsNotExist(err) || os.Getenv("UPDATE_GOLDEN") != "" {
		// Write golden file if it doesn't exist or when UPDATE_GOLDEN is set
		t.Fatalf("updated golden file: %v", os.WriteFile(goldenFile, raw, 0644))
	} else if err != nil {
		t.Fatalf("failed to load golden events: %v", err)
	}
	require.JSONEq(t, string(golden), string(raw))
}

func readFile(path string) ([]byte, error) {
	if b, err := os.ReadFile(path); err != nil {
		return nil, err
	} else if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("readFile: %w", err)
		}
		defer gz.Close()
		return io.ReadAll(gz)
	} else {
		return b, nil
	}
}
