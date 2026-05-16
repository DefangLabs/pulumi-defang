package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "development" // overwritten by -ldflags "-X main.version=..."

func main() {
	// All cleanup (signal.Stop, ctx cancels) lives in run() so its defers
	// actually fire — os.Exit skips deferred calls in the function it's in.
	os.Exit(run(os.Args...))
}

func run(args ...string) int {
	// --version is informational; skip signal/timeout setup so it stays cheap.
	if len(args) > 1 && args[1] == "--version" {
		fmt.Println(version)
		return 0
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ctx, cancelCause := context.WithCancelCause(context.Background())
	defer cancelCause(nil)
	go func() {
		if s, ok := <-sigCh; ok {
			cancelCause(&signalError{sig: s.(syscall.Signal)})
		}
	}()

	ctx, cancelTimeout := context.WithTimeoutCause(ctx, 60*time.Minute, &signalError{sig: syscall.SIGXCPU}) // like TS
	defer cancelTimeout()

	if err := cdMain(ctx, args...); err != nil {
		warn(err.Error())

		var usageErr *usageError
		if errors.As(err, &usageErr) {
			return 2 // usage error
		}
		var sigErr *signalError
		if errors.As(context.Cause(ctx), &sigErr) {
			return 128 + int(sigErr.sig) // SIGINT=130, SIGTERM=143, SIGXCPU=152 (timeout)
		}
		var pErr *pulumiError
		if errors.As(err, &pErr) && pErr.code > 0 {
			// Bubble up the nested Pulumi process exit code (e.g. 255). Skip
			// non-positive values (-2 unknownErrCode, 0) which aren't useful
			// process statuses — fall through to the generic 1.
			return pErr.code
		}
		return 1 // generic failure
	}
	return 0
}
