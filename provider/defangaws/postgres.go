package defangaws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	provideraws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AwsPostgres is the controller struct for the defang-aws:index:AwsPostgres component.
type AwsPostgres struct{}

// AwsPostgresInputs defines the inputs for a standalone AWS RDS Postgres instance.
type AwsPostgresInputs struct {
	Project     *string                `pulumi:"project"`
	Postgres    *shared.PostgresInput  `pulumi:"postgres,optional"`
	Image       *string                `pulumi:"image,optional"`
	Deploy      *shared.DeployConfig   `pulumi:"deploy,optional"`
	Environment map[string]*string     `pulumi:"environment,optional"`
	AWS         *shared.AWSConfigInput `pulumi:"aws,optional"`
}

// AwsPostgresOutputs holds the outputs of an AwsPostgres component.
type AwsPostgresOutputs struct {
	pulumi.ResourceState
	Endpoint pulumi.StringOutput `pulumi:"endpoint"`
}

// Construct implements the ComponentResource interface for AwsPostgres.
func (*AwsPostgres) Construct(ctx *pulumi.Context, name, typ string, inputs AwsPostgresInputs, opts pulumi.ResourceOption) (*AwsPostgresOutputs, error) {
	comp := &AwsPostgresOutputs{}
	if err := ctx.RegisterComponentResource(typ, name, comp, opts); err != nil {
		return nil, err
	}

	childOpt := pulumi.Parent(comp)
	svc := shared.ServiceInput{
		Postgres:    inputs.Postgres,
		Image:       inputs.Image,
		Deploy:      inputs.Deploy,
		Environment: inputs.Environment,
	}

	result, err := provideraws.BuildStandalonePostgres(ctx, name, svc, common.ToAWSConfig(inputs.AWS), childOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS Postgres: %w", err)
	}

	comp.Endpoint = result.Endpoint

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"endpoint": result.Endpoint,
	}); err != nil {
		return nil, err
	}

	return comp, nil
}
