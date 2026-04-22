package testutil

import (
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// ParentRecord captures the parent URN, type, and name reported for a single
// resource registration. It is populated by ParentTracker from the mock
// monitor's RegisterResource RPC.
type ParentRecord struct {
	Parent resource.URN
	Type   string
	Name   string
}

// ParentTracker records the parent URN of every resource registered through
// the mock monitor. Tests use it to assert hierarchy invariants (e.g. that
// every resource created inside a Project descends from the Project URN).
type ParentTracker struct {
	mu      sync.Mutex
	records []ParentRecord
}

// NewParentTracker returns a mock monitor paired with a tracker that captures
// the parent URN, type, and name of every resource registered during
// Construct. The mock echoes inputs back unchanged so it composes with any
// other test that only needs Construct to succeed.
func NewParentTracker() (*integration.MockResourceMonitor, *ParentTracker) {
	pt := &ParentTracker{}
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			var parent resource.URN
			// Parent comes from the RegisterResource RPC; mocks synthesized
			// without an RPC (e.g. ReadResource) leave it empty.
			if args.RegisterRPC != nil {
				parent = resource.URN(args.RegisterRPC.GetParent())
			}
			pt.mu.Lock()
			pt.records = append(pt.records, ParentRecord{
				Parent: parent,
				Type:   string(args.TypeToken),
				Name:   args.Name,
			})
			pt.mu.Unlock()
			return args.Name, args.Inputs, nil
		},
	}
	return mock, pt
}

// Records returns a copy of the recorded registrations.
func (pt *ParentTracker) Records() []ParentRecord {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	out := make([]ParentRecord, len(pt.records))
	copy(out, pt.records)
	return out
}

// AssertAllDescendFrom asserts that every recorded resource descends from
// projectURN. It checks the parent URN's qualified-type chain: a child of the
// project has chain == <projectType>, a grandchild has chain beginning with
// "<projectType>$". Resources whose own type token equals the project's type
// are treated as the root Project component and skipped.
//
// It does NOT enforce that every immediate parent is a component. Pulumi's
// SDK caution against parenting to custom resources (see the __childResources
// comment in pulumi/sdk/nodejs/resource.ts) is really about dependency-walk
// cycles, not a blanket ban — and ecosystem patterns like ARM-hierarchy
// children (VirtualNetworkLink→PrivateZone, RedisEnterpriseDatabase→cluster,
// PrivateDnsZoneGroup→PrivateEndpoint) and awsx's internal topology
// legitimately parent custom-to-custom. Keeping this check here as a strict
// invariant produced false positives on those patterns.
//
// Failures are reported as test errors (not fatals) so a single run surfaces
// every orphan resource at once.
func (pt *ParentTracker) AssertAllDescendFrom(t *testing.T, projectURN resource.URN) {
	t.Helper()
	projectType := string(projectURN.Type())
	projectQT := string(projectURN.QualifiedType())

	for _, r := range pt.Records() {
		// Skip the Project component itself; it's the root we're anchoring to.
		if r.Type == projectType {
			continue
		}
		if r.Parent == "" {
			t.Errorf("resource name=%q type=%s has no parent; expected to descend from %s",
				r.Name, r.Type, projectURN)
			continue
		}
		parentQT := string(r.Parent.QualifiedType())
		if parentQT == projectQT || strings.HasPrefix(parentQT, projectQT+"$") {
			continue
		}
		t.Errorf("resource name=%q type=%s has parent %q (qualifiedType=%q) which does not descend from Project (%s)",
			r.Name, r.Type, r.Parent, parentQT, projectURN)
	}
}
