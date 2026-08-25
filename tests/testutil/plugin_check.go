package testutil

import (
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// ownPackages are the resource-package prefixes this repo publishes. A type
// token starting with one of these names a resource served by our own plugin.
var ownPackages = []string{"defang-aws:", "defang-gcp:", "defang-azure:"}

// PluginRecord captures what a single resource registration told the engine
// about which plugin serves it.
type PluginRecord struct {
	Type              string
	Name              string
	Custom            bool
	PluginDownloadURL string
	Version           string
}

// PluginTracker records the plugin identity of every resource registered
// through the mock monitor.
type PluginTracker struct {
	mu      sync.Mutex
	records []PluginRecord
}

// NewPluginTracker returns a mock monitor paired with a tracker that captures
// the PluginDownloadURL of every resource registered during Construct. The
// mock echoes inputs back unchanged so it composes with any other test that
// only needs Construct to succeed.
func NewPluginTracker() (*integration.MockResourceMonitor, *PluginTracker) {
	pt := &PluginTracker{}
	mock := &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			rec := PluginRecord{Type: string(args.TypeToken), Name: args.Name}
			// Both fields come from the RegisterResource RPC; mocks
			// synthesized without one (e.g. ReadResource) leave them zero.
			if args.RegisterRPC != nil {
				rec.Custom = args.RegisterRPC.GetCustom()
				rec.PluginDownloadURL = args.RegisterRPC.GetPluginDownloadURL()
				rec.Version = args.RegisterRPC.GetVersion()
			}
			pt.mu.Lock()
			pt.records = append(pt.records, rec)
			pt.mu.Unlock()
			return args.Name, args.Inputs, nil
		},
	}
	return mock, pt
}

// Records returns a copy of the recorded registrations.
func (pt *PluginTracker) Records() []PluginRecord {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	out := make([]PluginRecord, len(pt.records))
	copy(out, pt.records)
	return out
}

// AssertOwnCustomResourcesCarryPluginIdentity asserts that every CUSTOM
// resource we register from one of our own packages tells the engine both
// where to fetch the plugin from and which version of it to use.
//
// Without the URL the engine synthesises a bare default provider and falls
// back to github.com/pulumi/pulumi-<name>, which does not exist for our
// packages. Without the version the checkpoint does not record which plugin
// built the resource, so later operations resolve whatever is newest and pay
// an unauthenticated GitHub API call to ask what "latest" is. Both faults are
// invisible during an up and strand the stack on destroy. See
// common.PluginIdentityFrom.
//
// Component resources are exempt: the engine deletes them without loading a
// plugin, so they never trigger the lookup.
//
// It asserts at least one own custom resource was seen, so a fixture that
// stops exercising the build path fails loudly instead of passing vacuously.
//
// Failures are reported as test errors (not fatals) so a single run surfaces
// every offending registration at once.
func (pt *PluginTracker) AssertOwnCustomResourcesCarryPluginIdentity(t *testing.T, wantURL, wantVersion string) {
	t.Helper()

	seen := 0
	for _, r := range pt.Records() {
		if !r.Custom || !isOwnPackage(r.Type) {
			continue
		}
		seen++
		if r.PluginDownloadURL != wantURL {
			t.Errorf("resource name=%q type=%s registered with PluginDownloadURL=%q, want %q",
				r.Name, r.Type, r.PluginDownloadURL, wantURL)
		}
		if r.Version != wantVersion {
			t.Errorf("resource name=%q type=%s registered with Version=%q, want %q",
				r.Name, r.Type, r.Version, wantVersion)
		}
	}
	if seen == 0 {
		t.Errorf("no custom resources from our own packages were registered; "+
			"the fixture no longer exercises the path this guards (records=%d)", len(pt.Records()))
	}
}

func isOwnPackage(typeToken string) bool {
	for _, p := range ownPackages {
		if strings.HasPrefix(typeToken, p) {
			return true
		}
	}
	return false
}
