package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/DefangLabs/defang/src/pkg/clouds/azure"
	"github.com/DefangLabs/defang/src/pkg/clouds/azure/aca"
	"github.com/DefangLabs/pulumi-defang/cd/program"
)

// cdCommandTriggerDown is the command the Azure self-destruct trigger job
// runs (see program.createAzureSelfDestruct). It only STARTS a regular down
// execution on the shared defang-cd Container Apps Job and exits; the down
// itself must not run inside the trigger job, because the destroy deletes
// that job — killing its own execution mid-destroy. On AWS/GCP the platform
// scheduler starts the external CD runner directly, so no equivalent exists.
const cdCommandTriggerDown = "trigger-down"

// triggerDown starts `/app/cd down` on the shared defang-cd job with this
// process's own (already filtered) environment. The trigger job's
// system-assigned identity holds Contributor on the defang-cd job only.
func triggerDown(ctx context.Context) error {
	image := os.Getenv("DEFANG_CD_IMAGE")
	if image == "" {
		return errors.New("missing required environment variable: DEFANG_CD_IMAGE")
	}
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		return errors.New("missing required environment variable: AZURE_SUBSCRIPTION_ID")
	}

	job := &aca.Job{
		Azure: azure.Azure{
			SubscriptionID: subscriptionID,
			Location:       azure.Location(os.Getenv("AZURE_LOCATION")),
		},
		ResourceGroup: "defang-cd", // the shared CD resource group; see program.createAzureSelfDestruct
	}
	execution, err := job.StartJobExecution(ctx, aca.JobRequest{
		Image:   image,
		Command: []string{"/app/cd", "down"}, // matches ByocAzure.runCdCommand
		Envs:    program.SelfDestructEnv(os.Environ()),
		Timeout: 30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("failed to start down execution: %w", err)
	}
	Println("Started down execution", execution)
	return nil
}
