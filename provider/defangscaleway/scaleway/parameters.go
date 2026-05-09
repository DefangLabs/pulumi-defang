package scaleway

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ConfigProvider reads config values from Scaleway Secret Manager.
// Secret names follow the convention: <prefix>_<project>_<stack>_<key>
// matching the CLI's getSecretName() format.
type ConfigProvider struct {
	prefix      string
	projectName string

	cache   map[string]pulumi.StringOutput
	mu      sync.Mutex
	fetched bool
}

// NewConfigProvider creates a ConfigProvider that reads secrets from
// Scaleway Secret Manager. The prefix and projectName parameters match
// the CLI's StackDir convention (e.g., prefix="Defang", projectName="myapp").
func NewConfigProvider(projectName string) *ConfigProvider {
	return &ConfigProvider{
		prefix:      "Defang",
		projectName: projectName,
		cache:       make(map[string]pulumi.StringOutput),
	}
}

func (cp *ConfigProvider) getSecretPrefix(stackName string) string {
	return fmt.Sprintf("%s_%s_%s_", cp.prefix, cp.projectName, stackName)
}

// GetConfigValue reads a secret from Scaleway Secret Manager by name.
func (cp *ConfigProvider) GetConfigValue(
	ctx *pulumi.Context, key string, opts ...pulumi.InvokeOption,
) pulumi.StringOutput {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Bulk-fetch all secrets with the project prefix on first access
	if !cp.fetched {
		values, err := cp.fetchSecrets(ctx.Stack())
		if err != nil {
			return compose.ConfigNotFoundOutput(key)
		}
		cp.fetched = true
		for k, v := range values {
			cp.cache[k] = pulumi.ToSecret(pulumi.String(v)).(pulumi.StringOutput)
		}
	}

	if val, ok := cp.cache[key]; ok {
		return val
	}
	return compose.ConfigNotFoundOutput(key)
}

// GetSecretRef is not supported by Scaleway ConfigProvider.
func (cp *ConfigProvider) GetSecretRef(
	_ *pulumi.Context, key string, _ ...pulumi.InvokeOption,
) (string, error) {
	return "", fmt.Errorf("Scaleway ConfigProvider does not support GetSecretRef for key %q", key)
}

// secretListResponse models the Scaleway Secret Manager list-secrets API response.
type secretListResponse struct {
	Secrets []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"secrets"`
}

// secretVersionAccessResponse models the access-secret-version API response.
type secretVersionAccessResponse struct {
	Data string `json:"data"` // base64-encoded
}

// fetchSecrets lists all secrets matching the project prefix and reads their
// latest versions. Uses the Scaleway Secret Manager REST API directly to avoid
// needing a Pulumi provider for data source invokes.
func (cp *ConfigProvider) fetchSecrets(stackName string) (map[string]string, error) {
	accessKey := os.Getenv("SCW_ACCESS_KEY")
	secretKey := os.Getenv("SCW_SECRET_KEY")
	region := os.Getenv("SCW_DEFAULT_REGION")
	projectID := os.Getenv("SCW_DEFAULT_PROJECT_ID")
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("SCW_ACCESS_KEY and SCW_SECRET_KEY must be set")
	}
	if region == "" {
		region = "fr-par"
	}

	prefix := cp.getSecretPrefix(stackName)

	// List secrets in the project
	listURL := fmt.Sprintf("https://api.scaleway.com/secret-manager/v1beta1/regions/%s/secrets?project_id=%s&page_size=100",
		region, url.QueryEscape(projectID))

	secrets, err := cp.listSecrets(listURL, secretKey)
	if err != nil {
		return nil, err
	}

	// Filter by prefix and read each secret's latest version
	result := make(map[string]string)
	for _, s := range secrets {
		if len(s.Name) <= len(prefix) || s.Name[:len(prefix)] != prefix {
			continue
		}
		key := s.Name[len(prefix):]
		if key == "" {
			continue
		}

		value, err := cp.accessSecretVersion(region, s.ID, secretKey)
		if err != nil {
			continue // skip secrets we can't read
		}
		result[key] = value
	}

	return result, nil
}

func (cp *ConfigProvider) listSecrets(apiURL, secretKey string) ([]struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", secretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listing secrets: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result secretListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Secrets, nil
}

func (cp *ConfigProvider) accessSecretVersion(region, secretID, secretKey string) (string, error) {
	apiURL := fmt.Sprintf("https://api.scaleway.com/secret-manager/v1beta1/regions/%s/secrets/%s/versions/latest/access",
		region, secretID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Auth-Token", secretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("accessing secret version: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result secretVersionAccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// The API returns the data as base64-encoded
	decoded, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return "", fmt.Errorf("decoding secret data: %w", err)
	}
	return string(decoded), nil
}
