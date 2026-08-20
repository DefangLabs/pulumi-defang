package defanggcp

import "testing"

// The VPC's name is what the RemovePeering call names, and the program passes the
// network in as an id rather than a bare name. Getting this wrong turns the whole
// resource into a 404 no-op, and its GCP call has no mock to catch that.
func TestNetworkName(t *testing.T) {
	const vpc = "my-vpc"
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"a network id", "projects/defang-playground-dev/global/networks/html-css-js-vpc-e99e23a", "html-css-js-vpc-e99e23a"},
		{"a self-link", "https://www.googleapis.com/compute/v1/projects/p/global/networks/" + vpc, vpc},
		{"a bare name", vpc, vpc},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networkName(tt.in); got != tt.want {
				t.Errorf("networkName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
