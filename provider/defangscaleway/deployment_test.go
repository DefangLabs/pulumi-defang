package defangscaleway

import (
	"encoding/json"
	"testing"

	"github.com/DefangLabs/defang/src/pkg/deployment"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deploymentMocks struct{ projectRecordingMocks }

func (m *deploymentMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	id, outputs, err := m.projectRecordingMocks.NewResource(args)
	if args.TypeToken == "scaleway:containers/container:Container" {
		id = "fr-par/cloud-generated-uuid"
		outputs["name"] = resource.NewStringProperty("cloud-assigned-name")
	}
	return id, outputs, err
}

func TestProjectDeploymentIsReadableByCLI(t *testing.T) {
	t.Parallel()
	mocks := &deploymentMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		result, err := buildProject(ctx, "demo", ProjectInputs{Etag: "attempt-1", Services: compose.Services{
			"web": {Image: strPtr("nginx:latest"), Ports: []compose.ServicePortConfig{{Target: 80}}},
			"db": {Postgres: &compose.PostgresConfig{},
				Environment: map[string]*string{"POSTGRES_PASSWORD": strPtr("DoNotPublish123!")}},
		}})
		if err != nil {
			return err
		}
		result.Deployment.ApplyT(func(encoded string) error {
			// Feed the producer's literal output through the CLI's state reader.
			state, err := json.Marshal(map[string]any{"version": 3, "deployment": map[string]any{
				"resources": []any{map[string]any{"outputs": map[string]any{deployment.OutputName: encoded}}},
			}})
			if err != nil {
				t.Errorf("encoding state: %v", err)
				return err
			}
			metadata, err := deployment.ReadState(state, "scaleway", "demo", "prod", "attempt-1")
			if err != nil {
				t.Errorf("reading metadata: %v", err)
				return err
			}
			assert.True(t, metadata.Summary)
			assert.Len(t, metadata.Services, 2)
			assert.Equal(t, "fr-par/cloud-generated-uuid", metadata.Services["web"].ID)
			assert.Equal(t, "cloud-assigned-name", metadata.Services["web"].Name)
			assert.Equal(t, "cloud-generated-uuid", metadata.Services["web"].LogLabels["resource_id"])
			assert.Equal(t, deployment.KindPostgres, metadata.Services["db"].Kind)
			assert.NotContains(t, encoded, "DoNotPublish123!")
			assert.NotContains(t, encoded, "connectionUrl")
			return nil
		})
		return nil
	}, pulumi.WithMocks("demo", "prod", mocks))
	require.NoError(t, err)
}

func TestRedisDeploymentDoesNotInheritSecretConnectionString(t *testing.T) {
	t.Parallel()
	var output pulumi.StringOutput
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		connection := pulumi.ToSecret("redis://user:NeverPublish@10.0.0.8:6379/0?token=NeverPublish").(pulumi.StringOutput)
		service := serviceDeployment(ctx, "cache", deployment.KindRedis, "attempt",
			pulumi.ID("fr-par/redis-id").ToIDOutput(), pulumi.String("cache").ToStringOutput(),
			publicRedisEndpoint(connection), "")
		output = projectDeployment(ctx, "attempt", []any{service})
		output.ApplyT(func(encoded string) error {
			assert.Contains(t, encoded, "10.0.0.8:6379")
			assert.NotContains(t, encoded, "NeverPublish")
			assert.NotContains(t, encoded, "redis://")
			return nil
		})
		return nil
	}, pulumi.WithMocks("demo", "prod", &projectRecordingMocks{}))
	require.NoError(t, err)
	assert.False(t, pulumi.IsSecret(output), "public metadata must not be encrypted")
}
