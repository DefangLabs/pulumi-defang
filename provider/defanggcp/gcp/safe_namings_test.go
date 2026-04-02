package gcp

import (
	"regexp"
	"strings"
	"testing"
)

const netPrefix = "net"

func TestHashTrim(t *testing.T) {
	const maxLen = 50 - 8
	for l := range 60 {
		s := strings.Repeat("x", l)
		got := hashTrim(s, maxLen)
		want := min(l, maxLen)
		if len(got) != want {
			t.Errorf("hashTrim() want length %d, got %d", want, len(got))
		}
	}

	// Test with hyphen (avoid consecutive dashes)
	got := hashTrim("xxxxxxxxxx-xxxxxxxx", 18)
	if len(got) != 18 {
		t.Errorf("hashTrim() with hyphen want length 18, got %d", len(got))
	}
	const expected = "xxxxxxxxxx-x1a4k3e"
	if got != expected {
		t.Errorf("hashTrim() with hyphen want %q, got %q", expected, got)
	}
}

func TestCloudrunServiceName(t *testing.T) {
	var cfg = &GlobalConfig{
		Prefix: "defang",
		Stack:  "dev",
	}
	// Regex following the gcp error message: only lowercase, digits, and hyphens;
	// must begin with letter, and cannot end with hyphen; must be less than 50 characters
	validationRegex := regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}[a-z0-9]$`)
	for l := range 60 {
		s := strings.Repeat("x", l)
		got := cloudrunServiceName(s, cfg) + "-1234567"
		if validationRegex.MatchString(got) == false {
			t.Errorf("cloudrunServiceName() produced invalid service name: %v", got)
		}
	}
}

//nolint: thelper,funlen // reporting the line in validate is more useful; table is intentionally long
func TestNetworkName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result string)
	}{
		{
			name:  "empty input should be prefixed with net-",
			input: "",
			validate: func(t *testing.T, result string) {
				if result != netPrefix {
					t.Errorf("expected 'net', got %q", result)
				}
			},
		},
		{
			name:  "only invalid characters should be replaced and prefixed",
			input: "!@#$%^&*()",
			validate: func(t *testing.T, result string) {
				if result != netPrefix {
					t.Errorf("expected 'net' after replacing invalid chars, got %q", result)
				}
			},
		},
		{
			name:  "input starting with number should be prefixed",
			input: "123network",
			validate: func(t *testing.T, result string) {
				if result != "net-123network" {
					t.Errorf("expected 'net-123network', got %q", result)
				}
			},
		},
		{
			name:  "input starting with uppercase should be lowercased and prefixed",
			input: "MyNetwork",
			validate: func(t *testing.T, result string) {
				if result != "mynetwork" {
					t.Errorf("expected 'mynetwork', got %q", result)
				}
			},
		},
		{
			name:  "input starting with hyphen should be prefixed",
			input: "-network",
			validate: func(t *testing.T, result string) {
				if result != "net-network" {
					t.Errorf("expected 'net-network', got %q", result)
				}
			},
		},
		{
			name:  "valid name starting with lowercase letter should be preserved",
			input: "validnetwork",
			validate: func(t *testing.T, result string) {
				if result != "validnetwork" {
					t.Errorf("expected 'validnetwork', got %q", result)
				}
			},
		},
		{
			name:  "valid name with hyphens and numbers",
			input: "my-network-123",
			validate: func(t *testing.T, result string) {
				if result != "my-network-123" {
					t.Errorf("expected 'my-network-123', got %q", result)
				}
			},
		},
		{
			name:  "name with mixed invalid characters should be replaced",
			input: "my_Network!@123",
			validate: func(t *testing.T, result string) {
				if result != "my-network--123" {
					t.Errorf("expected 'my-network--123', got %q", result)
				}
			},
		},
		{
			name:  "name ending with hyphen should have hyphen trimmed",
			input: "mynetwork-",
			validate: func(t *testing.T, result string) {
				if result != "mynetwork" {
					t.Errorf("expected 'mynetwork', got %q", result)
				}
			},
		},
		{
			name:  "exactly 63 characters starting with lowercase",
			input: "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyzz",
			validate: func(t *testing.T, result string) {
				if len(result) != 63 {
					t.Errorf("expected length 63, got %d", len(result))
				}
				if result != "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyzz" {
					t.Errorf("expected input to be preserved, got %q", result)
				}
			},
		},
		{
			name:  "62 characters starting with lowercase",
			input: "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz",
			validate: func(t *testing.T, result string) {
				if len(result) != 62 {
					t.Errorf("expected length 62, got %d", len(result))
				}
				if result != "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz" {
					t.Errorf("expected input to be preserved, got %q", result)
				}
			},
		},
		{
			name:  "64 characters should be trimmed to 63 with hash",
			input: "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyzzz",
			validate: func(t *testing.T, result string) {
				if len(result) != 63 {
					t.Errorf("expected length 63 after trim, got %d", len(result))
				}
				expected := networkName("abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyzzz")
				if result != expected {
					t.Errorf("hash should be deterministic, got different results")
				}
			},
		},
		{
			name:  "very long name should be trimmed and hashed",
			input: "this-is-a-very-long-network-name-that-exceeds-the-maximum-allowed-length-of-63-characters",
			validate: func(t *testing.T, result string) {
				if len(result) > 63 {
					t.Errorf("expected length <= 63, got %d", len(result))
				}
				if len(result) != 63 {
					t.Errorf("expected exactly 63 characters for long input, got %d", len(result))
				}
			},
		},
		{
			name:  "name with uppercase and special chars should be normalized",
			input: "My_NETWORK_123!@#",
			validate: func(t *testing.T, result string) {
				if result != "my-network-123" {
					t.Errorf("expected 'my-network-123', got %q", result)
				}
			},
		},
		{
			name:  "name with only hyphens should result in net- prefix",
			input: "-----",
			validate: func(t *testing.T, result string) {
				if result != netPrefix {
					t.Errorf("expected 'net' for hyphen-only input, got %q", result)
				}
			},
		},
		{
			name:  "VPC name pattern from codebase",
			input: "myproject-vpc",
			validate: func(t *testing.T, result string) {
				if result != "myproject-vpc" {
					t.Errorf("expected 'myproject-vpc', got %q", result)
				}
			},
		},
	}

	// Validation patterns
	allowedCharsPattern := regexp.MustCompile(`^[a-z0-9-]+$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := networkName(tt.input)

			// Run test-specific validation
			tt.validate(t, result)

			// Common validations for all results
			if len(result) > 63 {
				t.Errorf("result length %d exceeds maximum of 63", len(result))
			}

			if len(result) > 0 {
				if result[0] < 'a' || result[0] > 'z' {
					t.Errorf("result %q does not start with lowercase letter", result)
				}

				lastChar := result[len(result)-1]
				if (lastChar < 'a' || lastChar > 'z') && (lastChar < '0' || lastChar > '9') {
					t.Errorf("result %q does not end with lowercase letter or digit", result)
				}

				if !allowedCharsPattern.MatchString(result) {
					t.Errorf("result %q contains invalid characters", result)
				}
			}
		})
	}
}

func TestNetworkNameDeterministic(t *testing.T) {
	// Test that hashing is deterministic
	longInput := "this-is-a-very-long-network-name-that-definitely-exceeds-the-63-character-limit-for-gcp-network-names"

	result1 := networkName(longInput)
	result2 := networkName(longInput)

	if result1 != result2 {
		t.Errorf("networkName should be deterministic, got different results: %q vs %q", result1, result2)
	}

	if len(result1) != 63 {
		t.Errorf("expected length 63, got %d", len(result1))
	}
}
