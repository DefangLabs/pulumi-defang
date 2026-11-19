package program

import (
	"os"
	"testing"
	"time"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestScalewayDeploymentRegistersTeardownOnlyWithTTL(t *testing.T) {
	t.Setenv("DEFANG_SCW_CD_JOB_NAME", "defang-cd-dev")
	t.Setenv("DEFANG_SCW_CD_SECRET_ID", "secret-id")
	t.Setenv("SCW_SECRET_KEY", "secret-value")
	t.Setenv("DEFANG_EVENTS_UPLOAD_URL", "presigned-url")

	const payloadEtag = "deployment-id"
	for _, tc := range []struct {
		name        string
		ttl         string
		payloadEtag string
		wantEtag    string
	}{
		{name: "no TTL", payloadEtag: payloadEtag, wantEtag: payloadEtag},
		{name: "payload ETag", ttl: "48h", payloadEtag: payloadEtag, wantEtag: payloadEtag},
		{name: "config ETag", ttl: "48h", wantEtag: "fallback-id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocks := &scalewayScheduleMocks{}
			deployment := &defangv1.ProjectUpdate{
				CdVersion: "defangio/test-cd",
				Compose:   []byte("services:\n  web:\n    image: nginx\n"),
				Etag:      tc.payloadEtag,
			}
			before := time.Now().UTC()
			configure := func(info *pulumi.RunInfo) {
				info.Config = map[string]string{
					"defang:provider":    "scaleway",
					"defang:stateUrl":    "s3://defang-cd-dev?endpoint=https://s3.fr-par.scw.cloud",
					"defang:ttl":         tc.ttl,
					"defang:etag":        "fallback-id",
					"scaleway:projectId": "project-id",
					"scaleway:region":    "fr-par",
				}
			}
			require.NoError(t, pulumi.RunErr(NewRun(deployment), pulumi.WithMocks("app", "dev", mocks), configure))
			require.NotEmpty(t, mocks.projectUpdateFile)
			t.Cleanup(func() { _ = os.Remove(mocks.projectUpdateFile) })
			data, err := os.ReadFile(mocks.projectUpdateFile)
			require.NoError(t, err)
			var persisted defangv1.ProjectUpdate
			require.NoError(t, proto.Unmarshal(data, &persisted))
			assert.Equal(t, tc.wantEtag, persisted.GetEtag())

			if tc.ttl == "" {
				require.Nil(t, mocks.trigger)
				return
			}

			inputs := mocks.trigger
			require.NotNil(t, inputs)
			assert.Equal(t, "defangio/test-cd", inputs["imageUri"].StringValue())
			args := inputs["args"].ArrayValue()
			require.Len(t, args, 4)
			assert.Equal(t, TriggerDownCommand, args[0].StringValue())
			assert.Equal(t, "defang-cd-dev", args[1].StringValue())
			assert.Equal(t, tc.wantEtag, args[2].StringValue())
			assert.Equal(t, "projects/app/dev/project.pb", args[3].StringValue())
			cron := inputs["cron"].ObjectValue()
			// The deadline is measured when the schedule is registered.
			var want []string
			for d := 48*time.Hour + time.Minute; d <= 48*time.Hour+time.Since(before)+2*time.Minute; d += time.Minute {
				want = append(want, selfDestructCron(before.Add(d).Truncate(time.Minute)))
			}
			assert.Contains(t, want, cron["schedule"].StringValue())
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

func TestScalewayDeploymentRejectsTTLOverOneYear(t *testing.T) {
	t.Setenv("DEFANG_SCW_CD_JOB_NAME", "defang-cd-dev")
	t.Setenv("DEFANG_SCW_CD_SECRET_ID", "secret-id")
	deployment := &defangv1.ProjectUpdate{
		CdVersion: "defangio/test-cd",
		Compose:   []byte("services:\n  web:\n    image: nginx\n"),
		Etag:      "deployment-id",
	}
	configure := func(info *pulumi.RunInfo) {
		info.Config = map[string]string{
			"defang:provider":    "scaleway",
			"defang:stateUrl":    "s3://defang-cd-dev?endpoint=https://s3.fr-par.scw.cloud",
			"defang:ttl":         "9600h",
			"scaleway:projectId": "project-id",
			"scaleway:region":    "fr-par",
		}
	}
	err := pulumi.RunErr(NewRun(deployment), pulumi.WithMocks("app", "dev", &scalewayScheduleMocks{}), configure)
	require.ErrorIs(t, err, errCronTTLTooLong)
}

type scalewayScheduleMocks struct {
	trigger           resource.PropertyMap
	projectUpdateFile string
}

func (*scalewayScheduleMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m *scalewayScheduleMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == "scaleway:job/definition:Definition" {
		m.trigger = args.Inputs
	}
	if args.Name == "project-pb" {
		m.projectUpdateFile = args.Inputs["file"].StringValue()
	}
	return args.Name + "-id", args.Inputs, nil
}
