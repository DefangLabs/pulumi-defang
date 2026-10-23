package program

import (
	"strings"
	"testing"
	"time"

	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ptr[T any](v T) *T { return &v }

func validCdJob() armappcontainers.Job {
	return armappcontainers.Job{
		Location: ptr("westus"),
		Properties: &armappcontainers.JobProperties{
			EnvironmentID: ptr("/subscriptions/s/resourceGroups/defang-cd/providers/Microsoft.App/managedEnvironments/cd-env"),
			Template: &armappcontainers.JobTemplate{
				Containers: []*armappcontainers.Container{
					{Image: ptr("defangio/cd:public-beta")},
				},
			},
		},
	}
}

func TestAzureSelfDestructJobArgs(t *testing.T) {
	environ := []string{
		"PROJECT=myproj",
		"STACK=preview",
		"PULUMI_CONFIG_PASSPHRASE=hunter2",
		"DEFANG_STATE_URL=azblob://pulumi?storage_account=x",
		"PATH=/usr/bin", // dropped
	}
	fireAt := time.Date(2026, time.August, 17, 14, 30, 0, 0, time.UTC)

	args, err := azureSelfDestructJobArgs(validCdJob(), environ, fireAt, "Defang-myproj-preview-rg", "preview")
	if err != nil {
		t.Fatal(err)
	}

	if args.JobName != nil {
		t.Errorf("JobName must stay unset so Pulumi auto-names the job (a fixed name collides in the shared defang-cd environment), got %v", args.JobName)
	}
	if args.ResourceGroupName != pulumi.String("Defang-myproj-preview-rg") {
		t.Errorf("ResourceGroupName = %v", args.ResourceGroupName)
	}
	if args.Location != pulumi.String("westus") {
		t.Errorf("Location = %v", args.Location)
	}

	cfg := args.Configuration.(app.JobConfigurationArgs)
	if cfg.TriggerType != pulumi.String("Schedule") {
		t.Errorf("TriggerType = %v", cfg.TriggerType)
	}
	sched := cfg.ScheduleTriggerConfig.(app.JobConfigurationScheduleTriggerConfigArgs)
	if sched.CronExpression != pulumi.String("30 14 17 8 *") {
		t.Errorf("CronExpression = %v", sched.CronExpression)
	}

	tmpl := args.Template.(app.JobTemplateArgs)
	container := tmpl.Containers.(app.ContainerArray)[0].(app.ContainerArgs)
	if container.Image != pulumi.String("defangio/cd:public-beta") {
		t.Errorf("Image = %v", container.Image)
	}
	// The trigger must only START the down on the shared CD job — running
	// the destroy in-place would kill its own execution (see selfdestruct_azure.go).
	if args := container.Args.(pulumi.StringArray); len(args) != 1 || args[0] != pulumi.String("trigger-down") {
		t.Errorf("Args = %v, want [trigger-down]", args)
	}
	// All values ride as plain env vars, matching how the CLI passes them to
	// every CD execution (credentials never reach this point — SelfDestructEnv
	// drops them).
	var names []string
	env := map[string]string{}
	for _, ev := range container.Env.(app.EnvironmentVarArray) {
		v := ev.(app.EnvironmentVarArgs)
		name := string(v.Name.(pulumi.String))
		names = append(names, name)
		env[name] = string(v.Value.(pulumi.String))
	}
	if strings.Join(names, ",") != "DEFANG_CD_IMAGE,DEFANG_STATE_URL,PROJECT,PULUMI_CONFIG_PASSPHRASE,STACK" { // sorted, PATH dropped
		t.Errorf("env names = %v", names)
	}
	// trigger-down re-runs this image on the defang-cd job.
	if env["DEFANG_CD_IMAGE"] != "defangio/cd:public-beta" {
		t.Errorf("DEFANG_CD_IMAGE = %q", env["DEFANG_CD_IMAGE"])
	}
}

func TestAzureSelfDestructJobArgsValidation(t *testing.T) {
	environ := []string{"PROJECT=p"}
	fireAt := time.Now().Add(time.Hour)

	noEnv := validCdJob()
	noEnv.Properties.EnvironmentID = nil
	if _, err := azureSelfDestructJobArgs(noEnv, environ, fireAt, "rg", "s"); err == nil {
		t.Error("want error for missing environment id")
	}

	noImage := validCdJob()
	noImage.Properties.Template.Containers = nil
	if _, err := azureSelfDestructJobArgs(noImage, environ, fireAt, "rg", "s"); err == nil {
		t.Error("want error for missing container image")
	}
}
