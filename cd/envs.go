package main

import (
	"os"
	"strconv"
	"strings"
)

// Environment variables read at startup.
var (
	awsProfile          = os.Getenv("AWS_PROFILE")           // AWS only
	awsRegion           = getenv("AWS_REGION", region)       // AWS only
	azureLocation       = getenv("AZURE_LOCATION", region)   // Azure only
	azureSubscriptionId = os.Getenv("AZURE_SUBSCRIPTION_ID") // Azure only; the project RG and Key Vault names are derived from (project, stack, location) and (subscription, RG) respectively — see provider/defangazure/azure/azure.go
	cdImage             = os.Getenv("DEFANG_CD_IMAGE")       // GCP only; for cleanup
	delegationSetId     = os.Getenv("DELEGATION_SET_ID")     // AWS only
	domain              = os.Getenv("DOMAIN")
	etag                = getenv("DEFANG_ETAG", org)
	eventsUploadUrl     = os.Getenv("DEFANG_EVENTS_UPLOAD_URL")
	gcpProjectId        = getenv("GCP_PROJECT", os.Getenv("GCP_PROJECT_ID")) // GCP only; keep GCP_PROJECT for backward compat
	gcpRegion           = getenv("GCP_REGION", region)                       // GCP only
	jsonOutput          = getenvBool("DEFANG_JSON")
	mode                = getenv("DEFANG_MODE", "development")
	_, noColor          = os.LookupEnv("NO_COLOR")
	org                 = getenv("DEFANG_ORG", "defang")
	prefix              = getenv("DEFANG_PREFIX", "Defang")
	privateDomain       = os.Getenv("PRIVATE_DOMAIN") // AWS only
	projectName         = getenv("PROJECT", org)
	pulumiDebug         = getenvBool("DEFANG_PULUMI_DEBUG")
	pulumiDiff          = getenvBool("DEFANG_PULUMI_DIFF")
	pulumiTargets       = splitByComma(os.Getenv("DEFANG_PULUMI_TARGETS"))
	region              = os.Getenv("REGION")
	registryCredsArn    = os.Getenv("CI_REGISTRY_CREDENTIALS_ARN") // AWS only
	stackName           = os.Getenv("STACK")                       // required
	statesUploadUrl     = os.Getenv("DEFANG_STATES_UPLOAD_URL")
	stateUrl            = getenv("DEFANG_STATE_URL", os.Getenv("PULUMI_BACKEND_URL"))
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitByComma(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func getenvBool(key string) bool {
	val, _ := strconv.ParseBool(os.Getenv(key))
	return val
}
