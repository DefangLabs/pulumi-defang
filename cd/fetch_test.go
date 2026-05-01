package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
