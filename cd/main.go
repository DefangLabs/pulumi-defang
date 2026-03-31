package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/DefangLabs/pulumi-defang/examples/cd/program"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/debug"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
)

var version = "development" // overwritten by -ldflags "-X main.version=..."

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required environment variable: %s", key)
	}
	return v
}

type cdConfig struct {
	Org         string
	Project     string
	Stack       string
	Prefix      string
	Provider    string
	ComposePath string
	DebugMode   bool
	ShowDiff    bool
	Color       string // "always", "never", "auto"
	Targets     []string
	PreviewOnly bool
}

func loadConfig() cdConfig {
	org := envOrDefault("DEFANG_ORG", "defang")

	provider := "aws"
	if os.Getenv("GCP_PROJECT") != "" {
		provider = "gcp"
	} else if os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
		provider = "azure"
	}

	color := "always"
	if os.Getenv("NO_COLOR") != "" {
		color = "never"
	}

	var targets []string
	if t := os.Getenv("DEFANG_PULUMI_TARGETS"); t != "" {
		targets = strings.Split(t, ",")
	}

	return cdConfig{
		Org:         org,
		Project:     envOrDefault("PROJECT", org),
		Stack:       mustEnv("STACK"),
		Prefix:      envOrDefault("DEFANG_PREFIX", "Defang"),
		Provider:    provider,
		ComposePath: envOrDefault("DEFANG_COMPOSE", "./compose.yaml"),
		DebugMode:   os.Getenv("DEFANG_PULUMI_DEBUG") != "",
		ShowDiff:    os.Getenv("DEFANG_PULUMI_DIFF") != "",
		Color:       color,
		Targets:     targets,
		PreviewOnly: os.Getenv("DEFANG_PREVIEW") != "",
	}
}

// projectConfig returns config for Pulumi.yaml (project-level settings).
func (c cdConfig) projectConfig() map[string]workspace.ProjectConfigType {
	return map[string]workspace.ProjectConfigType{
		"pulumi:autonaming": {
			Value: map[string]any{
				"pattern": c.Prefix + "-${project}-${stack}-${name}-${hex(7)}",
				"providers": map[string]any{
					"aws": map[string]any{
						"resources": map[string]any{
							"aws:lb/loadBalancer:LoadBalancer":        map[string]string{"pattern": "${project}-${stack}-${hex(4)}"},
							"aws:lb/targetGroup:TargetGroup":          map[string]string{"pattern": "${name}-${hex(4)}"},
							"aws:elasticache/subnetGroup:SubnetGroup": map[string]string{"pattern": "defang-${project}-${stack}-${name}-${hex(7)}"},
							"aws:ecr/repository:Repository":           map[string]string{"pattern": "defang-${project}-${stack}-${name}-${hex(7)}"},
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

// stackConfig returns config for Pulumi.<stack>.yaml (stack-level settings).
func (c cdConfig) stackConfig() auto.ConfigMap {
	cfg := auto.ConfigMap{
		// Defang program config
		"defang:compose":  auto.ConfigValue{Value: c.ComposePath},
		"defang:provider": auto.ConfigValue{Value: c.Provider},
	}

	// Cloud provider config read by the explicit providers in the program
	switch c.Provider {
	case "aws":
		region := envOrDefault("AWS_REGION", os.Getenv("REGION"))
		if region == "" {
			log.Fatal("missing required environment variable: AWS_REGION or REGION")
		}
		cfg["aws:region"] = auto.ConfigValue{Value: region}
		if v := os.Getenv("AWS_PROFILE"); v != "" {
			cfg["aws:profile"] = auto.ConfigValue{Value: v}
		}

	case "gcp":
		cfg["gcp:project"] = auto.ConfigValue{Value: mustEnv("GCP_PROJECT")}
		cfg["gcp:region"] = auto.ConfigValue{Value: mustEnv("REGION")}

	case "azure":
		cfg["azure-native:location"] = auto.ConfigValue{Value: mustEnv("AZURE_LOCATION")}
	}

	// Defang recipe config
	cfg["defang:org"] = auto.ConfigValue{Value: c.Org}
	cfg["defang:prefix"] = auto.ConfigValue{Value: c.Prefix}
	cfg["defang:deploymentMode"] = auto.ConfigValue{Value: envOrDefault("DEFANG_MODE", "development")}
	if v := os.Getenv("DOMAIN"); v != "" {
		cfg["defang:domain"] = auto.ConfigValue{Value: v}
	}
	if v := os.Getenv("PRIVATE_DOMAIN"); v != "" {
		cfg["defang:privateDomain"] = auto.ConfigValue{Value: v}
	}

	return cfg
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c := loadConfig()
	userAgent := "defang/" + version
	program.Version = version

	stack, err := auto.UpsertStackInlineSource(ctx, c.Stack, c.Project, program.Run)
	if err != nil {
		log.Fatalf("failed to create/select stack: %v", err)
	}

	// Set workspace env vars (USER ends up in Pulumi lock files for debugging)
	if etag := envOrDefault("DEFANG_ETAG", c.Org); etag != "" {
		// stack.Workspace().SetEnvVars(map[string]string{"USER": etag})  doesn't work on linux
	}

	// Set project-level config (autonaming, disable-default-providers)
	ps, err := stack.Workspace().ProjectSettings(ctx)
	if err != nil {
		log.Fatalf("failed to get project settings: %v", err)
	}
	ps.Config = c.projectConfig()
	if err := stack.Workspace().SaveProjectSettings(ctx, ps); err != nil {
		log.Fatalf("failed to save project settings: %v", err)
	}

	// Set stack-level config (provider settings, defang config)
	if err := stack.SetAllConfig(ctx, c.stackConfig()); err != nil {
		log.Fatalf("failed to set config: %v", err)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	// Common option builders per command type
	debugLog := debug.LoggingOptions{Debug: c.DebugMode}

	switch command {
	case "up", "deploy":
		upOpts := []optup.Option{
			optup.UserAgent(userAgent),
			optup.Color(c.Color),
			optup.ProgressStreams(os.Stderr),
			optup.TargetDependents(),
		}
		if c.DebugMode {
			upOpts = append(upOpts, optup.DebugLogging(debugLog))
		}
		if c.ShowDiff {
			upOpts = append(upOpts, optup.Diff())
		}
		upOpts = append(upOpts, optup.Target(c.Targets))
		result, err := stack.Up(ctx, upOpts...)
		if err != nil {
			log.Fatalf("failed to deploy: %v", err)
		}
		fmt.Printf("endpoints: %v\n", result.Outputs["endpoints"].Value)
		fmt.Printf("loadBalancerDns: %v\n", result.Outputs["loadBalancerDns"].Value)

	case "preview":
		previewOpts := []optpreview.Option{
			optpreview.UserAgent(userAgent),
			optpreview.Color(c.Color),
			optpreview.ProgressStreams(os.Stderr),
			optpreview.TargetDependents(),
		}
		if c.ShowDiff {
			previewOpts = append(previewOpts, optpreview.Diff())
		}
		if c.DebugMode {
			previewOpts = append(previewOpts, optpreview.DebugLogging(debugLog))
		}
		previewOpts = append(previewOpts, optpreview.Target(c.Targets))
		_, err := stack.Preview(ctx, previewOpts...)
		if err != nil {
			log.Fatalf("failed to preview: %v", err)
		}

	case "destroy":
		destroyOpts := []optdestroy.Option{
			optdestroy.UserAgent(userAgent),
			optdestroy.Color(c.Color),
			optdestroy.ProgressStreams(os.Stderr),
			optdestroy.ContinueOnError(),
			optdestroy.Remove(),
		}
		if c.DebugMode {
			destroyOpts = append(destroyOpts, optdestroy.DebugLogging(debugLog))
		}
		_, err := stack.Destroy(ctx, destroyOpts...)
		if err != nil {
			log.Fatalf("failed to destroy: %v", err)
		}

	case "down":
		// down = refresh + destroy (consistent with legacy behavior)
		refreshOpts := []optrefresh.Option{
			optrefresh.UserAgent(userAgent),
			optrefresh.Color(c.Color),
			optrefresh.ProgressStreams(os.Stderr),
		}
		if c.DebugMode {
			refreshOpts = append(refreshOpts, optrefresh.DebugLogging(debugLog))
		}
		_, err := stack.Refresh(ctx, refreshOpts...)
		if err != nil {
			log.Fatalf("failed to refresh: %v", err)
		}

		destroyOpts := []optdestroy.Option{
			optdestroy.UserAgent(userAgent),
			optdestroy.Color(c.Color),
			optdestroy.ProgressStreams(os.Stderr),
			optdestroy.ContinueOnError(),
			optdestroy.Remove(),
		}
		if c.DebugMode {
			destroyOpts = append(destroyOpts, optdestroy.DebugLogging(debugLog))
		}
		_, err = stack.Destroy(ctx, destroyOpts...)
		if err != nil {
			log.Fatalf("failed to destroy: %v", err)
		}

	case "refresh":
		refreshOpts := []optrefresh.Option{
			optrefresh.UserAgent(userAgent),
			optrefresh.Color(c.Color),
			optrefresh.ProgressStreams(os.Stderr),
		}
		if c.DebugMode {
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
