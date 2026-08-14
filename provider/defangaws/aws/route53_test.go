package aws

import (
	"errors"
	"fmt"
	"testing"
)

// Upstream diagnostics the zone lookup has to recognise (or refuse to).
var (
	errNoMatchingZone       = errors.New("no matching Route 53 Hosted Zone found") // terraform-provider-aws >= v5.51
	errNoMatchingZoneLegacy = errors.New("no matching Route53Zone found")          // pulumi-aws <= v6.37
	errMultipleZones        = errors.New("multiple Route 53 Hosted Zones matched; use additional constraints")
	errCouldntFindResource  = errors.New("couldn't find resource") // the zone_id branch of the data source
	errAssumeRoleDenied     = errors.New("operation error STS: AssumeRole, api error AccessDenied: not authorized")
	errThrottled            = errors.New("operation error Route 53: ListHostedZonesByName, api error Throttling")
)

// TestAsZoneNotFound pins which upstream failures become ErrZoneNotFound — the
// signal that makes a caller skip BYOD records instead of failing the deploy —
// and, more importantly, which ones must stay opaque errors.
func TestAsZoneNotFound(t *testing.T) {
	// The shape a Pulumi Go caller actually sees: the engine and the bridge each
	// wrap the upstream diagnostic, and multierror formats the body.
	wrapped := func(err error) error {
		return fmt.Errorf("rpc error: code = Unknown desc = invocation of "+
			"aws:route53/getZone:getZone returned an error: invoking "+
			"aws:route53/getZone:getZone: 1 error occurred:\n\t* %w\n\n", err)
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{
			name: "current wording (terraform-provider-aws >= v5.51)",
			err:  wrapped(errNoMatchingZone),
			want: true,
		},
		{
			name: "legacy wording (pulumi-aws <= v6.37)",
			err:  wrapped(errNoMatchingZoneLegacy),
			want: true,
		},
		{
			name: "bare message, unwrapped",
			err:  errNoMatchingZone,
			want: true,
		},
		{
			// Must NOT be swallowed: skipping BYOD records here would hide a
			// misconfigured x-defang-dns-role behind a warning.
			name: "assume-role failure",
			err:  wrapped(errAssumeRoleDenied),
			want: false,
		},
		{
			name: "throttling",
			err:  wrapped(errThrottled),
			want: false,
		},
		{
			// The zone_id branch of the data source reports not-found differently;
			// we look zones up by name, so this must not silently match.
			name: "lookup-by-id not-found wording",
			err:  wrapped(errCouldntFindResource),
			want: false,
		},
		{
			name: "too many matches",
			err:  wrapped(errMultipleZones),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := asZoneNotFound(tt.err, "example.com")
			if isNotFound := errors.Is(got, ErrZoneNotFound); isNotFound != tt.want {
				t.Errorf("errors.Is(asZoneNotFound(), ErrZoneNotFound) = %v, want %v", isNotFound, tt.want)
			}
			if tt.err == nil {
				if got != nil {
					t.Errorf("asZoneNotFound(nil) = %v, want nil", got)
				}
				return
			}
			// Whatever the verdict, the original error stays reachable: a
			// not-found is wrapped, anything else is returned untouched.
			if !errors.Is(got, tt.err) {
				t.Errorf("asZoneNotFound() dropped the underlying error: %v", got)
			}
		})
	}
}
