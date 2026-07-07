package compose

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// taggableArgs mimics an AWS/Azure resource Args struct exposing Tags.
// ElementType makes it satisfy pulumi.Input, the type of ResourceTransformationArgs.Props.
type taggableArgs struct {
	Tags pulumi.StringMapInput
}

func (taggableArgs) ElementType() reflect.Type { return reflect.TypeOf(taggableArgs{}) }

// labelledArgs mimics a GCP resource Args struct exposing Labels.
type labelledArgs struct {
	Labels pulumi.StringMapInput
}

func (labelledArgs) ElementType() reflect.Type { return reflect.TypeOf(labelledArgs{}) }

// untaggableArgs has no Tags/Labels field (e.g. an attachment resource).
type untaggableArgs struct {
	Name pulumi.StringInput
}

func (untaggableArgs) ElementType() reflect.Type { return reflect.TypeOf(untaggableArgs{}) }

func TestLabelTagsTransformation_NilWhenNoLabels(t *testing.T) {
	assert.Nil(t, LabelTagsTransformation(nil, "aws", "Tags", nil))
	assert.Nil(t, LabelTagsTransformation(MapOrList[string]{}, "aws", "Tags", nil))
}

func TestLabelTagsTransformation_MergesExistingWins(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		tr := LabelTagsTransformation(
			MapOrList[string]{"com.acme.team": "core", "keep": "label"},
			"aws", "Tags", nil,
		)
		require.NotNil(t, tr)

		props := &taggableArgs{Tags: pulumi.StringMap{
			"defang:service": pulumi.String("web"),
			"keep":           pulumi.String("existing"),
		}}
		res := tr(&pulumi.ResourceTransformationArgs{
			Type: "aws:ecs/service:Service", Name: "web", Props: props,
		})
		require.NotNil(t, res)

		props.Tags.ToStringMapOutput().ApplyT(func(m map[string]string) string {
			assert.Equal(t, "core", m["com.acme.team"], "label applied verbatim (dots kept)")
			assert.Equal(t, "web", m["defang:service"], "functional tag preserved")
			assert.Equal(t, "existing", m["keep"], "existing tag wins over label on collision")
			return ""
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
}

func TestLabelTagsTransformation_SkipsWrongPrefix(t *testing.T) {
	tr := LabelTagsTransformation(MapOrList[string]{"a": "b"}, "aws", "Tags", nil)
	require.NotNil(t, tr)
	props := &taggableArgs{Tags: pulumi.StringMap{"x": pulumi.String("y")}}
	res := tr(&pulumi.ResourceTransformationArgs{
		Type: "gcp:cloudrunv2/service:Service", Name: "web", Props: props,
	})
	assert.Nil(t, res, "type token not matching prefix is left untouched")
}

func TestLabelTagsTransformation_SkipsUntaggable(t *testing.T) {
	tr := LabelTagsTransformation(MapOrList[string]{"a": "b"}, "aws", "Tags", nil)
	require.NotNil(t, tr)
	props := &untaggableArgs{Name: pulumi.String("x")}
	res := tr(&pulumi.ResourceTransformationArgs{
		Type: "aws:lb/targetGroupAttachment:TargetGroupAttachment", Name: "x", Props: props,
	})
	assert.Nil(t, res, "struct without a Tags field is skipped (no hard failure)")
}

func TestLabelTagsTransformation_Normalize(t *testing.T) {
	// Mimics GCP sanitization: dots → underscores, lowercased values.
	normalize := func(k, v string) (string, string) {
		return strings.ReplaceAll(k, ".", "_"), strings.ToLower(v)
	}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		tr := LabelTagsTransformation(MapOrList[string]{"com.acme.team": "Core"}, "gcp:", "Labels", normalize)
		require.NotNil(t, tr)
		props := &labelledArgs{}
		res := tr(&pulumi.ResourceTransformationArgs{
			Type: "gcp:cloudrunv2/service:Service", Name: "web", Props: props,
		})
		require.NotNil(t, res)
		require.NotNil(t, props.Labels)
		props.Labels.ToStringMapOutput().ApplyT(func(m map[string]string) string {
			assert.Equal(t, "core", m["com_acme_team"], "key sanitized, value lowercased")
			return ""
		})
		return nil
	}, pulumi.WithMocks("proj", "stack", testMocks{}))
	require.NoError(t, err)
}
