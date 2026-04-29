package defangazure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/pulumi/pulumi-go-provider/infer"
)

var (
	ErrNoACRRunID        = errors.New("failed to schedule run: no run ID returned")
	ErrACRRunTimedOut    = errors.New("ACR task run timed out")
	ErrACRRunFailed      = errors.New("ACR task run failed")
	ErrACREmptyUploadURL = errors.New("empty upload URL response from ACR")
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

	// Image name (the shared "builds" repo); per-service builds are
	// disambiguated by tag prefix (ServiceName).
	ImageName string `pulumi:"imageName"`

	// Logical Compose service name; encoded as the tag prefix
	// (e.g. tag "{serviceName}-{runID}") so all services in the project
	// share a single repo and benefit from cross-service layer reuse.
	ServiceName string `pulumi:"serviceName"`

	// Registry login server, e.g. "myregistry.azurecr.io"
	LoginServer string `pulumi:"loginServer"`

	// Build context URL — a bare blob URL pointing at the tar that the CLI
	// uploaded to the CD storage account. Restaged into ACR's own staging
	// area at run time (see stageBuildContextToACR) so that ACR Tasks can
	// fetch via its internal credentials, not via a SAS token in this URL.
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
		inputs.ServiceName,
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

// scheduleAndWaitACRRun stages the build context into ACR's own upload area,
// schedules an inline ACR task run, and polls until it reaches a terminal state.
//
// Source contextPath is the bare blob URL of the tar in the CD storage account.
// We download it via the CD task's managed identity and re-upload to ACR's
// staging slot (GetBuildSourceUploadURL); ACR Tasks then fetches via its own
// internal credentials, so neither the URL nor any SAS leaks into Pulumi state.
func scheduleAndWaitACRRun(
	ctx context.Context,
	subscriptionID, rgName, registryName string,
	encodedTaskContent, contextPath string,
	loginServer, imageName, serviceName string,
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

	sourceLocation, err := stageBuildContextToACR(ctx, cred, regClient, rgName, registryName, contextPath)
	if err != nil {
		return "", "", fmt.Errorf("staging build context: %w", err)
	}

	runType := "EncodedTaskRunRequest"
	isArchiveEnabled := false
	osLinux := armcontainerregistry.OSLinux
	cpu := int32(2)
	poller, err := regClient.BeginScheduleRun(ctx, rgName, registryName, &armcontainerregistry.EncodedTaskRunRequest{
		Type:               &runType,
		EncodedTaskContent: &encodedTaskContent,
		// Relative path returned by ACR's GetBuildSourceUploadURL — ACR Tasks
		// reads this from its own staging area without needing caller credentials.
		SourceLocation:   &sourceLocation,
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
		if image, done, runErr := checkRunStatus(scheduled.Properties, loginServer, imageName, serviceName, runID); done {
			if runErr != nil {
				return runID, image, fmt.Errorf("ACR run failed: %w", runErr)
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
		if image, done, runErr := checkRunStatus(resp.Properties, loginServer, imageName, serviceName, runID); done {
			if runErr != nil {
				return runID, image, fmt.Errorf("ACR run failed: %w", runErr)
			}
			return runID, image, nil
		}
	}
}

// checkRunStatus inspects run properties and returns (image, done, err).
// done is true when a terminal state is reached.
func checkRunStatus(
	props *armcontainerregistry.RunProperties,
	loginServer, imageName, serviceName, runID string,
) (string, bool, error) {
	switch *props.Status {
	case armcontainerregistry.RunStatusSucceeded:
		return buildImageURI(props.OutputImages, loginServer, imageName, serviceName, runID), true, nil
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

// buildImageURI constructs the image URI from the run's output images. With
// the shared "builds" repo, the per-service primary tag is "{serviceName}-{runID}";
// it's deterministic so the fallback (used when OutputImages is empty) can
// reconstruct it without inspecting the run output.
func buildImageURI(
	outputImages []*armcontainerregistry.ImageDescriptor,
	loginServer, imageName, serviceName, runID string,
) string {
	for _, img := range outputImages {
		if img.Digest != nil && *img.Digest != "" {
			return fmt.Sprintf("%s/%s@%s", loginServer, imageName, *img.Digest)
		}
		if img.Tag != nil && *img.Tag != "" {
			return fmt.Sprintf("%s/%s:%s", loginServer, imageName, *img.Tag)
		}
	}
	return fmt.Sprintf("%s/%s:%s-%s", loginServer, imageName, serviceName, runID)
}

// stageBuildContextToACR streams the tar at sourceURL (a bare blob URL in the
// CD storage account) into ACR's own upload staging slot, returning the
// relative path that EncodedTaskRunRequest.SourceLocation expects.
//
// The download uses the caller's managed identity; the upload uses a
// short-lived SAS URL returned by GetBuildSourceUploadURL. Streaming is done
// via blockblob.UploadStream, which stages blocks of bounded size and commits
// at the end — no full buffering in memory.
func stageBuildContextToACR(
	ctx context.Context,
	cred azcore.TokenCredential,
	rc *armcontainerregistry.RegistriesClient,
	rgName, registryName, sourceURL string,
) (string, error) {
	bc, err := blob.NewClient(sourceURL, cred, nil)
	if err != nil {
		return "", fmt.Errorf("creating blob client for %s: %w", sourceURL, err)
	}
	dl, err := bc.DownloadStream(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("opening download stream: %w", err)
	}
	defer func() { _ = dl.Body.Close() }()

	up, err := rc.GetBuildSourceUploadURL(ctx, rgName, registryName, nil)
	if err != nil {
		return "", fmt.Errorf("getting ACR upload URL: %w", err)
	}
	if up.UploadURL == nil || up.RelativePath == nil {
		return "", ErrACREmptyUploadURL
	}

	bbClient, err := blockblob.NewClientWithNoCredential(*up.UploadURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating blockblob client: %w", err)
	}
	if _, err := bbClient.UploadStream(ctx, dl.Body, nil); err != nil {
		return "", fmt.Errorf("uploading to ACR staging: %w", err)
	}

	return *up.RelativePath, nil
}
