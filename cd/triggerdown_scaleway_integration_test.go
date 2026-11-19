//go:build integration

package main

import (
	"context"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

func TestScalewayScheduledDestroyRetriesAfterLock(t *testing.T) {
	if _, err := exec.LookPath("pulumi"); err != nil {
		t.Skip("Pulumi CLI required for the local stack lock test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	const backendEnv, encryptionEnv = "PULUMI_BACKEND_URL", "PULUMI_CONFIG_PASSPHRASE"
	workspace := auto.EnvVars(map[string]string{
		backendEnv:    "file://" + t.TempDir(),
		encryptionEnv: "local-lock-test",
	})
	const project, stackName = "scaleway-ttl-lock-test", "dev"
	program := func(pctx *pulumi.Context) error {
		return pctx.RegisterComponentResource("test:ttl:Component", "workload", &pulumi.ResourceState{})
	}

	stack, err := auto.NewStackInlineSource(ctx, stackName, project, program, workspace)
	require.NoError(t, err)
	_, err = stack.Up(ctx)
	require.NoError(t, err)

	locked, release := make(chan struct{}), make(chan struct{})
	signalLocked := sync.OnceFunc(func() { close(locked) })
	releaseLock := sync.OnceFunc(func() { close(release) })
	defer releaseLock()
	holder, err := auto.SelectStackInlineSource(ctx, stackName, project, func(pctx *pulumi.Context) error {
		signalLocked()
		select {
		case <-release:
			return program(pctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}, workspace)
	require.NoError(t, err)

	holderDone := make(chan error, 1)
	go func() {
		_, err := holder.Up(ctx)
		holderDone <- err
	}()
	select {
	case <-locked:
	case err := <-holderDone:
		t.Fatalf("Pulumi up exited before holding the lock: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	attempts := 0
	err = retryScalewayDestroy(ctx, time.Millisecond, func() error {
		attempts++
		_, err := stack.Destroy(ctx, optdestroy.Remove())
		if attempts == 1 {
			require.True(t, auto.IsConcurrentUpdateError(err), "expected a real Pulumi lock conflict, got %v", err)
			releaseLock()
			require.NoError(t, <-holderDone)
		}
		return err
	})
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
}
