package gcp

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestInternalLBSourceRanges(t *testing.T) {
	ranges := internalLBSourceRanges(pulumi.String("100.64.0.0/24"))
	got := make([]string, 0, len(ranges))
	for _, r := range ranges {
		s, ok := r.(pulumi.String)
		if !ok {
			t.Fatalf("unexpected range type %T", r)
		}
		if s == "0.0.0.0/0" {
			t.Fatal("internal LB backend must not admit 0.0.0.0/0")
		}
		got = append(got, string(s))
	}
	found := false
	for _, s := range got {
		found = found || s == "100.64.0.0/24"
	}
	if !found {
		t.Fatalf("proxy subnet CIDR missing from %v", got)
	}
}
