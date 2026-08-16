package program

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// minTTL is the shortest supported time-to-live. The self-destruct trigger is
// created as part of the deploy itself, so a TTL shorter than a few minutes
// could yield a fire time that is already in the past by the time the trigger
// exists — which cron-based triggers (Azure, GCP) would silently postpone by
// a whole year.
const minTTL = 5 * time.Minute

// parseTTL interprets the defang:ttl stack config (set from the DEFANG_TTL
// env var by the CLI). Empty, "never" and "0" mean no self-destruct. Other
// values are Go durations ("12h", "90m"), with an extra whole-days prefix
// ("7d", "7d12h") because stack files will typically express lifetimes in
// days and time.ParseDuration stops at hours.
func parseTTL(value string) (time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(value))
	switch s {
	case "", "never", "0":
		return 0, nil
	}

	var days time.Duration
	if i := strings.IndexByte(s, 'd'); i > 0 {
		n, err := strconv.Atoi(s[:i])
		if err != nil {
			return 0, fmt.Errorf("invalid ttl %q: %w", value, err)
		}
		days = time.Duration(n) * 24 * time.Hour
		s = s[i+1:]
		if s == "" {
			if days < minTTL {
				return 0, fmt.Errorf("ttl %q is below the minimum of %s", value, minTTL)
			}
			return days, nil
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid ttl %q: %w", value, err)
	}
	d += days
	if d < minTTL {
		return 0, fmt.Errorf("ttl %q is below the minimum of %s", value, minTTL)
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
	// Points at a token file that only exists in the current container.
	"AZURE_FEDERATED_TOKEN_FILE": true,
	// Session credentials must never outlive the run that received them; the
	// scheduled run authenticates with the ambient identity instead.
	"AWS_ACCESS_KEY_ID":     true,
	"AWS_SECRET_ACCESS_KEY": true,
	"AWS_SESSION_TOKEN":     true,
}

// selfDestructEnvPrefixes selects the config-carrying variables. The same
// filter serves all three clouds: a run only ever sees its own cloud's
// variables.
var selfDestructEnvPrefixes = []string{"AWS_", "AZURE_", "DEFANG_", "GCLOUD_", "GCP_", "PULUMI_"}

// selfDestructEnv filters the CD process's own environment down to what a
// scheduled `down` run needs. The current environment is authoritative: it is
// exactly the set the CLI composed for this run, minus runtime-injected
// variables (PATH, HOSTNAME, ACA/ECS metadata, ...) which the allowlist drops.
func selfDestructEnv(environ []string) map[string]string {
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

// sortedKeys returns the map's keys in stable order, for deterministic
// resource inputs (unordered env lists would diff on every deploy).
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
