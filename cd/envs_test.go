package main

import "testing"

func TestGetenv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		fallback string
		want     string
	}{
		{"env set", "TEST_GETENV_SET", "fromenv", "fallback", "fromenv"},
		{"env empty", "TEST_GETENV_EMPTY", "", "fallback", "fallback"},
		{"env unset", "TEST_GETENV_UNSET_XYZ", "", "fallback", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.key, tt.envVal)
			}
			if got := getenv(tt.key, tt.fallback); got != tt.want {
				t.Errorf("Getenv(%q, %q) = %q, want %q", tt.key, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestSplitByComma(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"one,,three", []string{"one", "", "three"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitByComma(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("SplitByComma(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("SplitByComma(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitByComma(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"false", "false", false},
		{"0", "0", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_GETENVBOOL_" + tt.name
			t.Setenv(key, tt.val)
			if got := getenvBool(key); got != tt.want {
				t.Errorf("GetenvBool(%q) with value %q = %v, want %v", key, tt.val, got, tt.want)
			}
		})
	}
}
