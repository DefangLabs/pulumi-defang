package program

import (
	"testing"
	"time"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScalewayDeploymentRegistersTeardownOnlyWithTTL(t *testing.T) {
	t.Setenv("DEFANG_SCW_CD_JOB_NAME", "defang-cd-dev")
	t.Setenv("DEFANG_SCW_CD_SECRET_ID", "secret-id")
	t.Setenv("SCW_SECRET_KEY", "secret-value")
	t.Setenv("DEFANG_EVENTS_UPLOAD_URL", "presigned-url")

	for _, ttl := range []string{"", "9600h"} {
		t.Run("ttl="+ttl, func(t *testing.T) {
			mocks := &scalewayScheduleMocks{}
			deployment := &defangv1.ProjectUpdate{
				CdVersion: "defangio/test-cd",
				Compose:   []byte("services:\n  web:\n    image: nginx\n"),
				Etag:      "deployment-id",
			}
			before := time.Now().UTC()
			configure := func(info *pulumi.RunInfo) {
				info.Config = map[string]string{
					"defang:provider":    "scaleway",
					"defang:stateUrl":    "s3://defang-cd-dev?endpoint=https://s3.fr-par.scw.cloud",
					"defang:ttl":         ttl,
					"scaleway:projectId": "project-id",
					"scaleway:region":    "fr-par",
				}
			}
			require.NoError(t, pulumi.RunErr(NewRun(deployment), pulumi.WithMocks("app", "dev", mocks), configure))
			if ttl == "" {
				require.Nil(t, mocks.trigger)
				return
			}

			inputs := mocks.trigger
			require.NotNil(t, inputs)
			assert.Equal(t, "defangio/test-cd", inputs["imageUri"].StringValue())
			args := inputs["args"].ArrayValue()
			require.Len(t, args, 5)
			assert.Equal(t, ScalewayTriggerDownCommand, args[0].StringValue())
			assert.Equal(t, "defang-cd-dev", args[1].StringValue())
			assert.Equal(t, "deployment-id", args[3].StringValue())
			assert.Equal(t, "projects/app/dev/project.pb", args[4].StringValue())
			deadline, err := time.Parse(time.RFC3339, args[2].StringValue())
			require.NoError(t, err)
			assert.True(t, deadline.After(before.Add(9600*time.Hour)))
			assert.Zero(t, deadline.Second())
			cron := inputs["cron"].ObjectValue()
			assert.Equal(t, selfDestructCron(deadline), cron["schedule"].StringValue())
			assert.Equal(t, "Etc/UTC", cron["timezone"].StringValue())
			env := inputs["env"].ObjectValue()
			assert.Equal(t, "s3://defang-cd-dev", env["DEFANG_STATE_URL"].StringValue())
			assert.NotContains(t, env, resource.PropertyKey("SCW_SECRET_KEY"))
			assert.NotContains(t, env, resource.PropertyKey("DEFANG_EVENTS_UPLOAD_URL"))
			refs := inputs["secretReferences"].ArrayValue()
			require.Len(t, refs, 1)
			secret := refs[0].ObjectValue()
			assert.Equal(t, "secret-id", secret["secretId"].StringValue())
			assert.Equal(t, "SCW_SECRET_KEY", secret["environment"].StringValue())
			assert.Equal(t, "latest", secret["secretVersion"].StringValue())
		})
	}
}

type scalewayScheduleMocks struct {
	trigger resource.PropertyMap
}

func (*scalewayScheduleMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m *scalewayScheduleMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "scaleway:job/definition:Definition" {
		m.trigger = args.Inputs
	}
	return args.Name + "-id", args.Inputs, nil
}
