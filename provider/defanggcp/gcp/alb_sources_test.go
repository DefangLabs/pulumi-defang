package gcp

import "testing"

func TestInternalLBSourceRangesNotPublic(t *testing.T) {
	for _, r := range internalLBSourceRanges() {
		if s, ok := r.(interface{ String() string }); ok && s.String() == "0.0.0.0/0" {
			t.Fatal("internal LB backend must not admit 0.0.0.0/0")
		}
	}
}
