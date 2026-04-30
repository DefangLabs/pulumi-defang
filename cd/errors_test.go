package main

import (
	"errors"
	"fmt"
	"testing"
)

// autoErrorDump synthesizes the format Pulumi's auto SDK autoError.Error()
// produces — see auto/errors.go: fmt.Sprintf("%s\ncode: %d\nstdout: %s\nstderr: %s\n", ...).
// The autoError type itself is unexported (pulumi/pulumi#6212), so tests
// reproduce its serialized form rather than constructing it directly.
func autoErrorDump(wrapped string, code int, stdout, stderr string) error {
	return fmt.Errorf("%s\ncode: %d\nstdout: %s\nstderr: %s\n", wrapped, code, stdout, stderr)
}

// TestPulumiErrConcurrentUpdate is the Go port of byoc.test.ts's
// "concurrent update error mock" — synthesize the autoError dump for a
// stack-locked failure and verify pulumiErr extracts the lock message
// (with its indented continuation line) plus the nested exit code.
func TestPulumiErrConcurrentUpdate(t *testing.T) {
	stderr := "error: the stack is currently locked by 1 lock(s). Either wait for the other process(es) to end or delete the lock file with `pulumi cancel`.\n" +
		"  s3://defang-cd-bucket/.pulumi/locks/org/proj/stack/lock.json: created by user@host (pid 12345) at 2026-02-19T11:51:59-08:00"

	got := pulumiErr(autoErrorDump("Command failed with exit code 255", 255, "", stderr))

	var pErr *pulumiError
	if !errors.As(got, &pErr) {
		t.Fatalf("expected *pulumiError, got %T (%v)", got, got)
	}
	if pErr.code != 255 {
		t.Errorf("code = %d, want 255", pErr.code)
	}
	if pErr.msg != stderr {
		t.Errorf("msg mismatch:\n got: %q\nwant: %q", pErr.msg, stderr)
	}
}

// TestPulumiErrPassthrough verifies that errors without an "error:" block
// (e.g. plain Go errors from non-Pulumi codepaths like fetch/Unmarshal)
// pass through pulumiErr unchanged so callers' fmt.Errorf wrappers compose.
func TestPulumiErrPassthrough(t *testing.T) {
	orig := errors.New("plain network error")
	if got := pulumiErr(orig); got != orig {
		t.Errorf("expected pass-through of %v, got %v", orig, got)
	}
}

// TestPulumiErrMissingCode covers the "error: present, no code:" case —
// happens if pulumiErr is called on something that's not a true autoError
// but still happens to contain an error: line. code defaults to 0, which
// run() treats as the generic-failure fallback.
func TestPulumiErrMissingCode(t *testing.T) {
	got := pulumiErr(errors.New("error: something bad happened"))

	var pErr *pulumiError
	if !errors.As(got, &pErr) {
		t.Fatalf("expected *pulumiError, got %T", got)
	}
	if pErr.code != 0 {
		t.Errorf("code = %d, want 0", pErr.code)
	}
	if pErr.msg != "error: something bad happened" {
		t.Errorf("msg = %q", pErr.msg)
	}
}

// TestPulumiErrUnknownErrCode exercises Pulumi's -2 unknownErrCode path —
// the doubly-wrapped CommandError case the TS code special-cased. pulumiErr
// preserves the code as-is; run() filters non-positive codes out.
func TestPulumiErrUnknownErrCode(t *testing.T) {
	got := pulumiErr(autoErrorDump("wrapped", -2, "", "error: doubly wrapped"))

	var pErr *pulumiError
	if !errors.As(got, &pErr) {
		t.Fatalf("expected *pulumiError, got %T", got)
	}
	if pErr.code != -2 {
		t.Errorf("code = %d, want -2", pErr.code)
	}
}

// TestPulumiErrIndented covers a Pulumi up failure where stderr has a
// diagnostic block — the error: line is indented, not at column 0. The
// regex must still find it (this is the "drop the ^ anchor" fix).
func TestPulumiErrIndented(t *testing.T) {
	stderr := "Diagnostics:\n" +
		"  pulumi:pulumi:Stack (proj-stack):\n" +
		"    error: deploying urn:pulumi:lio::proj::resource failed: 403 Forbidden"

	got := pulumiErr(autoErrorDump("failed to run update: exit status 255", 255, "", stderr))

	var pErr *pulumiError
	if !errors.As(got, &pErr) {
		t.Fatalf("expected *pulumiError, got %T", got)
	}
	if pErr.code != 255 {
		t.Errorf("code = %d, want 255", pErr.code)
	}
	want := "error: deploying urn:pulumi:lio::proj::resource failed: 403 Forbidden"
	if pErr.msg != want {
		t.Errorf("msg = %q, want %q", pErr.msg, want)
	}
}
