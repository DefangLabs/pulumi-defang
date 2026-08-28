package program

import (
	"fmt"
	"strings"
	"time"

	"github.com/DefangLabs/defang/src/pkg/timeutils"
)

// CdTimeout is the wall-clock ceiling for one CD run. It is shared by the cd
// binary's own context timeout (cd/main.go) and the Cloud Build timeout the
// GCP self-destruct trigger requests, so a scheduled down gets exactly the
// budget an interactive run gets. (On Azure the ARM-side bound is the
// defang-cd job template's ReplicaTimeout, set by the CLI — see
// DefangLabs/defang issue 2213 for aligning that with this value.)
const CdTimeout = time.Hour

// maxTTL guards against typo'd far-future dates.
const maxTTL = 10 * 365 * 24 * time.Hour

// parseTTL interprets the defang:ttl stack config (set from the DEFANG_TTL
// env var by the CLI). Empty, "never" and "0" mean no self-destruct. Other
// values are Go durations with an optional whole-days prefix ("12h", "7d",
// "7d12h"), parsed by the same timeutils.ParseDuration the CLI validates
// the value with, so the two sides cannot drift. The CLI's ParseTTL already
// trims and lowercases before setting DEFANG_TTL, so this doesn't repeat it.
//
// There is deliberately no minimum here (the CLI's ParseTTL enforces a 1h
// floor for real deployments — see byoc.ParseTTL in the defang CLI repo) so a
// test or manual CD invocation that bypasses the CLI can use a short TTL
// (seconds or minutes) to verify the self-destruct trigger actually fires,
// without waiting an hour. A stack driven by the real CLI never sees
// anything shorter than the CLI's floor.
func parseTTL(value string) (time.Duration, error) {
	switch value {
	case "", "never", "0":
		return 0, nil
	}

	d, err := timeutils.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid ttl %q: %w", value, err)
	}
	if d <= 0 || d > maxTTL {
		return 0, fmt.Errorf("ttl %q must be between 0 and %s", value, maxTTL)
	}
	return d, nil
}

// selfDestructCron renders t as a 5-field cron expression matching one exact
// minute of the year, e.g. "30 14 17 8 *" (UTC). Neither ACA schedule
// triggers nor Cloud Scheduler support one-shot schedules, so the trigger
// would re-fire yearly — moot in practice, because a successful down deletes
// the trigger together with the rest of the stack, and after a failed down
// the re-fire acts as a retry.
func selfDestructCron(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%d %d %d %d *", t.Minute(), t.Hour(), t.Day(), int(t.Month()))
}

// selfDestructEnvExact are the unprefixed variable names a CD run needs (see
// ByocAzure.environment and its AWS/GCP counterparts in the CLI).
var selfDestructEnvExact = map[string]bool{
	"DOMAIN":                     true,
	"HOME":                       true,
	"NO_COLOR":                   true,
	"NPM_CONFIG_UPDATE_NOTIFIER": true,
	"PROJECT":                    true,
	"REGION":                     true,
	"STACK":                      true,
	"USER":                       true,
}

// selfDestructEnvExclude are prefixed variables that must NOT be frozen into
// the scheduled down run.
var selfDestructEnvExclude = map[string]bool{
	// Per-run upload URLs are presigned for THIS deploy and will have expired
	// long before the trigger fires.
	"DEFANG_EVENTS_UPLOAD_URL": true,
	"DEFANG_STATES_UPLOAD_URL": true,
	// The deploy's own identifiers and options; the scheduled down is a new run.
	"DEFANG_ETAG":           true,
	"DEFANG_PULUMI_TARGETS": true,
	"DEFANG_TTL":            true,
	// A one-off permission to take over another CD's state (see
	// cd/legacy_state.go). Freezing it into the trigger would carry the
	// override into every future scheduled run.
	"DEFANG_ALLOW_LEGACY_STATE_TAKEOVER": true,
	// Points at a token file that only exists in the current container.
	"AZURE_FEDERATED_TOKEN_FILE": true,
	// Credentials must never be frozen into a trigger resource; the scheduled
	// run authenticates with the ambient (managed) identity instead.
	"AWS_ACCESS_KEY_ID":                 true,
	"AWS_SECRET_ACCESS_KEY":             true,
	"AWS_SESSION_TOKEN":                 true,
	"AZURE_CLIENT_SECRET":               true,
	"AZURE_CLIENT_CERTIFICATE_PATH":     true,
	"AZURE_CLIENT_CERTIFICATE_PASSWORD": true,
}

// selfDestructEnvPrefixes selects the config-carrying variables. The same
// filter serves all three clouds: a run only ever sees its own cloud's
// variables.
var selfDestructEnvPrefixes = []string{"AWS_", "AZURE_", "DEFANG_", "GCLOUD_", "GCP_", "PULUMI_"}

// SelfDestructEnv filters the CD process's own environment down to what a
// scheduled `down` run needs. The current environment is authoritative: it is
// exactly the set the CLI composed for this run, minus runtime-injected
// variables (PATH, HOSTNAME, ACA/ECS metadata, ...) which the allowlist drops.
func SelfDestructEnv(environ []string) map[string]string {
	env := map[string]string{}
	for _, kv := range environ {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || selfDestructEnvExclude[k] {
			continue
		}
		if selfDestructEnvExact[k] {
			env[k] = v
			continue
		}
		for _, p := range selfDestructEnvPrefixes {
			if strings.HasPrefix(k, p) {
				env[k] = v
				break
			}
		}
	}
	return env
}
