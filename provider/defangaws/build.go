package defangaws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	cbtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

var (
	ErrNoBuildID        = errors.New("failed to start build: no build ID returned")
	ErrBuildTimedOut    = errors.New("build timed out")
	ErrBuildNotFound    = errors.New("build not found")
	ErrBuildFailed      = errors.New("build failed")
	ErrBuildFaulted     = errors.New("build faulted")
	ErrBuildStopped     = errors.New("build was stopped (ABORTED)")
	ErrCodeBuildTimeout = errors.New("build timed out on CodeBuild side")
)

// Build is a custom resource that triggers a CodeBuild build and waits for completion.
type Build struct{}

// BuildInputs are the inputs for the CodeBuild build resource.
type BuildInputs struct {
	// CodeBuild project name
	ProjectName string `pulumi:"projectName"`

	// AWS region
	Region string `pulumi:"region,optional"`

	// AWS profile
	Profile string `pulumi:"profile,optional"`

	// Destination image URL (e.g. "123456789.dkr.ecr.us-east-1.amazonaws.com/repo:tag")
	Destination string `pulumi:"destination,optional"`

	// Max wait time in seconds (default: 1200 = 20 minutes)
	MaxWaitTime *int `pulumi:"maxWaitTime,optional"`

	// Trigger replacements when these change (serialized to force replacement)
	Triggers []string `pulumi:"triggers,optional"`
}

// BuildState is the output state of the CodeBuild build resource.
type BuildState struct {
	BuildInputs

	// The CodeBuild build ID
	BuildID string `pulumi:"buildId"`

	// The built image URL (empty for non-image builds)
	Image string `pulumi:"image"`
}

// Create starts a CodeBuild build, waits for it to complete, and returns the image URL.
func (*Build) Create(
	ctx context.Context, req infer.CreateRequest[BuildInputs],
) (infer.CreateResponse[BuildState], error) {
	inputs := req.Inputs

	if req.DryRun {
		return infer.CreateResponse[BuildState]{
			ID: inputs.ProjectName,
			Output: BuildState{
				BuildInputs: inputs,
			},
		}, nil
	}

	maxWait := 1200
	if inputs.MaxWaitTime != nil {
		maxWait = *inputs.MaxWaitTime
	}

	// Initial wait for IAM role to sync
	time.Sleep(3 * time.Second)
	const waitDur = 5 * time.Second

	var buildID string
	var err error
	for attempt := range 2 {
		buildID, err = runCodeBuildBuild(ctx, inputs.ProjectName, inputs.Region, inputs.Profile, maxWait)
		if err == nil {
			break // success
		}
		if attempt == 1 || !isRetryable(err) {
			return infer.CreateResponse[BuildState]{}, fmt.Errorf("CodeBuild build failed: %w", err)
		}
		time.Sleep(waitDur)
	}

	image := inputs.Destination

	return infer.CreateResponse[BuildState]{
		ID: inputs.ProjectName,
		Output: BuildState{
			BuildInputs: inputs,
			BuildID:     buildID,
			Image:       image,
		},
	}, nil
}

func isRetryable(err error) bool {
	msg := err.Error()
	for _, s := range []string{"ABORTED", "Error while executing command: docker buildx build"} {
		if strings.Contains(msg, s) {
			return false
		}
	}
	return true
}

func runCodeBuildBuild(ctx context.Context, projectName, region, profile string, maxWaitSeconds int) (string, error) {
	opts := []func(*config.LoadOptions) error{}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("loading AWS config: %w", err)
	}

	client := codebuild.NewFromConfig(cfg)

	startOut, err := client.StartBuild(ctx, &codebuild.StartBuildInput{
		ProjectName: &projectName,
	})
	if err != nil {
		return "", fmt.Errorf("starting build: %w", err)
	}
	if startOut.Build == nil || startOut.Build.Id == nil {
		return "", ErrNoBuildID
	}
	buildID := *startOut.Build.Id

	deadline := time.Now().Add(time.Duration(maxWaitSeconds) * time.Second)
	pollInterval := 2 * time.Second

	for {
		if time.Now().After(deadline) {
			return buildID, fmt.Errorf("build %s timed out after %ds: %w", buildID, maxWaitSeconds, ErrBuildTimedOut)
		}

		time.Sleep(pollInterval)
		if pollInterval < 5*time.Second {
			pollInterval = 5 * time.Second
		}

		batchOut, err := client.BatchGetBuilds(ctx, &codebuild.BatchGetBuildsInput{
			Ids: []string{buildID},
		})
		if err != nil {
			return buildID, fmt.Errorf("polling build status: %w", err)
		}
		if len(batchOut.Builds) == 0 {
			return buildID, fmt.Errorf("build %s: %w", buildID, ErrBuildNotFound)
		}

		build := batchOut.Builds[0] // assume only one build per request
		switch build.BuildStatus {
		case cbtypes.StatusTypeSucceeded:
			return buildID, nil
		case cbtypes.StatusTypeInProgress:
			continue
		case cbtypes.StatusTypeFailed:
			return buildID, fmt.Errorf(`{"state":"%w","reason":%q}`, ErrBuildFailed, getBuildPhaseErrorContexts(build))
		case cbtypes.StatusTypeFault:
			return buildID, fmt.Errorf(`{"state":"%w","reason":%q}`, ErrBuildFaulted, getBuildPhaseErrorContexts(build))
		case cbtypes.StatusTypeStopped:
			return buildID, ErrBuildStopped
		case cbtypes.StatusTypeTimedOut:
			return buildID, ErrCodeBuildTimeout
		default:
			continue
		}
	}
}

func getBuildPhaseErrorContexts(build cbtypes.Build) string {
	var msgs []string
	for _, phase := range build.Phases {
		for _, c := range phase.Contexts {
			if c.Message != nil && len(*c.Message) > 0 {
				msgs = append(msgs, *c.Message)
			}
		}
	}
	return strings.Join(msgs, "\n")
}
