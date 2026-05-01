package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Wrap each sink in a LineWriter so a downstream WriteString call always
// receives a complete line. Required for cloud sinks (e.g. NewLabelsLogger)
// that emit one structured log entry per WriteString, and for the etag-prefix
// wrapper below to prepend exactly once per line; harmless over plain stdio.
var (
	stdoutLogger io.StringWriter = os.Stdout
	stderrLogger io.StringWriter = os.Stderr
)

// withEtagPrefix prepends [defang-etag=<etag>] to every line so the CLI can
// filter Azure ContainerAppConsoleLogs_CL by KQL `Log_s has "<etag>"`. Azure
// Container Apps doesn't auto-promote JSON-on-stdout into structured columns
// the way GCP Cloud Logging does, so per-line tagging is the only mechanism
// that works there. Etag empty → no prefix wrapper.
func withEtagPrefix(sw io.StringWriter) io.StringWriter {
	if etag == "" {
		return sw
	}
	return &prefixStringWriter{sw: sw, prefix: fmt.Sprintf("[defang-etag=%s] ", etag)}
}

// warn is like `console.warn` in JavaScript and logs to stderr.
func warn(v ...interface{}) {
	stderrLogger.WriteString(fmt.Sprintln(v...)) // nolint:errcheck
}

func Println(v ...interface{}) {
	stdoutLogger.WriteString(fmt.Sprintln(v...)) // nolint:errcheck
}

// Ignore Pulumi log lines containing any of these substrings, because they are
// not relevant to Defang users. (Consistent with pulumi/index.ts)
var ignoreSubstrs = []string{
	" `pulumi ",
	"press ^C",
	"grpc: the client connection is closing",
	") will not show as a stack output.",
}

func newProgressStream() *LineWriter {
	return NewLineWriter(NewIgnoreStringWriter(stdoutLogger, ignoreSubstrs))
}

func newErrorProgressStream() *LineWriter {
	return NewLineWriter(NewIgnoreStringWriter(stderrLogger, ignoreSubstrs))
}

// prefixStringWriter prepends a fixed prefix to every WriteString call. Used
// downstream of LineWriter, so each call is one full line; the prefix and the
// line are concatenated into a single downstream WriteString to preserve the
// "one WriteString = one log entry" contract that structured sinks (e.g.
// NewLabelsLogger) rely on.
type prefixStringWriter struct {
	sw     io.StringWriter
	prefix string
}

func (p *prefixStringWriter) WriteString(s string) (int, error) {
	n, err := p.sw.WriteString(p.prefix + s)
	// Report bytes from s only, not the added prefix, so callers' length
	// accounting matches what they handed us.
	return max(0, n-len(p.prefix)), err
}

type IgnoreWriter struct {
	ignoreSubstrs []string
	sw            io.StringWriter
}

func NewIgnoreStringWriter(sw io.StringWriter, ignoreSubstrs []string) *IgnoreWriter {
	return &IgnoreWriter{ignoreSubstrs: ignoreSubstrs, sw: sw}
}

func (iw *IgnoreWriter) WriteString(s string) (int, error) {
	for _, ignore := range iw.ignoreSubstrs {
		if strings.Contains(s, ignore) {
			return len(s), nil
		}
	}
	return iw.sw.WriteString(s)
}

type LineWriter struct {
	sw   io.StringWriter
	buf  bytes.Buffer
	lock sync.Mutex
}

func NewLineWriter(sw io.StringWriter) *LineWriter {
	return &LineWriter{sw: sw}
}

func (lw *LineWriter) Write(p []byte) (n int, err error) {
	lines := bytes.SplitAfter(p, []byte("\n"))

	lw.lock.Lock()
	defer lw.lock.Unlock()

	lw.buf.Write(lines[0])
	// All but the last are complete lines once joined with the buffer.
	for i := 1; i < len(lines); i++ {
		if _, err := lw.sw.WriteString(lw.buf.String()); err != nil {
			return 0, err
		}
		lw.buf.Reset()
		lw.buf.Write(lines[i])
	}
	return len(p), nil
}

// WriteString lets LineWriter stand in anywhere an io.StringWriter is wanted
// (warn/Println, NewIgnoreStringWriter). Each call is expected to pass a
// complete line; partial writes will buffer until the next newline.
func (lw *LineWriter) WriteString(s string) (int, error) {
	return lw.Write([]byte(s))
}

// Flush emits any buffered partial line to the underlying sink. Use before
// os.Exit so trailing output without a newline isn't dropped.
func (lw *LineWriter) Flush() error {
	lw.lock.Lock()
	defer lw.lock.Unlock()
	if lw.buf.Len() == 0 {
		return nil
	}
	_, err := lw.sw.WriteString(lw.buf.String())
	lw.buf.Reset()
	return err
}
