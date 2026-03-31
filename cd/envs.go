package main

import (
	"os"
	"strings"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Environment variables read at startup.
var (
	awsProfile        = os.Getenv("AWS_PROFILE")           // AWS only
	awsRegion         = envOrDefault("AWS_REGION", region) // AWS only
	azureLocation     = os.Getenv("AZURE_LOCATION")        // Azure only
	azureSubscription = os.Getenv("AZURE_SUBSCRIPTION_ID") // Azure only
	cdImage           = os.Getenv("DEFANG_CD_IMAGE")       // GCP only; for cleanup
	delegationSetId   = os.Getenv("DELEGATION_SET_ID")     // AWS only
	domain            = os.Getenv("DOMAIN")
	etag              = envOrDefault("DEFANG_ETAG", org)
	eventsUploadUrl   = os.Getenv("DEFANG_EVENTS_UPLOAD_URL")
	gcpProject        = os.Getenv("GCP_PROJECT") // GCP only
	jsonOutput        = os.Getenv("DEFANG_JSON") != ""
	mode              = envOrDefault("DEFANG_MODE", "development")
	noColor           = os.Getenv("NO_COLOR") != ""
	org               = envOrDefault("DEFANG_ORG", "defang")
	prefix            = envOrDefault("DEFANG_PREFIX", "Defang")
	privateDomain     = os.Getenv("PRIVATE_DOMAIN") // AWS only
	project           = envOrDefault("PROJECT", org)
	pulumiDebug       = os.Getenv("DEFANG_PULUMI_DEBUG") != ""
	pulumiDiff        = os.Getenv("DEFANG_PULUMI_DIFF") != ""
	pulumiTargets     = strings.Split(os.Getenv("DEFANG_PULUMI_TARGETS"), ",")
	region            = os.Getenv("REGION")
	registryCredsArn  = os.Getenv("CI_REGISTRY_CREDENTIALS_ARN") // AWS only
	stack             = os.Getenv("STACK")                       // required
	statesUploadUrl   = os.Getenv("DEFANG_STATES_UPLOAD_URL")
	stateUrl          = envOrDefault("DEFANG_STATE_URL", os.Getenv("PULUMI_BACKEND_URL"))
)
