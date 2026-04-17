package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// Redis is the controller struct for the defang-aws:index:Redis component.
type Redis struct{}

// RedisInputs defines the inputs for a standalone AWS ElastiCache Redis instance.
type RedisInputs struct {
	ProjectName string                      `pulumi:"project_name"`
	Redis       *compose.RedisConfig        `pulumi:"redis,optional"`
	Image       *string                     `pulumi:"image,optional"`
	Ports       []compose.ServicePortConfig `pulumi:"ports,optional"`
	Deploy      *compose.DeployConfig       `pulumi:"deploy,optional"`
	Environment map[string]string           `pulumi:"environment,optional"`
	AWS         *provideraws.SharedInfra    `pulumi:"aws,optional"`
}

// RedisOutputs holds the outputs of an AWS Redis component.
type RedisOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
	// Dependency is an internal-only handle (CNAME record when a private zone
	// exists, otherwise nil) used by downstream services for ordering. Untagged —
	// not part of the SDK schema.
	Dependency pulumi.Resource
}

// RedisComponentType is the Pulumi resource type token for the Redis component.
const RedisComponentType = "defang-aws:index:Redis"

// Construct implements the ComponentResource interface for Redis.
func (*Redis) Construct(
	ctx *pulumi.Context, name, typ string, inputs RedisInputs, opts pulumi.ResourceOption,
) (*RedisOutputs, error) {
	comp := &RedisOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	redis := inputs.Redis
	if redis == nil {
		redis = &compose.RedisConfig{}
	}

	svc := compose.ServiceConfig{
		Redis:       redis,
		Image:       inputs.Image,
		Ports:       inputs.Ports,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	if err := createRedis(ctx, comp, name, svc, inputs.AWS, nil); err != nil {
		return nil, err
	}
	return comp, nil
}

// createRedis creates the ElastiCache cluster (plus optional private-zone CNAME)
// under an already-registered Redis component, sets its Endpoint/Dependency, and
// registers its outputs. Shared between Construct and the project-level dispatcher.
func createRedis(
	ctx *pulumi.Context,
	comp *RedisOutputs,
	serviceName string,
	svc compose.ServiceConfig,
	infra *provideraws.SharedInfra,
	deps []pulumi.Resource,
) error {
	childOpt := pulumi.Parent(comp)

	redisResult, err := provideraws.CreateElasticache(
		ctx, serviceName, svc, infra.VpcID, infra.PrivateSubnetIDs, infra.PrivateSgID, deps, childOpt,
	)
	if err != nil {
		return fmt.Errorf("creating Redis for %s: %w", serviceName, err)
	}

	var dependency pulumi.Resource // stays nil when no CNAME and no explicit cluster handle
	if infra.PrivateZoneID != nil {
		privateFqdn := common.SafeLabel(serviceName) //+ "." + infra.PrivateDomain
		record, cnameErr := provideraws.CreateRecord(ctx, privateFqdn, common.RecordTypeCNAME, &route53.RecordArgs{
			ZoneId:  infra.PrivateZoneID.ToStringPtrOutput().Elem(),
			Records: pulumi.StringArray{redisResult.Address},
			Ttl:     pulumi.Int(300),
		}, childOpt)
		if cnameErr != nil {
			return fmt.Errorf("creating CNAME for %s: %w", serviceName, cnameErr)
		}
		dependency = record
	}
	comp.Dependency = dependency

	port := int32(6379)
	if len(svc.Ports) > 0 {
		port = svc.Ports[0].Target
	}
	comp.Endpoint = pulumi.StringOutput(pulumix.Apply(redisResult.Address, func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, port)
	}))

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{"endpoint": comp.Endpoint}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", serviceName, err)
	}
	return nil
}
