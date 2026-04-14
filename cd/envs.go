package main

import (
	"os"
	"strconv"
	"strings"
)

// Environment variables read at startup.
var (
	awsProfile        = os.Getenv("AWS_PROFILE")           // AWS only
	awsRegion         = Getenv("AWS_REGION", region)       // AWS only
	azureLocation      = os.Getenv("AZURE_LOCATION")        // Azure only
	azureResourceGroup = os.Getenv("AZURE_RESOURCE_GROUP")  // Azure only; import existing RG when set
	azureSubscription  = os.Getenv("AZURE_SUBSCRIPTION_ID") // Azure only
	cdImage           = os.Getenv("DEFANG_CD_IMAGE")       // GCP only; for cleanup
	delegationSetId   = os.Getenv("DELEGATION_SET_ID")     // AWS only
	domain            = os.Getenv("DOMAIN")
	etag              = Getenv("DEFANG_ETAG", org)
	eventsUploadUrl   = os.Getenv("DEFANG_EVENTS_UPLOAD_URL")
	gcpProject        = os.Getenv("GCP_PROJECT") // GCP only
	jsonOutput        = GetenvBool("DEFANG_JSON")
	mode              = Getenv("DEFANG_MODE", "development")
	_, noColor        = os.LookupEnv("NO_COLOR")
	org               = Getenv("DEFANG_ORG", "defang")
	prefix            = Getenv("DEFANG_PREFIX", "Defang")
	privateDomain     = os.Getenv("PRIVATE_DOMAIN") // AWS only
	project           = Getenv("PROJECT", org)
	pulumiDebug       = GetenvBool("DEFANG_PULUMI_DEBUG")
	pulumiDiff        = GetenvBool("DEFANG_PULUMI_DIFF")
	pulumiTargets     = SplitByComma(os.Getenv("DEFANG_PULUMI_TARGETS"))
	region            = os.Getenv("REGION")
	registryCredsArn  = os.Getenv("CI_REGISTRY_CREDENTIALS_ARN") // AWS only
	stack             = os.Getenv("STACK")                       // required
	statesUploadUrl   = os.Getenv("DEFANG_STATES_UPLOAD_URL")
	stateUrl          = Getenv("DEFANG_STATE_URL", os.Getenv("PULUMI_BACKEND_URL"))
)

func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func SplitByComma(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func GetenvBool(key string) bool {
	val, _ := strconv.ParseBool(os.Getenv(key))
	return val
}
