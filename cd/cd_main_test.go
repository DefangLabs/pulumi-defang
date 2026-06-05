package main

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
