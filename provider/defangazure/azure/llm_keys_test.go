package azure

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The error Azure returned for listKeys right after a tag-only account update.
var errListKeysConflict = errors.New(`request failed /subscriptions/x/resourceGroups/rg/providers/` +
	`Microsoft.CognitiveServices/accounts/a/listKeys: Status=409 Code="RequestConflict" ` +
	`Message="Cannot modify resource with id 'a' because the resource entity provisioning state is not terminal. ` +
	`Please wait for the provisioning state to become terminal and then retry the request."`)

var errListKeysForbidden = errors.New(`Status=403 Code="AuthorizationFailed"`)

func noSleep(context.Context, time.Duration) error { return nil }

func TestListAccountKeysWhenReady(t *testing.T) {
	delays := []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}

	t.Run("retries a provisioning conflict until the account is ready", func(t *testing.T) {
		calls := 0
		var slept []time.Duration
		got, err := listAccountKeysWhenReady(t.Context(), func() (string, error) {
			calls++
			if calls < 3 {
				return "", errListKeysConflict
			}
			return "key", nil
		}, delays, func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, "key", got)
		assert.Equal(t, 3, calls)
		assert.Equal(t, delays[:2], slept)
	})

	t.Run("returns other errors at once", func(t *testing.T) {
		calls := 0
		other := errListKeysForbidden
		_, err := listAccountKeysWhenReady(t.Context(), func() (string, error) {
			calls++
			return "", other
		}, delays, noSleep)
		require.ErrorIs(t, err, other)
		assert.Equal(t, 1, calls)
	})

	t.Run("gives up after the last delay", func(t *testing.T) {
		calls := 0
		_, err := listAccountKeysWhenReady(t.Context(), func() (string, error) {
			calls++
			return "", errListKeysConflict
		}, delays, noSleep)
		require.ErrorIs(t, err, errListKeysConflict)
		assert.Equal(t, len(delays)+1, calls)
	})

	t.Run("stops when the context ends", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		calls := 0
		_, err := listAccountKeysWhenReady(ctx, func() (string, error) {
			calls++
			return "", errListKeysConflict
		}, delays, sleepCtx)
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, errListKeysConflict)
		assert.Equal(t, 1, calls)
	})
}

func TestListKeysRetryDelaysBounded(t *testing.T) {
	var total time.Duration
	for _, d := range listKeysRetryDelays {
		total += d
	}
	assert.Greater(t, total, 5*time.Minute)
	assert.LessOrEqual(t, total, 15*time.Minute)
}

// listKeysConflictMocks fails listAccountKeys with a provisioning conflict
// for the first conflicts calls.
type listKeysConflictMocks struct {
	conflicts int32
	calls     atomic.Int32
}

func (*listKeysConflictMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outs := args.Inputs.Copy()
	if args.TypeToken == "azure-native:cognitiveservices:Account" {
		outs["name"] = resource.NewStringProperty("acct")
		outs["properties"] = resource.NewObjectProperty(resource.PropertyMap{
			"endpoint": resource.NewStringProperty("https://acct.services.ai.azure.com/"),
		})
	}
	if args.TypeToken == "random:index/randomString:RandomString" {
		outs["result"] = resource.NewStringProperty("abcd1234")
	}
	return args.Name + "_id", outs, nil
}

func (m *listKeysConflictMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if args.Token != "azure-native:cognitiveservices:listAccountKeys" {
		return args.Args, nil
	}
	if m.calls.Add(1) <= m.conflicts {
		return nil, errListKeysConflict
	}
	return resource.PropertyMap{"key1": resource.NewStringProperty("k1")}, nil
}

func TestCreateLLMInfraRetriesListKeysConflict(t *testing.T) {
	saved := listKeysRetryDelays
	listKeysRetryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { listKeysRetryDelays = saved })

	mocks := &listKeysConflictMocks{conflicts: 2}
	key := make(chan string, 1)
	secret := make(chan bool, 1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		rg, err := resources.NewResourceGroup(ctx, "rg", nil)
		if err != nil {
			return err
		}
		llm, err := CreateLLMInfra(ctx, "proj", &SharedInfra{ResourceGroup: rg})
		if err != nil {
			return err
		}
		// Exported so a listKeys failure fails the run, as the key's real
		// consumers (container app secrets) do.
		ctx.Export("apiKey", llm.APIKey)
		llm.APIKey.ApplyT(func(k string) string {
			key <- k
			return k
		})
		secret <- pulumi.IsSecret(llm.APIKey)
		return nil
	}, pulumi.WithMocks("project", "stack", mocks))
	require.NoError(t, err)
	assert.Equal(t, "k1", <-key)
	assert.True(t, <-secret)
	assert.Equal(t, int32(3), mocks.calls.Load())
}
