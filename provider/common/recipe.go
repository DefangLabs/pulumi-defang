package common

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const configNamespace = "defang"

// boolValue is a typed config accessor for a boolean value.
type boolValue struct {
	key string
	def bool
}

// intValue is a typed config accessor for an integer value.
type intValue struct {
	key string
	def int
}

// stringValue is a typed config accessor for a string value.
type stringValue struct {
	key string
	def string
}

// Bool creates a bool config accessor with the given key and default.
func Bool(key string, def bool) boolValue { return boolValue{key: key, def: def} }

// Int creates an int config accessor with the given key and default.
func Int(key string, def int) intValue { return intValue{key: key, def: def} }

// String creates a string config accessor with the given key and default.
func String(key string, def string) stringValue { return stringValue{key: key, def: def} }

// Get reads the config value from the Pulumi context, falling back to the default.
func (v boolValue) Get(ctx *pulumi.Context) bool {
	cfg := config.New(ctx, configNamespace)
	val, err := cfg.TryBool(v.key)
	if err != nil {
		return v.def
	}
	return val
}

// Get reads the config value from the Pulumi context, falling back to the default.
func (v intValue) Get(ctx *pulumi.Context) int {
	cfg := config.New(ctx, configNamespace)
	val, err := cfg.TryInt(v.key)
	if err != nil {
		return v.def
	}
	return val
}

// Get reads the config value from the Pulumi context, falling back to the default.
func (v stringValue) Get(ctx *pulumi.Context) string {
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
	Version            = String("version", "development")
)
