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
		Identity: &armappcontainers.ManagedServiceIdentity{
			Type: ptr(armappcontainers.ManagedServiceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armappcontainers.UserAssignedIdentity{
				"/subscriptions/s/resourceGroups/defang-cd/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cd": {},
			},
		},
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

	if args.JobName != pulumi.String(selfDestructJobName) {
		t.Errorf("JobName = %v", args.JobName)
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

	// The passphrase must be a secret ref, not plain text.
	secrets := cfg.Secrets.(app.SecretArray)
	if len(secrets) != 1 {
		t.Fatalf("secrets = %v, want exactly the passphrase", secrets)
	}
	secret := secrets[0].(app.SecretArgs)
	if secret.Name != pulumi.String("pulumi-config-passphrase") || secret.Value != pulumi.String("hunter2") {
		t.Errorf("secret = %+v", secret)
	}

	tmpl := args.Template.(app.JobTemplateArgs)
	container := tmpl.Containers.(app.ContainerArray)[0].(app.ContainerArgs)
	if container.Image != pulumi.String("defangio/cd:public-beta") {
		t.Errorf("Image = %v", container.Image)
	}
	var plain, secretRefs []string
	for _, ev := range container.Env.(app.EnvironmentVarArray) {
		v := ev.(app.EnvironmentVarArgs)
		name := string(v.Name.(pulumi.String))
		if v.SecretRef != nil {
			secretRefs = append(secretRefs, name)
		} else {
			plain = append(plain, name)
		}
	}
	if strings.Join(secretRefs, ",") != "PULUMI_CONFIG_PASSPHRASE" {
		t.Errorf("secretRefs = %v", secretRefs)
	}
	if strings.Join(plain, ",") != "DEFANG_STATE_URL,PROJECT,STACK" { // sorted, PATH dropped
		t.Errorf("plain env = %v", plain)
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

	noIdentity := validCdJob()
	noIdentity.Identity = nil
	if _, err := azureSelfDestructJobArgs(noIdentity, environ, fireAt, "rg", "s"); err == nil {
		t.Error("want error for missing identity")
	}
}
