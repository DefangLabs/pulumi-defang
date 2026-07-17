package gcp

import "testing"

func TestGcpPostgresVersion(t *testing.T) {
	tests := []struct {
		version int
		want    string
	}{
		{9, "POSTGRES_9_6"},
		{10, "POSTGRES_10"},
		{11, "POSTGRES_11"},
		{12, "POSTGRES_12"},
		{13, "POSTGRES_13"},
		{14, "POSTGRES_14"},
		{15, "POSTGRES_15"},
		{16, "POSTGRES_16"},
		{17, "POSTGRES_17"},
		{0, "POSTGRES_17"},  // unparseable tag → latest
		{99, "POSTGRES_17"}, // unknown major → latest
	}
	for _, tt := range tests {
		if got := gcpPostgresVersion(tt.version); got != tt.want {
			t.Errorf("gcpPostgresVersion(%d) = %q, want %q", tt.version, got, tt.want)
		}
	}
}
