package aws

import (
	"fmt"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type rdsResult struct {
	instance *rds.Instance
}

// rdsInstanceClass maps CPU/memory to an RDS instance class.
func rdsInstanceClass(cpus float64, memMiB int) string {
	switch {
	case cpus <= 1 && memMiB <= 2048:
		return "db.t4g.micro"
	case cpus <= 1 && memMiB <= 4096:
		return "db.t4g.small"
	case cpus <= 2 && memMiB <= 8192:
		return "db.t4g.medium"
	case cpus <= 2:
		return "db.t4g.large"
	case cpus <= 4:
		return "db.m7g.xlarge"
	default:
		return "db.m7g.2xlarge"
	}
}

// postgresEngineVersion maps a major version to an RDS engine version string.
func postgresEngineVersion(version int) string {
	switch version {
	case 14:
		return "14"
	case 15:
		return "15"
	case 16:
		return "16"
	case 17:
		return "17"
	default:
		return "17"
	}
}

// createRDS creates a managed RDS Postgres instance for a service.
func createRDS(
	ctx *pulumi.Context,
	serviceName string,
	svc common.ServiceConfig,
	vpcID pulumi.StringOutput,
	privateSubnetIDs pulumi.StringArrayOutput,
	serviceSG *ec2.SecurityGroup,
	recipe common.AWSRecipe,
	opts ...pulumi.ResourceOption,
) (*rdsResult, error) {
	pg := svc.Postgres
	if pg == nil {
		return nil, fmt.Errorf("postgres config is nil")
	}

	// Create DB subnet group
	subnetGroup, err := rds.NewSubnetGroup(ctx, serviceName, &rds.SubnetGroupArgs{
		Description: pulumi.String(fmt.Sprintf("Subnet group for %s postgres", serviceName)),
		SubnetIds:   privateSubnetIDs,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating DB subnet group: %w", err)
	}

	// Create security group for RDS
	rdsSG, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       vpcID,
		Description: pulumi.String(fmt.Sprintf("RDS security group for %s", serviceName)),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(5432),
				ToPort:         pulumi.Int(5432),
				SecurityGroups: pulumi.StringArray{serviceSG.ID()},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating RDS security group: %w", err)
	}

	instanceClass := rdsInstanceClass(svc.GetCPUs(), svc.GetMemoryMiB())

	instance, err := rds.NewInstance(ctx, serviceName, &rds.InstanceArgs{
		AllocatedStorage:        pulumi.Int(20),
		MaxAllocatedStorage:     pulumi.Int(500),
		Engine:                  pulumi.String("postgres"),
		EngineVersion:           pulumi.String(postgresEngineVersion(pg.Version)),
		InstanceClass:           pulumi.String(instanceClass),
		DbName:                  pulumi.String(pg.DBName),
		Username:                pulumi.String(pg.Username),
		Password:                pulumi.String(pg.Password),
		DbSubnetGroupName:       subnetGroup.Name,
		VpcSecurityGroupIds:     pulumi.StringArray{rdsSG.ID()},
		SkipFinalSnapshot:       pulumi.Bool(true),
		PubliclyAccessible:      pulumi.Bool(false),
		DeletionProtection:      pulumi.Bool(recipe.DeletionProtection),
		StorageEncrypted:        pulumi.Bool(recipe.StorageEncrypted),
		AutoMinorVersionUpgrade: pulumi.Bool(true),
		BackupRetentionPeriod:   pulumi.Int(recipe.BackupRetentionDays),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating RDS instance: %w", err)
	}

	return &rdsResult{
		instance: instance,
	}, nil
}
