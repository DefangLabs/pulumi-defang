//go:build integration

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

// TestInterpolation runs `pulumi up` against the current directory, then GETs
// each cloud's `web` endpoint and verifies the container's env output matches
// the expected interpolation of testEnvironment.
//
// Tagged `integration` — skipped by default; to run:
//
//	go test -tags integration -run Interpolation -v -timeout 30m
//
// Reads the stack name from PULUMI_STACK (default "dev").
func TestInterpolation(t *testing.T) {
	ctx := t.Context()

	stackName := os.Getenv("PULUMI_STACK")
	if stackName == "" {
		stackName = "dev"
	}
	stack, err := auto.UpsertStackLocalSource(ctx, stackName, ".")
	if err != nil {
		t.Fatalf("selecting stack: %v", err)
	}

	res, err := stack.Up(ctx)
	if err != nil {
		t.Fatalf("pulumi up: %v", err)
	}

	// Expected values mirror testEnvironment with ${CONFIG} resolved to
	// configValue. Compose modifier semantics (CONFIG is set):
	//   ${CONFIG?required}     → configValue   (required: fail if unset)
	//   ${CONFIG-defaultValue} → configValue   (default: used only if unset)
	//   ${CONFIG+altValue}     → "altValue"    (alt: used only if set)
	expected := map[string]string{
		"TEST_LITERAL":           "verbatim",
		"TEST_CONFIG":            configValue,
		"TEST_OTHER":             configValue,
		"TEST_INTERPOLATED":      "prefix" + configValue + "suffix",
		"TEST_EMPTY":             "",
		"TEST_MODIFIER_REQUIRED": configValue,
		"TEST_MODIFIER_DEFAULT":  configValue,
		"TEST_MODIFIER_ALT":      "altValue",
	}

	for _, cloud := range []string{"aws", "gcp", "azure"} {
		t.Run(cloud, func(t *testing.T) {
			t.Parallel()
			out, ok := res.Outputs[cloud+"-endpoints"]
			if !ok {
				t.Skipf("no %s-endpoints output", cloud)
			}
			endpoints, ok := out.Value.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T: %v", out.Value, out.Value)
			}
			url, ok := endpoints["web"].(string)
			if !ok {
				t.Fatalf("no 'web' in %s-endpoints: %v", cloud, endpoints)
			}

			body := getWithRetry(t, url)
			got := parseEnv(body)
			for k, want := range expected {
				g, ok := got[k]
				switch {
				case !ok:
					t.Errorf("%s: missing; body=%q", k, body)
				case g != want:
					t.Errorf("%s: got %q, want %q", k, g, want)
				}
			}
		})
	}
}

// getWithRetry polls url until it returns 200, up to 5 minutes — covers the
// delay between `pulumi up` returning and the container actually serving.
func getWithRetry(t *testing.T, url string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
		} else if resp.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			return string(body)
		} else {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %s", resp.Status)
		}
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("GET %s after 5m: %v", url, lastErr)
	return ""
}

// parseEnv turns `KEY=VALUE\n…` (the output of `env`) into a map.
func parseEnv(body string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(body, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		result[k] = v
	}
	return result
}

func TestInterpolationDestroy(t *testing.T) {
	stackName := os.Getenv("PULUMI_STACK")
	if stackName == "" {
		stackName = "dev"
	}
	stack, err := auto.UpsertStackLocalSource(t.Context(), stackName, ".")
	if err != nil {
		t.Fatalf("selecting stack: %v", err)
	}

	_, err = stack.Destroy(context.Background())
	if err != nil {
		t.Errorf("pulumi destroy: %v", err)
	}
}
