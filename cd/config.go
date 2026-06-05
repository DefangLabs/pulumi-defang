package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v4"
)

func stackConfigJson(recipePulumiConfig string) (string, error) {
	var config configMap
	if len(recipePulumiConfig) != 0 {
		// Parse all stack config from the recipe's PulumiConfig field
		var err error
		config, err = parseRecipePulumiConfig(recipePulumiConfig)
		if err != nil {
			return "", err
		}
	} else {
		// Legacy path: init stack config
		prefix := getenv("DEFANG_PREFIX", "Defang")
		config = defaultStackConfig(prefix)
	}
	// For compat, apply any config from env vars on top of the recipe's PulumiConfig.
	// Eventually the CLI can provide all config through the recipe JSON/YAML.
	err := addStackConfigFromEnv(config)
	if err != nil {
		return "", err
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal stack config: %w", err)
	}
	return string(configJson), nil
}

// parseRecipePulumiConfig returns stack config from the recipe's PulumiConfig.
// It accepts either a Pulumi stack settings YAML (everything nested under a
// top-level "config:" key) or the flat JSON emitted by `pulumi config --json`
// (a map of "namespace:key" to a {value: "<string>"} wrapper). Objects and
// arrays are serialized to their compact JSON string form, matching how Pulumi
// stores structured config.
func parseRecipePulumiConfig(recipePulumiConfig string) (configMap, error) {
	// JSON is a subset of YAML, so a single YAML decoder handles both formats.
	var root map[string]any
	if err := yaml.Unmarshal([]byte(recipePulumiConfig), &root); err != nil {
		return nil, err
	}

	// Stack settings nest config under "config:"; `pulumi config --json` is flat
	// and wraps each value in a {value: "<string>"} object. We only unwrap in the
	// flat case so a genuine object literal named "value" in stack settings is
	// preserved rather than mistaken for a wrapper.
	section, flat := root, true
	if cfg, ok := root["config"].(map[string]any); ok {
		section, flat = cfg, false
	}

	config := configMap{}
	for key, val := range section {
		if flat {
			if wrapper, ok := val.(map[string]any); ok {
				if v, ok := wrapper["objectValue"]; ok { // pulumi checks objectValue first
					val = v
				} else if v, ok := wrapper["value"]; ok { // pulumi emits "value" for scalar config
					val = v
				}
			}
		}
		config[key] = configValue{Value: val}
	}
	return config, nil
}

func defaultStackConfig(prefix string) configMap {
	if prefix != "" {
		prefix += "-"
	}
	lowerPrefix := strings.ToLower(prefix)
	return configMap{
		"defang:prefix": configValue{Value: prefix},
		"pulumi:autonaming": configValue{Value: map[string]any{
			"pattern": prefix + "${project}-${stack}-${name}-${hex(7)}",
			"providers": map[string]any{
				"aws": map[string]any{
					"resources": map[string]any{
						"aws:lb/loadBalancer:LoadBalancer": map[string]string{"pattern": "${project}-${stack}-${hex(4)}"},
						"aws:lb/targetGroup:TargetGroup":   map[string]string{"pattern": "${name}-${hex(4)}"},
						// ecs.Service is always scoped to an ecs.Cluster, so the cluster's
						// full prefix already disambiguates it; no need to repeat it here.
						"aws:ecs/service:Service":                 map[string]string{"pattern": "${name}-${hex(7)}"},
						"aws:elasticache/subnetGroup:SubnetGroup": map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"}, // lowercase
						"aws:ecr/repository:Repository":           map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"}, // lowercase
						"aws:rds/subnetGroup:SubnetGroup":         map[string]string{"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}"}, // lowercase
					},
				},
				"azure-native": map[string]any{
					"resources": map[string]any{
						// ACR registry names must be alphanumeric only (^[a-zA-Z0-9]*$, 5–50 chars).
						// The default pattern includes hyphens from project/stack names, so override it.
						// ${name} is already sanitized to alphanumeric by sanitizeRegistryName() in image.go.
						// ${stack} is safe to include: stacks are lowercase with no hyphens.
						"azure-native:containerregistry:Registry": map[string]string{"pattern": "${name}${stack}${hex(7)}"}, // name = sanitized project name
						// https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules#microsoftcontainerregistry
						// 5-50	Alphanumerics, hyphens, and underscores
						"azure-native:containerregistry:Task": map[string]string{"pattern": "${name}-${hex(7)}"},
						// Requirements for Container App Environment resource names:
						// Between 2 and 60 characters long.
						// This resource name is not case-sensitive even though it is written as lowercase only in the docs.
						// Numbers and hyphens are also allowed.
						// https://azure.github.io/PSRule.Rules.Azure/en/rules/Azure.ContainerApp.EnvNaming
						"azure-native:app:ManagedEnvironment": map[string]string{"pattern": prefix + "${project}-${stack}-${hex(7)}"},
						// Log Analytics workspace names must be 4–63 chars. The workspace's
						// logical ${name} is the project name, so the default pattern repeats
						// the project twice and overflows on longer names; drop ${name}.
						// https://learn.microsoft.com/en-us/azure/templates/microsoft.operationalinsights/workspaces?pivots=deployment-language-bicep#microsoftoperationalinsightsworkspaces
						"azure-native:operationalinsights:Workspace": map[string]string{"pattern": prefix + "${project}-${stack}-${hex(7)}"},
					},
				},
				// Most GCP resources require names matching ^[a-z][-a-z0-9]{0,61}[a-z0-9]$
				// (lowercase only, max 63 chars). The default prefix may contain capitals
				// (e.g. "Defang-"), so force the entire pattern to use the lowercased prefix.
				"gcp": map[string]any{
					"pattern": lowerPrefix + "${project}-${stack}-${name}-${hex(7)}", // TODO: sanitize project name
					"resources": map[string]any{
						// Service account ID must be between 6 and 30 characters.
						// Service account ID must start with a lower case letter, followed by one or more lower case alphanumerical characters that can be separated by hyphens.
						"gcp:serviceaccount/account:Account": map[string]string{"pattern": "${name}-${hex(4)}"},
						// Cloud Run service name max 49 chars (^[a-z][a-z0-9-]{0,47}[a-z0-9]$).
						// Default prefix-project-stack pattern overflows on longer inputs;
						// drop the prefix to mirror old cloudrunServiceName (49 char budget).
						"gcp:cloudrunv2/service:Service": map[string]string{"pattern": "${project}-${stack}-${name}-${hex(7)}"}, // TODO: sanitize project name
						// Memorystore Redis instance ID max 40 chars (^[a-z][a-z0-9-]{0,38}[a-z0-9]$).
						// Drop the prefix to mirror old redisInstanceName (40 char budget).
						"gcp:redis/instance:Instance": map[string]string{"pattern": "${project}-${name}-${hex(7)}"}, // TODO: sanitize project name
					},
				},
			},
		}},
	}
}

// addStackConfigFromEnv adds any stack config from legacy environment variables.
func addStackConfigFromEnv(config configMap) error {
	region := os.Getenv("REGION")
	awsProfile := os.Getenv("AWS_PROFILE")                    // AWS only
	awsRegion := getenv("AWS_REGION", region)                 // AWS only
	azureLocation := getenv("AZURE_LOCATION", region)         // Azure only
	azureSubscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID") // Azure only; the project RG and Key Vault names are derived from (project, stack, location) and (subscription, RG) respectively — see provider/defangazure/azure/azure.go
	cdImage := os.Getenv("DEFANG_CD_IMAGE")                   // GCP only; for cleanup
	delegationSetId := os.Getenv("DELEGATION_SET_ID")         // AWS only
	domain := os.Getenv("DOMAIN")
	org := getenv("DEFANG_ORG", "defang")
	etag := getenv("DEFANG_ETAG", org)
	gcpProject := getenv("GCLOUD_PROJECT", os.Getenv("GCP_PROJECT")) // GCP only; keep GCP_PROJECT for old CLI compat
	gcpRegion := getenv("GCLOUD_REGION", region)                     // GCP only
	privateDomain := os.Getenv("PRIVATE_DOMAIN")                     // AWS only
	registryCredsArn := os.Getenv("CI_REGISTRY_CREDENTIALS_ARN")     // AWS only
	stateUrl := getenv("DEFANG_STATE_URL", os.Getenv("PULUMI_BACKEND_URL"))

	// Defang program config
	config["defang:cdImage"] = configValue{Value: cdImage}
	config["defang:etag"] = configValue{Value: etag} // deployment ID; recorded in state, surfaced in tags/env
	config["defang:org"] = configValue{Value: org}
	config["defang:stateUrl"] = configValue{Value: stateUrl}
	config["defang:version"] = configValue{Value: version}

	// Cloud provider config read by the explicit providers in the program
	var providers []string
	if awsRegion != "" {
		providers = append(providers, "aws")
		config["aws:region"] = configValue{Value: awsRegion}
		if awsProfile != "" {
			config["aws:profile"] = configValue{Value: awsProfile}
		}
		if privateDomain != "" {
			config["defang-aws:privateDomain"] = configValue{Value: privateDomain}
		}
		if delegationSetId != "" {
			config["defang-aws:delegationSetId"] = configValue{Value: delegationSetId}
		}
		if registryCredsArn != "" {
			config["defang-aws:ciRegistryCredentialsArn"] = configValue{Value: registryCredsArn}
		}
	}

	if gcpProject != "" {
		providers = append(providers, "gcp")
		config["gcp:project"] = configValue{Value: gcpProject}
		if gcpRegion != "" {
			config["gcp:region"] = configValue{Value: gcpRegion}
		}
		// TODO: configure label-logger
	}

	if azureSubscriptionId != "" {
		providers = append(providers, "azure")
		config["azure-native:subscriptionId"] = configValue{Value: azureSubscriptionId}
		if azureLocation != "" {
			config["azure-native:location"] = configValue{Value: azureLocation}
		}
		config["azure-native:useMsi"] = configValue{Value: "true"}
		// The project RG name and Key Vault name are derived deterministically
		// from (project, stack, location) and (subscription, RG) respectively
		// inside the provider — matching the CLI's conventions. No need to
		// pass them through as stack config or env vars.
	}

	if len(providers) == 0 {
		return &usageError{msg: "no cloud provider configured: set AWS_REGION, GCLOUD_PROJECT, or AZURE_SUBSCRIPTION_ID environment variable"}
	} else if len(providers) > 1 {
		return &usageError{msg: fmt.Sprintf("conflicting cloud providers configured: %v", providers)}
	}
	config["defang:provider"] = configValue{Value: providers[0]}

	// Defang recipe config
	if domain != "" {
		config["defang:domain"] = configValue{Value: domain}
	}
	return nil
}

// configMap is like auto.ConfigMap but with our own `configValue` struct type.
type configMap map[string]configValue

// configValue wraps any config value and implements custom JSON marshaling to
// match how Pulumi's CLI emits config in `pulumi config --json`, which writes
// a "value" field (string) for scalar values and an optional "objectValue"
// field (JSON object) for structured values like objects and arrays.
type configValue struct {
	Value any
}

func (c configValue) MarshalJSON() ([]byte, error) {
	switch v := c.Value.(type) {
	case string, float32, float64, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		// scalar values are stored as strings in Pulumi config
		return json.Marshal(map[string]any{"value": fmt.Sprint(v)})
	default:
		// objects and arrays are stored as JSON strings in Pulumi config, but also include the original value
		value, err := json.Marshal(c.Value)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"value": string(value), "objectValue": json.RawMessage(value)})
	}
}
