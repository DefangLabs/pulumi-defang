package common

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// boolValue is a typed accessor for a boolean recipe setting. affordable is the
// cheapest/development baseline, returned when the stack config omits the key.
type boolValue struct {
	namespace  string
	key        string
	affordable bool
}

// intValue is a typed accessor for an integer recipe setting. See boolValue.
type intValue struct {
	namespace  string
	key        string
	affordable int
}

// stringValue is a typed accessor for a string recipe setting. See boolValue.
type stringValue struct {
	namespace  string
	key        string
	affordable string
}

// Get reads the value from the stack config, falling back to the affordable default.
func (v boolValue) Get(ctx *pulumi.Context) bool {
	cfg := config.New(ctx, v.namespace)
	val, err := cfg.TryBool(v.key)
	if err != nil {
		return v.affordable
	}
	return val
}

// Get reads the value from the stack config, falling back to the affordable default.
func (v intValue) Get(ctx *pulumi.Context) int {
	cfg := config.New(ctx, v.namespace)
	val, err := cfg.TryInt(v.key)
	if err != nil {
		return v.affordable
	}
	return val
}

// Get reads the value from the stack config, falling back to the affordable default.
func (v stringValue) Get(ctx *pulumi.Context) string {
	cfg := config.New(ctx, v.namespace)
	val, err := cfg.Try(v.key)
	if err != nil {
		return v.affordable
	}
	return val
}

// Recipe is a namespaced set of config accessors. Construct one per provider.
type Recipe struct{ namespace string }

// NewRecipe returns a recipe bound to the given config namespace.
func NewRecipe(namespace string) Recipe { return Recipe{namespace: namespace} }

// Bool declares a bool setting in this recipe's namespace with the given
// affordable (development) default.
func (r Recipe) Bool(key string, affordable bool) boolValue {
	return boolValue{namespace: r.namespace, key: key, affordable: affordable}
}

// Int declares an int setting in this recipe's namespace with the given
// affordable (development) default.
func (r Recipe) Int(key string, affordable int) intValue {
	return intValue{namespace: r.namespace, key: key, affordable: affordable}
}

// String declares a string setting in this recipe's namespace with the given
// affordable (development) default.
func (r Recipe) String(key string, affordable string) stringValue {
	return stringValue{namespace: r.namespace, key: key, affordable: affordable}
}

// Cloud-agnostic config values shared across all providers (defang: namespace).
// deletion-protection and log-retention-days are intentionally NOT here: each
// provider declares its own (defang-aws:/defang-gcp:/defang-azure:) since the
// resources they apply to differ per cloud.
var (
	// Defang is the cloud-agnostic recipe namespace, shared by all providers.
	Defang  = NewRecipe("defang")
	Prefix  = Defang.String("prefix", "Defang")
	Version = Defang.String("version", "Defang")
)
