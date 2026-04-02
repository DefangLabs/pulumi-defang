package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
	"github.com/DefangLabs/pulumi-defang/examples/cd/program"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/debug"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"google.golang.org/protobuf/encoding/protowire"
)

var version = "development" // overwritten by -ldflags "-X main.version=..."

func provider() string {
	switch {
	case awsRegion != "":
		return "aws"
	case gcpProject != "":
		return "gcp"
	case azureSubscription != "":
		return "azure"
	}
	log.Fatal("missing required environment variable: must set one of: AWS_REGION, GCP_PROJECT, or AZURE_SUBSCRIPTION")
	return ""
}

func color() string {
	if noColor {
		return "never"
	}
	return "always"
}

// projectConfig returns config for Pulumi.yaml (project-level settings).
func projectConfig() map[string]workspace.ProjectConfigType {
	prefix := prefix
	if prefix != "" {
		prefix += "-"
	}
	lowerPrefix := strings.ToLower(prefix)
	return map[string]workspace.ProjectConfigType{
		"pulumi:autonaming": {
			Value: map[string]any{
				"pattern": prefix + "${project}-${stack}-${name}-${hex(7)}",
				"providers": map[string]any{
					"aws": map[string]any{
						"resources": map[string]any{
							"aws:lb/loadBalancer:LoadBalancer":        map[string]string{"pattern": "${project}-${stack}-${hex(4)}"},
							"aws:lb/targetGroup:TargetGroup":          map[string]string{"pattern": "${name}-${hex(4)}"},
							"aws:elasticache/subnetGroup:SubnetGroup": map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"},
							"aws:ecr/repository:Repository":           map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"},
						},
					},
				},
			},
		},
		"pulumi:disable-default-providers": {
			Value: []string{"eks", "kubernetes", "aws"},
		},
	}
}

// fetchPayload retrieves the ProjectUpdate protobuf from s3://, gs://, https://, base64, or a local file.
func fetchPayload(ctx context.Context, uri string) ([]byte, error) {
	switch {
	case strings.HasPrefix(uri, "s3://"):
		return fetchS3(ctx, uri)
	case strings.HasPrefix(uri, "gs://"):
		return fetchGCS(ctx, uri)
	case strings.HasPrefix(uri, "http://"), strings.HasPrefix(uri, "https://"):
		return fetchHTTP(ctx, uri)
	default:
		return base64.StdEncoding.DecodeString(uri)
	}
}

func fetchS3(ctx context.Context, uri string) ([]byte, error) {
	parts := strings.SplitN(strings.TrimPrefix(uri, "s3://"), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid S3 URI: %v", uri)
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s3.NewFromConfig(cfg).GetObject(ctx, &s3.GetObjectInput{
		Bucket: &parts[0],
		Key:    &parts[1],
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func fetchGCS(ctx context.Context, uri string) ([]byte, error) {
	parts := strings.SplitN(strings.TrimPrefix(uri, "gs://"), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GCS URI: %v", uri)
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	rc, err := client.Bucket(parts[0]).Object(parts[1]).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func fetchHTTP(ctx context.Context, uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("GET %s returned %s", uri, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// extractComposeYaml extracts the compose bytes (field 4) from a ProjectUpdate protobuf
// without importing the full defang CLI proto package.
func extractComposeYaml(projectUpdate []byte) ([]byte, error) {
	for len(projectUpdate) > 0 {
		num, typ, n := protowire.ConsumeTag(projectUpdate)
		if n < 0 {
			return nil, errors.New("invalid protobuf tag")
		}
		projectUpdate = projectUpdate[n:]
		switch typ {
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(projectUpdate)
			if n < 0 {
				return nil, errors.New("invalid protobuf bytes field")
			}
			if num == 4 {
				return v, nil
			}
			projectUpdate = projectUpdate[n:]
		case protowire.VarintType:
			_, n := protowire.ConsumeVarint(projectUpdate)
			if n < 0 {
				return nil, errors.New("invalid protobuf varint field")
			}
			projectUpdate = projectUpdate[n:]
		default:
			return nil, fmt.Errorf("unexpected protobuf wire type %d", typ)
		}
	}
	return nil, errors.New("ProjectUpdate has no compose field")
}

// stackConfig returns config for Pulumi.<stack>.yaml (stack-level settings).
func stackConfig() auto.ConfigMap {
	cfg := auto.ConfigMap{
		// Defang program config
		"defang:provider": auto.ConfigValue{Value: provider()},
	}

	// Cloud provider config read by the explicit providers in the program
	switch provider() {
	case "aws":
		if awsRegion == "" {
			log.Fatal("missing required environment variable: AWS_REGION or REGION")
		}
		cfg["aws:region"] = auto.ConfigValue{Value: awsRegion}
		if awsProfile != "" {
			cfg["aws:profile"] = auto.ConfigValue{Value: awsProfile}
		}

	case "gcp":
		if gcpProject == "" {
			log.Fatal("missing required environment variable: GCP_PROJECT")
		}
		cfg["gcp:project"] = auto.ConfigValue{Value: gcpProject}
		if region == "" {
			log.Fatal("missing required environment variable: REGION")
		}
		cfg["gcp:region"] = auto.ConfigValue{Value: region}

	case "azure":
		if azureLocation == "" {
			log.Fatal("missing required environment variable: AZURE_LOCATION")
		}
		cfg["azure-native:location"] = auto.ConfigValue{Value: azureLocation}
	}

	// Defang recipe config
	cfg["defang:org"] = auto.ConfigValue{Value: org}
	cfg["defang:prefix"] = auto.ConfigValue{Value: prefix}
	cfg["defang:deploymentMode"] = auto.ConfigValue{Value: mode}
	if domain != "" {
		cfg["defang:domain"] = auto.ConfigValue{Value: domain}
	}
	if privateDomain != "" {
		cfg["defang:privateDomain"] = auto.ConfigValue{Value: privateDomain}
	}
	if delegationSetId != "" {
		cfg["defang:delegationSetId"] = auto.ConfigValue{Value: delegationSetId}
	}
	if registryCredsArn != "" {
		cfg["defang:ciRegistryCredentialsArn"] = auto.ConfigValue{Value: registryCredsArn}
	}

	switch mode {
	case "development":
	}

	return cfg
}

func upload(ctx context.Context, url string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal upload payload: %v", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		log.Printf("failed to create upload request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to upload to %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("upload to %s failed: status=%d body=%q", url, resp.StatusCode, string(body))
	}
}

func uploadEvents(ctx context.Context, evts []events.EngineEvent) {
	if eventsUploadUrl == "" || len(evts) == 0 {
		return
	}
	log.Printf("Sending %d deployment events to Portal...", len(evts))
	upload(ctx, eventsUploadUrl, map[string]any{"events": evts})
}

func uploadState(ctx context.Context, s auto.Stack) {
	if statesUploadUrl == "" {
		return
	}
	log.Print("Sending deployment state to Portal...")
	state, err := s.Export(ctx)
	if err != nil {
		log.Printf("failed to export stack state: %v", err)
		return
	}
	upload(ctx, statesUploadUrl, state)
}

func collectEvents() (chan events.EngineEvent, *[]events.EngineEvent) {
	ch := make(chan events.EngineEvent)
	var collected []events.EngineEvent
	go func() {
		for evt := range ch {
			if jsonOutput {
				if evt.ResourcePreEvent != nil {
					data, _ := json.Marshal(evt.ResourcePreEvent.Metadata)
					fmt.Println(string(data))
				}
			}
			collected = append(collected, evt)
		}
	}()
	return ch, &collected
}

func main() {
	if stack == "" {
		log.Fatal("missing required environment variable: STACK")
	}
	if stack != strings.ToLower(stack) {
		log.Fatal("STACK name must be lowercase")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Hour)
	defer cancel()

	userAgent := "defang/" + version
	program.Version = version

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var s auto.Stack
	var err error
	switch command {
	case "up", "deploy", "preview":
		// Payload URL from args (like old code): cd <command> <payload>
		if len(os.Args) <= 2 {
			log.Fatalf("missing required argument: payload")
		}
		payload := os.Args[2]
		// Fetch protobuf and extract compose YAML
		var composeYaml []byte
		if payload != "" {
			projectUpdate, err := fetchPayload(ctx, payload)
			if err != nil {
				log.Fatalf("failed to fetch payload: %v", err)
			}
			composeYaml, err = extractComposeYaml(projectUpdate)
			if err != nil {
				log.Fatalf("failed to extract compose: %v", err)
			}
		}
		s, err = auto.UpsertStackInlineSource(ctx, stack, project, program.NewRun(composeYaml))
	default:
		s, err = auto.SelectStackInlineSource(ctx, stack, project, nil)
	}
	if err != nil {
		log.Fatalf("failed to create/select stack: %v", err)
	}
	stack := s

	// Set workspace env vars
	if etag != "" {
		// USER ends up in Pulumi lock files for debugging; FIXME: doens't work on linux
		stack.Workspace().SetEnvVar("USER", etag)
	}

	// Set project-level config (autonaming, disable-default-providers)
	ps, err := stack.Workspace().ProjectSettings(ctx)
	if err != nil {
		log.Fatalf("failed to get project settings: %v", err)
	}
	ps.Config = projectConfig()
	if err := stack.Workspace().SaveProjectSettings(ctx, ps); err != nil {
		log.Fatalf("failed to save project settings: %v", err)
	}

	// Set stack-level config (provider settings, defang config)
	if err := stack.SetAllConfig(ctx, stackConfig()); err != nil {
		log.Fatalf("failed to set config: %v", err)
	}

	// Common option builders per command type
	debugLog := debug.LoggingOptions{Debug: pulumiDebug, LogToStdErr: true}

	switch command {
	case "up", "deploy":
		evtCh, evts := collectEvents()
		upOpts := []optup.Option{
			optup.UserAgent(userAgent),
			optup.Color(color()),
			optup.ProgressStreams(os.Stderr),
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
		uploadEvents(ctx, *evts)
		uploadState(ctx, stack)
		if err != nil {
			log.Fatalf("failed to deploy: %v", err)
		}

	case "preview":
		evtCh, evts := collectEvents()
		previewOpts := []optpreview.Option{
			optpreview.UserAgent(userAgent),
			optpreview.Color(color()),
			optpreview.ProgressStreams(os.Stderr),
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
		uploadEvents(ctx, *evts)
		if err != nil {
			log.Fatalf("failed to preview: %v", err)
		}

	case "destroy":
		evtCh, evts := collectEvents()
		destroyOpts := []optdestroy.Option{
			optdestroy.UserAgent(userAgent),
			optdestroy.Color(color()),
			optdestroy.ProgressStreams(os.Stderr),
			optdestroy.EventStreams(evtCh),
			optdestroy.ContinueOnError(),
			optdestroy.Remove(),
		}
		if pulumiDebug {
			destroyOpts = append(destroyOpts, optdestroy.DebugLogging(debugLog))
		}
		_, err := stack.Destroy(ctx, destroyOpts...)
		uploadEvents(ctx, *evts)
		uploadState(ctx, stack)
		if err != nil {
			log.Fatalf("failed to destroy: %v", err)
		}

	case "down":
		// down = refresh + destroy (consistent with legacy behavior)
		refreshOpts := []optrefresh.Option{
			optrefresh.UserAgent(userAgent),
			optrefresh.Color(color()),
			optrefresh.ProgressStreams(os.Stderr),
		}
		if pulumiDebug {
			refreshOpts = append(refreshOpts, optrefresh.DebugLogging(debugLog))
		}
		_, err := stack.Refresh(ctx, refreshOpts...)
		if err != nil {
			log.Fatalf("failed to refresh: %v", err)
		}

		evtCh, evts := collectEvents()
		destroyOpts := []optdestroy.Option{
			optdestroy.UserAgent(userAgent),
			optdestroy.Color(color()),
			optdestroy.ProgressStreams(os.Stderr),
			optdestroy.EventStreams(evtCh),
			optdestroy.ContinueOnError(),
			optdestroy.Remove(),
		}
		if pulumiDebug {
			destroyOpts = append(destroyOpts, optdestroy.DebugLogging(debugLog))
		}
		_, err = stack.Destroy(ctx, destroyOpts...)
		uploadEvents(ctx, *evts)
		uploadState(ctx, stack)
		if err != nil {
			log.Fatalf("failed to destroy: %v", err)
		}

	case "refresh":
		refreshOpts := []optrefresh.Option{
			optrefresh.UserAgent(userAgent),
			optrefresh.Color(color()),
			optrefresh.ProgressStreams(os.Stderr),
		}
		if pulumiDebug {
			refreshOpts = append(refreshOpts, optrefresh.DebugLogging(debugLog))
		}
		_, err := stack.Refresh(ctx, refreshOpts...)
		if err != nil {
			log.Fatalf("failed to refresh: %v", err)
		}

	case "cancel":
		if err := stack.Cancel(ctx); err != nil {
			log.Fatalf("failed to cancel: %v", err)
		}

	case "outputs":
		outputs, err := stack.Outputs(ctx)
		if err != nil {
			log.Fatalf("failed to get outputs: %v", err)
		}
		data, _ := json.MarshalIndent(outputs, "", "  ")
		fmt.Println(string(data))

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "usage: cd [up|preview|destroy|down|refresh|cancel|outputs]")
		os.Exit(1)
	}
}
