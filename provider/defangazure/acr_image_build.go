package defangazure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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

	// Name of the ACR task — used only to declare a Pulumi dependency on the Task resource.
	TaskName string `pulumi:"taskName"`

	// Image name (without tag) pushed by the task, e.g. "myservice"
	ImageName string `pulumi:"imageName"`

	// Registry login server, e.g. "myregistry.azurecr.io"
	LoginServer string `pulumi:"loginServer"`

	// Full context URL for the build (may include a SAS token query string).
	// Passed at run time via EncodedTaskRunRequest so the ARM API never strips the token.
	ContextPath string `pulumi:"contextPath"`

	// Base64-encoded ACR task YAML (from generateTaskYAML).
	EncodedTaskContent string `pulumi:"encodedTaskContent"`

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
		inputs.EncodedTaskContent,
		inputs.ContextPath,
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

// scheduleAndWaitACRRun schedules an inline ACR task run using EncodedTaskRunRequest
// and polls until it reaches a terminal state.
// Using EncodedTaskRunRequest (rather than TaskRunRequest) means the context URL is
// passed in the request body, which is never persisted — so the ARM API cannot strip
// the SAS query string the way it does when storing a task definition.
func scheduleAndWaitACRRun(
	ctx context.Context,
	subscriptionID, rgName, registryName string,
	encodedTaskContent, contextPath string,
	loginServer, imageName string,
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

	runType := "EncodedTaskRunRequest"
	isArchiveEnabled := false
	osLinux := armcontainerregistry.OSLinux
	cpu := int32(2)
	poller, err := regClient.BeginScheduleRun(ctx, rgName, registryName, &armcontainerregistry.EncodedTaskRunRequest{
		Type:               &runType,
		EncodedTaskContent: &encodedTaskContent,
		// SourceLocation carries the full context URL including SAS token.
		// EncodedTaskRunRequest is never persisted by ARM, so the query string is preserved.
		SourceLocation:   &contextPath,
		IsArchiveEnabled: &isArchiveEnabled,
		Platform: &armcontainerregistry.PlatformProperties{
			OS: &osLinux,
		},
		AgentConfiguration: &armcontainerregistry.AgentProperties{
			CPU: &cpu,
		},
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
		if image, done, runErr := checkRunStatus(scheduled.Properties, loginServer, imageName, runID); done {
			if runErr != nil {
				log := fetchRunLog(ctx, runsClient, rgName, registryName, runID)
				return runID, image, fmt.Errorf("%w\n--- ACR run log ---\n%s", runErr, log)
			}
			return runID, image, nil
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
		if image, done, runErr := checkRunStatus(resp.Properties, loginServer, imageName, runID); done {
			if runErr != nil {
				log := fetchRunLog(ctx, runsClient, rgName, registryName, runID)
				return runID, image, fmt.Errorf("%w\n--- ACR run log ---\n%s", runErr, log)
			}
			return runID, image, nil
		}
	}
}

// fetchRunLog fetches the run log content via the ACR log SAS URL.
// Returns a best-effort string; errors are embedded in the returned string.
func fetchRunLog(
	ctx context.Context,
	runsClient *armcontainerregistry.RunsClient,
	rgName, registryName, runID string,
) string {
	resp, err := runsClient.GetLogSasURL(ctx, rgName, registryName, runID, nil)
	if err != nil {
		return fmt.Sprintf("(could not get log URL: %v)", err)
	}
	if resp.LogLink == nil {
		return "(no log URL returned)"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *resp.LogLink, nil)
	if err != nil {
		return fmt.Sprintf("(could not build log request: %v)", err)
	}
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("(could not fetch log: %v)", err)
	}
	defer func() { _ = httpResp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 64*1024))
	if err != nil {
		return fmt.Sprintf("(could not read log: %v)", err)
	}
	return string(body)
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
	case armcontainerregistry.RunStatusQueued,
		armcontainerregistry.RunStatusRunning,
		armcontainerregistry.RunStatusStarted:
		return "", false, nil
	}
	return "", false, nil
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
