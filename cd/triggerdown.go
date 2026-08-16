package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/clouds/azure"
	"github.com/DefangLabs/defang/src/pkg/clouds/azure/aca"
	"github.com/DefangLabs/pulumi-defang/cd/program"
)

// cdCommandTriggerDown is the command the Azure self-destruct trigger job
// runs (see program.createAzureSelfDestruct). It only STARTS a regular down
// execution on the shared defang-cd Container Apps Job and exits.
//
// Why the indirection: on every cloud the trigger is an in-stack resource,
// but AWS/GCP schedulers are pure control-plane pointers — when they fire,
// the down runs in EXTERNAL compute (CodeBuild / Cloud Build), so the destroy
// deleting the already-fired pointer is harmless. An ACA scheduled job cannot
// call an API; it can only run its own container, and executions are child
// resources of the job itself. If that container ran the destroy directly,
// the destroy would delete the trigger job early (a dependent of the project)
// and thereby terminate its own running execution. trigger-down splits the
// two roles: the in-stack job stays a short-lived pointer, and the down runs
// in the external defang-cd job — same shape as the other clouds.
const cdCommandTriggerDown = client.CdCommand("trigger-down")

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
	// No Timeout: StartJobExecution never reads it — the effective bound is
	// the defang-cd job template's ReplicaTimeout, set by the CLI's SetUpJob.
	execution, err := job.StartJobExecution(ctx, aca.JobRequest{
		Image:   image,
		Command: []string{"/app/cd", string(client.CdCommandDown)}, // matches ByocAzure.runCdCommand
		Envs:    program.SelfDestructEnv(os.Environ()),
	})
	if err != nil {
		return fmt.Errorf("failed to start down execution: %w", err)
	}
	Println("Started down execution", execution)
	return nil
}
