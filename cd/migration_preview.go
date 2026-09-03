package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// legacyStatePreparation is the non-secret result of inspecting an existing
// stack. resources contains only adopted data-bearing resource identities; it
// is empty for ordinary current-CD stacks, so they do not pay for an extra
// preview before up.
type legacyStatePreparation struct {
	override  bool
	resources []migrationPreviewResource
}

// migrationPreviewResource keeps both sides of an alias. Pulumi may report
// the legacy URN in Old and the current URN in New/URN, depending on which
// replacement step is being described, so matching only one side is unsafe.
type migrationPreviewResource struct {
	oldURN resource.URN
	newURN resource.URN
}

func migrationPreviewResources(
	adoptions []migrationAdoption,
	projectName, stackName, cloud string,
) []migrationPreviewResource {
	pkg := thisCDPackages[cloud]
	resources := make([]migrationPreviewResource, 0, len(adoptions))
	for _, adoption := range adoptions {
		parentType := tokens.Type(pkg + ":index:Project$" + pkg + ":index:" + string(adoption.spec.serviceKind))
		resources = append(resources, migrationPreviewResource{
			oldURN: adoption.resource.urn,
			newURN: resource.NewURN(
				tokens.QName(stackName),
				tokens.PackageName(projectName),
				parentType,
				tokens.Type(adoption.spec.resourceType),
				adoption.service+adoption.spec.currentSuffix,
			),
		})
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].newURN < resources[j].newURN
	})
	return resources
}

type stackPreviewer interface {
	Preview(ctx context.Context, opts ...optpreview.Option) (auto.PreviewResult, error)
}

type migrationPreviewStep struct {
	op  apitype.OpType
	urn resource.URN
}

func (s migrationPreviewStep) display() string {
	return fmt.Sprintf("%s %s::%s", s.op, s.urn.QualifiedType(), s.urn.Name())
}

type migrationPreviewCollector struct {
	events chan events.EngineEvent
	stop   chan struct{}
	done   chan struct{}
	steps  []migrationPreviewStep
	bad    bool
}

func newMigrationPreviewCollector(resources []migrationPreviewResource) *migrationPreviewCollector {
	c := &migrationPreviewCollector{
		events: make(chan events.EngineEvent),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go func() {
		defer close(c.done)
		seen := map[string]bool{}
		for {
			select {
			case event, ok := <-c.events:
				if !ok {
					return
				}
				if event.Error != nil {
					// Event errors can contain command output. Record only that the
					// stream was incomplete, never the error text.
					c.bad = true
					continue
				}
				if event.ResourcePreEvent == nil {
					continue
				}
				metadata := event.ResourcePreEvent.Metadata
				matchedURN, affected := affectedMigrationResource(metadata, resources)
				if !affected || migrationPreviewOpIsSafe(metadata.Op) {
					continue
				}
				key := string(metadata.Op) + "\x00" + string(matchedURN)
				if !seen[key] {
					seen[key] = true
					c.steps = append(c.steps, migrationPreviewStep{op: metadata.Op, urn: matchedURN})
				}
			case <-c.stop:
				return
			}
		}
	}()
	return c
}

func (c *migrationPreviewCollector) abort() {
	close(c.stop)
	<-c.done
}

func (c *migrationPreviewCollector) finish() ([]migrationPreviewStep, bool) {
	// A successful Automation API preview closes every event channel before it
	// returns. Waiting here proves the complete provider-backed plan was seen.
	<-c.done
	sort.Slice(c.steps, func(i, j int) bool {
		if c.steps[i].urn == c.steps[j].urn {
			return c.steps[i].op < c.steps[j].op
		}
		return c.steps[i].urn < c.steps[j].urn
	})
	return c.steps, c.bad
}

func affectedMigrationResource(
	metadata apitype.StepEventMetadata,
	resources []migrationPreviewResource,
) (resource.URN, bool) {
	urns := []string{metadata.URN}
	if metadata.Old != nil {
		urns = append(urns, metadata.Old.URN)
	}
	if metadata.New != nil {
		urns = append(urns, metadata.New.URN)
	}
	for _, protected := range resources {
		for _, urn := range urns {
			switch resource.URN(urn) {
			case protected.oldURN:
				return protected.oldURN, true
			case protected.newURN:
				return protected.newURN, true
			}
		}
	}
	return "", false
}

func migrationPreviewOpIsSafe(op apitype.OpType) bool {
	return op == apitype.OpSame || op == apitype.OpUpdate
}

func migrationPreviewOptions(
	userAgent, color string,
	targets []string,
	eventStream chan<- events.EngineEvent,
) []optpreview.Option {
	return []optpreview.Option{
		optpreview.UserAgent(userAgent),
		optpreview.Color(color),
		optpreview.SuppressProgress(),
		optpreview.SuppressOutputs(),
		optpreview.EventStreams(eventStream),
		optpreview.TargetDependents(),
		optpreview.Target(targets),
	}
}

func verifyMigrationPreview(
	ctx context.Context,
	stack stackPreviewer,
	preparation legacyStatePreparation,
	userAgent, color string,
	targets []string,
) error {
	if len(preparation.resources) == 0 {
		return nil
	}

	warn(fmt.Sprintf(
		"Checking %d adopted data-bearing resource(s) with a provider-backed Pulumi preview before up.",
		len(preparation.resources),
	))
	collector := newMigrationPreviewCollector(preparation.resources)
	_, err := stack.Preview(ctx, migrationPreviewOptions(userAgent, color, targets, collector.events)...)
	if err != nil {
		collector.abort()
		if preparation.override {
			warn("Warning: the migration safety preview failed, but the exact-stack takeover override " +
				"permits up to continue. Existing data may be replaced or deleted.")
			return nil
		}
		return &migrationPreviewError{reason: "the provider-backed preview failed"}
	}

	steps, streamFailed := collector.finish()
	if streamFailed {
		if preparation.override {
			warn("Warning: the migration safety preview event stream was incomplete, but the exact-stack " +
				"takeover override permits up to continue. Existing data may be replaced or deleted.")
			return nil
		}
		return &migrationPreviewError{reason: "the provider-backed preview event stream was incomplete"}
	}
	if len(steps) != 0 {
		if preparation.override {
			warnMigrationPreviewSteps(
				fmt.Sprintf(
					"Warning: the exact-stack takeover override permits up despite %d destructive operation(s) "+
						"in the provider-backed preview. Existing data may be replaced or deleted.",
					len(steps),
				),
				steps,
			)
			return nil
		}
		return &destructiveMigrationPreviewError{steps: steps}
	}

	warn(fmt.Sprintf(
		"Provider-backed migration safety preview passed for %d adopted data-bearing resource(s).",
		len(preparation.resources),
	))
	return nil
}

func warnMigrationPreviewSteps(header string, steps []migrationPreviewStep) {
	warn(header)
	for i, step := range steps {
		if i == maxReportedResources {
			warn(fmt.Sprintf("  ...and %d more", len(steps)-maxReportedResources))
			break
		}
		warn("  " + step.display())
	}
}

type migrationPreviewError struct {
	reason string
}

func (e *migrationPreviewError) Error() string {
	return "cannot verify that the adopted data-bearing resources are safe to update: " + e.reason +
		". No cloud resources have been changed. Run `preview` for diagnostics, then retry or contact " +
		"Defang support. See " +
		migrationRunbook
}

type destructiveMigrationPreviewError struct {
	steps []migrationPreviewStep
}

func (e *destructiveMigrationPreviewError) Error() string {
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"the provider-backed preview would replace or delete adopted data-bearing infrastructure (%d operation(s)):\n",
		len(e.steps),
	)
	for i, step := range e.steps {
		if i == maxReportedResources {
			fmt.Fprintf(&b, "  ...and %d more\n", len(e.steps)-maxReportedResources)
			break
		}
		fmt.Fprintf(&b, "  %s\n", step.display())
	}
	b.WriteString("\nNo cloud resources have been changed. Correct the immutable configuration difference " +
		"or use a blue/green migration. See " + migrationRunbook)
	return b.String()
}
