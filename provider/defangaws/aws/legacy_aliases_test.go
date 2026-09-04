package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shapes below were read out of a stack the legacy defang-mvp CD deployed:
// its Pulumi project is "cd", its resources hang off the stack, and its names
// carry that same "cd" where this CD would carry the compose project. If the
// reconstruction drifts from defang-mvp's pulumi/shared/config.ts and
// pulumi/shared/aws/safe_namings.ts, migrations stop adopting and start
// replacing -- so pin the exact strings.
func TestLegacyNamesMatchTheMvpCD(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		// stack(service): the SG and the ElastiCache subnet group keep the
		// prefix's capital D, because stack() sanitizes nothing.
		assert.Equal(t, "Defang-cd-prod1-postgres", legacyStackName(ctx, "postgres"))
		assert.Equal(t, "Defang-cd-prod1-redis", legacyStackName(ctx, engineRedis))

		// awsIdentifierSafe(stack(service), 55) for the RDS instance.
		assert.Equal(t, "defang-cd-prod1-postgres", legacyRdsInstanceName(ctx, "postgres"))
		return nil
	}, pulumi.WithMocks("myproject", "prod1", noopAWSMocks{}))
	require.NoError(t, err)
}

// Playground stacks set no prefix, so stack() began at the project.
func TestLegacyNamesWithoutPrefix(t *testing.T) {
	t.Setenv("PULUMI_CONFIG", `{"defang:prefix": ""}`)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		assert.Equal(t, "cd-prod1-redis", legacyStackName(ctx, engineRedis))
		return nil
	}, pulumi.WithMocks("myproject", "prod1", noopAWSMocks{}))
	require.NoError(t, err)
}

// truncate() kept the head of the name and appended six hex characters of its
// hash, with no separator. A service name long enough to trip that must trim
// the same way or the alias names a URN that never existed.
func TestLegacyIdentifierTrimsLikeTheMvpCD(t *testing.T) {
	long := "a-very-long-service-name-that-will-certainly-overflow-the-limit"
	got := legacyTruncate("defang-cd-prod1-"+long, legacyRdsInstanceNameMaxLength)

	assert.Len(t, got, legacyRdsInstanceNameMaxLength)
	assert.Equal(t, "defang-cd-prod1-a-very-long-service-name-that-wil", got[:49])
	assert.Regexp(t, `^[0-9a-f]{6}$`, got[49:])
}

// awsIdentifierSafe() lowercased and folded each RUN of invalid characters to a
// single hyphen.
func TestLegacyIdentifierSanitization(t *testing.T) {
	assert.Equal(t, "defang-cd-prod1-my-svc", legacySanitize("Defang-cd-prod1-My_Svc"))
	assert.Equal(t, "a-b", legacySanitize("A__..B"))
}

// noopAWSMocks answers resource registrations without recording them.
type noopAWSMocks struct{}

func (noopAWSMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (noopAWSMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// legacyAliasSpy records the legacy identity each registered resource declared.
type legacyAliasSpy struct {
	noopAWSMocks
	aliases map[string][]legacyAliasSpec
}

type legacyAliasSpec struct {
	name     string
	project  string
	noParent bool
}

func (m *legacyAliasSpy) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if m.aliases == nil {
		m.aliases = map[string][]legacyAliasSpec{}
	}
	for _, alias := range args.RegisterRPC.GetAliases() {
		if spec := alias.GetSpec(); spec != nil {
			m.aliases[args.Name] = append(m.aliases[args.Name], legacyAliasSpec{
				name:     spec.GetName(),
				project:  spec.GetProject(),
				noParent: spec.GetNoParent(),
			})
		}
	}
	return args.Name + "_id", args.Inputs, nil
}

// Without these aliases an `up` over a legacy defang-mvp stack plans a create
// plus a delete for the database rather than an update, and the delete takes
// the data with it.
func TestRDSAdoptsLegacyMvpNames(t *testing.T) {
	spy := &legacyAliasSpy{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateRDS(ctx, &compose.DryRunConfigProvider{}, "postgres", compose.ServiceConfig{
			Image:    pulumi.String("postgres:16"),
			Postgres: &compose.PostgresConfig{},
			Environment: compose.Environment{
				"POSTGRES_PASSWORD": pulumi.String("literal-password"),
			},
		},
			pulumi.String("vpc-123"),
			pulumi.StringArray{pulumi.String("subnet-1"), pulumi.String("subnet-2")},
			nil, nil, nil, pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("myproject", "prod1", spy))
	require.NoError(t, err)

	assert.Contains(t, spy.aliases["postgres"], legacyAliasSpec{
		name: "defang-cd-prod1-postgres", project: legacyProject, noParent: true,
	}, "RDS instance would be replaced instead of adopted")
	assert.Contains(t, spy.aliases["postgres"], legacyAliasSpec{
		name: "Defang-cd-prod1-postgres", project: legacyProject, noParent: true,
	}, "RDS security group would be replaced instead of adopted")
	assert.Contains(t, spy.aliases["postgres"], legacyAliasSpec{
		name: "postgres", project: legacyProject, noParent: true,
	}, "RDS subnet group would be replaced instead of adopted")
}

func TestElasticacheAdoptsLegacyMvpNames(t *testing.T) {
	const service = engineRedis // the compose service is named after its engine here
	spy := &legacyAliasSpy{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := CreateElasticache(ctx, service, compose.ServiceConfig{
			Image: pulumi.String("redis:7"),
			Redis: &compose.RedisConfig{},
		},
			pulumi.String("vpc-123"),
			pulumi.StringArray{pulumi.String("subnet-1"), pulumi.String("subnet-2")},
			nil, nil, nil, pulumi.Parent(nil))
		return err
	}, pulumi.WithMocks("myproject", "prod1", spy))
	require.NoError(t, err)

	assert.Contains(t, spy.aliases[service], legacyAliasSpec{
		name: service, project: legacyProject, noParent: true,
	}, "replication group would be replaced instead of adopted")
	assert.Contains(t, spy.aliases[service], legacyAliasSpec{
		name: "Defang-cd-prod1-" + service, project: legacyProject, noParent: true,
	}, "ElastiCache subnet group and security group would be replaced instead of adopted")
}
