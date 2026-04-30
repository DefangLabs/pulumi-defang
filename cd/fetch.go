package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const maxPayloadBytes = 16 << 20 // 16 MiB

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxPayloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPayloadBytes {
		return nil, fmt.Errorf("payload exceeds %d bytes", maxPayloadBytes)
	}
	return data, nil
}

// fetchPayload retrieves the ProjectUpdate protobuf from s3://, gs://, https://, or base64.
func fetchPayload(ctx context.Context, uri string) ([]byte, error) {
	switch {
	case strings.HasPrefix(uri, "s3://"):
		return fetchS3(ctx, uri)
	case strings.HasPrefix(uri, "gs://"):
		return fetchGCS(ctx, uri)
	case strings.Contains(uri, ".blob.core.windows.net/"):
		return fetchAzureBlob(ctx, uri)
	case strings.HasPrefix(uri, "http://"), strings.HasPrefix(uri, "https://"):
		return fetchHTTP(ctx, uri)
	default:
		return base64.StdEncoding.DecodeString(uri)
	}
}

func fetchS3(ctx context.Context, uri string) ([]byte, error) {
	bucket, key, found := strings.Cut(strings.TrimPrefix(uri, "s3://"), "/")
	if !found {
		return nil, fmt.Errorf("invalid S3 URI: %v", uri)
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s3.NewFromConfig(cfg).GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return readLimited(result.Body)
}

func fetchGCS(ctx context.Context, uri string) ([]byte, error) {
	bucket, object, found := strings.Cut(strings.TrimPrefix(uri, "gs://"), "/")
	if !found {
		return nil, fmt.Errorf("invalid GCS URI: %v", uri)
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return readLimited(rc)
}

func fetchAzureBlob(ctx context.Context, uri string) ([]byte, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure credential: %w", err)
	}
	client, err := blob.NewClient(uri, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure blob client: %w", err)
	}
	resp, err := client.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading Azure blob: %w", err)
	}
	defer resp.Body.Close()
	return readLimited(resp.Body)
}

func fetchHTTP(ctx context.Context, uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("GET %s returned %s", uri, resp.Status)
	}
	return readLimited(resp.Body)
}
