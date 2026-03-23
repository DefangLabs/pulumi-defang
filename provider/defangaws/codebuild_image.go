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
	ErrBuildStopped     = errors.New("build was stopped (ABORTED)")
	ErrCodeBuildTimeout = errors.New("build timed out on CodeBuild side")
)

// CodeBuildImageBuild is a custom resource that triggers a CodeBuild build and waits for completion.
type CodeBuildImageBuild struct{}

// CodeBuildImageBuildInputs are the inputs for the CodeBuild build resource.
type CodeBuildImageBuildInputs struct {
	// CodeBuild project name
	ProjectName string `pulumi:"projectName"`

	// AWS region
	Region string `pulumi:"region,optional"`

	// Destination image URL (e.g. "123456789.dkr.ecr.us-east-1.amazonaws.com/repo:tag")
	Destination string `pulumi:"destination,optional"`

	// Max wait time in seconds (default: 1200 = 20 minutes)
	MaxWaitTime *int `pulumi:"maxWaitTime,optional"`

	// Trigger replacements when these change (serialized to force replacement)
	Triggers []string `pulumi:"triggers,optional"`
}

// CodeBuildImageBuildState is the output state of the CodeBuild build resource.
type CodeBuildImageBuildState struct {
	CodeBuildImageBuildInputs

	// The CodeBuild build ID
	BuildID string `pulumi:"buildId"`

	// The built image URL (empty for non-image builds)
	Image string `pulumi:"image"`
}

// Create starts a CodeBuild build, waits for it to complete, and returns the image URL.
func (*CodeBuildImageBuild) Create(
	ctx context.Context, req infer.CreateRequest[CodeBuildImageBuildInputs],
) (infer.CreateResponse[CodeBuildImageBuildState], error) {
	inputs := req.Inputs

	if req.DryRun {
		return infer.CreateResponse[CodeBuildImageBuildState]{
			ID: inputs.ProjectName,
			Output: CodeBuildImageBuildState{
				CodeBuildImageBuildInputs: inputs,
			},
		}, nil
	}

	maxWait := 1200
	if inputs.MaxWaitTime != nil {
		maxWait = *inputs.MaxWaitTime
	}

	// Initial wait for IAM role to sync
	time.Sleep(3 * time.Second)

	var buildID string
	var err error
	for attempt := range 2 {
		buildID, err = runCodeBuildBuild(ctx, inputs.ProjectName, inputs.Region, maxWait)
		if err == nil {
			break
		}
		if attempt == 1 || !isRetryable(err) {
			return infer.CreateResponse[CodeBuildImageBuildState]{}, fmt.Errorf("CodeBuild build failed: %w", err)
		}
		time.Sleep(5 * time.Second)
	}

	image := inputs.Destination

	return infer.CreateResponse[CodeBuildImageBuildState]{
		ID: inputs.ProjectName,
		Output: CodeBuildImageBuildState{
			CodeBuildImageBuildInputs: inputs,
			BuildID:                   buildID,
			Image:                     image,
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

func runCodeBuildBuild(ctx context.Context, projectName, region string, maxWaitSeconds int) (string, error) {
	opts := []func(*config.LoadOptions) error{}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
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

		build := batchOut.Builds[0]
		switch build.BuildStatus {
		case cbtypes.StatusTypeSucceeded:
			return buildID, nil
		case cbtypes.StatusTypeInProgress:
			continue
		case cbtypes.StatusTypeFailed, cbtypes.StatusTypeFault:
			msg := "build failed"
			if build.Phases != nil {
				for _, phase := range build.Phases {
					if phase.PhaseStatus == cbtypes.StatusTypeFailed && len(phase.Contexts) > 0 {
						for _, c := range phase.Contexts {
							if c.Message != nil {
								msg = *c.Message
							}
						}
					}
				}
			}
			return buildID, fmt.Errorf("%s: %s: %w", build.BuildStatus, msg, ErrBuildFailed)
		case cbtypes.StatusTypeStopped:
			return buildID, ErrBuildStopped
		case cbtypes.StatusTypeTimedOut:
			return buildID, ErrCodeBuildTimeout
		default:
			continue
		}
	}
}
