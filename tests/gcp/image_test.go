package gcp

// Tests for external registry image handling: Artifact Registry remote
// repository creation and image URL rewriting for Cloud Run and Compute Engine.
//
// When a service uses an image from a registry not natively supported by Cloud
// Run (e.g. ghcr.io, quay.io), the provider must:
//  1. Create an AR REMOTE_REPOSITORY that proxies the external registry, and
//  2. Rewrite the image reference to point at that AR path.
//
// Both steps are required: the rewrite without the repo → 404 on pull;
// the repo without the rewrite → Cloud Run rejects the registry URL.

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

const gcpARRepositoryType = "gcp:artifactregistry/repository:Repository"

// isRemoteRepo returns true if the AR repository record represents a
// REMOTE_REPOSITORY (pull-through cache), not a regular push repository.
func isRemoteRepo(m property.Map) bool {
	return m.Get("mode").AsString() == "REMOTE_REPOSITORY"
}

// TestExternalRegistryCreatesRemoteRepo verifies that deploying a service with
// an image from an unsupported registry (ghcr.io) creates one AR remote repo.
func TestExternalRegistryCreatesRemoteRepo(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"chat": testutil.ServiceWithPorts("ghcr.io/berriai/litellm:main-stable", testutil.HostPort(4000)),
		}),
	})

	require.NoError(t, err)

	remoteRepos := countTypeWhere(*records, gcpARRepositoryType, isRemoteRepo)
	assert.Equal(t, 1, remoteRepos, "expected exactly one AR remote repo for ghcr.io")
}

// TestExternalRegistryRemoteRepoHasCorrectConfig verifies the remote repo is
// configured with the right ID and upstream URI.
func TestExternalRegistryRemoteRepoHasCorrectConfig(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"chat": testutil.ServiceWithPorts("ghcr.io/berriai/litellm:main-stable", testutil.HostPort(4000)),
		}),
	})

	require.NoError(t, err)

	repo := findTypeWhere(*records, gcpARRepositoryType, isRemoteRepo)
	require.NotNil(t, repo, "expected an AR remote repo")
	assert.Equal(t, "ghcr-io", repo.inputs.Get("repositoryId").AsString(),
		"repo ID should be sanitized registry name")
	assert.Equal(t, "REMOTE_REPOSITORY", repo.inputs.Get("mode").AsString())

	uri := repo.inputs.
		Get("remoteRepositoryConfig").AsMap().
		Get("commonRepository").AsMap().
		Get("uri").AsString()
	assert.Equal(t, "https://ghcr.io", uri, "remote repo URI should point at the original registry")
}

// TestSupportedRegistryDoesNotCreateRemoteRepo verifies that Cloud-Run-native
// registries (docker.io, gcr.io, Artifact Registry) don't trigger remote repo creation.
func TestSupportedRegistryDoesNotCreateRemoteRepo(t *testing.T) {
	for _, image := range []string{
		"nginx:latest",
		"docker.io/library/nginx:1.25",
		"gcr.io/my-project/app:v1",
		"us-central1-docker.pkg.dev/proj/repo/app:v1",
	} {
		t.Run(image, func(t *testing.T) {
			mock, records := collectResources()
			server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

			_, err := server.Construct(p.ConstructRequest{
				Urn: testutil.GcpURN("Project"),
				Inputs: testutil.ServicesMap(map[string]property.Value{
					"app": testutil.ServiceWithPorts(image, testutil.IngressPort(8080)),
				}),
			})

			require.NoError(t, err)
			remoteRepos := countTypeWhere(*records, gcpARRepositoryType, isRemoteRepo)
			assert.Equal(t, 0, remoteRepos,
				"supported registry %q should not create an AR remote repo", image)
		})
	}
}

// TestExternalRegistryDeduplicatedAcrossServices verifies that two services
// using the same external registry only create one remote repo.
func TestExternalRegistryDeduplicatedAcrossServices(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"chat":      testutil.ServiceWithPorts("ghcr.io/berriai/litellm:main-stable", testutil.HostPort(4000)),
			"embedding": testutil.ServiceWithPorts("ghcr.io/berriai/litellm:main-stable", testutil.HostPort(4000)),
		}),
	})

	require.NoError(t, err)

	remoteRepos := countTypeWhere(*records, gcpARRepositoryType, isRemoteRepo)
	assert.Equal(t, 1, remoteRepos,
		"two services from the same external registry should share one remote repo")
}

// TestMultipleExternalRegistriesCreateSeparateRemoteRepos verifies that two
// services from different external registries each get their own remote repo.
func TestMultipleExternalRegistriesCreateSeparateRemoteRepos(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			"a": testutil.ServiceWithImage("ghcr.io/owner/image:latest"),
			"b": testutil.ServiceWithImage("quay.io/prometheus/node-exporter:v1.8.0"),
		}),
	})

	require.NoError(t, err)

	remoteRepos := countTypeWhere(*records, gcpARRepositoryType, isRemoteRepo)
	assert.Equal(t, 2, remoteRepos,
		"two services from different external registries should create two remote repos")

	ghcrRepo := findTypeWhere(*records, gcpARRepositoryType, func(m property.Map) bool {
		return isRemoteRepo(m) && m.Get("repositoryId").AsString() == "ghcr-io"
	})
	quayRepo := findTypeWhere(*records, gcpARRepositoryType, func(m property.Map) bool {
		return isRemoteRepo(m) && m.Get("repositoryId").AsString() == "quay-io"
	})
	assert.NotNil(t, ghcrRepo, "expected remote repo for ghcr.io")
	assert.NotNil(t, quayRepo, "expected remote repo for quay.io")
}

// TestExternalRegistryImageIsRewrittenInCloudInit verifies that a Compute Engine
// service's cloud-init user-data references the rewritten AR path, not the
// original external registry URL.
func TestExternalRegistryImageIsRewrittenInCloudInit(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			// host port → Compute Engine
			"chat": testutil.ServiceWithPorts("ghcr.io/berriai/litellm:main-stable", testutil.HostPort(4000)),
		}),
	})

	require.NoError(t, err)

	it := findTypeWhere(*records, gcpInstanceTemplateType, func(_ property.Map) bool { return true })
	require.NotNil(t, it, "expected an instance template for Compute Engine service")

	userData := it.inputs.Get("metadata").AsMap().Get("user-data").AsString()
	assert.NotContains(t, userData, "ghcr.io",
		"cloud-init should not reference ghcr.io directly; image should be rewritten to AR")
	assert.Contains(t, userData, "docker.pkg.dev",
		"cloud-init should reference the Artifact Registry path for the rewritten image")
	assert.Contains(t, userData, "ghcr-io",
		"cloud-init should contain the remote repo path segment (ghcr-io)")
}

// TestExternalRegistryImageIsRewrittenForCloudRun verifies that a Cloud Run
// service's image reference is rewritten from the external registry to the AR path.
func TestExternalRegistryImageIsRewrittenForCloudRun(t *testing.T) {
	mock, records := collectResources()
	server := testutil.MakeGcpTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.GcpURN("Project"),
		Inputs: testutil.ServicesMap(map[string]property.Value{
			// ingress port → Cloud Run
			"app": testutil.ServiceWithPorts("ghcr.io/owner/myapp:v1", testutil.IngressPort(8080)),
		}),
	})

	require.NoError(t, err)

	cr := findTypeWhere(*records, gcpCloudRunServiceType, func(_ property.Map) bool { return true })
	require.NotNil(t, cr, "expected a Cloud Run service")

	containers := cr.inputs.Get("template").AsMap().Get("containers").AsArray()
	require.Equal(t, 1, containers.Len())
	image := containers.Get(0).AsMap().Get("image").AsString()

	assert.NotContains(t, image, "ghcr.io",
		"Cloud Run image should not reference ghcr.io directly; got: %s", image)
	assert.Contains(t, image, "docker.pkg.dev",
		"Cloud Run image should reference Artifact Registry path; got: %s", image)
	assert.Contains(t, image, "ghcr-io",
		"Cloud Run image should contain the remote repo path segment; got: %s", image)
}
