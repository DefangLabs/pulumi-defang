package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
)

func TestUpload(t *testing.T) {
	var got map[string]string
	var receivedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		receivedContentType = r.Header.Get("Content-Type")
		_ = decodeBody(r.Body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	payload := map[string]string{"key": "value"}
	if err := doUpload(t.Context(), srv.URL, payload); err != nil {
		t.Fatalf("doUpload failed: %v", err)
	}

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", receivedContentType)
	}

	if got["key"] != "value" {
		t.Errorf("got key=%q, want %q", got["key"], "value")
	}
}

func TestUploadEventsEmpty(t *testing.T) {
	// Should still send request when events are empty
	var called atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
	}))
	t.Cleanup(srv.Close)

	uploadEvents[any](t.Context(), srv.URL, nil)
	uploadEvents(t.Context(), srv.URL, []events.EngineEvent{})

	if called.Load() != 2 {
		t.Error("expected 2 HTTP requests for empty events")
	}
}

func TestUploadEventsNoUrl(t *testing.T) {
	// Should not send request when URL is empty
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
	}))
	t.Cleanup(srv.Close)

	uploadEvents(t.Context(), "", []events.EngineEvent{{}})

	if called.Load() {
		t.Error("expected no HTTP request when URL is empty")
	}
}

func TestUploadEventsSendsPayload(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeBody(r.Body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	uploadEvents(t.Context(), srv.URL, []events.EngineEvent{{}})

	evts, ok := got["events"].([]any)
	if !ok || len(evts) != 1 {
		t.Errorf("expected 1 event, got %v", got)
	}
}

func TestUploadEventsSurvivesCanceledContext(t *testing.T) {
	// Regression test for https://github.com/DefangLabs/pulumi-defang/issues/103:
	// SIGINT/SIGTERM/timeout cancel the run's ctx right before the final
	// upload starts, which used to abort the upload before anything was sent.
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // simulate SIGINT/SIGTERM/timeout firing before the final upload

	uploadEvents(ctx, srv.URL, []events.EngineEvent{{}})

	if !called.Load() {
		t.Error("expected the upload to still be sent after ctx was canceled")
	}
}

func TestDetachUploadContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if ctx.Err() == nil {
		t.Fatal("test setup: parent ctx should be canceled")
	}

	detached, cancelDetached := detachUploadContext(ctx)
	defer cancelDetached()

	if err := detached.Err(); err != nil {
		t.Errorf("expected detached context to not be canceled, got %v", err)
	}
	if _, ok := detached.Deadline(); !ok {
		t.Error("expected detached context to carry its own deadline")
	}
}

func decodeBody(body io.Reader, v any) error {
	if gr, err := gzip.NewReader(body); err == nil {
		defer gr.Close()
		body = gr
	}
	return json.NewDecoder(body).Decode(v)
}
