package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
)

// installEtagPrefix rewires os.Stdout/os.Stderr (and the standard log package)
// so every line written by this process is prefixed with [defang-etag=<etag>] .
// The prefix lets the CLI filter Container App console logs by etag via KQL
// `Log_s has "<etag>"` against ContainerAppConsoleLogs_CL — Azure Container
// Apps doesn't auto-promote JSON-on-stdout into structured columns the way GCP
// Cloud Logging does, so log-line tagging is the only mechanism that works.
//
// Returns a flush func that drains any unterminated buffered output. Callers
// must defer it before the process exits or the last unterminated line is lost.
//
// Etag empty → no-op; returns a flush that does nothing.
func installEtagPrefix(etag string) (flush func()) {
	if etag == "" {
		return func() {}
	}
	prefix := []byte(fmt.Sprintf("[defang-etag=%s] ", etag))

	stdoutFlush := redirectFD(&os.Stdout, prefix)
	stderrFlush := redirectFD(&os.Stderr, prefix)
	// log.* defaults to os.Stderr; reassign so the new value is picked up.
	log.SetOutput(os.Stderr)

	return func() {
		stdoutFlush()
		stderrFlush()
	}
}

// redirectFD replaces *fp with a pipe writer and spawns a goroutine that reads
// the pipe, prefixes each newline-terminated chunk, and writes back to the
// original file. This intercepts every write that goes through the global
// os.Stdout / os.Stderr — including the standard log package, fmt.Println, and
// any third-party library that writes to those handles.
//
// Returns a flush func that closes the pipe-writer side and waits for the
// reader goroutine to drain — required so a final unterminated line is
// emitted with its prefix before the process exits.
func redirectFD(fp **os.File, prefix []byte) func() {
	original := *fp
	r, w, err := os.Pipe()
	if err != nil {
		// Pipe creation should never fail under normal conditions; if it does,
		// silently fall through and leave the original FD in place. Logging
		// here would risk recursion through the very stream we're wrapping.
		return func() {}
	}
	*fp = w

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// bufio.Scanner with default buffer caps lines at 64 KiB which is
		// fine for log output but trips on Pulumi's huge resource-diff dumps.
		// Use a larger explicit buffer.
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			// Reuse a single buffer for prefix+line+newline. fmt.Fprintf into
			// the original file would route back through Sprintf allocation —
			// direct Write is cheaper and stays inside one syscall most of
			// the time.
			out := make([]byte, 0, len(prefix)+len(line)+1)
			out = append(out, prefix...)
			out = append(out, line...)
			out = append(out, '\n')
			_, _ = original.Write(out)
		}
		// Scanner stops on EOF (pipe close) or error. Both are terminal — no
		// recovery path. If error is non-EOF we swallow it because surfacing
		// it would require writing to the very FD we just tore down.
	}()

	return func() {
		// Close the writer side so the scanner sees EOF.
		_ = w.Close()
		wg.Wait()
		// Restore the original FD so any post-flush writes (e.g. from
		// log.Fatalf during shutdown) still reach the user's terminal.
		*fp = original
	}
}

