package aws

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQualifiedContainerName(t *testing.T) {
	// The CLI splits on the LAST "_" and treats the tail as the etag.
	assert.Equal(t, "app_kg7lqwrfnfxa", QualifiedContainerName("app", "kg7lqwrfnfxa"))
	// A compose container_name is qualified the same way.
	assert.Equal(t, "sidecar_kg7lqwrfnfxa", QualifiedContainerName("sidecar", "kg7lqwrfnfxa"))
	// Standalone Service callers have no deployment etag; leave the name bare.
	assert.Equal(t, "app", QualifiedContainerName("app", ""))
}

func TestECSServiceResourceName(t *testing.T) {
	// The CLI takes the text between the FIRST "_" and the LAST "-" of the
	// autonamed physical name, so both the project and the service may
	// themselves contain hyphens.
	assert.Equal(t, "html-css-js_app", ECSServiceResourceName("html-css-js", "app"))
	assert.Equal(t, "proj_my-service", ECSServiceResourceName("proj", "my-service"))
	// Standalone Service callers: no project to qualify with.
	assert.Equal(t, "app", ECSServiceResourceName("", "app"))
}

// TestBuildDependsOnQualifiesNames checks that an intra-task depends_on keeps
// pointing at the container it names once container names carry the etag.
func TestBuildDependsOnQualifiesNames(t *testing.T) {
	sidecars := map[string]compose.ServiceConfig{
		"helper": {},
		"named":  {ContainerName: ptr.String("custom")},
	}
	deps := buildDependsOn(compose.DependsOnConfig{
		"helper":  {Condition: "service_healthy"},
		"named":   {Condition: "service_started"},
		"outside": {Condition: "service_started"}, // not in this task; dropped
	}, sidecars, "kg7lqwrfnfxa")

	got := make([]string, 0, len(deps))
	for _, d := range deps {
		got = append(got, *d.ContainerName)
	}
	assert.ElementsMatch(t, []string{"helper_kg7lqwrfnfxa", "custom_kg7lqwrfnfxa"}, got)
}

// TestBuildVolumesFromQualifiesNames is the volumes_from counterpart.
func TestBuildVolumesFromQualifiesNames(t *testing.T) {
	sidecars := map[string]compose.ServiceConfig{"helper": {}}
	vols := buildVolumesFrom([]string{"helper:ro"}, sidecars, "kg7lqwrfnfxa")
	require.Len(t, vols, 1)
	assert.Equal(t, "helper_kg7lqwrfnfxa", *vols[0].SourceContainer)
	assert.True(t, *vols[0].ReadOnly)
}
