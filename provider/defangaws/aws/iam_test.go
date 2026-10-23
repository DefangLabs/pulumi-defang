package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicyName(t *testing.T) {
	assert.Equal(t, "MyPolicy", policyName("MyPolicy"))
	assert.Equal(t, "AmazonS3ReadOnlyAccess", policyName("arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"))
	assert.Equal(t, "MyPolicy", policyName("arn:aws:iam::123456789012:policy/MyPolicy"))
	assert.Equal(t, "MyPolicy", policyName("arn:aws:iam::123456789012:policy/path/MyPolicy"))
}
