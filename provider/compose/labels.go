package compose

import (
	"reflect"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// stringMapInputType is the reflect.Type of pulumi.StringMapInput, used to
// detect taggable Args structs by reflection.
var stringMapInputType = reflect.TypeOf((*pulumi.StringMapInput)(nil)).Elem()

// labelsToStringMap converts Compose labels to a pulumi.StringMap, applying the
// optional normalize func to each key/value. Returns nil when there are no
// labels so callers can skip wiring up a transformation entirely.
func labelsToStringMap(labels MapOrList[string], normalize func(k, v string) (string, string)) pulumi.StringMap {
	if len(labels) == 0 {
		return nil
	}
	out := make(pulumi.StringMap, len(labels))
	for k, v := range labels {
		if normalize != nil {
			k, v = normalize(k, v)
		}
		out[k] = pulumi.String(v)
	}
	return out
}

// LabelTagsTransformation returns a Pulumi resource transformation that merges
// the given Compose labels into the named tag/label field of every resource
// whose type token starts with typePrefix and whose Args struct exposes that
// field as a pulumi.StringMapInput. fieldName is "Tags" for AWS/Azure and
// "Labels" for GCP.
//
// Unlike the TS implementation (which gates on a hand-maintained allowlist of
// taggable type tokens), this relies on Go's typed Args structs: it only sets
// the field when reflection finds it with the right type, so non-taggable
// resources — and AWS's autoscaling Group, whose Tags field is a list rather
// than a StringMap — are skipped automatically.
//
// Existing (functional/explicit) tag values win over labels on key collision,
// so defang-managed tags such as "defang:service" can never be clobbered by a
// user label. normalize optionally rewrites keys/values (e.g. GCP label
// sanitization); pass nil to apply labels verbatim. Returns nil when there are
// no labels, so callers can skip attaching the transformation.
//
// Attach the result via pulumi.Transformations(...) on a component's opts;
// Pulumi cascades component-level transformations to all children (including
// resources created under sub-providers), which is why this reaches every
// downstream resource without threading a provider through each call.
func LabelTagsTransformation(
	labels MapOrList[string],
	typePrefix string,
	fieldName string,
	normalize func(k, v string) (string, string),
) pulumi.ResourceTransformation {
	sm := labelsToStringMap(labels, normalize)
	if len(sm) == 0 {
		return nil
	}
	labelsOut := sm.ToStringMapOutput()
	return func(args *pulumi.ResourceTransformationArgs) *pulumi.ResourceTransformationResult {
		if !strings.HasPrefix(args.Type, typePrefix) {
			return nil
		}
		v := reflect.ValueOf(args.Props)
		if v.Kind() != reflect.Ptr || v.IsNil() {
			return nil
		}
		v = v.Elem()
		if v.Kind() != reflect.Struct {
			return nil
		}
		f := v.FieldByName(fieldName)
		if !f.IsValid() || !f.CanSet() {
			return nil
		}
		if !f.Type().Implements(stringMapInputType) && f.Type() != stringMapInputType {
			return nil
		}

		// props.Tags may be a plain StringMap or a StringMapOutput; normalize
		// both to a StringMapOutput before merging.
		existingOut := pulumi.StringMap{}.ToStringMapOutput()
		if existing, ok := f.Interface().(pulumi.StringMapInput); ok && existing != nil {
			existingOut = existing.ToStringMapOutput()
		}

		merged := pulumi.All(labelsOut, existingOut).ApplyT(func(parts []interface{}) map[string]string {
			out := map[string]string{}
			if lbls, ok := parts[0].(map[string]string); ok {
				for k, v := range lbls {
					out[k] = v // labels first
				}
			}
			if existing, ok := parts[1].(map[string]string); ok {
				for k, v := range existing {
					out[k] = v // existing/functional tags win on collision
				}
			}
			return out
		}).(pulumi.StringMapOutput)

		f.Set(reflect.ValueOf(merged))
		return &pulumi.ResourceTransformationResult{Props: args.Props, Opts: args.Opts}
	}
}
