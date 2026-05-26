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
	prefix := getenv("DEFANG_PREFIX", "Defang")
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
		if len(projectUpdate.PulumiConfig) != 0 {
			err = stack.SetAllConfigJson(ctx, string(projectUpdate.PulumiConfig), nil)
		} else {
			cfg, err := stackConfigFromEnv()
			if err != nil {
				return err
			}
			err = stack.SetAllConfig(ctx, cfg)
		}
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

	// Set project-level config (autonaming, disable-default-providers)
	ps, err := stack.Workspace().ProjectSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get project settings: %w", err)
	}
	ps.Config = projectConfig(prefix)
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
							// Requirements for Container App Environment resource names:
							// Between 2 and 60 characters long.
							// This resource name is not case-sensitive even though it is written as lowercase only in the docs.
							// Numbers and hyphens are also allowed.
							// https://azure.github.io/PSRule.Rules.Azure/en/rules/Azure.ContainerApp.EnvNaming
							"azure-native:app:ManagedEnvironment": map[string]string{"pattern": prefix + "${project}-${stack}-${hex(7)}"},
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

// stackConfigFromEnv returns config for Pulumi.<stack>.yaml (stack-level settings).
func stackConfigFromEnv() (auto.ConfigMap, error) {
	region := os.Getenv("REGION")
	awsProfile := os.Getenv("AWS_PROFILE")                    // AWS only
	awsRegion := getenv("AWS_REGION", region)                 // AWS only
	azureLocation := getenv("AZURE_LOCATION", region)         // Azure only
	azureSubscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID") // Azure only; the project RG and Key Vault names are derived from (project, stack, location) and (subscription, RG) respectively — see provider/defangazure/azure/azure.go
	cdImage := os.Getenv("DEFANG_CD_IMAGE")                   // GCP only; for cleanup
	delegationSetId := os.Getenv("DELEGATION_SET_ID")         // AWS only
	domain := os.Getenv("DOMAIN")
	org := getenv("DEFANG_ORG", "defang")
	etag := getenv("DEFANG_ETAG", org)
	gcpProject := getenv("GCLOUD_PROJECT", os.Getenv("GCP_PROJECT")) // GCP only; keep GCP_PROJECT for old CLI compat
	gcpRegion := getenv("GCLOUD_REGION", region)                     // GCP only
	privateDomain := os.Getenv("PRIVATE_DOMAIN")                     // AWS only
	registryCredsArn := os.Getenv("CI_REGISTRY_CREDENTIALS_ARN")     // AWS only
	stateUrl := getenv("DEFANG_STATE_URL", os.Getenv("PULUMI_BACKEND_URL"))

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

	if gcpProject != "" {
		providers = append(providers, "gcp")
		cfg["gcp:project"] = auto.ConfigValue{Value: gcpProject}
		if gcpRegion != "" {
			cfg["gcp:region"] = auto.ConfigValue{Value: gcpRegion}
		}
		// TODO: configure label-logger
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
		return nil, &usageError{msg: "no cloud provider configured: set AWS_REGION, GCLOUD_PROJECT, or AZURE_SUBSCRIPTION_ID environment variable"}
	} else if len(providers) > 1 {
		return nil, &usageError{msg: fmt.Sprintf("conflicting cloud providers configured: %v", providers)}
	} else {
		cfg["defang:provider"] = auto.ConfigValue{Value: providers[0]}
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
	return cfg, nil
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
