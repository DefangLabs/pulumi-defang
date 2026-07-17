// Package aliases is an offline harness for evaluating strategies to add
// pulumi.Aliases to the Go providers so stacks deployed by the old defang-mvp
// CD keep their resources (instead of delete+create) when redeployed with
// this repo's providers. See README.md for the scenario and the findings.
//
// It runs the real Pulumi engine against a file:// backend with the random
// provider standing in for cloud resources, so it needs no credentials — but
// it does need the pulumi CLI and network access (plugin download), so it is
// skipped under -short (the CI provider-test mode).
package aliases

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	// The old TS CD's Pulumi project was fixed (Pulumi.yaml "name: cd");
	// the Go cd uses the compose project name as the Pulumi project, so
	// every URN changes at the project segment on migration.
	oldProject = "cd"
	newProject = "kittens"
	stackName  = "teststack"

	// Old shared component the bootstrap resources hung off of.
	defangStackType = "defang-mvp:shared/ecs/defang:Defang"
	defangStackName = "defangStack"

	// New component types (mirrors provider/defangaws type tokens).
	projectType = "defang-aws:index:Project"
	serviceType = "defang-aws:index:Service"
	redisType   = "defang-aws:index:Redis"
)

// aliasMode selects how the new-shape program attaches aliases.
type aliasMode string

const (
	modeNone          aliasMode = "none"           // baseline: no aliases
	modeURN           aliasMode = "urn"            // explicit old URN per resource (x-defang-aliases style)
	modeSpec          aliasMode = "spec"           // Alias{Name/Type/Project/Parent} computed by convention
	modeParent        aliasMode = "parent-inherit" // alias only on components; children rely on inherited aliases
	modeParentProject aliasMode = "parent+project" // like parent-inherit, plus Alias{Project} on children
)

func randomArgs() *random.RandomStringArgs {
	return &random.RandomStringArgs{Length: pulumi.Int(8)}
}

// oldProgram models the defang-mvp CD stack shape: a shared DefangStack
// component owning the bootstrap resources, and per-service resources
// registered flat at the stack root (the old code created them without
// service-level parents; names came from safe_namings).
func oldProgram(ctx *pulumi.Context) error {
	ds := &pulumi.ResourceState{}
	if err := ctx.RegisterComponentResource(defangStackType, defangStackName, ds); err != nil {
		return err
	}
	for _, name := range []string{"vpc", "cluster"} {
		if _, err := random.NewRandomString(ctx, name, randomArgs(), pulumi.Parent(ds)); err != nil {
			return err
		}
	}
	for _, name := range []string{"web-task", "web-sg", "cache-subnet-group", "cache-cluster"} {
		if _, err := random.NewRandomString(ctx, name, randomArgs()); err != nil {
			return err
		}
	}
	return nil
}

// oldURN reconstructs an old-stack URN from the naming convention — this is
// what provider code could compute mechanically at migration time.
func oldURN(resType, name string, parentType string) string {
	qualified := resType
	if parentType != "" {
		qualified = parentType + "$" + resType
	}
	return fmt.Sprintf("urn:pulumi:%s::%s::%s::%s", stackName, oldProject, qualified, name)
}

// newProgram models this repo's shape: everything nested under a Project
// component, per-service resources under Service/Redis components. The
// aliasFor callback returns the alias options to attach per logical resource.
func newProgram(mode aliasMode, urns map[string]string) pulumi.RunFunc {
	aliasFor := func(name string) []pulumi.ResourceOption {
		return aliasesFor(mode, urns, name)
	}
	return func(ctx *pulumi.Context) error {
		proj := &pulumi.ResourceState{}
		if err := ctx.RegisterComponentResource(projectType, newProject, proj, aliasFor(newProject)...); err != nil {
			return err
		}
		for _, name := range []string{"vpc", "cluster"} {
			opts := append([]pulumi.ResourceOption{pulumi.Parent(proj)}, aliasFor(name)...)
			if _, err := random.NewRandomString(ctx, name, randomArgs(), opts...); err != nil {
				return err
			}
		}

		web := &pulumi.ResourceState{}
		if err := ctx.RegisterComponentResource(serviceType, "web", web,
			append([]pulumi.ResourceOption{pulumi.Parent(proj)}, aliasFor("web")...)...); err != nil {
			return err
		}
		for _, name := range []string{"web-task", "web-sg"} {
			opts := append([]pulumi.ResourceOption{pulumi.Parent(web)}, aliasFor(name)...)
			if _, err := random.NewRandomString(ctx, name, randomArgs(), opts...); err != nil {
				return err
			}
		}

		cache := &pulumi.ResourceState{}
		if err := ctx.RegisterComponentResource(redisType, "cache", cache,
			append([]pulumi.ResourceOption{pulumi.Parent(proj)}, aliasFor("cache")...)...); err != nil {
			return err
		}
		for _, name := range []string{"cache-subnet-group", "cache-cluster"} {
			opts := append([]pulumi.ResourceOption{pulumi.Parent(cache)}, aliasFor(name)...)
			if _, err := random.NewRandomString(ctx, name, randomArgs(), opts...); err != nil {
				return err
			}
		}
		return nil
	}
}

// aliasesFor maps each new-shape logical resource to its alias options under
// the given mode. urns is the by-name URN index of the exported old stack
// (used only by modeURN; the other modes compute alias specs by convention).
func aliasesFor(mode aliasMode, urns map[string]string, name string) []pulumi.ResourceOption {
	alias := func(a pulumi.Alias) []pulumi.ResourceOption {
		return []pulumi.ResourceOption{pulumi.Aliases([]pulumi.Alias{a})}
	}

	switch mode {
	case modeNone:
		return nil

	case modeURN:
		// The Project component maps onto the old DefangStack; web/cache
		// components are new. Everything else matches by logical name.
		key := name
		if name == newProject {
			key = defangStackName
		}
		urn, ok := urns[key]
		if !ok {
			return nil
		}
		return alias(pulumi.Alias{URN: pulumi.URN(urn)})

	case modeSpec:
		switch name {
		case newProject: // was the DefangStack component, different name+type, at the root
			return alias(pulumi.Alias{
				Name:     pulumi.String(defangStackName),
				Type:     pulumi.String(defangStackType),
				Project:  pulumi.String(oldProject),
				NoParent: pulumi.Bool(true),
			})
		case "web", "cache": // new components, nothing to alias
			return nil
		case "vpc", "cluster": // were children of the DefangStack component
			return alias(pulumi.Alias{
				Project:   pulumi.String(oldProject),
				ParentURN: pulumi.URN(oldURN(defangStackType, defangStackName, "")),
			})
		default: // per-service resources were flat at the stack root
			return alias(pulumi.Alias{
				Project:  pulumi.String(oldProject),
				NoParent: pulumi.Bool(true),
			})
		}

	case modeParent, modeParentProject:
		// Alias only the components; children have no explicit URN/parent
		// mapping and rely on the SDK's inherited-alias computation.
		if name == newProject {
			return alias(pulumi.Alias{
				Name:     pulumi.String(defangStackName),
				Type:     pulumi.String(defangStackType),
				Project:  pulumi.String(oldProject),
				NoParent: pulumi.Bool(true),
			})
		}
		if name == "web" || name == "cache" {
			return nil
		}
		if mode == modeParentProject {
			return alias(pulumi.Alias{Project: pulumi.String(oldProject)})
		}
		return nil
	}
	return nil
}

// envs returns the per-workspace environment for a file backend. The legacy
// (pre-project-scoped) DIY layout is pinned because that's what the defang CD
// buckets use: the stack file path has no project segment, so the new program
// reads the old project's state in place — no relocation step. A backend with
// the modern project-scoped layout would shelve the old state under
// stacks/cd/ and the new project would start empty.
func envs(backendDir string) auto.LocalWorkspaceOption {
	return auto.EnvVars(map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + backendDir,
		"PULUMI_CONFIG_PASSPHRASE": "test",
		// current + former name of the same knob, for pulumi-version drift
		"PULUMI_DIY_BACKEND_LEGACY_LAYOUT":        "true",
		"PULUMI_SELF_MANAGED_STATE_LEGACY_LAYOUT": "true",
	})
}

// urnsByName indexes the exported deployment's resource URNs by logical name.
func urnsByName(t *testing.T, deployment apitype.UntypedDeployment) map[string]string {
	t.Helper()
	var d apitype.DeploymentV3
	if err := json.Unmarshal(deployment.Deployment, &d); err != nil {
		t.Fatalf("unmarshal deployment: %v", err)
	}
	urns := make(map[string]string, len(d.Resources))
	for _, r := range d.Resources {
		urns[r.URN.Name()] = string(r.URN)
	}
	return urns
}

// preview runs a preview and tallies planned ops per resource, excluding the
// stack and provider pseudo-resources.
func preview(t *testing.T, ctx context.Context, stack auto.Stack) map[apitype.OpType]int {
	t.Helper()
	evtCh := make(chan events.EngineEvent)
	counts := make(map[apitype.OpType]int)
	var lines []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range evtCh {
			pre := evt.ResourcePreEvent
			if pre == nil {
				continue
			}
			typ := pre.Metadata.Type
			if typ == "pulumi:pulumi:Stack" || strings.HasPrefix(typ, "pulumi:providers:") {
				continue
			}
			counts[pre.Metadata.Op]++
			lines = append(lines, fmt.Sprintf("  %-7s %s", pre.Metadata.Op, pre.Metadata.URN))
		}
	}()
	if _, err := stack.Preview(ctx, optpreview.EventStreams(evtCh)); err != nil {
		t.Fatalf("preview: %v", err)
	}
	<-done
	for _, l := range lines {
		t.Log(l)
	}
	return counts
}

func TestAliasMigrationScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("needs the real pulumi engine; run without -short")
	}
	if _, err := exec.LookPath("pulumi"); err != nil {
		t.Skip("pulumi CLI not on PATH")
	}
	ctx := context.Background()

	// Deploy the old-shape stack for real against a throwaway file backend.
	// The scenarios below select the SAME stack from the same backend under
	// the NEW project name — with the legacy DIY layout (see envs) the stack
	// file has no project segment, so the old state is found in place, exactly
	// like a defang CD bucket. The export below is only used to harvest the
	// old URNs for the explicit-URN mode.
	backend := t.TempDir()
	oldStack, err := auto.UpsertStackInlineSource(ctx, stackName, oldProject, oldProgram, envs(backend))
	if err != nil {
		t.Fatalf("upsert old stack: %v", err)
	}
	if err := oldStack.Workspace().InstallPlugin(ctx, "random", "v4.19.2"); err != nil {
		t.Fatalf("install random plugin: %v", err)
	}
	if _, err := oldStack.Up(ctx, optup.SuppressProgress()); err != nil {
		t.Fatalf("up old stack: %v", err)
	}
	deployment, err := oldStack.Export(ctx)
	if err != nil {
		t.Fatalf("export old stack: %v", err)
	}
	urns := urnsByName(t, deployment)

	// The old stack holds 7 resources (DefangStack component + 6 randoms);
	// the new shape has 9 (Project + 2 infra + Service/Redis components + 4).
	scenarios := []struct {
		mode  aliasMode
		check func(t *testing.T, ops map[apitype.OpType]int)
	}{
		{modeNone, func(t *testing.T, ops map[apitype.OpType]int) {
			t.Helper()
			if ops[apitype.OpSame] != 0 || ops[apitype.OpDelete] != 7 || ops[apitype.OpCreate] != 9 {
				t.Errorf("baseline should replace the world: %v", ops)
			}
		}},
		{modeURN, func(t *testing.T, ops map[apitype.OpType]int) {
			t.Helper()
			if ops[apitype.OpDelete] != 0 || ops[apitype.OpSame] != 7 {
				t.Errorf("URN aliases should preserve all 7 old resources: %v", ops)
			}
		}},
		{modeSpec, func(t *testing.T, ops map[apitype.OpType]int) {
			t.Helper()
			if ops[apitype.OpDelete] != 0 || ops[apitype.OpSame] != 7 {
				t.Errorf("spec aliases should preserve all 7 old resources: %v", ops)
			}
		}},
		// The two inheritance modes are exploratory: they document how far
		// component-level aliases get you (see README findings). No hard
		// assertions — the tally in the test log is the deliverable.
		{modeParent, nil},
		{modeParentProject, nil},
	}

	for _, sc := range scenarios {
		t.Run(string(sc.mode), func(t *testing.T) {
			stack, err := auto.UpsertStackInlineSource(ctx, stackName, newProject,
				newProgram(sc.mode, urns), envs(backend))
			if err != nil {
				t.Fatalf("upsert new stack: %v", err)
			}
			if err := stack.Workspace().InstallPlugin(ctx, "random", "v4.19.2"); err != nil {
				t.Fatalf("install random plugin: %v", err)
			}
			ops := preview(t, ctx, stack)
			t.Logf("%s: %v", sc.mode, ops)
			if sc.check != nil {
				sc.check(t, ops)
			}
		})
	}
}
