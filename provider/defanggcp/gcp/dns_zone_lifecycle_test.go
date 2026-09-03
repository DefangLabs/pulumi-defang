package gcp

import (
	"context"
	"sync/atomic"
	"testing"

	lt "github.com/pulumi/pulumi/pkg/v3/engine/lifecycletest/framework"
	pkgresource "github.com/pulumi/pulumi/pkg/v3/resource"
	"github.com/pulumi/pulumi/pkg/v3/resource/deploy"
	"github.com/pulumi/pulumi/pkg/v3/resource/deploy/deploytest"
	"github.com/pulumi/pulumi/pkg/v3/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lifecycleOperations backs the engine-level transition tests below. They use
// Pulumi's real deployment planner rather than language mocks, so a replacement
// or delete is visible as an actual provider operation.
type lifecycleOperations struct {
	creates atomic.Int32
	updates atomic.Int32
	deletes atomic.Int32
	reads   atomic.Int32
}

func (o *lifecycleOperations) provider() *deploytest.Provider {
	return &deploytest.Provider{
		CreateF: func(_ context.Context, req plugin.CreateRequest) (plugin.CreateResponse, error) {
			o.creates.Add(1)
			return plugin.CreateResponse{
				ID:         "managed-zone-id",
				Properties: req.Properties,
				Status:     resource.StatusOK,
			}, nil
		},
		UpdateF: func(_ context.Context, req plugin.UpdateRequest) (plugin.UpdateResponse, error) {
			o.updates.Add(1)
			return plugin.UpdateResponse{Properties: req.NewInputs, Status: resource.StatusOK}, nil
		},
		DeleteF: func(_ context.Context, _ plugin.DeleteRequest) (plugin.DeleteResponse, error) {
			o.deletes.Add(1)
			return plugin.DeleteResponse{Status: resource.StatusOK}, nil
		},
		ReadF: func(_ context.Context, req plugin.ReadRequest) (plugin.ReadResponse, error) {
			o.reads.Add(1)
			return plugin.ReadResponse{
				ReadResult: plugin.ReadResult{
					ID:      req.ID,
					Inputs:  req.Inputs,
					Outputs: req.State,
				},
				Status: resource.StatusOK,
			}, nil
		},
	}
}

func managedZoneProgram(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
	_, err := monitor.RegisterResource(managedZoneToken, "public-dns", true, deploytest.ResourceOptions{
		Inputs: resource.PropertyMap{
			"description": resource.NewStringProperty("Public DNS zone for myproject"),
			"dnsName":     resource.NewStringProperty("myproject.tenant.defang.app."),
		},
	})
	return err
}

func externalZoneProgram(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
	_, _, err := monitor.ReadResource(
		managedZoneToken,
		externalDelegateZoneLogicalName,
		"defang-myproject-tenant-defang-app",
		"",
		nil,
		"",
		"",
		"",
		nil,
		"",
		"",
	)
	return err
}

func zoneState(snapshot *deploy.Snapshot, logicalName string) *pkgresource.State {
	if snapshot == nil {
		return nil
	}
	for _, state := range snapshot.Resources {
		if string(state.Type) == managedZoneToken && state.URN.Name() == logicalName {
			return state
		}
	}
	return nil
}

func TestManagedDelegateZoneLifecycleTransitionStaysManaged(t *testing.T) {
	ops := &lifecycleOperations{}

	// This first update is the old implementation: public-dns is a managed
	// resource with an auto-named physical Name because Name is absent from inputs.
	lt.NewTestBuilder(t, nil).
		WithProvider("gcp", "1.0.0", ops.provider()).
		RunUpdate(managedZoneProgram, true).
		Then(func(oldSnapshot *deploy.Snapshot, err error) {
			require.NoError(t, err)
			oldZone := zoneState(oldSnapshot, "public-dns")
			require.NotNil(t, oldZone)
			assert.False(t, oldZone.External)
			assert.Equal(t, resource.ID("managed-zone-id"), oldZone.ID)
			assert.NotContains(t, oldZone.Inputs, resource.PropertyKey("name"))

			// The new managed path registers the identical type/logical name/inputs.
			// Pulumi must retain the same managed state, with no delete, replacement,
			// update, or second create.
			lt.NewTestBuilder(t, oldSnapshot).
				WithProvider("gcp", "1.0.0", ops.provider()).
				RunUpdate(managedZoneProgram, true).
				Then(func(newSnapshot *deploy.Snapshot, err error) {
					require.NoError(t, err)
					newZone := zoneState(newSnapshot, "public-dns")
					require.NotNil(t, newZone)
					assert.False(t, newZone.External)
					assert.Equal(t, oldZone.ID, newZone.ID)
					assert.Equal(t, int32(1), ops.creates.Load())
					assert.Zero(t, ops.updates.Load())
					assert.Zero(t, ops.deletes.Load())
				})
		})
}

func TestExternalDelegateZoneLifecycleTransitionNeverDeletes(t *testing.T) {
	ops := &lifecycleOperations{}

	lt.NewTestBuilder(t, nil).
		WithProvider("gcp", "1.0.0", ops.provider()).
		RunUpdate(externalZoneProgram, true).
		Then(func(readSnapshot *deploy.Snapshot, err error) {
			require.NoError(t, err)
			externalZone := zoneState(readSnapshot, externalDelegateZoneLogicalName)
			require.NotNil(t, externalZone)
			assert.True(t, externalZone.External)
			assert.Equal(t, resource.ID("defang-myproject-tenant-defang-app"), externalZone.ID)
			assert.Equal(t, int32(1), ops.reads.Load())
			assert.Zero(t, ops.creates.Load())

			// Removing an external read from the program is the lifecycle shape of
			// teardown: the engine discards it from state without calling Delete.
			lt.NewTestBuilder(t, readSnapshot).
				WithProvider("gcp", "1.0.0", ops.provider()).
				RunUpdate(func(_ plugin.RunInfo, _ *deploytest.ResourceMonitor) error { return nil }, true).
				Then(func(emptySnapshot *deploy.Snapshot, err error) {
					require.NoError(t, err)
					assert.Nil(t, zoneState(emptySnapshot, externalDelegateZoneLogicalName))
					assert.Zero(t, ops.deletes.Load())
					assert.Zero(t, ops.updates.Load())
				})
		})
}
