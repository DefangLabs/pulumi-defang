package main

import (
	"compress/gzip"
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

	saved := eventsUploadUrl
	eventsUploadUrl = srv.URL
	t.Cleanup(func() { eventsUploadUrl = saved })

	uploadEvents(t.Context(), nil)
	uploadEvents(t.Context(), []events.EngineEvent{})

	if called.Load() != 2 {
		t.Errorf("expected 2 HTTP requests for empty events, got %d", called.Load())
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
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeBody(r.Body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	saved := eventsUploadUrl
	eventsUploadUrl = srv.URL
	t.Cleanup(func() { eventsUploadUrl = saved })

	uploadEvents(t.Context(), []events.EngineEvent{{}})

	evts, ok := got["events"].([]any)
	if !ok || len(evts) != 1 {
		t.Errorf("expected 1 event, got %v", got)
	}
}

func decodeBody(body io.Reader, v any) error {
	if gr, err := gzip.NewReader(body); err == nil {
		defer gr.Close()
		body = gr
	}
	return json.NewDecoder(body).Decode(v)
}
