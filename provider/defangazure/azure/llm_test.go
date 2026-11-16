package azure

import (
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/cognitiveservices/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// llmDeploymentMocks gives every ModelSelection the target in selected, and
// every Foundry deployment a name output the test can tell from its input.
type llmDeploymentMocks struct{ selected resource.PropertyMap }

func (m *llmDeploymentMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outs := args.Inputs.Copy()
	switch {
	case strings.HasSuffix(args.TypeToken, ":index:ModelSelection"):
		for k, v := range m.selected {
			outs[k] = v
		}
	case args.TypeToken == "azure-native:cognitiveservices:Deployment":
		outs["name"] = resource.NewStringProperty("deployed-" + args.Inputs["deploymentName"].StringValue())
	}
	return args.Name + "_id", outs, nil
}

func (*llmDeploymentMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// deployShared creates the "shared" deployment against a selection holding
// text-embedding-3-small version 2, and returns the deployment's recorded name.
func deployShared(t *testing.T, candidate compose.ModelCandidate, switched bool) (string, error) {
	t.Helper()
	mocks := &llmDeploymentMocks{selected: resource.PropertyMap{
		"model":        resource.NewStringProperty("text-embedding-3-small"),
		"version":      resource.NewStringProperty("2"),
		"format":       resource.NewStringProperty("OpenAI"),
		"sku":          resource.NewStringProperty("Standard"),
		"selectedRole": resource.NewStringProperty(common.ModelRoleEmbedding),
	}}
	recorded := make(chan string, 1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rg, err := resources.NewResourceGroup(ctx, "rg", nil)
		if err != nil {
			return err
		}
		account, err := cognitiveservices.NewAccount(ctx, "ai", &cognitiveservices.AccountArgs{
			ResourceGroupName: rg.Name,
		})
		if err != nil {
			return err
		}
		plan := common.ModelPlan{
			Request: "shared", Source: common.ModelSourceRecipe, Candidates: []compose.ModelCandidate{candidate},
		}
		out, err := CreateLLMDeployment(ctx, "shared", plan, nil, switched,
			&LLMInfra{Account: account}, &SharedInfra{ResourceGroup: rg}, common.PluginIdentity{})
		if err != nil {
			return err
		}
		out.ApplyT(func(name string) string {
			recorded <- name
			return name
		})
		return nil
	}, pulumi.WithMocks("project", "stack", mocks))
	if err != nil {
		return "", err
	}
	return <-recorded, nil
}

func TestCreateLLMDeploymentRecordsTheDeployment(t *testing.T) {
	// The recorded name is the deployment's own output, so a deployment that
	// fails is never recorded as the model's new deployment.
	name, err := deployShared(t, compose.ModelCandidate{Model: "text-embedding-3-small"}, false)
	require.NoError(t, err)
	assert.Equal(t, "deployed-shared", name)
}

func TestCreateLLMDeploymentChecksASwitch(t *testing.T) {
	t.Run("unpinned-embedding-is-refused", func(t *testing.T) {
		_, err := deployShared(t, compose.ModelCandidate{Model: "text-embedding-3-small"}, true)
		require.ErrorContains(t, err, common.ErrEmbeddingMigration.Error())
		assert.ErrorContains(t, err, "needs an exact target")
	})
	t.Run("pin-of-the-held-target-switches", func(t *testing.T) {
		name, err := deployShared(t, compose.ModelCandidate{Model: "text-embedding-3-small", Version: "2"}, true)
		require.NoError(t, err)
		assert.Equal(t, "deployed-shared", name)
	})
	t.Run("pin-of-another-target-is-refused", func(t *testing.T) {
		// The shared selection keeps version 2 for the models already using it,
		// so this model would not get the version it pins.
		_, err := deployShared(t, compose.ModelCandidate{Model: "text-embedding-3-small", Version: "1"}, true)
		require.ErrorContains(t, err, common.ErrEmbeddingMigration.Error())
		assert.ErrorContains(t, err, "not the pinned text-embedding-3-small version 1")
	})
}
