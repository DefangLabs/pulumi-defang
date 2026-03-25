// Ported from https://github.com/DefangLabs/defang-mvp/blob/main/pulumi/shared/aws/common.ts
package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)


func getCallerAccountId(ctx *pulumi.Context, opts ...pulumi.InvokeOption) (string, error) {
	ac, err := aws.GetCallerIdentity(ctx, nil, opts...)
	if err != nil {
		return "", err
	}
	return ac.AccountId, nil
}


func getCallerRegion(ctx *pulumi.Context, opts ...pulumi.InvokeOption) (aws.Region, error) {
	r, err := aws.GetRegion(ctx, nil, opts...)
	if err != nil {
		return "", err
	}
	return aws.Region(r.Region), nil
}
