package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// newRedisComponent registers a component resource for a managed Redis service,
// creates its ElastiCache children, registers outputs, and returns the host:port endpoint.
func newRedisComponent(
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
	if err := ctx.RegisterComponentResource("defang-aws:index:AwsRedis", serviceName, comp, parentOpt); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering redis component %s: %w", serviceName, err)
	}
	opts := []pulumi.ResourceOption{pulumi.Parent(comp)}

	redisResult, err := createElasticache(ctx, configProvider, serviceName, svc, vpcID, privateSubnetIDs, sg, recipe, opts...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating Redis for %s: %w", serviceName, err)
	}

	port := 6379
	if len(svc.Ports) > 0 {
		port = svc.Ports[0].Target
	}
	endpoint := pulumi.StringOutput(pulumix.Apply(redisResult.address, func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, port)
	}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": endpoint}); err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return endpoint, nil
}
