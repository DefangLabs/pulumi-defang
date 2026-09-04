package program

import (
	"testing"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gcpProgramMocks struct {
	projectInputs resource.PropertyMap
}

func (m *gcpProgramMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}
	for key, value := range args.Inputs {
		outputs[key] = value
	}
	if args.TypeToken == "defang-gcp:index:Project" {
		m.projectInputs = args.Inputs
		outputs["endpoints"] = resource.NewObjectProperty(resource.PropertyMap{})
		outputs["loadBalancerDns"] = resource.NewNullProperty()
		outputs["serviceIds"] = resource.NewObjectProperty(resource.PropertyMap{})
	}
	return args.Name + "_id", outputs, nil
}

func (*gcpProgramMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestToGCPArgsCopiesProjectDomain(t *testing.T) {
	cf := &compose.Project{Name: "proj", Services: compose.Services{}}

	withDomain := toGCPArgs(cf, "proj.example.com", "etag")
	require.NotNil(t, withDomain.Domain)
	assert.Equal(t, "proj.example.com", *withDomain.Domain)

	withoutDomain := toGCPArgs(cf, "", "etag")
	assert.Nil(t, withoutDomain.Domain)
}

// This exercises the whole CD boundary: NewRun reads defang:domain, passes it
// through deployGCP, and toGCPArgs puts it on the remote Project component.
// A converter-only test would not catch either caller silently dropping it.
func TestNewRunThreadsDomainIntoGCPProject(t *testing.T) {
	t.Setenv("PULUMI_CONFIG", `{
		"defang:provider":"gcp",
		"defang:domain":"proj.example.com",
		"gcp:project":"test-project",
		"gcp:region":"us-central1"
	}`)
	mocks := &gcpProgramMocks{}
	err := pulumi.RunErr(NewRun(&defangv1.ProjectUpdate{
		Compose: []byte("services:\n  api:\n    image: nginx:alpine\n"),
	}), pulumi.WithMocks("proj", "stack", mocks))
	require.NoError(t, err)
	require.NotNil(t, mocks.projectInputs, "GCP Project component was not registered")
	require.Contains(t, mocks.projectInputs, resource.PropertyKey("domain"))
	assert.Equal(t, "proj.example.com", mocks.projectInputs["domain"].StringValue())
}
