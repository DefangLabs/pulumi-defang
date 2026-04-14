package azure

import (
	"sync"

	"github.com/pulumi/pulumi-azure-native-sdk/appconfiguration/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

// GetConfig reads a secret config value from Azure App Configuration.
// The defang CLI stores user-defined config (e.g. POSTGRES_PASSWORD) as key-value
// entries in the store, labelled with the Pulumi stack name.
//
// Returns a resolved-empty-string Output on error or miss. Never returns a
// zero-value pulumi.StringOutput{} because that has a nil internal state, which
// causes a nil-pointer dereference inside Pulumi's reflection walk (gatherJoinSet)
// when the output is embedded in a resource's args.
func (p *ConfigProvider) GetConfig(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) pulumi.StringOutput {
	if ctx.DryRun() {
		return pulumi.Sprintf("dry-run-%s", key).ToStringOutput()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if val, ok := p.cache[key]; ok {
		return val
	}

	empty := pulumi.String("").ToStringOutput()

	result, err := appconfiguration.LookupKeyValue(ctx, &appconfiguration.LookupKeyValueArgs{
		ConfigStoreName:   p.storeName,
		ResourceGroupName: p.resourceGroupName,
		// KeyValueName mirrors the AWS SSM path convention: /Defang/{project}/{stack}/{key}
		KeyValueName: "/Defang/" + p.projectName + "/" + ctx.Stack() + "/" + key,
	}, opts...)
	if err != nil || result == nil || result.Value == nil {
		return empty
	}

	out := pulumi.String(*result.Value).ToStringOutput()
	p.cache[key] = out
	return out
}
