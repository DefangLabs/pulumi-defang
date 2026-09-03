package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/require"
)

type fakeMigrationPreviewer struct {
	calls  int
	events []events.EngineEvent
	err    error
	opts   optpreview.Options
}

func (f *fakeMigrationPreviewer) Preview(
	_ context.Context,
	opts ...optpreview.Option,
) (auto.PreviewResult, error) {
	f.calls++
	for _, opt := range opts {
		opt.ApplyOption(&f.opts)
	}
	for _, event := range f.events {
		for _, stream := range f.opts.EventStreams {
			stream <- event
		}
	}
	// Exercise the production helper's error-path shutdown: Automation API can
	// return before creating an event watcher, in which case it cannot close
	// caller-provided channels.
	if f.err == nil {
		for _, stream := range f.opts.EventStreams {
			close(stream)
		}
	}
	return auto.PreviewResult{}, f.err
}

func testPreviewResource(n int) migrationPreviewResource {
	return migrationPreviewResource{
		oldURN: resource.URN(fmt.Sprintf(
			"urn:pulumi:beta::my-app::gcp:sql/databaseInstance:DatabaseInstance::legacy-db-%d", n,
		)),
		newURN: resource.URN(fmt.Sprintf(
			"urn:pulumi:beta::my-app::defang-gcp:index:Project$defang-gcp:index:Postgres$"+
				"gcp:sql/databaseInstance:DatabaseInstance::db-%d", n,
		)),
	}
}

func previewResourceEvent(
	protected migrationPreviewResource,
	op apitype.OpType,
) events.EngineEvent {
	return events.EngineEvent{EngineEvent: apitype.EngineEvent{
		ResourcePreEvent: &apitype.ResourcePreEvent{Metadata: apitype.StepEventMetadata{
			Op:   op,
			URN:  string(protected.newURN),
			Type: string(protected.newURN.Type()),
			Old: &apitype.StepEventStateMetadata{
				URN:     string(protected.oldURN),
				Type:    string(protected.oldURN.Type()),
				Inputs:  map[string]any{"password": "TOP-SECRET"},
				Outputs: map[string]any{"password": "TOP-SECRET"},
			},
			New: &apitype.StepEventStateMetadata{
				URN:     string(protected.newURN),
				Type:    string(protected.newURN.Type()),
				Inputs:  map[string]any{"password": "TOP-SECRET"},
				Outputs: map[string]any{"password": "TOP-SECRET"},
			},
		}},
	}}
}

func TestMigrationPreviewAllowsProviderSameAndUpdate(t *testing.T) {
	first, second := testPreviewResource(1), testPreviewResource(2)
	previewer := &fakeMigrationPreviewer{events: []events.EngineEvent{
		previewResourceEvent(first, apitype.OpSame),
		previewResourceEvent(second, apitype.OpUpdate),
	}}

	err := verifyMigrationPreview(t.Context(), previewer, legacyStatePreparation{
		resources: []migrationPreviewResource{first, second},
	}, "test-agent", "never", nil)
	require.NoError(t, err)
	require.Equal(t, 1, previewer.calls)
}

func TestMigrationPreviewBlocksEveryDestructiveProviderOperation(t *testing.T) {
	for _, op := range []apitype.OpType{
		apitype.OpReplace,
		apitype.OpDelete,
		apitype.OpDeleteReplaced,
		apitype.OpCreateReplacement,
		apitype.OpCreate,
		apitype.OpReadReplacement,
		apitype.OpDiscardReplaced,
		apitype.OpRemovePendingReplace,
		apitype.OpImportReplacement,
	} {
		t.Run(string(op), func(t *testing.T) {
			protected := testPreviewResource(1)
			previewer := &fakeMigrationPreviewer{events: []events.EngineEvent{
				previewResourceEvent(protected, op),
			}}

			err := verifyMigrationPreview(t.Context(), previewer, legacyStatePreparation{
				resources: []migrationPreviewResource{protected},
			}, "test-agent", "never", nil)
			var destructiveErr *destructiveMigrationPreviewError
			require.ErrorAs(t, err, &destructiveErr)
			require.Contains(t, err.Error(), string(op))
			require.Contains(t, err.Error(), "Nothing has been changed")
			require.NotContains(t, err.Error(), "TOP-SECRET")
		})
	}
}

func TestMigrationPreviewUsesSameTargetsAndTargetDependentsAsUp(t *testing.T) {
	protected := testPreviewResource(1)
	targets := []string{"urn:pulumi:beta::my-app::pkg:index:Component::target"}
	previewer := &fakeMigrationPreviewer{events: []events.EngineEvent{
		previewResourceEvent(protected, apitype.OpSame),
	}}

	require.NoError(t, verifyMigrationPreview(t.Context(), previewer, legacyStatePreparation{
		resources: []migrationPreviewResource{protected},
	}, "test-agent", "never", targets))
	require.Equal(t, targets, previewer.opts.Target)
	require.True(t, previewer.opts.TargetDependents)
	require.True(t, previewer.opts.SuppressOutputs)
}

func TestMigrationPreviewFailureAndEventErrorsFailClosedWithoutSecrets(t *testing.T) {
	protected := testPreviewResource(1)
	for _, tt := range []struct {
		name      string
		previewer *fakeMigrationPreviewer
	}{
		{
			name:      "preview command",
			previewer: &fakeMigrationPreviewer{err: errors.New("provider failed with password TOP-SECRET")},
		},
		{
			name: "event stream",
			previewer: &fakeMigrationPreviewer{events: []events.EngineEvent{
				{Error: errors.New("event decode failed with password TOP-SECRET")},
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			originalLogger := stderrLogger
			stderrLogger = &logs
			t.Cleanup(func() { stderrLogger = originalLogger })

			err := verifyMigrationPreview(t.Context(), tt.previewer, legacyStatePreparation{
				resources: []migrationPreviewResource{protected},
			}, "test-agent", "never", nil)
			var previewErr *migrationPreviewError
			require.ErrorAs(t, err, &previewErr)
			require.NotContains(t, err.Error(), "TOP-SECRET")
			require.NotContains(t, logs.String(), "TOP-SECRET")
		})
	}
}

func TestMigrationPreviewExactStackOverrideIsLoudBoundedAndRedacted(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := stderrLogger
	stderrLogger = &logs
	t.Cleanup(func() { stderrLogger = originalLogger })

	resources := make([]migrationPreviewResource, 7)
	events := make([]events.EngineEvent, 7)
	for i := range resources {
		resources[i] = testPreviewResource(i)
		events[i] = previewResourceEvent(resources[i], apitype.OpReplace)
	}
	previewer := &fakeMigrationPreviewer{events: events}

	require.NoError(t, verifyMigrationPreview(t.Context(), previewer, legacyStatePreparation{
		override:  true,
		resources: resources,
	}, "test-agent", "never", nil))
	log := logs.String()
	require.Contains(t, log, "exact-stack takeover override permits up")
	require.Contains(t, log, "...and 2 more")
	require.NotContains(t, log, "db-5")
	require.NotContains(t, log, "db-6")
	require.NotContains(t, log, "TOP-SECRET")
}

func TestMigrationPreviewExactStackOverrideCanBypassSanitizedPreviewFailure(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := stderrLogger
	stderrLogger = &logs
	t.Cleanup(func() { stderrLogger = originalLogger })

	previewer := &fakeMigrationPreviewer{err: errors.New("provider failed with password TOP-SECRET")}
	require.NoError(t, verifyMigrationPreview(t.Context(), previewer, legacyStatePreparation{
		override:  true,
		resources: []migrationPreviewResource{testPreviewResource(1)},
	}, "test-agent", "never", nil))
	require.Contains(t, logs.String(), "exact-stack takeover override")
	require.NotContains(t, logs.String(), "TOP-SECRET")
}

func TestNonMigrationUpDoesNotRunAnExtraPreview(t *testing.T) {
	previewer := &fakeMigrationPreviewer{}
	require.NoError(t, verifyMigrationPreview(
		t.Context(), previewer, legacyStatePreparation{}, "test-agent", "never", nil,
	))
	require.Zero(t, previewer.calls)
}

func TestPrepareLegacyStateTracksExactOldAndNewDataBearingURNs(t *testing.T) {
	for _, tt := range []struct {
		name, fixture, blockedType, cloud, compose string
		want                                       []string
	}{
		{
			name:        "Cloud SQL and Memorystore",
			fixture:     "legacy-state-gcp.json",
			blockedType: "cloudbuild:index:CloudBuild",
			cloud:       cloudGCP,
			compose:     gcpManagedCompose,
			want: []string{
				"urn:pulumi:beta::my-app::gcp:sql/databaseInstance:DatabaseInstance::my-app-postgres-service-postgres -> " +
					"urn:pulumi:beta::my-app::defang-gcp:index:Project$defang-gcp:index:Postgres$" +
					"gcp:sql/databaseInstance:DatabaseInstance::postgres-service",
				"urn:pulumi:beta::my-app::gcp:redis/instance:Instance::my-app-redis-service-redis -> " +
					"urn:pulumi:beta::my-app::defang-gcp:index:Project$defang-gcp:index:Redis$" +
					"gcp:redis/instance:Instance::redis-service",
			},
		},
		{
			name:        "RDS and ElastiCache",
			fixture:     "legacy-state-aws.json",
			blockedType: "pulumi-nodejs:dynamic:Resource",
			cloud:       cloudAWS,
			compose:     awsManagedCompose,
			want: []string{
				"urn:pulumi:s***::cd::aws:rds/instance:Instance::e-cd-s***-postgres -> " +
					"urn:pulumi:beta::my-app::defang-aws:index:Project$defang-aws:index:Postgres$" +
					"aws:rds/instance:Instance::postgres",
				"urn:pulumi:s***::cd::aws:elasticache/replicationGroup:ReplicationGroup::redis -> " +
					"urn:pulumi:beta::my-app::defang-aws:index:Project$defang-aws:index:Redis$" +
					"aws:elasticache/replicationGroup:ReplicationGroup::redis",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			deployment := fixtureWithoutTypes(t, tt.fixture, tt.blockedType)
			preparation, err := prepareLegacyState(
				t.Context(), fakeStack{deployment: deployment}, "", testProjectName, testStackName,
				[]byte(tt.compose), tt.cloud, configFor(tt.cloud), program.ServiceAliases{}, true,
			)
			require.NoError(t, err)

			identities := make([]string, 0, len(preparation.resources))
			for _, protected := range preparation.resources {
				identities = append(identities, string(protected.oldURN)+" -> "+string(protected.newURN))
			}
			require.ElementsMatch(t, tt.want, identities)
		})
	}
}
