package gcp

import (
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ghcrRegistry = "ghcr.io"

// pulumiAutonamingConfigKey is the stack config key AutonamingPrefix reads.
// Shared across test files so the literal isn't repeated (goconst).
const pulumiAutonamingConfigKey = "pulumi:autonaming"

type artifactRepositoryRegistration struct {
	logicalName    string
	repositoryID   string
	retainOnDelete bool
}

type artifactRepositoryMocks struct {
	mu           sync.Mutex
	repositories []artifactRepositoryRegistration
}

func (m *artifactRepositoryMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Copy()
	if args.TypeToken == "gcp:artifactregistry/repository:Repository" {
		registration := artifactRepositoryRegistration{
			logicalName:  args.Name,
			repositoryID: args.Inputs["repositoryId"].StringValue(),
		}
		if args.RegisterRPC != nil {
			registration.retainOnDelete = args.RegisterRPC.GetRetainOnDelete()
		}

		m.mu.Lock()
		m.repositories = append(m.repositories, registration)
		m.mu.Unlock()

		outputs["name"] = resource.NewStringProperty(
			"projects/gcp-project/locations/us-central1/repositories/" + registration.repositoryID,
		)
	}
	if args.TypeToken == "gcp:serviceaccount/account:Account" {
		outputs["email"] = resource.NewStringProperty(args.Name + "@gcp-project.iam.gserviceaccount.com")
		outputs["name"] = resource.NewStringProperty(args.Name)
	}
	if args.TypeToken == "gcp:storage/bucket:Bucket" {
		outputs["name"] = resource.NewStringProperty(args.Name)
	}
	return args.Name + "_id", outputs, nil
}

func (m *artifactRepositoryMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func withPulumiConfig(config map[string]string) pulumi.RunOption {
	return func(info *pulumi.RunInfo) {
		info.Config = config
	}
}

func repositoryIDForTest(
	t *testing.T,
	stack, composeProject, logicalName string,
	config map[string]string,
) string {
	t.Helper()

	var repositoryID string
	opts := []pulumi.RunOption{pulumi.WithMocks("pulumi-project", stack, testMocks{})}
	if config != nil {
		opts = append(opts, withPulumiConfig(config))
	}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		repositoryID = artifactRegistryRepositoryID(ctx, composeProject, logicalName)
		return nil
	}, opts...)
	require.NoError(t, err)
	return repositoryID
}

func TestArtifactRegistryRepositoryIDFallsBackToLegacyScope(t *testing.T) {
	devID := repositoryIDForTest(t, "dev", "compose-project", "repo", nil)
	prodID := repositoryIDForTest(t, "prod", "compose-project", "repo", nil)
	renamedID := repositoryIDForTest(t, "dev", "renamed-project", "repo", nil)

	assert.Equal(t, "defang-compose-project-dev-repo", devID)
	assert.Equal(t, "defang-compose-project-prod-repo", prodID)
	assert.Equal(t, "defang-renamed-project-dev-repo", renamedID)
	assert.NotEqual(t, devID, prodID)
	assert.NotEqual(t, devID, renamedID)
}

func TestArtifactRegistryRepositoryIDHonorsCustomAutonaming(t *testing.T) {
	config := map[string]string{
		pulumiAutonamingConfigKey: `{"pattern":"Custom-${project}-${stack}-${name}-${hex(7)}"}`,
	}

	repositoryID := repositoryIDForTest(t, "dev", "compose-project", "repo", config)
	assert.Equal(t, "custom-pulumi-project-dev-repo", repositoryID)
}

func TestArtifactRegistryRepositoryIDCustomLongNamesAreTruncated(t *testing.T) {
	config := map[string]string{
		pulumiAutonamingConfigKey: `{"pattern":"${project}-${stack}-${name}-${hex(7)}"}`,
	}
	commonPrefix := strings.Repeat("long", 20)

	firstID := repositoryIDForTest(t, "dev", "compose-project", commonPrefix+"first", config)
	secondID := repositoryIDForTest(t, "dev", "compose-project", commonPrefix+"second", config)

	require.LessOrEqual(t, len(firstID), artifactRegistryRepositoryIDMaxLength)
	require.LessOrEqual(t, len(secondID), artifactRegistryRepositoryIDMaxLength)
	assert.Equal(t, firstID, repositoryIDForTest(t, "dev", "compose-project", commonPrefix+"first", config),
		"repository IDs must be deterministic")
	assert.NotEqual(t, firstID, secondID, "distinct names sharing a long prefix must not collide")
}

func TestCreateRemoteReposKeepsLogicalNameAndDoesNotRetain(t *testing.T) {
	mocks := &artifactRepositoryMocks{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		_, err := createRemoteRepos(ctx, "compose-project", []string{ghcrRegistry})
		return err
	}, pulumi.WithMocks("pulumi-project", "dev", mocks))
	require.NoError(t, err)

	require.Len(t, mocks.repositories, 1)
	assert.Equal(t, artifactRepositoryRegistration{
		logicalName:    ghcrRegistry,
		repositoryID:   "defang-compose-project-dev-ghcr-io",
		retainOnDelete: false,
	}, mocks.repositories[0])
}

func TestCreateBuildInfraUsesScopedRepositoryIDInImageURL(t *testing.T) {
	mocks := &artifactRepositoryMocks{}
	url := make(chan string, 1)
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		infra, err := createBuildInfra(ctx, "compose-project")
		if err != nil {
			return err
		}
		infra.RepositoryURL.ApplyT(func(repositoryURL string) string {
			url <- repositoryURL
			return repositoryURL
		})
		return nil
	},
		pulumi.WithMocks("pulumi-project", "dev", mocks),
		withPulumiConfig(map[string]string{"gcp:project": "gcp-project"}),
	)
	require.NoError(t, err)

	require.Len(t, mocks.repositories, 1)
	assert.Equal(t, artifactRepositoryRegistration{
		logicalName:    "repo",
		repositoryID:   "defang-compose-project-dev-repo",
		retainOnDelete: false,
	}, mocks.repositories[0])
	assert.Equal(t,
		"us-central1-docker.pkg.dev/gcp-project/defang-compose-project-dev-repo",
		<-url,
	)
}

// TestCdSourceBucket checks that the shared CD bucket is parsed out of the
// defang:stateUrl config the CD program sets from DEFANG_STATE_URL, and that a
// non-GCS (or absent) value yields no bucket instead of a bogus name.
func TestCdSourceBucket(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected string
	}{
		{name: "unset", config: `{}`, expected: ""},
		{name: "gcs url", config: `{"defang:stateUrl": "gs://defang-cd-abc123"}`, expected: "defang-cd-abc123"},
		{
			name:     "gcs url with path",
			config:   `{"defang:stateUrl": "gs://defang-cd-abc123/pulumi"}`,
			expected: "defang-cd-abc123",
		},
		{name: "s3 url", config: `{"defang:stateUrl": "s3://defang-cd-abc123"}`, expected: ""},
		{name: "not a url", config: `{"defang:stateUrl": ":://"}`, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PULUMI_CONFIG", tt.config)
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				assert.Equal(t, tt.expected, cdSourceBucket(ctx))
				return nil
			}, pulumi.WithMocks("proj", "stack", testMocks{}))
			require.NoError(t, err)
		})
	}
}
