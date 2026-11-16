package gcp

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestInternalLBSourceRanges(t *testing.T) {
	var got []string
	for _, r := range internalLBSourceRanges(pulumi.String("100.64.0.0/24")) {
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
