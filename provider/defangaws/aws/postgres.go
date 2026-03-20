package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// newPostgresComponent registers a component resource for a managed Postgres service,
// creates its RDS children, registers outputs, and returns the host:port endpoint.
func newPostgresComponent(
	ctx *pulumi.Context,
	configProvider shared.ConfigProvider,
	serviceName string,
	svc shared.ServiceInput,
	vpcID pulumix.Output[string],
	privateSubnetIDs pulumix.Output[[]string],
	sg *ec2.SecurityGroup,
	recipe Recipe,
	parentOpt pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	comp := &serviceComponent{}
	if err := ctx.RegisterComponentResource("defang-aws:index:AwsPostgres", serviceName, comp, parentOpt); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering postgres component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	rdsResult, err := CreateRDS(ctx, configProvider, serviceName, svc, vpcID, privateSubnetIDs, sg, recipe, opts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating RDS for %s: %w", serviceName, err)
	}

	endpoint := pulumi.StringOutput(pulumix.Apply(pulumix.Output[string](rdsResult.Instance.Address), func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, 5432)
	}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return endpoint, nil
}
