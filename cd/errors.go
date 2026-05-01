package main

import (
	"regexp"
	"strconv"
	"syscall"
)

// usageError signals a CLI/config validation failure → exit 2.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// signalError is set as the cancel cause of ctx when SIGINT/SIGTERM arrives, so
// exitCode can recover which signal fired. Plain ctx.Err() loses this.
type signalError struct{ sig syscall.Signal }

func (e *signalError) Error() string { return "received signal " + e.sig.String() }

// pulumiErrRE captures Pulumi's "error: ..." line plus any indented
// continuation lines. The auto SDK's autoError.Error() emits the wrapped err
// followed by the full code/stdout/stderr blob — we already streamed those
// live, so all that's worth repeating at the end is this line.
//
// No ^ anchor: the autoError formatter inlines stderr after "stderr: " on the
// same line, so an error: that lives at the head of stderr appears as
// "stderr: error: ..." in the dump — never at column 0. Match anywhere.
var pulumiErrRE = regexp.MustCompile(`(?i)error:.+(\n[ \t]{2,}.+)*`)

// pulumiCodeRE captures the exit code embedded in autoError.Error(). The
// autoError type and its `code int` field are unexported (pulumi/pulumi#6212),
// so the only way to get at the nested process's exit status is to grep the
// formatted text. Format is fixed to "\ncode: %d\n".
var pulumiCodeRE = regexp.MustCompile(`\ncode: (-?\d+)\n`)

// pulumiError preserves the "error:" line plus the nested Pulumi process exit
// code so run() can propagate it as the cd binary's own exit status.
type pulumiError struct {
	msg  string
	code int
}

func (e *pulumiError) Error() string { return e.msg }

// pulumiErr replaces an auto SDK error with just its "error: ..." block and
// the nested process exit code. The SDK's wrapped err and full code/stdout/
// stderr dump have already been streamed live, and the extracted line already
// names the operation (e.g. "error: deploying urn:..."), so an outer
// "failed to deploy:" wrapper would just duplicate that. Returns err unchanged
// if no such block is found.
func pulumiErr(err error) error {
	text := err.Error()
	msg := pulumiErrRE.FindString(text)
	if msg == "" {
		return err
	}
	code := 0
	if m := pulumiCodeRE.FindStringSubmatch(text); m != nil {
		// Atoi only fails if the regex matched non-digits, which it won't.
		code, _ = strconv.Atoi(m[1])
	}
	return &pulumiError{msg: msg, code: code}
}
