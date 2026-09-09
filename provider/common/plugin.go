package common

import (
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// PluginDownloadURL is where Pulumi fetches this plugin from when it is not
// already in the local plugin cache. Each provider's schema metadata publishes
// it, so the generated SDKs bake it into every resource an end-user program
// registers. It is also the fallback for PluginIdentityFrom when a caller
// supplied none.
const PluginDownloadURL = "github://api.github.com/DefangLabs/pulumi-defang"

// PluginIdentity says which plugin serves our own resource types: where to
// fetch it from, and — only when a caller explicitly pinned one — which
// version of it.
//
// The engine hands both to Construct — the Pulumi Go SDK copies them out of
// the ConstructRequest onto the component's resource options — but they stop
// there. A component's children do not inherit them, so a resource of our own
// package that we register from inside the provider process has to be given
// them explicitly.
//
// Omitting the URL is not visible at deploy time: the engine synthesises a
// bare default provider and falls back to the conventional plugin location
// github.com/pulumi/pulumi-<name>, which does not exist for any of our
// packages. Nothing fails while the plugin binary is still in the local cache,
// so the stack strands later, on an operation run from a workspace whose cache
// is empty and which therefore has to fetch the plugin.
//
// The version is a different matter, and PluginIdentityFrom deliberately does
// not supply one — see the reasoning there.
type PluginIdentity struct {
	// DownloadURL is the plugin's pluginDownloadURL. Never empty after
	// PluginIdentityFrom: it falls back to the PluginDownloadURL constant.
	DownloadURL string
	// Version is the plugin's semver, or "" — the normal case — to let the
	// engine accept whichever build of our plugin is at hand.
	Version string
}

// PluginIdentityFrom determines the plugin identity to use for our own child
// resources: our PluginDownloadURL, plus a version only if the caller pinned
// one.
//
// It deliberately does NOT fall back to the provider package's linker-stamped
// Version, even though that variable is in scope at every call site.
//
// A version in a registration is recorded in the checkpoint, as part of the
// default provider's own URN — pulumi:providers:defang-azure::default_2_7_2_…
// — and every later operation on those resources (up, refresh, destroy) has to
// re-resolve that exact provider. Pulumi resolves a *versioned* request by
// exact match (workspace.SelectCompatiblePlugin), so it is satisfied only by a
// locally installed plugin of precisely that version, or by a GitHub release
// tagged with it. Our CD image bakes exactly one version of the plugin: its
// own. A pinned version is therefore resolvable by the image that wrote it,
// and by no other image, unless the version happens to name a published
// release. It usually does not:
//
//   - Only release.yml passes a real PROVIDER_VERSION. Every other build — the
//     per-commit "alpha" CD images that test.yml publishes, a local `make
//     image`, a bare `go build` — carries the Dockerfile's placeholder default
//     (0.0.1) or a pulumictl pre-release string. Neither is ever tagged on
//     GitHub, so a stack whose Build resources were first registered by such
//     an image can never be operated on by any other image again: the engine
//     404s on .../releases/tags/v0.0.1 and there is nothing anywhere to
//     install. This is not hypothetical — it stranded a production stack whose
//     first deploy used an alpha image and whose next deploy used a released
//     one.
//   - Even between two published releases the pin costs a download of the
//     superseded plugin on every upgrade, because the new image does not have
//     it, and it makes the recorded provider URN churn on every version bump.
//
// A registration with no version resolves through the legacy path
// (workspace.LegacySelectCompatiblePlugin), which accepts any locally
// installed plugin of that name — so every CD image can serve the resources it
// did not create — and falls back to "latest" from PluginDownloadURL when the
// cache really is empty. That is also what the generated SDKs already do: their
// SdkVersion is the zero value, so PkgResourceDefaultOpts emits the URL and no
// version. Leaving the version out makes our self-registered resources agree
// with every other resource in the same stack instead of being the one
// exception.
//
// The versionless path does pay one extra unauthenticated GitHub API call on a
// genuinely cold cache: releases/latest to learn the version, then
// releases/tags/vX to find the asset. Pinning buys nothing back there. For an
// organisation other than "pulumi", the download itself goes through the same
// rate-limited api.github.com endpoint either way (see githubSource.Download),
// so a pin saves one request out of two and costs the stack its ability to be
// served by any other build.
//
// Reading the options first mirrors GetChildOptions in pulumi-kubernetes' yaml
// SDK, which propagates the caller's Version and PluginDownloadURL to the
// children it registers. Today that read always comes back empty for us:
// pulumi-go-provider's own ConstructRequest carries neither field, so its RPC
// builder cannot populate them and the Go SDK has nothing to copy onto the
// options it hands to Construct. Only Providers survives that trip. The read is
// kept because it costs nothing, it is the behaviour we want the moment the
// framework forwards them, and it lets a caller pin a version deliberately —
// which, coming from a program that resolved a published SDK, names a version
// that really can be fetched.
//
// Thread the result down to the registration that needs it. It must not travel
// as a ResourceOption: Version is per-package, so applying ours to a
// neighbouring aws:/gcp:/azure: resource would send the engine looking for that
// provider at our version.
func PluginIdentityFrom(opts ...pulumi.ResourceOption) PluginIdentity {
	id := PluginIdentity{DownloadURL: PluginDownloadURL}
	snapshot, err := pulumi.NewResourceOptions(opts...)
	if err != nil {
		// Malformed options are the caller's problem and will resurface at
		// registration; fall back rather than failing here.
		return id
	}
	if snapshot.PluginDownloadURL != "" {
		id.DownloadURL = snapshot.PluginDownloadURL
	}
	if snapshot.Version != "" {
		id.Version = pulumiPluginVersion(snapshot.Version)
	}
	return id
}

// pulumiPluginVersion converts a Git tag-shaped version to the semver string
// expected by pulumi.Version. Callers may pin a tag such as "v2.7.1"; passing
// that prefix through makes the engine's strict parser reject the child
// resource registration with "Invalid character(s) found in major number
// \"v2\"".
func pulumiPluginVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

// ResourceOptions prefixes opts with this identity, for registering one of our
// own resource types. Later options win in the Pulumi Go SDK, so an explicit
// value in opts still overrides.
func (id PluginIdentity) ResourceOptions(opts ...pulumi.ResourceOption) []pulumi.ResourceOption {
	defaults := make([]pulumi.ResourceOption, 0, len(opts)+2)
	if id.DownloadURL != "" {
		defaults = append(defaults, pulumi.PluginDownloadURL(id.DownloadURL))
	}
	if id.Version != "" {
		defaults = append(defaults, pulumi.Version(id.Version))
	}
	return append(defaults, opts...)
}
