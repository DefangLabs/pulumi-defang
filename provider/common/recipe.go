package common

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const configNamespace = "defang"

// BoolValue is a typed config accessor for a boolean value.
type BoolValue struct {
	key string
	def bool
}

// IntValue is a typed config accessor for an integer value.
type IntValue struct {
	key string
	def int
}

// StringValue is a typed config accessor for a string value.
type StringValue struct {
	key string
	def string
}

// Bool creates a bool config accessor with the given key and default.
func Bool(key string, def bool) BoolValue { return BoolValue{key: key, def: def} }

// Int creates an int config accessor with the given key and default.
func Int(key string, def int) IntValue { return IntValue{key: key, def: def} }

// String creates a string config accessor with the given key and default.
func String(key string, def string) StringValue { return StringValue{key: key, def: def} }

// Get reads the config value from the Pulumi context, falling back to the default.
func (v BoolValue) Get(ctx *pulumi.Context) bool {
	cfg := config.New(ctx, configNamespace)
	val, err := cfg.TryBool(v.key)
	if err != nil {
		return v.def
	}
	return val
}

// Get reads the config value from the Pulumi context, falling back to the default.
func (v IntValue) Get(ctx *pulumi.Context) int {
	cfg := config.New(ctx, configNamespace)
	val, err := cfg.TryInt(v.key)
	if err != nil {
		return v.def
	}
	return val
}

// Get reads the config value from the Pulumi context, falling back to the default.
func (v StringValue) Get(ctx *pulumi.Context) string {
	cfg := config.New(ctx, configNamespace)
	val, err := cfg.Try(v.key)
	if err != nil {
		return v.def
	}
	return val
}

// Cloud-agnostic config values shared across all providers.
var (
	DeletionProtection = Bool("deletion-protection", false)
	LogRetentionDays   = Int("log-retention-days", 1)
	Prefix             = String("prefix", "Defang")
)
