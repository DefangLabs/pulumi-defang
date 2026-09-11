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
		if _, err := resolveECRDigest(context.Background(), dest); !errors.Is(err, ErrNotECRReference) {
			t.Errorf("resolveECRDigest(%q): got %v, want %v", dest, err, ErrNotECRReference)
		}
	}
}

// The registry the image was actually pushed to is the source of truth for
// which region to query -- resolveECRDigest must use it even when it
// differs from the build's own region (e.g. a standalone Build targeting an
// ECR registry in another region than the one it runs CodeBuild in).
func TestECRRegistryRECapturesRegistryRegion(t *testing.T) {
	cases := []struct {
		registry, wantRegion string
	}{
		{"123456789012.dkr.ecr.us-east-1.amazonaws.com", "us-east-1"},
		{"123456789012.dkr.ecr.eu-west-2.amazonaws.com", "eu-west-2"},
		{"123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn", "cn-north-1"},
	}
	for _, c := range cases {
		m := ecrRegistryRE.FindStringSubmatch(c.registry)
		if m == nil {
			t.Fatalf("ecrRegistryRE didn't match %q", c.registry)
		}
		if m[1] != c.wantRegion {
			t.Errorf("ecrRegistryRE(%q): got region %q, want %q", c.registry, m[1], c.wantRegion)
		}
	}
}
