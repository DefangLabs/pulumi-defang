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
		"nginx:latest", // Docker Hub reference, not ECR
		"123456789012.dkr.ecr.us-east-1.amazonaws.com/myrepo", // no tag
	}
	for _, dest := range cases {
		if _, err := resolveECRDigest(context.Background(), "us-east-1", dest); !errors.Is(err, ErrNotECRReference) {
			t.Errorf("resolveECRDigest(%q): got %v, want %v", dest, err, ErrNotECRReference)
		}
	}
}
