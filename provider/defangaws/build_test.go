package defangaws

import (
	"context"
	"errors"
	"testing"
)

func TestResolveECRDigestRejectsNonECRDestination(t *testing.T) {
	cases := []string{
		"",
		"myrepo",
		"nginx:latest",                   // Docker Hub reference, not ECR
		"ghcr.io/acme/app:tag",           // fully qualified, but not ECR
		"docker.io/library/nginx:latest", // fully qualified, but not ECR
		"123456789012.dkr.ecr.us-east-1.amazonaws.com/myrepo",              // no tag
		"123456789012.dkr.ecr.us-east-1.amazonaws.com.evil.com/myrepo:tag", // lookalike host
	}
	for _, dest := range cases {
		if _, err := resolveECRDigest(context.Background(), "us-east-1", dest); !errors.Is(err, ErrNotECRReference) {
			t.Errorf("resolveECRDigest(%q): got %v, want %v", dest, err, ErrNotECRReference)
		}
	}
}
