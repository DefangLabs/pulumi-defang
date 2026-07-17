package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyPolicy(t *testing.T) {
	tests := []struct {
		entry string
		want  PolicyCloud
	}{
		{"arn:aws:iam::123456789012:policy/deployer", PolicyCloudAWS},
		{"arn:aws-us-gov:iam::123456789012:policy/deployer", PolicyCloudAWS},
		{"roles/run.developer", PolicyCloudGCP},
		{"projects/myproj/roles/deployer", PolicyCloudGCP},
		{"organizations/123/roles/deployer", PolicyCloudGCP},
		{"/subscriptions/000/providers/Microsoft.Authorization/roleDefinitions/abc", PolicyCloudAzure},
		{"/providers/Microsoft.Authorization/roleDefinitions/abc", PolicyCloudAzure},
		{"deployer", PolicyCloudAny},
		{"Storage Blob Data Contributor", PolicyCloudAny},
	}
	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			assert.Equal(t, tt.want, ClassifyPolicy(tt.entry))
		})
	}
}

func TestPoliciesFor(t *testing.T) {
	const awsArn = "arn:aws:iam::123456789012:policy/deployer"
	const gcpRole = "roles/run.developer"
	const azureID = "/subscriptions/000/providers/Microsoft.Authorization/roleDefinitions/abc"
	policies := []string{awsArn, gcpRole, azureID, "deployer"}

	tests := []struct {
		cloud          PolicyCloud
		wantApplicable []string
		wantSkipped    []string
	}{
		{PolicyCloudAWS, []string{awsArn, "deployer"}, []string{gcpRole, azureID}},
		{PolicyCloudGCP, []string{gcpRole, "deployer"}, []string{awsArn, azureID}},
		{PolicyCloudAzure, []string{azureID, "deployer"}, []string{awsArn, gcpRole}},
	}
	for _, tt := range tests {
		t.Run(string(tt.cloud), func(t *testing.T) {
			applicable, skipped := PoliciesFor(tt.cloud, policies)
			assert.Equal(t, tt.wantApplicable, applicable)
			assert.Equal(t, tt.wantSkipped, skipped)
		})
	}

	applicable, skipped := PoliciesFor(PolicyCloudAWS, nil)
	assert.Empty(t, applicable)
	assert.Empty(t, skipped)
}
