package gcp

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
)

func TestGetComputeBootImage(t *testing.T) {
	arm := "linux/arm64"
	amd := "linux/amd64"

	svc := compose.ServiceConfig{Platform: &arm}
	assert.Equal(t, "projects/cos-cloud/global/images/family/cos-arm64-stable", getComputeBootImage(svc),
		"ARM platforms must boot the arm64 COS family (T2A cannot boot x86 images)")

	svc = compose.ServiceConfig{Platform: &amd}
	assert.Equal(t, "projects/cos-cloud/global/images/family/cos-stable", getComputeBootImage(svc))

	assert.Equal(t, "projects/cos-cloud/global/images/family/cos-stable", getComputeBootImage(compose.ServiceConfig{}),
		"no platform defaults to x86")
}
