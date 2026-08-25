package common

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// PluginDownloadURL is where Pulumi fetches this plugin from when it is not
// already in the local plugin cache. Each provider's schema metadata publishes
// it, so the generated SDKs bake it into every resource an end-user program
// registers. It is also the fallback for PluginIdentityFrom when a caller
// supplied none.
const PluginDownloadURL = "github://api.github.com/DefangLabs/pulumi-defang"

// PluginIdentity says which plugin serves our own resource types: where to
// fetch it from, and which version of it to use.
//
// The engine hands both to Construct — the Pulumi Go SDK copies them out of
// the ConstructRequest onto the component's resource options — but they stop
// there. A component's children do not inherit them, so a resource of our own
// package that we register from inside the provider process has to be given
// them explicitly.
//
// Omitting them is not visible at deploy time. Without the URL the engine
// synthesises a bare default provider and falls back to the conventional
// plugin location github.com/pulumi/pulumi-<name>, which does not exist for
// any of our packages. Without the version the checkpoint does not record
// which plugin built the resource, so later operations resolve whatever is
// newest and pay an unauthenticated GitHub API call to ask what "latest" is —
// a call that is rate-limited per source IP. Neither costs anything during an
// up, because resolving the SDK-registered provider has already put the plugin
// binary in the local cache. Both strand the stack on destroy, from a fresh
// workspace whose cache is empty, where nothing populates it.
type PluginIdentity struct {
	// DownloadURL is the plugin's pluginDownloadURL. Never empty after
	// PluginIdentityFrom: it falls back to the PluginDownloadURL constant.
	DownloadURL string
	// Version is the plugin's semver, or "" for a build that carries none
	// (a dev build has no -ldflags -X, and the generated SDKs omit the
	// option in that case too).
	Version string
}

// PluginIdentityFrom determines the plugin identity to use for our own child
// resources, preferring what the caller asked for and falling back to what
// this build knows about itself.
//
// fallbackVersion is the provider package's linker-initialized Version. Pass
// it from Construct, where that variable is in scope. It may be "": a dev
// build has no -ldflags -X, and the generated SDKs omit the version option in
// that case too.
//
// Reading the options first mirrors GetChildOptions in pulumi-kubernetes' yaml
// SDK, which propagates the caller's Version and PluginDownloadURL to the
// children it registers. Today that read always comes back empty for us:
// pulumi-go-provider's own ConstructRequest carries neither field, so its RPC
// builder cannot populate them and the Go SDK has nothing to copy onto the
// options it hands to Construct. Only Providers survives that trip. The read
// is kept because it costs nothing, it is the behaviour we want the moment the
// framework forwards them, and it lets a caller override.
//
// Thread the result down to the registration that needs it. It must not travel
// as a ResourceOption: Version is per-package, so applying ours to a
// neighbouring aws:/gcp:/azure: resource would send the engine looking for
// that provider at our version.
func PluginIdentityFrom(fallbackVersion string, opts ...pulumi.ResourceOption) PluginIdentity {
	id := PluginIdentity{DownloadURL: PluginDownloadURL, Version: fallbackVersion}
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
		id.Version = snapshot.Version
	}
	return id
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
