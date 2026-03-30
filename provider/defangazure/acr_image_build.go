package defangazure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/pulumi/pulumi-go-provider/infer"
)

var (
	ErrNoACRRunID     = errors.New("failed to schedule run: no run ID returned")
	ErrACRRunTimedOut = errors.New("ACR task run timed out")
	ErrACRRunFailed   = errors.New("ACR task run failed")
)

// ACRImageBuild is a custom resource that schedules an ACR task run and waits for completion.
// Analogous to Build (CodeBuild) in the AWS provider.
type ACRImageBuild struct{}

// ACRImageBuildInputs are the inputs for the ACR image build resource.
type ACRImageBuildInputs struct {
	// Azure subscription ID
	SubscriptionID string `pulumi:"subscriptionId"`

	// Resource group containing the registry
	ResourceGroupName string `pulumi:"resourceGroupName"`

	// Name of the Azure Container Registry
	RegistryName string `pulumi:"registryName"`

	// Name of the ACR task to run
	TaskName string `pulumi:"taskName"`

	// Image name (without tag) pushed by the task, e.g. "myservice"
	ImageName string `pulumi:"imageName"`

	// Registry login server, e.g. "myregistry.azurecr.io"
	LoginServer string `pulumi:"loginServer"`

	// Max wait time in seconds (default: 3600)
	MaxWaitTime *int `pulumi:"maxWaitTime,optional"`

	// Trigger replacements when these change (hash of build inputs)
	Triggers []string `pulumi:"triggers,optional"`
}

// ACRImageBuildState is the output state of the ACR image build resource.
type ACRImageBuildState struct {
	ACRImageBuildInputs

	// The ACR run ID
	RunID string `pulumi:"runId"`

	// The full image URI pushed by the build (loginServer/imageName:tag or @digest)
	Image string `pulumi:"image"`
}

// Create schedules an ACR task run and waits for it to complete.
func (*ACRImageBuild) Create(
	ctx context.Context, req infer.CreateRequest[ACRImageBuildInputs],
) (infer.CreateResponse[ACRImageBuildState], error) {
	inputs := req.Inputs

	if req.DryRun {
		return infer.CreateResponse[ACRImageBuildState]{
			ID: inputs.TaskName,
			Output: ACRImageBuildState{
				ACRImageBuildInputs: inputs,
			},
		}, nil
	}

	maxWait := 3600
	if inputs.MaxWaitTime != nil {
		maxWait = *inputs.MaxWaitTime
	}

	runID, image, err := scheduleAndWaitACRRun(
		ctx,
		inputs.SubscriptionID,
		inputs.ResourceGroupName,
		inputs.RegistryName,
		inputs.TaskName,
		inputs.LoginServer,
		inputs.ImageName,
		maxWait,
	)
	if err != nil {
		return infer.CreateResponse[ACRImageBuildState]{}, fmt.Errorf("ACR image build failed: %w", err)
	}

	return infer.CreateResponse[ACRImageBuildState]{
		ID: inputs.TaskName,
		Output: ACRImageBuildState{
			ACRImageBuildInputs: inputs,
			RunID:               runID,
			Image:               image,
		},
	}, nil
}

// scheduleAndWaitACRRun schedules an ACR task run and polls until it reaches a terminal state.
func scheduleAndWaitACRRun(
	ctx context.Context,
	subscriptionID, rgName, registryName, taskName, loginServer, imageName string,
	maxWaitSeconds int,
) (string, string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", "", fmt.Errorf("getting Azure credential: %w", err)
	}

	regClient, err := armcontainerregistry.NewRegistriesClient(subscriptionID, cred, nil)
	if err != nil {
		return "", "", fmt.Errorf("creating registries client: %w", err)
	}

	runsClient, err := armcontainerregistry.NewRunsClient(subscriptionID, cred, nil)
	if err != nil {
		return "", "", fmt.Errorf("creating runs client: %w", err)
	}

	taskID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerRegistry/registries/%s/tasks/%s",
		subscriptionID, rgName, registryName, taskName,
	)
	taskRunType := "TaskRunRequest"
	isArchiveEnabled := false
	poller, err := regClient.BeginScheduleRun(ctx, rgName, registryName, &armcontainerregistry.TaskRunRequest{
		Type:             &taskRunType,
		TaskID:           &taskID,
		IsArchiveEnabled: &isArchiveEnabled,
	}, nil)
	if err != nil {
		return "", "", fmt.Errorf("scheduling ACR task run: %w", err)
	}

	// Wait for the run to be created (LRO resolves once the run resource exists).
	scheduled, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("waiting for ACR run to be scheduled: %w", err)
	}
	if scheduled.Properties == nil || scheduled.Properties.RunID == nil {
		return "", "", ErrNoACRRunID
	}
	runID := *scheduled.Properties.RunID

	// Check whether the run already reached a terminal state.
	if scheduled.Properties.Status != nil {
		if image, done, err := checkRunStatus(scheduled.Properties, loginServer, imageName, runID); done {
			return runID, image, err
		}
	}

	// Poll until terminal.
	deadline := time.Now().Add(time.Duration(maxWaitSeconds) * time.Second)
	pollInterval := 5 * time.Second

	for {
		if time.Now().After(deadline) {
			return runID, "", fmt.Errorf("run %s timed out after %ds: %w", runID, maxWaitSeconds, ErrACRRunTimedOut)
		}

		time.Sleep(pollInterval)
		if pollInterval < 30*time.Second {
			pollInterval = min(pollInterval*2, 30*time.Second)
		}

		resp, err := runsClient.Get(ctx, rgName, registryName, runID, nil)
		if err != nil {
			return runID, "", fmt.Errorf("polling run %s: %w", runID, err)
		}
		if resp.Properties == nil || resp.Properties.Status == nil {
			continue
		}
		if image, done, err := checkRunStatus(resp.Properties, loginServer, imageName, runID); done {
			return runID, image, err
		}
	}
}

// checkRunStatus inspects run properties and returns (image, done, err).
// done is true when a terminal state is reached.
func checkRunStatus(
	props *armcontainerregistry.RunProperties,
	loginServer, imageName, runID string,
) (string, bool, error) {
	switch *props.Status {
	case armcontainerregistry.RunStatusSucceeded:
		return buildImageURI(props.OutputImages, loginServer, imageName, runID), true, nil
	case armcontainerregistry.RunStatusFailed, armcontainerregistry.RunStatusError:
		return "", true, fmt.Errorf("run %s %s: %w", runID, string(*props.Status), ErrACRRunFailed)
	case armcontainerregistry.RunStatusCanceled:
		return "", true, fmt.Errorf("run %s was canceled: %w", runID, ErrACRRunFailed)
	case armcontainerregistry.RunStatusTimeout:
		return "", true, fmt.Errorf("run %s timed out on ACR side: %w", runID, ErrACRRunFailed)
	default:
		return "", false, nil
	}
}

// buildImageURI constructs the image URI from the run's output images.
// Falls back to loginServer/imageName:runID when OutputImages is empty.
func buildImageURI(
	outputImages []*armcontainerregistry.ImageDescriptor,
	loginServer, imageName, runID string,
) string {
	for _, img := range outputImages {
		if img.Digest != nil && *img.Digest != "" {
			return fmt.Sprintf("%s/%s@%s", loginServer, imageName, *img.Digest)
		}
		if img.Tag != nil && *img.Tag != "" {
			return fmt.Sprintf("%s/%s:%s", loginServer, imageName, *img.Tag)
		}
	}
	return fmt.Sprintf("%s/%s:%s", loginServer, imageName, runID)
}
