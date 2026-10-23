package aws

// ObjectStore is the standalone S3 bucket component for AWS. These tests verify
// that the ObjectStore component constructs successfully and surfaces non-empty
// outputs for a declared bucket name.

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DefangLabs/pulumi-defang/tests/testutil"
)

func TestConstructAwsObjectStore(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	resp, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("ObjectStore"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"objectStore": property.New(property.NewMap(map[string]property.Value{
				"bucket": property.New("myproject-uploads"),
			})),
			"aws": awsConfig,
		}),
	})
	require.NoError(t, err)

	// endpoint/region/arn are cloud-computed fields the mock resource monitor
	// doesn't synthesize (it only echoes back explicit inputs), so only their
	// presence is checked, same as the Project ALB-ARN outputs in
	// project_test.go. "bucket" is an explicit input (via BucketArgs.Bucket),
	// so the mock echoes it back and its value can be asserted directly.
	for _, key := range []string{"endpoint", "bucket", "region", "arn"} {
		_, ok := resp.State.GetOk(key)
		assert.True(t, ok, "missing output %q", key)
	}

	v, ok := resp.State.GetOk("bucket")
	require.True(t, ok, "missing output %q", "bucket")
	assert.Equal(t, "myproject-uploads", v.AsString())
	assert.NotEmpty(t, v.AsString())
}

func TestConstructAwsObjectStoreNoInfra(t *testing.T) {
	server := testutil.MakeAwsTestServer()

	_, err := server.Construct(p.ConstructRequest{
		Urn: testutil.AwsURN("ObjectStore"),
		Inputs: property.NewMap(map[string]property.Value{
			"projectName": property.New("myproject"),
			"objectStore": property.New(property.NewMap(map[string]property.Value{
				"bucket": property.New("myproject-uploads"),
			})),
		}),
	})
	require.NoError(t, err)
}

// bucketArnMock echoes inputs back like the default mock, but synthesizes the
// `arn` output for S3 buckets — the ObjectStore hands that ARN to the task-role
// grant, so without it the policy document would never resolve — and records
// every inline role policy the project creates.
type bucketArnMock struct {
	mu       sync.Mutex
	policies map[string]string // resource name -> policy document JSON
	ecsNames []string
}

func newBucketArnMock() (*bucketArnMock, *integration.MockResourceMonitor) {
	m := &bucketArnMock{policies: map[string]string{}}
	return m, &integration.MockResourceMonitor{
		NewResourceF: func(args integration.MockResourceArgs) (string, property.Map, error) {
			m.mu.Lock()
			defer m.mu.Unlock()
			switch string(args.TypeToken) {
			case "aws:s3/bucket:Bucket":
				bucket := args.Inputs.Get("bucket").AsString()
				return args.Name, args.Inputs.Set("arn", property.New("arn:aws:s3:::"+bucket)), nil
			case "aws:iam/rolePolicy:RolePolicy":
				if policy, ok := args.Inputs.GetOk("policy"); ok && policy.IsString() {
					m.policies[args.Name] = policy.AsString()
				}
			case "aws:ecs/service:Service":
				m.ecsNames = append(m.ecsNames, args.Name)
			}
			return args.Name, args.Inputs, nil
		},
	}
}

func (m *bucketArnMock) policy(name string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := m.policies[name]
	return doc, ok
}

func (m *bucketArnMock) services() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.ecsNames)
}

// objectStoreProject is a project with one x-defang-s3 service and one
// container service that declares depends_on it (or not, per dependsOn).
func objectStoreProject(dependsOn bool) property.Map {
	app := map[string]property.Value{"image": property.New("myapp:latest")}
	if dependsOn {
		app["dependsOn"] = property.New(property.NewMap(map[string]property.Value{
			"uploads": property.New(property.NewMap(map[string]property.Value{
				"required": property.New(true),
			})),
		}))
	}
	return testutil.ServicesMap(map[string]property.Value{
		"uploads": property.New(property.NewMap(map[string]property.Value{
			"image": property.New("minio/minio"),
			"objectStore": property.New(property.NewMap(map[string]property.Value{
				"bucket": property.New("myproject-uploads"),
			})),
		})),
		"app": property.New(property.NewMap(app)),
	})
}

// A service that depends_on an x-defang-s3 store gets an inline task-role
// policy for that bucket. The bucket-level half matters as much as the object
// half: HeadBucket — the credential probe many S3 clients run at startup — is
// authorized by s3:ListBucket on the bucket ARN, not by any object action.
func TestConstructAwsProjectObjectStoreGrant(t *testing.T) {
	m, mock := newBucketArnMock()
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AwsURN("Project"),
		Inputs: objectStoreProject(true),
	})
	require.NoError(t, err)

	doc, ok := m.policy("app-objectstore-uploads")
	require.True(t, ok, "no object-store policy attached to the app task role")

	var policy struct {
		Statement []struct {
			Sid      string   `json:"Sid"`
			Effect   string   `json:"Effect"`
			Action   []string `json:"Action"`
			Resource string   `json:"Resource"`
		} `json:"Statement"`
	}
	require.NoError(t, json.Unmarshal([]byte(doc), &policy))
	require.Len(t, policy.Statement, 2)

	bucket, objects := policy.Statement[0], policy.Statement[1]
	assert.Equal(t, "arn:aws:s3:::myproject-uploads", bucket.Resource)
	assert.Contains(t, bucket.Action, "s3:ListBucket")
	assert.Contains(t, bucket.Action, "s3:GetBucketLocation")
	assert.Equal(t, "arn:aws:s3:::myproject-uploads/*", objects.Resource)
	assert.Contains(t, objects.Action, "s3:GetObject")
	assert.Contains(t, objects.Action, "s3:PutObject")
	for _, stmt := range policy.Statement {
		assert.Equal(t, "Allow", stmt.Effect)
	}

	// The store itself is a bucket, not a task: it must not reach ECS even
	// though the compose service carries a minio image anchor. The name is
	// project-qualified ("<project>_<service>", and AwsURN names the component
	// "name") because the CLI parses the service out of the ECS event.
	assert.Equal(t, []string{"name_app"}, m.services())
}

// depends_on is the wiring contract — it is what the CLI injects
// <STORE>_BUCKET off — so a service that does not declare it gets no grant.
func TestConstructAwsProjectObjectStoreNoGrantWithoutDependsOn(t *testing.T) {
	m, mock := newBucketArnMock()
	server := testutil.MakeAwsTestServer(integration.WithMocks(mock))

	_, err := server.Construct(p.ConstructRequest{
		Urn:    testutil.AwsURN("Project"),
		Inputs: objectStoreProject(false),
	})
	require.NoError(t, err)

	_, ok := m.policy("app-objectstore-uploads")
	assert.False(t, ok, "granted bucket access to a service that does not depend on the store")
}
