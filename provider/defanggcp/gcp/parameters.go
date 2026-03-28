package gcp

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type ConfigProvider struct {
	projectName string
}

func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{projectName: projectName}
}

func (p *ConfigProvider) GetConfig(ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption) pulumi.StringOutput {
	return pulumi.String("unimplemented").ToStringOutput()
}
