package azure

import (
	"sync"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type ConfigProvider struct {
	storeName         string
	resourceGroupName string
	projectName       string
	cache             map[string]pulumi.StringOutput
	mu                sync.Mutex
}

func NewConfigProvider(storeName, resourceGroupName, projectName string) *ConfigProvider {
	return &ConfigProvider{
		storeName:         storeName,
		resourceGroupName: resourceGroupName,
		projectName:       projectName,
		cache:             make(map[string]pulumi.StringOutput),
	}
}

// GetConfig returns a user-defined config value (e.g. POSTGRES_PASSWORD) as a
// pulumi.StringOutput.
//
// The defang CLI stores these in Azure App Configuration under the key
// "/{prefix}/{project}/{stack}/{name}"; at deploy time, the CD container reads
// them via fetchAzureUserConfig and forwards them into the Pulumi stack config
// under the "{project}:" namespace (see cd/main.go). This function reads them
// back from that namespace. Going through Pulumi config — instead of an ARM
// LookupKeyValue invoke — avoids data-plane permission issues on the App
// Configuration store and treats the values as Pulumi secrets end-to-end.
//
// Returns a resolved-empty-string Output when unset. Never returns a zero-value
// pulumi.StringOutput{} because that has a nil internal state, which causes a
// nil-pointer dereference inside Pulumi's reflection walk (gatherJoinSet) when
// the output is embedded in a resource's args.
func (p *ConfigProvider) GetConfig(ctx *pulumi.Context, key string, _ ...pulumi.InvokeOption) pulumi.StringOutput {
	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.cache[key]; ok {
		return val
	}

	out := config.New(ctx, p.projectName).GetSecret(key)
	p.cache[key] = out
	return out
}
