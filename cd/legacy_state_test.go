package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/require"
)

// The state fixtures under testdata/ are derived from real deployments, not
// hand-written, so they carry the URN shapes this guard actually has to tell
// apart:
//
//   - legacy-state-gcp.json  defang-mvp's Go GCP CD, from that repo's
//     fabric/pkg/estimator/testdata/gcp-prod.log
//   - legacy-state-aws.json  defang-mvp's TypeScript CD, from
//     fabric/pkg/estimator/testdata/potato.log (real stack raphaeltm-prod1)
//   - new-state-{aws,gcp,azure}.json  this CD, from the recorded
//     preview-events-*.json in this directory
//   - mixed-state-gcp.json  a takeover that ran and failed: this CD's resources
//     on top of surviving legacy ones
//   - empty-state.json  a stack that was created but never deployed
func fixtureDeployment(t *testing.T, name string) apitype.UntypedDeployment {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	var doc apitype.UntypedDeployment
	require.NoError(t, json.Unmarshal(data, &doc))
	return doc
}

func loadFixture(t *testing.T, name string) []apitype.ResourceV3 {
	t.Helper()
	var snapshot apitype.DeploymentV3
	require.NoError(t, json.Unmarshal(fixtureDeployment(t, name).Deployment, &snapshot))
	return snapshot.Resources
}

// The whole guard, over every captured state, through the real entry point.
func TestCheckLegacyStateOnRealStates(t *testing.T) {
	tests := []struct {
		fixture string
		block   bool
	}{
		{"empty-state.json", false},
		{"new-state-aws.json", false},
		{"new-state-gcp.json", false},
		{"new-state-azure.json", false},
		{"legacy-state-gcp.json", true},
		{"legacy-state-aws.json", true},
		// A denylist of known-legacy types would get the mixed state wrong:
		// this CD's marker resources are present, so "did our CD write this?"
		// answers yes, yet the legacy resources are still there to be deleted.
		{"mixed-state-gcp.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			err := checkLegacyState(context.Background(), fakeStack{deployment: fixtureDeployment(t, tt.fixture)}, "", testProjectName, testStackName)
			foreign := foreignResources(loadFixture(t, tt.fixture))
			if !tt.block {
				require.NoError(t, err)
				require.Emptyf(t, foreign, "state must pass unchanged, but %d resources were called foreign: %v",
					len(foreign), foreign)
				return
			}
			require.Error(t, err)
			require.NotEmpty(t, foreign)
			// None of this CD's own resources may be reported, even in the
			// mixed state -- the message has to point at the legacy ones.
			for _, urn := range foreign {
				require.NotContains(t, string(urn), "defang-aws:")
				require.NotContains(t, string(urn), "defang-gcp:")
				require.NotContains(t, string(urn), "defang-azure:")
			}
		})
	}
}

// Every real legacy resource type token seen in the two legacy fixtures must be
// flagged. This pins the discriminator against the actual type tokens rather
// than a summary of them.
func TestForeignResourcesFlagsKnownLegacyTypes(t *testing.T) {
	wantFlagged := []string{
		// Legacy TypeScript CD component type.
		"defang-mvp:shared/ecs/defang:Defang",
		// The TypeScript CD is the only Node.js program; this CD is Go.
		"pulumi-nodejs:dynamic:Resource",
		// The legacy GCP CD's own in-tree provider.
		"cloudbuild:index:CloudBuild",
		// Neither legacy CD parented its cloud resources under a component of
		// its own, so they are flat cloud types directly under the stack.
		"gcp:sql/databaseInstance:DatabaseInstance",
		"gcp:cloudrunv2/service:Service",
		"gcp:redis/instance:Instance",
		"aws:ecs/service:Service",
		"awsx:ec2:Vpc",
		// The resources whose deletion is the whole point of this guard.
		"aws:rds/instance:Instance",
		"aws:elasticache/replicationGroup:ReplicationGroup",
	}

	flagged := map[string]bool{}
	for _, fixture := range []string{"legacy-state-gcp.json", "legacy-state-aws.json"} {
		for _, urn := range foreignResources(loadFixture(t, fixture)) {
			for _, part := range strings.Split(string(urn.QualifiedType()), "$") {
				flagged[part] = true
			}
		}
	}
	for _, typ := range wantFlagged {
		require.Truef(t, flagged[typ], "legacy type %q was not flagged as foreign", typ)
	}
}

// Legacy component types that no captured state in this repo happens to
// contain, so they are checked against the type tokens in the source that
// creates them. DigitalOcean matters because this CD does not support that
// cloud at all: a DO state must abort with the migration message rather than
// with "unsupported provider".
func TestForeignResourcesFlagsLegacyComponentsWithoutAFixture(t *testing.T) {
	// Type tokens read from defang-mvp at commit d191a5f6:
	//   pulumi/cd/do/kaniko_image.ts:29   pulumi/shared/aws/vpcx.ts:33
	//   pulumi/ecs/loki.ts:75             pulumi/cd/aws/tenant_stack.ts:59
	legacyTypes := []string{
		"defang-mvp:cd/do/kaniko_image:KanikoImage",
		"defang-mvp:shared/vpc:Vpc",
		"defang-mvp:shared/ecs/loki:Loki",
		"defang-mvp:cd/tenant_stack:TenantStack",
		"digitalocean:index/app:App",
		"digitalocean:index/databaseCluster:DatabaseCluster",
	}
	for _, typ := range legacyTypes {
		t.Run(typ, func(t *testing.T) {
			state := []apitype.ResourceV3{
				mkRes("urn:pulumi:beta::my-app::pulumi:pulumi:Stack::my-app-beta", ""),
				mkRes("urn:pulumi:beta::my-app::"+typ+"::thing",
					"urn:pulumi:beta::my-app::pulumi:pulumi:Stack::my-app-beta"),
			}
			require.Len(t, foreignResources(state), 1)
		})
	}
}

// "defang-mvp" and "defang-gcp" share a prefix. Matching on the package name
// rather than a string prefix keeps the legacy package out.
func TestForeignResourcesDoesNotConfuseDefangMvpWithThisCD(t *testing.T) {
	require.False(t, thisCDPackages["defang-mvp"])
	state := []apitype.ResourceV3{
		mkRes("urn:pulumi:beta::my-app::pulumi:pulumi:Stack::my-app-beta", ""),
		mkRes("urn:pulumi:beta::my-app::defang-mvp:shared/ecs/defang:Defang$aws:rds/instance:Instance::db",
			"urn:pulumi:beta::my-app::defang-mvp:shared/ecs/defang:Defang::defang"),
	}
	require.Len(t, foreignResources(state), 1)
}

const (
	testStackURN = "urn:pulumi:gcp::cd-test::pulumi:pulumi:Stack::cd-test-gcp"
	testProject  = "urn:pulumi:gcp::cd-test::defang-gcp:index:Project::cd-test"

	// The project and stack of the run under test, i.e. what PROJECT and STACK
	// would be. An override has to name exactly this pair.
	testProjectName = "my-app"
	testStackName   = "beta"
	testTarget      = testProjectName + "/" + testStackName
)

// mkRes builds the two fields the check reads, plus the type token the URN
// already implies, so the fixture matches what Pulumi writes.
func mkRes(urn, parent string) apitype.ResourceV3 {
	u := resource.URN(urn)
	return apitype.ResourceV3{URN: u, Type: u.Type(), Parent: resource.URN(parent)}
}

// This CD registers a few resources at the top level, outside the Project
// component, so they carry a plain cloud type token. They must not be mistaken
// for another CD's leftovers -- a stack with a TTL, or any successful deploy,
// has them. The allowlist is a name AND a position, so a resource that borrows
// one of those names but hangs off something else is still foreign.
func TestForeignResourcesAllowsTopLevelNamesOnlyAtTopLevel(t *testing.T) {
	const legacyParent = "urn:pulumi:gcp::cd-test::defang-mvp:shared/ecs/defang:Defang::defang"
	for name := range thisCDTopLevelNames {
		t.Run(name, func(t *testing.T) {
			// Deliberately a raw cloud type with no defang package.
			atTopLevel := mkRes("urn:pulumi:gcp::cd-test::gcp:cloudscheduler/job:Job::"+name, testStackURN)
			underLegacy := mkRes("urn:pulumi:gcp::cd-test::defang-mvp:shared/ecs/defang:Defang$"+
				"gcp:storage/bucketObject:BucketObject::"+name, legacyParent)
			base := []apitype.ResourceV3{mkRes(testStackURN, ""), mkRes(testProject, testStackURN)}

			require.Empty(t, foreignResources(append(base, atTopLevel)))
			require.Len(t, foreignResources(append(base, underLegacy)), 1)
		})
	}
}

// thisCDTopLevelNames is hand-maintained, and getting it wrong is expensive in
// one direction: a top-level resource missing from the list makes the guard
// abort every existing stack's next deploy. Nothing in the compiler catches
// that, and the state fixtures cannot either -- they were captured from runs
// with no TTL and no state URL, so they contain none of these resources. The
// recorded preview goldens cannot either: those tests are behind the
// `integration` build tag and skip without cloud credentials.
//
// So read the registrations back out of the source that makes them. cd/program
// is the CD's own Pulumi program: it holds no components of its own, so every
// resource it registers on the run context lands at the top level of the stack.
func TestThisCDTopLevelNamesCoversEveryRootRegistration(t *testing.T) {
	// e.g. `_, err = s3.NewBucketObject(ctx, "project-pb", &s3.BucketObjectArgs{`
	registration := regexp.MustCompile(`\.New([A-Za-z]+)\((?:ctx|pctx), ("[^"]*"|[\w.]+)[,)]`)
	// Some registrations name the resource through a constant, so collect the
	// package's string constants to resolve those.
	assignment := regexp.MustCompile(`(?m)^\s*(?:const\s+|var\s+)?(\w+)\s*=\s*"([^"]*)"`)

	sources, err := filepath.Glob(filepath.Join("program", "*.go"))
	require.NoError(t, err)

	bodies := map[string]string{}
	constants := map[string]string{}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		body, err := os.ReadFile(source)
		require.NoError(t, err)
		bodies[source] = string(body)
		for _, match := range assignment.FindAllStringSubmatch(string(body), -1) {
			constants[match[1]] = match[2]
		}
	}

	found := 0
	for source, body := range bodies {
		for _, match := range registration.FindAllStringSubmatch(body, -1) {
			kind, arg := match[1], match[2]
			// Providers are structural, and the Project component carries our
			// package in its type, so neither needs to be named.
			if kind == "Provider" || kind == "Project" {
				continue
			}
			name, err := strconv.Unquote(arg)
			if err != nil {
				name = constants[arg]
				require.NotEmptyf(t, name, "%s names a resource with %q, which this test cannot resolve", source, arg)
			}
			found++
			require.Truef(t, thisCDTopLevelNames[name],
				"%s registers top-level resource %q, but cd/legacy_state.go does not list it. Every "+
					"existing stack would abort on its next deploy. Add it to thisCDTopLevelNames.", source, name)
		}
	}
	// Pinned, not a floor: if a refactor stops the scan from matching a call it
	// used to match, this fails instead of silently checking nothing. Bump it
	// when you genuinely add or remove a top-level resource.
	require.Equal(t, 8, found, "the scan no longer matches the source it is meant to police")
}

// The allowlist is a name in a position, so a legacy CD that happened to name
// something "self-destruct" at the top level would slip through. Check the real
// legacy states do not.
func TestLegacyStatesDoNotUseOurTopLevelNames(t *testing.T) {
	for _, fixture := range []string{"legacy-state-gcp.json", "legacy-state-aws.json"} {
		t.Run(fixture, func(t *testing.T) {
			for _, res := range loadFixture(t, fixture) {
				if isStructural(res) || !isRootChild(res) {
					continue
				}
				require.Falsef(t, thisCDTopLevelNames[res.URN.Name()],
					"legacy resource %s collides with an allowlisted top-level name", res.URN)
			}
		})
	}
}

// The root stack and the provider resources exist in every state, this CD's and
// every older one's, so on their own they must not trip the guard. A stack
// whose deploy failed before it created anything looks like this.
func TestForeignResourcesIgnoresStructuralResources(t *testing.T) {
	state := []apitype.ResourceV3{
		mkRes(testStackURN, ""),
		mkRes("urn:pulumi:gcp::cd-test::pulumi:providers:gcp::default_8_26_0", ""),
		mkRes("urn:pulumi:gcp::cd-test::pulumi:providers:cloudbuild::default", ""),
		mkRes("urn:pulumi:gcp::cd-test::pulumi:providers:pulumi-nodejs::default", ""),
		mkRes("urn:pulumi:gcp::cd-test::pulumi:providers:defang-gcp::default", ""),
	}
	require.Empty(t, foreignResources(state))
}

// A user recipe may add resources of its own. They still have to sit under the
// Project component, so they are recognised; anything genuinely outside it is
// not something this CD can safely delete.
func TestForeignResourcesAllowsResourcesUnderOurProject(t *testing.T) {
	state := []apitype.ResourceV3{
		mkRes(testStackURN, ""),
		mkRes(testProject, testStackURN),
		mkRes("urn:pulumi:gcp::cd-test::defang-gcp:index:Project$custom:mod:Thing::x", testProject),
		mkRes("urn:pulumi:gcp::cd-test::defang-gcp:index:Project$defang-gcp:index:Service$gcp:sql/user:User::db",
			testProject),
	}
	require.Empty(t, foreignResources(state))
}

func TestForeignResourcesInEmptyDeployment(t *testing.T) {
	// A stack that exists but has never been deployed exports a deployment
	// with no resources key at all (verified against pulumi v3.259.0).
	fresh := apitype.UntypedDeployment{Version: 3, Deployment: json.RawMessage(
		`{"manifest":{"time":"2026-08-28T00:00:00Z","magic":"m","version":"v3.259.0"}}`)}
	foreign, err := foreignResourcesIn(fresh)
	require.NoError(t, err)
	require.Empty(t, foreign)

	// A stack that does not exist yet exports nothing at all.
	foreign, err = foreignResourcesIn(apitype.UntypedDeployment{})
	require.NoError(t, err)
	require.Empty(t, foreign)
}

// fakeStack drives checkLegacyState without a Pulumi backend.
type fakeStack struct {
	deployment apitype.UntypedDeployment
	err        error
}

func (f fakeStack) Export(context.Context) (apitype.UntypedDeployment, error) {
	return f.deployment, f.err
}

func TestLegacyStateErrorMessage(t *testing.T) {
	err := checkLegacyState(context.Background(), fakeStack{deployment: fixtureDeployment(t, "legacy-state-gcp.json")}, "", testProjectName, testStackName)
	var legacyErr *legacyStateError
	require.ErrorAs(t, err, &legacyErr)

	msg := err.Error()
	// The message has to say what was found, what it would cost, where the
	// runbook is, and how a human gets unblocked.
	require.Contains(t, msg, "older version of Defang")
	require.Contains(t, msg, "deleted")
	require.Contains(t, msg, "databases")
	require.Contains(t, msg, migrationRunbook)
	// It must say the stack is unchanged and steer the reader away from `down`,
	// which is not blocked and deletes everything it just listed.
	require.Contains(t, msg, "Nothing has been changed")
	require.Contains(t, msg, "`down`")
	// It must not name either override. Neither is settable by the person
	// reading this, so both are noise that invites a workaround.
	require.NotContains(t, msg, allowTakeoverConfigKey)
	require.NotContains(t, msg, allowTakeoverEnv)
	// It must name real detected resources, not just a count...
	require.Contains(t, msg, "gcp:")
	// ...but must not print the whole stack.
	require.LessOrEqual(t, strings.Count(msg, "\n"), 20)
	require.Contains(t, msg, "more")
}

// The override channel that is actually reachable: Defang sets the key in the
// tenant's recipe, and the CD receives it in the `up` payload. The recipe is
// free-form YAML or JSON, so every shape below is a real input.
func TestCheckLegacyStateRecipeOverride(t *testing.T) {
	tests := []struct {
		name   string
		recipe string
		allow  bool
	}{
		{"stack settings yaml", "config:\n  defang:allowLegacyStateTakeover: " + testTarget + "\n", true},
		{"stack settings quoted", "config:\n  defang:allowLegacyStateTakeover: \"" + testTarget + "\"\n", true},
		{"flat json", `{"defang:allowLegacyStateTakeover": {"value": "` + testTarget + `"}}`, true},

		{"empty", "", false},
		{"unrelated config", "config:\n  gcp:region: us-central1\n", false},
		{"empty string", `{"defang:allowLegacyStateTakeover": {"value": ""}}`, false},
		{"nested object", `{"defang:allowLegacyStateTakeover": {"objectValue": {"enabled": true}}}`, false},
		{"malformed", "config:\n\tthis is not: [valid", false},
		{"lookalike key", "config:\n  defang:allowlegacystatetakeover: " + testTarget + "\n", false},

		// A boolean used to be enough. It must not be, because a recipe is
		// shared by every project a tenant deploys in that mode.
		{"bare true", "config:\n  defang:allowLegacyStateTakeover: true\n", false},
		{"string true", `{"defang:allowLegacyStateTakeover": {"value": "true"}}`, false},

		// The whole point of naming the target: an authorization left in a
		// shared recipe must not free a stack nobody was migrating.
		{"another stack, same project", "config:\n  defang:allowLegacyStateTakeover: " + testProjectName + "/prod\n", false},
		{"another project, same stack", "config:\n  defang:allowLegacyStateTakeover: other/" + testStackName + "\n", false},
		{"project only", "config:\n  defang:allowLegacyStateTakeover: " + testProjectName + "\n", false},
		{"trailing slash", "config:\n  defang:allowLegacyStateTakeover: " + testTarget + "/\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLegacyState(context.Background(),
				fakeStack{deployment: fixtureDeployment(t, "legacy-state-gcp.json")}, tt.recipe, testProjectName, testStackName)
			if tt.allow {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// The break-glass channel, for a CD run started by hand with no recipe. It
// names its target for the same reason the recipe does.
func TestCheckLegacyStateEnvOverride(t *testing.T) {
	tests := []struct {
		value string
		allow bool
	}{
		{testTarget, true},
		{"", false},
		{"1", false},
		{"true", false},
		{testProjectName + "/prod", false},
		{"other/" + testStackName, false},
	}
	for _, tt := range tests {
		t.Run("value="+tt.value, func(t *testing.T) {
			t.Setenv(allowTakeoverEnv, tt.value)
			err := checkLegacyState(context.Background(),
				fakeStack{deployment: fixtureDeployment(t, "legacy-state-gcp.json")}, "", testProjectName, testStackName)
			if tt.allow {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// A run with no project or stack name must never match an override, however
// the value is shaped.
func TestCheckLegacyStateOverrideNeedsBothNames(t *testing.T) {
	for _, target := range []string{"/", "my-app/", "/beta", ""} {
		t.Run("target="+target, func(t *testing.T) {
			t.Setenv(allowTakeoverEnv, target)
			require.False(t, takeoverAllowed("", "", testStackName))
			require.False(t, takeoverAllowed("", testProjectName, ""))
			require.False(t, takeoverAllowed("", "", ""))
		})
	}
}

// thisCDTopLevelNames is the union of every name this CD has ever used, so
// dropping one deadlocks every stack that still holds that resource: `up`
// aborts, and `up` is the only thing that would have deleted it. The drift
// scanner cannot see this direction -- it only checks current source against
// the list -- so pin the historical names here.
func TestThisCDTopLevelNamesNeverShrinks(t *testing.T) {
	for _, name := range []string{"project-pb", "self-destruct", "defang-self-destruct", "self-destruct-starter"} {
		require.Truef(t, thisCDTopLevelNames[name],
			"%q was removed from thisCDTopLevelNames; stacks deployed by an older build still hold it and "+
				"would abort on their next deploy", name)
	}
}

// The env override must not survive into the scheduled self-destruct run: that
// trigger is a persisted resource, so a one-off permission frozen into it would
// apply to every future firing. This is the only thing tying the literal in
// cd/program/ttl.go to allowTakeoverEnv here -- the packages cannot share a
// constant, because program cannot import main.
func TestEnvOverrideIsNotFrozenIntoSelfDestruct(t *testing.T) {
	// SelfDestructEnv forwards every DEFANG_* variable by prefix, so the
	// override reaches the trigger unless it is excluded by name.
	env := program.SelfDestructEnv([]string{"PROJECT=app", allowTakeoverEnv + "=1"})
	require.Contains(t, env, "PROJECT")
	require.NotContains(t, env, allowTakeoverEnv)
}

// An unreadable state fails open: a second read of the state backend must not
// become a new way for every deploy to fail. Nothing here may panic -- the URN
// accessors assert on a malformed URN, and a failed assert kills the CD.
func TestCheckLegacyStateFailsOpen(t *testing.T) {
	tests := []struct {
		name  string
		stack fakeStack
	}{
		{"backend error", fakeStack{err: errors.New("could not export stack: connection reset")}},
		{"invalid json", fakeStack{deployment: deploymentOf(`{{{`)}},
		{"resource with no urn", fakeStack{deployment: deploymentOf(`{"resources":[{}]}`)}},
		{"null resource", fakeStack{deployment: deploymentOf(`{"resources":[null]}`)}},
		{"urn is not a urn", fakeStack{deployment: deploymentOf(`{"resources":[{"urn":"not-a-urn"}]}`)}},
		{"parent is not a urn", fakeStack{deployment: deploymentOf(
			`{"resources":[{"urn":"urn:pulumi:s::p::aws:rds/instance:Instance::db","parent":"garbage"}]}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				require.NoError(t, checkLegacyState(context.Background(), tt.stack, "", testProjectName, testStackName))
			})
		})
	}
}

// A deployment shape this cannot read must not decode to "no resources", because
// an empty result here means "go ahead and delete everything".
func TestCheckLegacyStateRejectsUnknownDeploymentVersion(t *testing.T) {
	// A V1/V2-shaped body decodes cleanly into DeploymentV3 as zero resources.
	legacyShape := apitype.UntypedDeployment{Version: 1, Deployment: json.RawMessage(
		`{"latest":{"resources":[{"urn":"urn:pulumi:beta::my-app::gcp:sql/databaseInstance:DatabaseInstance::db"}]}}`)}
	_, err := foreignResourcesIn(legacyShape)
	require.Error(t, err, "an unreadable version must not look like an empty stack")

	// ...and it fails open rather than aborting the deploy.
	require.NoError(t, checkLegacyState(context.Background(), fakeStack{deployment: legacyShape}, "",
		testProjectName, testStackName))
}

func deploymentOf(body string) apitype.UntypedDeployment {
	return apitype.UntypedDeployment{Version: apitype.DeploymentSchemaVersionCurrent, Deployment: json.RawMessage(body)}
}
