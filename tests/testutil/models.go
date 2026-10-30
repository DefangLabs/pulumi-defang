package testutil

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/DefangLabs/pulumi-defang/provider/modelselect"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ModelFixture reads a payload from tests/testdata/model-requests, relative to
// a test package directory under tests/.
func ModelFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := fs.ReadFile(os.DirFS("../testdata/model-requests"), name)
	require.NoError(t, err)
	return string(data)
}

// ComposeInputs converts a Compose YAML payload, as the CLI uploads it, into
// Project Construct inputs: the same decoding the CD does (provider/compose)
// followed by the schema's property names (`pulumi` tags).
func ComposeInputs(t *testing.T, composeYAML string) property.Map {
	t.Helper()
	var project compose.Project
	require.NoError(t, yaml.Unmarshal([]byte(composeYAML), &project))
	services := toProperty(reflect.ValueOf(project.Services))
	inputs := map[string]property.Value{"services": services}
	if len(project.Networks) > 0 {
		inputs["networks"] = toProperty(reflect.ValueOf(project.Networks))
	}
	if len(project.Models) > 0 {
		inputs["models"] = toProperty(reflect.ValueOf(project.Models))
	}
	return property.NewMap(inputs)
}

func pulumiName(f reflect.StructField) string {
	tag, ok := f.Tag.Lookup("pulumi")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

var stringInputType = reflect.TypeFor[pulumi.StringInput]()

func toProperty(v reflect.Value) property.Value {
	if !v.IsValid() {
		return property.New(property.Null)
	}
	if v.Type() == stringInputType || v.Type().Implements(stringInputType) && v.Kind() != reflect.Struct {
		if v.Kind() == reflect.Interface && v.IsNil() {
			return property.New(property.Null)
		}
		if s, ok := v.Interface().(pulumi.String); ok {
			return property.New(string(s))
		}
		panic(fmt.Sprintf("toProperty: dynamic input %T", v.Interface()))
	}
	switch v.Kind() { //nolint:exhaustive // compose types use no other kinds
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return property.New(property.Null)
		}
		return toProperty(v.Elem())
	case reflect.String:
		return property.New(v.String())
	case reflect.Bool:
		return property.New(v.Bool())
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64:
		return property.New(float64(v.Convert(reflect.TypeFor[int64]()).Int()))
	case reflect.Float32, reflect.Float64:
		return property.New(v.Float())
	case reflect.Slice:
		items := make([]property.Value, v.Len())
		for i := range items {
			items[i] = toProperty(v.Index(i))
		}
		return property.New(property.NewArray(items))
	case reflect.Map:
		m := map[string]property.Value{}
		for _, k := range v.MapKeys() {
			m[k.String()] = toProperty(v.MapIndex(k))
		}
		return property.New(property.NewMap(m))
	case reflect.Struct:
		m := map[string]property.Value{}
		for i := range v.NumField() {
			f := v.Type().Field(i)
			name := pulumiName(f)
			if name == "" || v.Field(i).IsZero() {
				continue
			}
			m[name] = toProperty(v.Field(i))
		}
		return property.New(property.NewMap(m))
	default:
		panic(fmt.Sprintf("toProperty: unsupported kind %s", v.Kind()))
	}
}

// fromProperty decodes a property value into a struct with `pulumi` tags.
func fromProperty(pv property.Value, v reflect.Value) {
	if pv.IsNull() || pv.IsComputed() {
		return
	}
	switch v.Kind() { //nolint:exhaustive // ModelSelectionInputs uses no other kinds
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fromProperty(pv, v.Elem())
	case reflect.String:
		v.SetString(pv.AsString())
	case reflect.Bool:
		v.SetBool(pv.AsBool())
	case reflect.Slice:
		arr := pv.AsArray()
		out := reflect.MakeSlice(v.Type(), arr.Len(), arr.Len())
		for i := range arr.Len() {
			fromProperty(arr.Get(i), out.Index(i))
		}
		v.Set(out)
	case reflect.Struct:
		m := pv.AsMap()
		for i := range v.NumField() {
			f := v.Type().Field(i)
			if f.Anonymous {
				fromProperty(pv, v.Field(i))
				continue
			}
			if name := pulumiName(f); name != "" {
				fromProperty(m.Get(name), v.Field(i))
			}
		}
	default:
		panic(fmt.Sprintf("fromProperty: unsupported kind %s", v.Kind()))
	}
}

// ModelSelections runs registered ModelSelection resources through the
// real selection lifecycle with the given resolver, recording each result.
type ModelSelections struct {
	Selector modelselect.ModelSelector

	mu       sync.Mutex
	Inputs   map[string]common.ModelSelectionInputs
	Selected map[string]common.ModelSelectionState
}

// Handle creates args if it is a ModelSelection and returns its outputs.
func (m *ModelSelections) Handle(args integration.MockResourceArgs) (property.Map, bool, error) {
	if !strings.HasSuffix(string(args.TypeToken), ":index:"+common.ModelSelectionType) {
		return property.Map{}, false, nil
	}
	var in common.ModelSelectionInputs
	fromProperty(property.New(args.Inputs), reflect.ValueOf(&in).Elem())
	resp, err := m.Selector.Create(context.Background(), infer.CreateRequest[common.ModelSelectionInputs]{
		Name: args.Name, Inputs: in,
	})
	if err != nil {
		return property.Map{}, true, err
	}
	m.mu.Lock()
	if m.Inputs == nil {
		m.Inputs = map[string]common.ModelSelectionInputs{}
		m.Selected = map[string]common.ModelSelectionState{}
	}
	m.Inputs[args.Name] = in
	m.Selected[args.Name] = resp.Output
	m.mu.Unlock()
	return SelectionOutputs(resp.Output), true, nil
}

// SelectionOutputs encodes a selection state as resource outputs. Every
// output is present, empty or not, as the engine reports them.
func SelectionOutputs(s common.ModelSelectionState) property.Map {
	return property.NewMap(map[string]property.Value{
		"request":      property.New(s.Request),
		"source":       property.New(s.Source),
		"model":        property.New(s.Model),
		"version":      property.New(s.Version),
		"location":     property.New(s.Location),
		"format":       property.New(s.Format),
		"sku":          property.New(s.Sku),
		"selectedRole": property.New(s.SelectedRole),
	})
}
