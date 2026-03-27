package common

import "testing"

func TestNormalizeDNS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.bar.", "foo.bar"},
		{"Foo.bar", "foo.bar"},
		{"foo.bar", "foo.bar"},
		{"FOO.BAR.", "foo.bar"},
		{"already.lower", "already.lower"},
	}
	for _, tt := range tests {
		if got := NormalizeDNS(tt.input); got != tt.want {
			t.Errorf("NormalizeDNS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSafeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "name"},
		{"Name", "name"},
		{"name123", "name123"},
		{"name-with-hyphen", "name-with-hyphen"},
		{"name_hyphen", "name_hyphen"},
		{"name.hyphen", "name-hyphen"},
		{"name__hyphen", "name__hyphen"},
		{"12_34", "12_34"},
		{"1234", "1234"},
		{"1234.com", "1234-com"},
		{"_", "_"},
		{"...", "---"},
		{"---", "---"},
	}
	for _, tt := range tests {
		if got := SafeLabel(tt.input); got != tt.want {
			t.Errorf("SafeLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsPrivateZone(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"foo.local", true},
		{"internal.", true},
		{"flan.", false},
		{"example.com", false},
		{"foo.internal", true},
		{"bar.private", true},
		{"corp.", true},
		{"home.arpa", true},
		{"foo.lan", true},
		{"foo.intranet", true},
	}
	for _, tt := range tests {
		if got := IsPrivateZone(tt.input); got != tt.want {
			t.Errorf("IsPrivateZone(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGetZoneName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bar.com", "bar.com"},
		{"bar.co.", "bar.co."},
		{"foo.bar.com", "bar.com"},
		{"foo.bar.co.", "bar.co."},
	}
	for _, tt := range tests {
		if got := GetZoneName(tt.input); got != tt.want {
			t.Errorf("GetZoneName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
