package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

type migrationAliasMocks struct {
	aliases map[string][]string
}

func (m *migrationAliasMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if m.aliases == nil {
		m.aliases = map[string][]string{}
	}
	if args.RegisterRPC != nil {
		aliases := append([]string{}, args.RegisterRPC.GetAliasURNs()...)
		for _, alias := range args.RegisterRPC.GetAliases() {
			if urn := alias.GetUrn(); urn != "" {
				aliases = append(aliases, urn)
			}
		}
		m.aliases[args.TypeToken] = aliases
	}
	return args.Name + "_id", args.Inputs, nil
}

func (*migrationAliasMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestRDSRegistersExactMigrationAliases(t *testing.T) {
	const (
		instanceURN = "urn:pulumi:prod::app::aws:rds/instance:Instance::legacy-db"
		subnetURN   = "urn:pulumi:prod::app::aws:rds/subnetGroup:SubnetGroup::legacy-db"
		securityURN = "urn:pulumi:prod::app::aws:ec2/securityGroup:SecurityGroup::legacy-db"
	)
	mocks := &migrationAliasMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateRDS(ctx, &compose.DryRunConfigProvider{}, "db", compose.ServiceConfig{
			Image:    pulumi.String("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Aliases: map[string]string{
				compose.AliasInstance:      instanceURN,
				compose.AliasSubnetGroup:   subnetURN,
				compose.AliasSecurityGroup: securityURN,
			},
		}, pulumi.String("vpc"), pulumi.StringArray{pulumi.String("subnet")}, pulumi.StringPtr("source-sg"), nil, nil)
		return err
	}, pulumi.WithMocks("app", "prod", mocks))
	require.NoError(t, err)
	require.Contains(t, mocks.aliases["aws:rds/instance:Instance"], instanceURN)
	require.Contains(t, mocks.aliases["aws:rds/subnetGroup:SubnetGroup"], subnetURN)
	require.Contains(t, mocks.aliases["aws:ec2/securityGroup:SecurityGroup"], securityURN)
}

func TestElastiCacheRegistersExactMigrationAlias(t *testing.T) {
	const clusterURN = "urn:pulumi:prod::app::aws:elasticache/replicationGroup:ReplicationGroup::legacy-cache"
	mocks := &migrationAliasMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateElasticache(ctx, "cache", compose.ServiceConfig{
			Image: pulumi.String("redis:7.2"),
			Redis: &compose.RedisConfig{},
			Aliases: map[string]string{
				compose.AliasCluster: clusterURN,
			},
		}, pulumi.String("vpc"), nil, nil, nil, nil)
		return err
	}, pulumi.WithMocks("app", "prod", mocks))
	require.NoError(t, err)
	require.Contains(t, mocks.aliases["aws:elasticache/replicationGroup:ReplicationGroup"], clusterURN)
}
