package main

// Guard against taking over a Pulumi state that a different Defang CD wrote.
//
// The Defang CLI hands every CD the same state location: same bucket, same
// project, same stack, same passphrase. So when a customer is switched from an
// older CD to this one, this CD opens the older CD's state.
//
// That is not a safe upgrade. The two CDs share no resource URNs, and there is
// no alias or adoption code. Pulumi therefore reads the old resources as
// "gone", plans to create a full new set of infrastructure, and then plans to
// delete every old resource. The delete list includes managed databases.
// Outside production mode, deletion protection and backups are both off, so the
// data goes with them.
//
// The supported way to move between CDs is blue/green: deploy to a NEW stack,
// move traffic, then tear the old stack down. See docs/legacy-cd-migration.md.
//
// Only `up` is guarded, because only `up` applies this CD's desired state to a
// snapshot it did not write, which is what produces create-everything-then-
// delete-everything. `down` and `destroy` delete exactly what the caller asked
// them to, and they are the last step of the migration above, so blocking them
// would leave the old stack with no supported way down. `preview` is the
// diagnostic that shows the danger. `refresh`, `outputs`, `list` and `cancel`
// delete no infrastructure.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

// allowTakeoverConfigKey opts out of the guard. This is the channel that is
// reachable through the product: the recipe's pulumi_config is free-form and
// reaches the CD in the `up` payload, so a takeover can be authorized without
// shipping a CLI or a CD image.
//
// The value is the "<project>/<stack>" it applies to, NOT a boolean. A Fabric
// recipe is keyed by (tenant, mode) — not by stack — so one recipe is shared by
// every project that tenant deploys in that mode. A boolean would disarm the
// guard for all of them, including the ones nobody was migrating. Naming the
// target keeps a tenant-wide setting to a single-stack effect.
const allowTakeoverConfigKey = "defang:allowLegacyStateTakeover"

// allowTakeoverEnv is the same opt-out for a CD run started by hand, which has
// no recipe to carry config. It takes the same "<project>/<stack>" value, so a
// stale or committed value cannot disarm the guard for some other stack.
const allowTakeoverEnv = "DEFANG_ALLOW_LEGACY_STATE_TAKEOVER"

// migrationRunbook is the blue/green migration guide.
const migrationRunbook = "https://github.com/DefangLabs/pulumi-defang/blob/main/docs/legacy-cd-migration.md"

// maxReportedResources keeps the message readable on a stack with hundreds of
// foreign resources.
const maxReportedResources = 5

// thisCDPackages are the Pulumi package names of this CD's own components.
//
// Every resource this CD creates through its providers is a descendant of a
// defang-<cloud>:index:Project component, so its URN carries one of these
// packages somewhere in the qualified type. Verified against the recorded
// preview events in cd/testdata/preview-events-{aws,gcp,azure}.json, e.g.
//
//	urn:pulumi:gcp::cd-test::defang-gcp:index:Project$gcp:compute/network:Network::cd-test-vpc
//
// The older CDs have no equivalent: the TypeScript CD used defang-mvp:* and
// pulumi-nodejs:dynamic:Resource, and the Go GCP CD registered no components at
// all — its state is flat gcp:* resources plus cloudbuild:index:CloudBuild.
// "Ours" is deliberately not the word here: defang-mvp:* is Defang's too, and
// it is exactly what this has to exclude.
var thisCDPackages = map[string]bool{
	"defang-aws":   true,
	"defang-gcp":   true,
	"defang-azure": true,
}

// thisCDTopLevelNames are the names this CD gives the few resources it
// registers at the top level of the stack, outside the Project component. Their
// URNs carry a plain cloud package, so the name and the position are the only
// things that tell them apart from an older CD's leftovers.
//
// The list exists only because those resources escape the Project component. If
// they were ever re-parented under it, their URNs would carry a defang package
// and this list, isThisCD's position rule, and their tests would all go away.
//
// NEVER REMOVE AN ENTRY. This is the union of every name this CD has ever
// used, not the set it uses today: stacks deployed by an older build still hold
// the old resource, and dropping its name here would abort their next `up` —
// which is the only operation that would ever have deleted it. Deadlock, for
// every stack that has one.
//
// Adding is just as unforgiving and nothing in the compiler helps: a new
// top-level resource declared with a fresh literal still compiles, and would
// abort every existing stack's next deploy. The state fixtures cannot catch it
// either — they were captured from runs with no TTL and no state URL, so they
// contain none of these resources. TestThisCDTopLevelNamesCoversEveryRootRegistration
// reads the registrations back out of the source instead.
var thisCDTopLevelNames = map[string]bool{
	"project-pb":            true, // program/{aws,gcp,azure}.go, saveProjectPb*
	"self-destruct":         true, // program/selfdestruct_{aws,gcp}.go
	"defang-self-destruct":  true, // program/selfdestruct_azure.go, the trigger job
	"self-destruct-starter": true, // program/selfdestruct_azure.go, its role assignment
}

// isThisCD reports whether a resource was created by this CD.
func isThisCD(res apitype.ResourceV3) bool {
	// The qualified type is the full parent chain, "$"-separated, so a match
	// anywhere in it means the resource hangs off one of our components.
	for _, typ := range strings.Split(string(res.URN.QualifiedType()), "$") {
		pkg, _, _ := strings.Cut(typ, ":")
		if thisCDPackages[pkg] {
			return true
		}
	}
	return isRootChild(res) && thisCDTopLevelNames[res.URN.Name()]
}

// isStructural reports whether a resource says nothing about who owns the
// infrastructure. The root stack and the provider resources exist in every
// state, old and new, so they are not evidence either way.
func isStructural(res apitype.ResourceV3) bool {
	typ := res.URN.QualifiedType()
	return typ == resource.RootStackType || strings.HasPrefix(string(typ), "pulumi:providers:")
}

// isRootChild reports whether a resource sits directly under the root stack. A
// "$" in the qualified type means the resource is a component's child by
// construction, so it is not at the root even if the snapshot has lost its
// parent field.
func isRootChild(res apitype.ResourceV3) bool {
	if strings.Contains(string(res.URN.QualifiedType()), "$") {
		return false
	}
	return res.Parent == "" || res.Parent.QualifiedType() == resource.RootStackType
}

// foreignResources returns the URNs in a deployment that this CD does not
// recognize as its own. An empty result means the state is safe to operate on:
// either it is empty, or this CD wrote all of it.
//
// This is deliberately an allowlist rather than a list of known-legacy types. A
// new type token in an old CD would slip past a denylist; it cannot slip past
// this. It also catches a half-finished takeover, where our resources and the
// old CD's resources sit in the same state.
//
// It returns URNs, not resources: apitype.ResourceV3 carries decrypted inputs
// and outputs, and nothing downstream needs them.
func foreignResources(resources []apitype.ResourceV3) []resource.URN {
	var foreign []resource.URN
	for _, res := range resources {
		if isStructural(res) || isThisCD(res) {
			continue
		}
		foreign = append(foreign, res.URN)
	}
	return foreign
}

// stackExporter is the part of auto.Stack this needs. It exists so the check
// can be tested against a state fixture without a Pulumi backend.
type stackExporter interface {
	Export(context.Context) (apitype.UntypedDeployment, error)
}

// takeoverAllowed reports whether this exact stack has been authorized for a
// deliberate takeover. Both channels name their target, so an authorization
// left behind in a shared recipe or a shell cannot free a different stack.
func takeoverAllowed(recipePulumiConfig, project, stack string) bool {
	if project == "" || stack == "" {
		return false // never let a half-built target match a half-built value
	}
	target := project + "/" + stack
	return recipeTakeoverTarget(recipePulumiConfig) == target || os.Getenv(allowTakeoverEnv) == target
}

// recipeTakeoverTarget reads the opt-in key out of the recipe, using the same
// parser that turns the recipe into stack config. A malformed recipe is not an
// opt-in; stackConfigJson reports that error a few lines later.
func recipeTakeoverTarget(recipePulumiConfig string) string {
	if recipePulumiConfig == "" {
		return ""
	}
	config := configMap{}
	if err := unmarshalRecipe(recipePulumiConfig, config); err != nil {
		return ""
	}
	target, _ := config[allowTakeoverConfigKey].Value.(string)
	return target
}

// checkLegacyState reads the stack's current state and refuses to continue if a
// different CD wrote it. It detects "not this CD" in either direction: a newer
// CD's state would trip it too.
//
// It fails open. If the state cannot be read it warns and returns nil, because
// making every deploy depend on a second successful read of the state backend
// would turn a transient backend error into an outage for every customer. The
// exposure from failing open is small: Pulumi has to read the same state to do
// any work, so a state this cannot read is one the deploy will not get far on
// either.
func checkLegacyState(ctx context.Context, exporter stackExporter, recipePulumiConfig, projectName, stackName string) error {
	if takeoverAllowed(recipePulumiConfig, projectName, stackName) {
		warn("Warning: " + projectName + "/" + stackName + " is allowed to take over a state written by another " +
			"Defang CD. If this stack belongs to another CD, this deploy will replace every resource in it, " +
			"including databases.")
		return nil
	}

	deployment, err := exporter.Export(ctx)
	if err != nil {
		// Deliberately not logging err. auto.Stack.Export runs `pulumi stack
		// export --show-secrets`, and on failure the SDK packs the command's
		// entire stdout into the error, so printing it would put the decrypted
		// state into the deploy log. Set DEFANG_PULUMI_DEBUG=1 to debug this.
		warn("Skipping the check for another Defang CD's state: could not read the existing state.")
		return nil
	}

	foreign, err := foreignResourcesIn(deployment)
	if err != nil {
		warn("Skipping the check for another Defang CD's state: could not parse the existing state:", err)
		return nil
	}
	if len(foreign) == 0 {
		return nil
	}
	return &legacyStateError{foreign: foreign}
}

// foreignResourcesIn decodes an exported deployment and returns the URNs this
// CD does not own. A stack that has never been deployed exports a deployment
// with no resources at all, which yields an empty result.
func foreignResourcesIn(deployment apitype.UntypedDeployment) ([]resource.URN, error) {
	// A stack that does not exist yet exports nothing. Handled here so a first
	// deploy never prints a parse warning.
	if len(deployment.Deployment) == 0 {
		return nil, nil
	}
	// A shape this cannot read must not decode to "no resources", because here
	// an empty result means "go ahead and delete everything".
	if deployment.Version != apitype.DeploymentSchemaVersionCurrent {
		return nil, fmt.Errorf("unsupported deployment version %d", deployment.Version)
	}
	// The exported deployment holds decrypted secrets, so none of it is logged
	// and nothing but URNs and parents leaves this function.
	var snapshot apitype.DeploymentV3
	if err := json.Unmarshal(deployment.Deployment, &snapshot); err != nil {
		return nil, err
	}
	// URN accessors assert on a malformed URN, and the assert panics. A
	// hand-repaired snapshot is exactly what an operator produces mid-migration,
	// so check first and let it fail open rather than crash the deploy. The URNs
	// themselves are not reported, only the position, to keep state out of logs.
	for i, res := range snapshot.Resources {
		if !res.URN.IsValid() {
			return nil, fmt.Errorf("resource %d has an unreadable URN", i)
		}
		if res.Parent != "" && !res.Parent.IsValid() {
			return nil, fmt.Errorf("resource %d has an unreadable parent URN", i)
		}
	}
	return foreignResources(snapshot.Resources), nil
}

// legacyStateError explains what was found and how to proceed. It reaches the
// customer through main.go, which prints it and exits 1 — the same exit code as
// a failed deploy, so nothing downstream can single it out.
type legacyStateError struct {
	foreign []resource.URN
}

func (e *legacyStateError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "this stack has %d resources that this version of Defang did not create, "+
		"so an older version of Defang deployed it:\n", len(e.foreign))
	for i, urn := range e.foreign {
		if i == maxReportedResources {
			fmt.Fprintf(&b, "  ...and %d more\n", len(e.foreign)-maxReportedResources)
			break
		}
		fmt.Fprintf(&b, "  %s::%s\n", urn.QualifiedType(), urn.Name())
	}
	b.WriteString("\nContinuing would replace all of it: a new set of resources would be created and the ones " +
		"above deleted, databases and their data included. Nothing has been changed.\n" +
		"\nTo move this project, deploy it to a new stack, then shut this one down. See\n  " +
		migrationRunbook + "\n" +
		"\nThis stack is not stuck, so do not run `down` or `destroy` on it to clear the error. Those are not " +
		"blocked, and they delete everything above.\n" +
		"\nIf this is wrong, contact Defang support.")
	return b.String()
}
