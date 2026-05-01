package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/examples/cd/program"
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

var version = "development" // overwritten by -ldflags "-X main.version=..."

func color() string {
	if noColor {
		return "never"
	}
	return "always"
}

// projectConfig returns config for Pulumi.yaml (project-level settings).
func projectConfig(prefix string) map[string]workspace.ProjectConfigType {
	if prefix != "" {
		prefix += "-"
	}
	lowerPrefix := strings.ToLower(prefix)
	// TODO: we'll need a lowercase version of the project name as well
	return map[string]workspace.ProjectConfigType{
		"pulumi:autonaming": {
			Value: map[string]any{
				"pattern": prefix + "${project}-${stack}-${name}-${hex(7)}",
				"providers": map[string]any{
					"aws": map[string]any{
						"resources": map[string]any{
							"aws:lb/loadBalancer:LoadBalancer": map[string]string{"pattern": "${project}-${stack}-${hex(4)}"},
							"aws:lb/targetGroup:TargetGroup":   map[string]string{"pattern": "${name}-${hex(4)}"},
							// ecs.Service is always scoped to an ecs.Cluster, so the cluster's
							// full prefix already disambiguates it; no need to repeat it here.
							"aws:ecs/service:Service":                 map[string]string{"pattern": "${name}-${hex(7)}"},
							"aws:elasticache/subnetGroup:SubnetGroup": map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"}, // lowercase
							"aws:ecr/repository:Repository":           map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"}, // lowercase
							"aws:rds/subnetGroup:SubnetGroup":         map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"}, // lowercase
						},
					},
					"azure-native": map[string]any{
						"resources": map[string]any{
							// ACR registry names must be alphanumeric only (^[a-zA-Z0-9]*$, 5–50 chars).
							// The default pattern includes hyphens from project/stack names, so override it.
							// ${name} is already sanitized to alphanumeric by sanitizeRegistryName() in image.go.
							// ${stack} is safe to include: stacks are lowercase with no hyphens.
							"azure-native:containerregistry:Registry": map[string]string{"pattern": "${name}${stack}${hex(7)}"}, // name = sanitized project name
							// https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules#microsoftcontainerregistry
							// 5-50	Alphanumerics, hyphens, and underscores
							"azure-native:containerregistry:Task": map[string]string{"pattern": "${name}-${hex(7)}"},
						},
					},
					// Most GCP resources require names matching ^[a-z][-a-z0-9]{0,61}[a-z0-9]$
					// (lowercase only, max 63 chars). The default prefix may contain capitals
					// (e.g. "Defang-"), so force the entire pattern to use the lowercased prefix.
					"gcp": map[string]any{
						"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}", // TODO: sanitize project name
						"resources": map[string]any{
							// Service account ID must be between 6 and 30 characters.
							// Service account ID must start with a lower case letter, followed by one or more lower case alphanumerical characters that can be separated by hyphens.
							"gcp:serviceaccount/account:Account": map[string]string{"pattern": "${name}-${hex(4)}"},
							// Cloud Run service name max 49 chars (^[a-z][a-z0-9-]{0,47}[a-z0-9]$).
							// Default prefix-project-stack pattern overflows on longer inputs;
							// drop the prefix to mirror old cloudrunServiceName (49 char budget).
							"gcp:cloudrunv2/service:Service": map[string]string{"pattern": "${project}-${stack}-${name}-${hex(7)}"}, // TODO: sanitize project name
							// Memorystore Redis instance ID max 40 chars (^[a-z][a-z0-9-]{0,38}[a-z0-9]$).
							// Drop the prefix to mirror old redisInstanceName (40 char budget).
							"gcp:redis/instance:Instance": map[string]string{"pattern": "${project}-${name}-${hex(7)}"}, // TODO: sanitize project name
						},
					},
				},
			},
		},
		"pulumi:disable-default-providers": {
			// Ensure we create one provider per cloud by disabling the automatic default providers
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
}

// stackConfig returns config for Pulumi.<stack>.yaml (stack-level settings).
func stackConfig() (auto.ConfigMap, error) {
	cfg := auto.ConfigMap{
		// Defang program config
		"defang:cdImage":  auto.ConfigValue{Value: cdImage},
		"defang:etag":     auto.ConfigValue{Value: etag}, // deployment ID; recorded in state, surfaced in tags/env
		"defang:org":      auto.ConfigValue{Value: org},
		"defang:stateUrl": auto.ConfigValue{Value: stateUrl},
		"defang:version":  auto.ConfigValue{Value: version},
	}

	// Cloud provider config read by the explicit providers in the program
	var providers []string
	if awsRegion != "" {
		providers = append(providers, "aws")
		cfg["aws:region"] = auto.ConfigValue{Value: awsRegion}
		if awsProfile != "" {
			cfg["aws:profile"] = auto.ConfigValue{Value: awsProfile}
		}
	}

	if gcpProjectId != "" {
		providers = append(providers, "gcp")
		cfg["gcp:project"] = auto.ConfigValue{Value: gcpProjectId}
		if gcpRegion != "" {
			cfg["gcp:region"] = auto.ConfigValue{Value: gcpRegion}
		}
	}

	if azureSubscriptionId != "" {
		providers = append(providers, "azure")
		cfg["azure-native:subscriptionId"] = auto.ConfigValue{Value: azureSubscriptionId}
		if azureLocation != "" {
			cfg["azure-native:location"] = auto.ConfigValue{Value: azureLocation}
		}
		cfg["azure-native:useMsi"] = auto.ConfigValue{Value: "true"}
		// The project RG name and Key Vault name are derived deterministically
		// from (project, stack, location) and (subscription, RG) respectively
		// inside the provider — matching the CLI's conventions. No need to
		// pass them through as stack config or env vars.
	}

	if len(providers) == 0 {
		return nil, &usageError{msg: "no cloud provider configured: set AWS_REGION, GCP_PROJECT_ID, or AZURE_SUBSCRIPTION_ID environment variable"}
	} else if len(providers) > 1 {
		return nil, &usageError{msg: fmt.Sprintf("conflicting cloud providers configured: %v", providers)}
	}

	// Defang recipe config
	if domain != "" {
		cfg["defang:domain"] = auto.ConfigValue{Value: domain}
	}
	if privateDomain != "" {
		cfg["defang:privateDomain"] = auto.ConfigValue{Value: privateDomain}
	}
	if delegationSetId != "" {
		// FIXME: should use defang-aws namespace
		cfg["defang:delegationSetId"] = auto.ConfigValue{Value: delegationSetId}
	}
	if registryCredsArn != "" {
		// FIXME: should use defang-aws namespace
		cfg["defang:ciRegistryCredentialsArn"] = auto.ConfigValue{Value: registryCredsArn}
	}
	// TODO: set recipe values based on deployment mode from `mode` var
	return cfg, nil
}

// collectEvents returns a channel to feed engine events into and a wait
// function that blocks until the channel is closed and the collector
// goroutine has drained it, then returns the events. The wait-function
// pattern provides a happens-before edge between the goroutine's appends
// and the caller's read, avoiding a data race on the slice.
func collectEvents(ctx context.Context) chan<- events.EngineEvent {
	if eventsUploadUrl == "" && !jsonOutput {
		return nil
	}
	eventsChannel := make(chan events.EngineEvent)
	go func() {
		var engineEvents []events.EngineEvent
		defer func() {
			uploadEvents(ctx, engineEvents)
		}()
		for evt := range eventsChannel {
			engineEvents = append(engineEvents, evt)
			if jsonOutput {
				if evt.ResourcePreEvent != nil {
					data, _ := json.Marshal(evt.ResourcePreEvent.Metadata)
					Println(string(data)) // jsonl
				}
			}
		}
	}()
	return eventsChannel
}

func main() {
	// All cleanup (signal.Stop, ctx cancels) lives in run() so its defers
	// actually fire — os.Exit skips deferred calls in the function it's in.
	os.Exit(run())
}

func run() int {
	// --version is informational; skip signal/timeout setup so it stays cheap.
	if len(os.Args) > 1 && os.Args[1] == "--version" {
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

	if err := cdMain(ctx); err != nil {
		warn(err.Error())

		var usageErr *usageError
		if errors.As(err, &usageErr) {
			return 2
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

func cdMain(ctx context.Context) error {
	// Wrap stdout/stderr so every log line emitted by the Pulumi engine, the
	// standard log package, and any library writing to the global file handles
	// is prefixed with the etag. Lets the CLI filter ContainerAppConsoleLogs_CL
	// by KQL `Log_s has "<etag>"`. Must run BEFORE any other write.
	flushEtag := installEtagPrefix(etag)
	defer flushEtag()

	if stackName == "" {
		return &usageError{msg: "missing required environment variable: STACK"}
	}
	if stackName != strings.ToLower(stackName) {
		return &usageError{msg: "STACK name must be lowercase"}
	}

	var command string
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var stack auto.Stack
	var err error
	switch command {
	case "", "help":
		return &usageError{msg: "usage: cd [up <payload>|preview <payload>|destroy|down|refresh|cancel|outputs]"}
	case "up", "deploy", "preview":
		// Payload URL from args (like old code): cd <command> <payload>
		if len(os.Args) <= 2 {
			return &usageError{msg: "missing required argument: payload"}
		}
		payload := os.Args[2]
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
		}
		stack, err = auto.UpsertStackInlineSource(ctx, stackName, projectName, program.NewRun(projectUpdate))
	default:
		stack, err = auto.SelectStackInlineSource(ctx, stackName, projectName, nil)
	}
	if err != nil {
		return pulumiErr(err)
	}

	// Set workspace env vars
	if etag != "" {
		// USER ends up in Pulumi lock files for debugging; FIXME: doesn't work on linux
		stack.Workspace().SetEnvVar("USER", etag)
	}

	// Set project-level config (autonaming, disable-default-providers)
	ps, err := stack.Workspace().ProjectSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get project settings: %w", err)
	}
	ps.Config = projectConfig(prefix)
	if err := stack.Workspace().SaveProjectSettings(ctx, ps); err != nil {
		return fmt.Errorf("failed to save project settings: %w", err)
	}

	// Set stack-level config (provider settings, defang config)
	cfg, err := stackConfig()
	if err != nil {
		return err
	}
	if err := stack.SetAllConfig(ctx, cfg); err != nil {
		return pulumiErr(err)
	}

	// Common option builders per command type
	userAgent := "defang/" + version
	debugLog := debug.LoggingOptions{Debug: pulumiDebug, LogToStdErr: true}
	progressStream := newProgressStream()
	defer progressStream.Flush()
	errorProgressStream := newErrorProgressStream()
	defer errorProgressStream.Flush()

	switch command {
	case "up", "deploy":
		evtCh := collectEvents(ctx)
		upOpts := []optup.Option{
			optup.UserAgent(userAgent),
			optup.Color(color()),
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
		uploadState(ctx, stack)
		if err != nil {
			// TODO: run a refresh on failure to update the state with any partial changes, like the CLI does
			return pulumiErr(err)
		}

	case "preview":
		evtCh := collectEvents(ctx)
		previewOpts := []optpreview.Option{
			optpreview.UserAgent(userAgent),
			optpreview.Color(color()),
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

	case "down", "destroy":
		evtCh := collectEvents(ctx)
		destroyOpts := []optdestroy.Option{
			optdestroy.UserAgent(userAgent),
			optdestroy.Color(color()),
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
		uploadState(ctx, stack)
		if err != nil {
			return pulumiErr(err)
		}

	case "refresh":
		evtCh := collectEvents(ctx)
		refreshOpts := []optrefresh.Option{
			optrefresh.UserAgent(userAgent),
			optrefresh.Color(color()),
			optrefresh.ProgressStreams(progressStream),
			optrefresh.ErrorProgressStreams(errorProgressStream),
			optrefresh.EventStreams(evtCh),
		}
		if pulumiDebug {
			refreshOpts = append(refreshOpts, optrefresh.DebugLogging(debugLog))
		}
		_, err := stack.Refresh(ctx, refreshOpts...)
		uploadState(ctx, stack)
		if err != nil {
			return pulumiErr(err)
		}

	case "cancel":
		if err := stack.Cancel(ctx); err != nil {
			return pulumiErr(err)
		}

	case "outputs":
		outputs, err := stack.Outputs(ctx)
		if err != nil {
			return pulumiErr(err)
		}
		data, _ := json.MarshalIndent(outputs, "", "  ")
		Println(string(data))

	default:
		return &usageError{msg: fmt.Sprintf("unknown command: %s", command)}
	}
	return nil
}
