package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
)

func uploadEvents(ctx context.Context, engineEvents []events.EngineEvent) {
	if eventsUploadUrl == "" {
		return
	}
	Println("Sending", len(engineEvents), "deployment events to Portal...")
	if err := doUpload(ctx, eventsUploadUrl, map[string]any{"events": engineEvents}); err != nil {
		warn("Failed to send events:", err)
	}
}

func uploadState(ctx context.Context, stack auto.Stack) {
	if statesUploadUrl == "" {
		return
	}
	Println("Sending deployment state to Portal...")
	state, err := stack.Export(ctx)
	if err != nil {
		warn("Failed to export stack state:", err)
		return
	}
	if err := doUpload(ctx, statesUploadUrl, state); err != nil {
		warn("Failed to send state:", err)
	}
}

func doUpload(ctx context.Context, url string, body any) error {
	// Stream JSON straight into the gzip writer so we don't allocate a
	// separate uncompressed buffer (state payloads can be MBs). Buffer the
	// gzipped output instead of using io.Pipe: retryablehttp re-reads the
	// body on retry, which requires a seekable reader. Content-Encoding is
	// unsigned in the SigV4 presigned URL (Fabric doesn't include it in
	// SignedHeaders), so adding it here is safe; S3 still persists it as
	// object metadata, and the Portal read path checks response.ContentEncoding
	// to decide whether to gunzip.
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if err := json.NewEncoder(gw).Encode(body); err != nil {
		return fmt.Errorf("failed to encode body: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// For testing, allow "uploads" to local files.
	if path, ok := strings.CutPrefix(url, "file://"); ok {
		return os.WriteFile(path, gzBuf.Bytes(), 0600)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPut, url, &gzBuf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip") // TODO: skip compression for small bodies
	// ContentLength is set automatically by NewRequestWithContext for *bytes.Buffer.

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil                          // Disable logging
	retryClient.HTTPClient.Timeout = 30 * time.Second // Per-attempt; overall budget = retries × this
	resp, err := retryClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"upload failed: status=%d body=%q",
			resp.StatusCode,
			string(respBody),
		)
	}
	return nil
}
