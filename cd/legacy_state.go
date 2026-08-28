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
// This file makes the unsafe path fail loudly instead of silently.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

// allowLegacyStateTakeoverEnv opts out of the guard. It is for deliberate,
// supervised use only: it lets Pulumi replace every resource in the stack.
const allowLegacyStateTakeoverEnv = "DEFANG_ALLOW_LEGACY_STATE_TAKEOVER"

// migrationRunbook is the blue/green migration guide. A sibling workstream owns
// the file; keep this path in sync with it.
const migrationRunbook = "https://github.com/DefangLabs/pulumi-defang/blob/main/docs/legacy-cd-migration.md"

// maxReportedResources caps how many foreign resources the error lists, so the
// message stays readable on a stack with hundreds of them.
const maxReportedResources = 5

// ourPackages are the Pulumi package names of this CD's own components.
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
var ourPackages = map[string]bool{
	"defang-aws":   true,
	"defang-gcp":   true,
	"defang-azure": true,
}

// isOurs reports whether a resource was created by this CD.
func isOurs(res apitype.ResourceV3) bool {
	// The qualified type is the full parent chain, "$"-separated, so a match
	// anywhere in it means the resource hangs off one of our components.
	for _, typ := range strings.Split(string(res.URN.QualifiedType()), "$") {
		pkg, _, _ := strings.Cut(typ, ":")
		if ourPackages[pkg] {
			return true
		}
	}
	// A handful of resources are registered at the top level, outside the
	// Project component, so they carry a plain cloud type token. Accept them
	// only in that position, and only under the names this CD gives them.
	return isRootChild(res) && program.TopLevelResourceNames[res.URN.Name()]
}

// isStructural reports whether a resource says nothing about who owns the
// infrastructure. The root stack and the provider resources exist in every
// state, old and new, so they are not evidence either way.
func isStructural(res apitype.ResourceV3) bool {
	typ := res.URN.QualifiedType()
	return typ == resource.RootStackType || strings.HasPrefix(string(typ), "pulumi:providers:")
}

// isRootChild reports whether a resource sits directly under the root stack
// rather than under some component.
func isRootChild(res apitype.ResourceV3) bool {
	return res.Parent == "" || res.Parent.QualifiedType() == resource.RootStackType
}

// foreignResources returns the resources in a deployment that this CD does not
// recognize as its own. An empty result means the state is safe to operate on:
// either it is empty, or this CD wrote all of it.
//
// This is deliberately an allowlist rather than a list of known-legacy types. A
// new type token in an old CD would slip past a denylist; it cannot slip past
// this. It also catches a half-finished takeover, where our resources and the
// old CD's resources sit in the same state.
func foreignResources(resources []apitype.ResourceV3) []apitype.ResourceV3 {
	var foreign []apitype.ResourceV3
	for _, res := range resources {
		if isStructural(res) || isOurs(res) {
			continue
		}
		foreign = append(foreign, res)
	}
	return foreign
}

// checkLegacyState reads the stack's current state and refuses to continue if
// another CD wrote it.
//
// It fails open. If the state cannot be read it warns and returns nil, because
// making every deploy depend on a second successful read of the state backend
// would turn a transient backend error into an outage for every customer. The
// exposure from failing open is small: Pulumi has to read the same state to do
// any work, so a state this cannot read is one the deploy will not get far on
// either.
func checkLegacyState(ctx context.Context, stack auto.Stack) error {
	if getenvBool(allowLegacyStateTakeoverEnv) {
		Println("Warning: " + allowLegacyStateTakeoverEnv + " is set. Skipping the check for a state written by " +
			"another Defang CD. If this stack belongs to another CD, this deploy will replace every resource in " +
			"it, including databases.")
		return nil
	}

	deployment, err := stack.Export(ctx)
	if err != nil {
		Println("Warning: could not read the existing stack state, so this skipped the check for a state written " +
			"by another Defang CD: " + err.Error())
		return nil
	}

	foreign, err := foreignResourcesIn(deployment)
	if err != nil {
		Println("Warning: could not parse the existing stack state, so this skipped the check for a state written " +
			"by another Defang CD: " + err.Error())
		return nil
	}
	if len(foreign) == 0 {
		return nil
	}
	return &legacyStateError{foreign: foreign}
}

// foreignResourcesIn decodes an exported deployment and returns the resources
// this CD does not own. A stack that has never been deployed exports a
// deployment with no resources at all, which yields an empty result.
func foreignResourcesIn(deployment apitype.UntypedDeployment) ([]apitype.ResourceV3, error) {
	if len(deployment.Deployment) == 0 {
		return nil, nil
	}
	// Only the URNs and parents are read. The exported deployment holds
	// decrypted secrets, so none of it is ever logged.
	var snapshot apitype.DeploymentV3
	if err := json.Unmarshal(deployment.Deployment, &snapshot); err != nil {
		return nil, err
	}
	return foreignResources(snapshot.Resources), nil
}

// legacyStateError explains what was found and how to proceed.
type legacyStateError struct {
	foreign []apitype.ResourceV3
}

func (e *legacyStateError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "this stack holds %d resources that this version of the Defang CD does not manage, "+
		"so it was created by a different CD:\n", len(e.foreign))
	for i, res := range e.foreign {
		if i == maxReportedResources {
			fmt.Fprintf(&b, "  ...and %d more\n", len(e.foreign)-maxReportedResources)
			break
		}
		fmt.Fprintf(&b, "  %s::%s\n", res.URN.QualifiedType(), res.URN.Name())
	}
	b.WriteString("\nContinuing would replace all of this infrastructure. Pulumi would create a new set of " +
		"resources and then delete the ones above, including any database. That data would be lost.\n" +
		"\nTo move this project to this CD, deploy it to a NEW stack and migrate to it. See\n  " +
		migrationRunbook + "\n" +
		"\nTo continue anyway and accept the deletion, set " + allowLegacyStateTakeoverEnv + "=1.")
	return b.String()
}
