package program

import (
	"reflect"
	"testing"
	"time"
)

func TestParseTTL(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"", 0, false},
		{"never", 0, false},
		{"0", 0, false},
		{"12h", 12 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"1h30m", 90 * time.Minute, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"7d12h", 7*24*time.Hour + 12*time.Hour, false},
		{"1h", time.Hour, false}, // exactly the minimum
		{"59m", 0, true},         // below the minimum
		{"5m", 0, true},          // below the minimum
		{"-1h", 0, true},         // negative
		{"-1d48h", 0, true},      // negative day prefix
		{"213504d", 0, true},     // would overflow time.Duration
		{"3651d", 0, true},       // above the maximum
		{"tomorrow", 0, true},
		{"12", 0, true},   // bare number (only "0" is special)
		{"d", 0, true},    // no digits
		{"1.5d", 0, true}, // fractional days unsupported
	}
	for _, tt := range tests {
		got, err := parseTTL(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseTTL(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseTTL(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestSelfDestructCron(t *testing.T) {
	// 14:30:59 UTC on Aug 17 → seconds truncated, UTC enforced
	utc := time.Date(2026, time.August, 17, 14, 30, 59, 0, time.UTC)
	if got := selfDestructCron(utc); got != "30 14 17 8 *" {
		t.Errorf("selfDestructCron(utc) = %q, want %q", got, "30 14 17 8 *")
	}
	// non-UTC input is converted (UTC+2 → 12:05 UTC)
	cest := time.Date(2026, time.January, 2, 14, 5, 0, 0, time.FixedZone("CEST", 2*60*60))
	if got := selfDestructCron(cest); got != "5 12 2 1 *" {
		t.Errorf("selfDestructCron(cest) = %q, want %q", got, "5 12 2 1 *")
	}
}

func TestSelfDestructEnv(t *testing.T) {
	environ := []string{
		// kept: exact names
		"PROJECT=myproj",
		"STACK=preview",
		"DOMAIN=preview.myproj.tenant.defang.app",
		"HOME=/root",
		"USER=root",
		"NPM_CONFIG_UPDATE_NOTIFIER=false",
		"NO_COLOR=1",
		// kept: prefixes
		"DEFANG_STATE_URL=azblob://pulumi?storage_account=x",
		"DEFANG_PREFIX=Defang",
		"DEFANG_MODE=development",
		"PULUMI_BACKEND_URL=azblob://pulumi?storage_account=x",
		"PULUMI_CONFIG_PASSPHRASE=hunter2",
		"AZURE_SUBSCRIPTION_ID=sub",
		"AZURE_LOCATION=westus",
		// dropped: per-run / runtime / credentials
		"DEFANG_ETAG=abc123",
		"DEFANG_TTL=12h",
		"DEFANG_STATES_UPLOAD_URL=https://presigned",
		"DEFANG_EVENTS_UPLOAD_URL=https://presigned",
		"DEFANG_PULUMI_TARGETS=urn:x",
		"AZURE_FEDERATED_TOKEN_FILE=/tmp/token",
		"AZURE_CLIENT_SECRET=shh",
		"AZURE_CLIENT_CERTIFICATE_PATH=/tmp/cert.pem",
		"AZURE_CLIENT_CERTIFICATE_PASSWORD=shh",
		"AWS_ACCESS_KEY_ID=AKIA",
		"AWS_SECRET_ACCESS_KEY=shh",
		"AWS_SESSION_TOKEN=shh",
		"PATH=/usr/bin",
		"HOSTNAME=aca-abc",
		"CONTAINER_APP_JOB_NAME=defang-cd",
		"malformed-no-equals",
	}
	want := map[string]string{
		"PROJECT":                    "myproj",
		"STACK":                      "preview",
		"DOMAIN":                     "preview.myproj.tenant.defang.app",
		"HOME":                       "/root",
		"USER":                       "root",
		"NPM_CONFIG_UPDATE_NOTIFIER": "false",
		"NO_COLOR":                   "1",
		"DEFANG_STATE_URL":           "azblob://pulumi?storage_account=x",
		"DEFANG_PREFIX":              "Defang",
		"DEFANG_MODE":                "development",
		"PULUMI_BACKEND_URL":         "azblob://pulumi?storage_account=x",
		"PULUMI_CONFIG_PASSPHRASE":   "hunter2",
		"AZURE_SUBSCRIPTION_ID":      "sub",
		"AZURE_LOCATION":             "westus",
	}
	got := SelfDestructEnv(environ)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("selfDestructEnv() = %#v, want %#v", got, want)
	}
}
