package azure

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-azure-native-sdk/app/v3"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v3"
	azureconfig "github.com/pulumi/pulumi-azure-native-sdk/v3/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SharedInfra holds resources shared across all services in a project.
type SharedInfra struct {
	ResourceGroup      *resources.ResourceGroup
	Environment        *app.ManagedEnvironment
	BuildInfra         *BuildInfra            // nil when no services require image builds
	Networking         *NetworkingResult      // nil when no VNet-integrated services are present
	DNS                *DNSResult             // nil when no VNet-integrated services are present
	LLMInfra           *LLMInfra              // nil when no LLM services are present
	ConfigProvider     compose.ConfigProvider // reads project secrets (set via `defang config set`)
	KeyVaultURL        string                 // Key Vault URL for secret references (empty if no vault)
	KeyVaultIdentityID pulumi.StringPtrOutput // user-assigned identity for KV access (zero if no vault)
	// Etag is the deployment ID supplied by the CD program; empty for
	// standalone Service callers.
	Etag string
}

// BaseTags returns the project-wide tag map: defang-project (Pulumi project),
// defang-stack (Pulumi stack), and defang-etag (deployment ID, if non-empty).
// These are applied to every azure-native resource by the cascading
// transformation installed via DefaultTagsTransformation — callers should not
// set them manually.
func BaseTags(ctx *pulumi.Context, etag string) pulumi.StringMap {
	tags := pulumi.StringMap{
		"defang-org":     pulumi.String(ctx.Organization()), // TODO: doesn't work with DIY backends
		"defang-project": pulumi.String(ctx.Project()),
		"defang-stack":   pulumi.String(ctx.Stack()),
	}
	if etag != "" {
		tags["defang-etag"] = pulumi.String(etag)
	}
	return tags
}

// ServiceTags returns the per-resource tag map for resources scoped to a single
// service. Only sets defang-service; the project/stack/etag tags are added by
// the cascading transformation installed via DefaultTagsTransformation.
func ServiceTags(serviceName string) pulumi.StringMap {
	return pulumi.StringMap{
		"defang-service": pulumi.String(serviceName),
	}
}

// stringMapInputType is the reflect.Type of pulumi.StringMapInput, used by the
// default-tags transformation to detect taggable Args structs.
var stringMapInputType = reflect.TypeOf((*pulumi.StringMapInput)(nil)).Elem()

// DefaultTagsTransformation returns a resource transformation that merges
// baseTags into every azure-native resource's Tags field. Existing per-resource
// tag values win on key collision so callers can still override
// (e.g. defang-service).
//
// azure-native's Provider has no DefaultTags option, and pulumi-go-provider's
// Construct context lacks a stack so RegisterStackTransformation panics.
// Pass the result via pulumi.Transformations(...) on the parent component
// (Project) — Pulumi cascades component-level transformations to all children.
func DefaultTagsTransformation(baseTags pulumi.StringMap) pulumi.ResourceTransformation {
	if len(baseTags) == 0 {
		return nil
	}
	baseOut := baseTags.ToStringMapOutput()
	return func(args *pulumi.ResourceTransformationArgs) *pulumi.ResourceTransformationResult {
		if !strings.HasPrefix(args.Type, "azure-native:") {
			return nil
		}
		v := reflect.ValueOf(args.Props)
		if v.Kind() != reflect.Ptr || v.IsNil() {
			return nil
		}
		v = v.Elem()
		if v.Kind() != reflect.Struct {
			return nil
		}
		f := v.FieldByName("Tags")
		if !f.IsValid() || !f.CanSet() {
			return nil
		}
		if !f.Type().Implements(stringMapInputType) && f.Type() != stringMapInputType {
			return nil
		}

		var existingOut pulumi.StringMapOutput
		if iface := f.Interface(); iface != nil {
			if existing, ok := iface.(pulumi.StringMapInput); ok {
				existingOut = existing.ToStringMapOutput()
			} else {
				existingOut = pulumi.StringMap{}.ToStringMapOutput()
			}
		} else {
			existingOut = pulumi.StringMap{}.ToStringMapOutput()
		}

		merged := pulumi.All(baseOut, existingOut).ApplyT(func(parts []interface{}) map[string]string {
			out := map[string]string{}
			if base, ok := parts[0].(map[string]string); ok {
				for k, v := range base {
					out[k] = v
				}
			}
			if existing, ok := parts[1].(map[string]string); ok {
				for k, v := range existing {
					out[k] = v
				}
			}
			return out
		}).(pulumi.StringMapOutput)

		f.Set(reflect.ValueOf(merged))
		return &pulumi.ResourceTransformationResult{Props: args.Props, Opts: args.Opts}
	}
}

// KeyVaultName returns the deterministic Key Vault name for the given Defang
// Compose project in this stack, derived from (subscription, resource group)
// per the defang CLI convention (see
// defang/src/pkg/clouds/azure/keyvault.VaultName).
//
// composeProject is the Defang Compose project name (e.g. "crewai"), which
// may differ from ctx.Project() — a single Pulumi project can host multiple
// Defang Compose projects.
func KeyVaultName(ctx *pulumi.Context, composeProject string) string {
	subID := azureconfig.GetSubscriptionId(ctx)
	rg := ProjectResourceGroupName(ctx, composeProject)
	h := sha256.Sum256([]byte(subID + "|" + rg))
	return "defang-config-" + hex.EncodeToString(h[:])[:8]
}

// ProjectResourceGroupName returns the deterministic name of the Defang
// project's Azure resource group, derived from (composeProject, stack,
// location) per the defang CLI convention (see
// defang/src/pkg/cli/client/byoc/azure.projectResourceGroupName). The CLI
// creates this RG before invoking the CD task; the provider imports it.
//
// composeProject is the Defang Compose project name (typically from the
// compose file's top-level `name:`), which may differ from ctx.Project() —
// a single Pulumi project can host multiple Defang Compose projects.
func ProjectResourceGroupName(ctx *pulumi.Context, composeProject string) string {
	prefix := common.Prefix.Get(ctx)
	if prefix != "" {
		prefix += "-"
	}
	return prefix + composeProject + "-" + ctx.Stack()
}
