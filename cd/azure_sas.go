package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"gopkg.in/yaml.v3"
)

// addAzureBlobSASTokens walks the compose YAML and replaces any Azure Blob
// build-context URLs with user-delegation SAS URLs so the ACR task can
// download the build context without needing its own managed identity access.
func addAzureBlobSASTokens(ctx context.Context, composeYaml []byte) ([]byte, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(composeYaml, &doc); err != nil {
		return composeYaml, fmt.Errorf("parsing compose YAML: %w", err)
	}

	services, ok := doc["services"].(map[string]interface{})
	if !ok {
		return composeYaml, nil
	}

	modified := false
	for svcName, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		buildRaw, ok := svc["build"]
		if !ok {
			continue
		}
		build, ok := buildRaw.(map[string]interface{})
		if !ok {
			continue
		}
		contextRaw, ok := build["context"]
		if !ok {
			continue
		}
		contextURL, ok := contextRaw.(string)
		if !ok || !strings.Contains(contextURL, ".blob.core.windows.net") {
			continue
		}
		sasURL, err := blobToSASURL(ctx, contextURL)
		if err != nil {
			log.Printf("warning: failed to generate SAS URL for %s build context: %v", svcName, err)
			continue
		}
		build["context"] = sasURL
		modified = true
		log.Printf("Generated SAS URL for %s build context", svcName)
	}

	if !modified {
		return composeYaml, nil
	}
	return yaml.Marshal(doc)
}

// blobToSASURL generates a 2-hour user-delegation SAS URL for the given
// Azure Blob Storage URL using the default Azure credential (managed identity).
func blobToSASURL(ctx context.Context, blobURL string) (string, error) {
	u, err := url.Parse(blobURL)
	if err != nil {
		return "", fmt.Errorf("parsing blob URL: %w", err)
	}
	// host is "{account}.blob.core.windows.net"
	account := strings.SplitN(u.Host, ".", 2)[0]
	// path is "/{container}/{blob}"
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected blob URL path %q (want /container/blob)", u.Path)
	}
	containerName, blobName := parts[0], parts[1]

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", fmt.Errorf("creating Azure credential: %w", err)
	}

	svcClient, err := service.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net/", account),
		cred, nil,
	)
	if err != nil {
		return "", fmt.Errorf("creating storage service client: %w", err)
	}

	now := time.Now().UTC().Add(-10 * time.Second) // small buffer for clock skew
	expiry := now.Add(2 * time.Hour)
	udc, err := svcClient.GetUserDelegationCredential(ctx, service.KeyInfo{
		Start:  strPtr(now.Format(sas.TimeFormat)),
		Expiry: strPtr(expiry.Format(sas.TimeFormat)),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("getting user delegation key: %w", err)
	}

	sasParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     now,
		ExpiryTime:    expiry,
		Permissions:   (&sas.BlobPermissions{Read: true}).String(),
		ContainerName: containerName,
		BlobName:      blobName,
	}.SignWithUserDelegation(udc)
	if err != nil {
		return "", fmt.Errorf("signing SAS: %w", err)
	}

	return blobURL + "?" + sasParams.Encode(), nil
}

func strPtr(s string) *string { return &s }
