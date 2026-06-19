package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/DefangLabs/defang/src/pkg/cli/client"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/debug"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"google.golang.org/protobuf/proto"
)

func cdMain(ctx context.Context, args ...string) error {
	etag := os.Getenv("DEFANG_ETAG")
	eventsUploadUrl := os.Getenv("DEFANG_EVENTS_UPLOAD_URL")
	jsonOutput := getenvBool("DEFANG_JSON")
	projectName := os.Getenv("PROJECT") // required
	pulumiDebug := getenvBool("DEFANG_PULUMI_DEBUG")
	pulumiDiff := getenvBool("DEFANG_PULUMI_DIFF")
	pulumiTargets := splitByComma(os.Getenv("DEFANG_PULUMI_TARGETS"))
	stackName := os.Getenv("STACK")
	statesUploadUrl := os.Getenv("DEFANG_STATES_UPLOAD_URL")

	if projectName == "" {
		return &usageError{msg: "missing required environment variable: PROJECT"}
	}
	if stackName != strings.ToLower(stackName) {
		return &usageError{msg: "STACK name must be lowercase"}
	}

	var command client.CdCommand
	if len(args) > 1 {
		command = client.CdCommand(args[1])
	}

	var stack auto.Stack
	var err error
	switch command {
	case "", "help":
		help := fmt.Sprintf("usage: cd [%s <payload>|%s <payload>|%s|%s|%s|%s|%s|%s]",
			client.CdCommandUp, client.CdCommandPreview, // commands with payload
			client.CdCommandDown, client.CdCommandDestroy, client.CdCommandRefresh,
			client.CdCommandCancel, client.CdCommandOutputs, client.CdCommandList)
		Println(help)
		return nil
	case client.CdCommandUp, client.CdCommandPreview:
		// Payload URL from args (like old code): cd <command> <payload>
		if len(args) <= 2 {
			return &usageError{msg: "missing required argument: payload"}
		}
		payload := args[2]
		// Fetch the ProjectUpdate protobuf and pass it through to the Pulumi
		// program; the program extracts the compose YAML itself and uploads
		// the protobuf as a Pulumi-managed blob after the deploy succeeds —
		// so the file only appears on success and is tracked in state (vs. a
		// pre-Pulumi SDK upload that would leave stale records on failure).
		projectUpdate := &defangv1.ProjectUpdate{}
		if payload != "" {
			projectPb, err := fetchPayload(ctx, payload)
			if err != nil {
				return fmt.Errorf("failed to fetch payload: %w", err)
			}
			if err := proto.Unmarshal(projectPb, projectUpdate); err != nil {
				return fmt.Errorf("failed to unmarshal ProjectUpdate protobuf: %w", err)
			}
			// FIXME: what to do when Compose project name does not match PROJECT env var?
		}
		stack, err = auto.UpsertStackInlineSource(ctx, stackName, projectName, program.NewRun(projectUpdate))
		if err != nil {
			return pulumiErr(err)
		}
		// Set stack-level config (provider settings, defang config)
		configJson, err := stackConfigJson(projectUpdate.Recipe.GetPulumiConfig())
		if err != nil {
			return err
		}
		err = stack.SetAllConfigJson(ctx, configJson, nil)
		if err != nil {
			return pulumiErr(err)
		}
	case client.CdCommandDestroy, client.CdCommandDown, client.CdCommandRefresh, client.CdCommandCancel, client.CdCommandOutputs:
		stack, err = auto.SelectStackInlineSource(ctx, stackName, projectName, nil)
		if err != nil {
			return pulumiErr(err)
		}
	case client.CdCommandList:
		// List doesn't need a real stack, but select something so we can call ListStacks on the workspace.
		stack, _ = auto.SelectStackInlineSource(ctx, stackName, projectName, nil)
	default:
		return &usageError{msg: fmt.Sprintf("unknown command: %s", command)}
	}

	// Set workspace env vars
	if etag != "" {
		// USER ends up in Pulumi lock files for debugging; FIXME: this hack doesn't work on linux
		stack.Workspace().SetEnvVar("USER", etag)
	}

	// Set project-level config (disable-default-providers)
	ps, err := stack.Workspace().ProjectSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get project settings: %w", err)
	}
	ps.Config = map[string]workspace.ProjectConfigType{
		// Ensure we create one provider per cloud by disabling the automatic default providers
		"pulumi:disable-default-providers": {
			Value: []string{
				"aws-native",
				"aws",
				"azure-native",
				"azure",
				"eks",
				"gcp",
				"google-beta",
				"google-native",
				"kubernetes",
			},
		},
	}
	if err := stack.Workspace().SaveProjectSettings(ctx, ps); err != nil {
		return fmt.Errorf("failed to save project settings: %w", err)
	}

	// Common option builders per command type
	userAgent := "defang/" + version
	debugLog := debug.LoggingOptions{Debug: pulumiDebug, LogToStdErr: true}
	progressStream := newProgressStream()
	defer progressStream.Flush()
	errorProgressStream := newErrorProgressStream()
	defer errorProgressStream.Flush()

	switch command {
	case client.CdCommandUp:
		evtCh, waitEvents := collectEvents(ctx, eventsUploadUrl, jsonOutput)
		defer waitEvents()
		upOpts := []optup.Option{
			optup.UserAgent(userAgent),
			optup.Color(color()),
			optup.SuppressProgress(),
			optup.ProgressStreams(progressStream),
			optup.ErrorProgressStreams(errorProgressStream),
			optup.EventStreams(evtCh),
			optup.TargetDependents(),
			optup.Target(pulumiTargets),
		}
		if pulumiDebug {
			upOpts = append(upOpts, optup.DebugLogging(debugLog))
		}
		if pulumiDiff {
			upOpts = append(upOpts, optup.Diff())
		}
		_, err := stack.Up(ctx, upOpts...)
		uploadState(ctx, statesUploadUrl, stack)
		if err != nil {
			// TODO: run a refresh on failure to update the state with any partial changes, like the CLI does
			return pulumiErr(err)
		}

	case client.CdCommandPreview:
		evtCh, waitEvents := collectEvents(ctx, eventsUploadUrl, jsonOutput)
		defer waitEvents()
		previewOpts := []optpreview.Option{
			optpreview.UserAgent(userAgent),
			optpreview.Color(color()),
			optpreview.SuppressProgress(),
			optpreview.ProgressStreams(progressStream),
			optpreview.ErrorProgressStreams(errorProgressStream),
			optpreview.EventStreams(evtCh),
			optpreview.TargetDependents(),
		}
		if pulumiDiff {
			previewOpts = append(previewOpts, optpreview.Diff())
		}
		if pulumiDebug {
			previewOpts = append(previewOpts, optpreview.DebugLogging(debugLog))
		}
		previewOpts = append(previewOpts, optpreview.Target(pulumiTargets))
		_, err := stack.Preview(ctx, previewOpts...)
		if err != nil {
			return pulumiErr(err)
		}

	case client.CdCommandDown, client.CdCommandDestroy:
		evtCh, waitEvents := collectEvents(ctx, eventsUploadUrl, jsonOutput)
		defer waitEvents()
		destroyOpts := []optdestroy.Option{
			optdestroy.UserAgent(userAgent),
			optdestroy.Color(color()),
			optdestroy.SuppressProgress(),
			optdestroy.ProgressStreams(progressStream),
			optdestroy.ErrorProgressStreams(errorProgressStream),
			optdestroy.EventStreams(evtCh),
			optdestroy.ContinueOnError(),
			optdestroy.Remove(),
		}
		// down = refresh + destroy (consistent with legacy behavior)
		if command == "down" {
			destroyOpts = append(destroyOpts, optdestroy.Refresh())
		}
		if pulumiDebug {
			destroyOpts = append(destroyOpts, optdestroy.DebugLogging(debugLog))
		}
		_, err = stack.Destroy(ctx, destroyOpts...)
		uploadState(ctx, statesUploadUrl, stack) // TODO: this prints a warning if the destroy succeeded
		if err != nil {
			return pulumiErr(err)
		}

	case client.CdCommandRefresh:
		evtCh, waitEvents := collectEvents(ctx, eventsUploadUrl, jsonOutput)
		defer waitEvents()
		refreshOpts := []optrefresh.Option{
			optrefresh.UserAgent(userAgent),
			optrefresh.Color(color()),
			optrefresh.SuppressProgress(),
			optrefresh.ProgressStreams(progressStream),
			optrefresh.ErrorProgressStreams(errorProgressStream),
			optrefresh.EventStreams(evtCh),
		}
		if pulumiDebug {
			refreshOpts = append(refreshOpts, optrefresh.DebugLogging(debugLog))
		}
		_, err := stack.Refresh(ctx, refreshOpts...)
		uploadState(ctx, statesUploadUrl, stack)
		if err != nil {
			return pulumiErr(err)
		}

	case client.CdCommandCancel:
		if err := stack.Cancel(ctx); err != nil {
			return pulumiErr(err)
		}

	case client.CdCommandOutputs:
		outputs, err := stack.Outputs(ctx)
		if err != nil {
			return pulumiErr(err)
		}
		data, err := json.MarshalIndent(outputs, "", "  ")
		Println(string(data))
		return err

	case client.CdCommandList:
		summary, err := stack.Workspace().ListStacks(ctx)
		if err != nil {
			return pulumiErr(err)
		}
		data, err := json.MarshalIndent(summary, "", "  ")
		Println(string(data))
		return err

	default:
		panic("unknown command " + command) // unreachable due to check in switch above
	}
	return nil
}

func color() string {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return "never"
	}
	return "always"
}

// collectEvents returns a channel to feed engine events into and a wait
// function that blocks until the collector goroutine has drained the
// channel (closed by Pulumi when the operation finishes) and finished
// uploading. Callers must defer the wait function so the final upload
// completes before run() returns and its deferred ctx cancels fire —
// otherwise the in-flight uploadEvents request gets canceled.
func collectEvents(ctx context.Context, eventsUploadUrl string, jsonOutput bool) (chan<- events.EngineEvent, func()) {
	// Always create a real channel even when there's no consumer (no upload URL,
	// no JSON output). Returning nil makes Pulumi's auto SDK block forever on
	// a chan-send to a nil channel — the events.StreamEvents goroutine
	// deadlocks and the deploy never completes.
	eventsChannel := make(chan events.EngineEvent)
	done := make(chan struct{})
	go func() {
		// LIFO: close(done) runs last so waitEvents() only unblocks after
		// uploadEvents returns, even on panic in the loop.
		defer close(done)
		// The events are marshaled asap so we capture the original event data
		// before any of it gets mutated. This also allows the GC to collect
		// the original event objects sooner instead of keeping them alive.
		var engineEvents []json.RawMessage
		defer func() {
			uploadEvents(ctx, eventsUploadUrl, engineEvents)
		}()
		// Pulumi automation will close the channel when done: https://github.com/pulumi/pulumi/blob/master/sdk/go/auto/stack.go#L1956
		for evt := range eventsChannel {
			if eventsUploadUrl != "" {
				bytes, _ := json.Marshal(evt)
				engineEvents = append(engineEvents, json.RawMessage(bytes))
			}
			if jsonOutput {
				if evt.ResourcePreEvent != nil {
					data, _ := json.Marshal(evt.ResourcePreEvent.Metadata)
					Println(string(data)) // jsonl
				}
			}
		}
	}()
	return eventsChannel, func() { <-done }
}
