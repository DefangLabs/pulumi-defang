//go:build integration

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/stretchr/testify/require"
)

func TestRunVersion(t *testing.T) {
	if run("cd", "--version") != 0 {
		t.Error("run --version flag should exit with 0")
	}
}

func TestRunUsageError(t *testing.T) {
	if run() != 2 {
		t.Error("run should exit with 2 on usage error")
	}
}

func TestRunPreviewAzure(t *testing.T) {
	t.Setenv("AZURE_SUBSCRIPTION_ID", "f311c4db-e998-4c94-906c-7e2637303a05")
	t.Setenv("AZURE_LOCATION", "westus")

	testProviderPreview(t, "azure")
}

func TestRunPreviewAWS(t *testing.T) {
	t.Setenv("AWS_PROFILE", "defang-lab")
	t.Setenv("AWS_REGION", "us-west-2")

	testProviderPreview(t, "aws")
}

func TestRunPreviewGCP(t *testing.T) {
	t.Setenv("GCP_PROJECT", "liotest-443018") // TODO: pick a neutral project for integration testing
	t.Setenv("GCP_REGION", "us-central1")

	testProviderPreview(t, "gcp")
}

func testProviderPreview(t *testing.T, provider string) {
	t.Helper()

	if err := exec.CommandContext(t.Context(), "make", "-C", "..", "install_defang-"+provider).Run(); err != nil {
		t.Fatalf("failed to install defang-%s provider: %v", provider, err)
	}

	eventsFile := t.TempDir() + "/events.json.gz"
	t.Setenv("DEFANG_EVENTS_UPLOAD_URL", "file://"+eventsFile)
	statesFile := t.TempDir() + "/states.json.gz"
	t.Setenv("DEFANG_STATES_UPLOAD_URL", "file://"+statesFile)
	t.Setenv("DEFANG_PULUMI_DIFF", "true")
	t.Setenv("DEFANG_PULUMI_DEBUG", "false")
	t.Setenv("NO_COLOR", "")
	t.Setenv("PROJECT", "cd-test")
	t.Setenv("STACK", provider)

	if run("cd", "preview", "IlpzZXJ2aWNlczoKICBuZ2lueDoKICAgIGltYWdlOiBuZ2lueAogICAgcG9ydHM6CiAgICAgIC0gdGFyZ2V0OiA4MAogICAgICAgIHB1Ymxpc2hlZDogIjgwIgo=") != 0 {
		t.Fatal("preview should exit with 0")
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
