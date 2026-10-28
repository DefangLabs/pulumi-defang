package defangscaleway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"
)

// fakeScaleway serves just enough of the Instance and Block APIs for
// teardownServer: the server, its terminate action, and DELETEs of its
// leftovers. The volume refuses deletion while "attached", as the real
// block API does until terminate has detached it.
type fakeScaleway struct {
	mu             sync.Mutex
	serverGone     bool
	localGone      bool // terminate deletes the local volume; DELETE /servers leaves it
	terminateFails bool // a never-booted server rejects the terminate action
	attachedPolls  int  // volume DELETEs refused before the volume comes free
	volumeDenied   bool // volume DELETE answers 403, which waiting cannot fix
	getFails       bool // GET /servers answers 500
	calls          []string
	deleted        []string
}

func (f *fakeScaleway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)

	const zone = "/instance/v1/zones/fr-par-1"
	switch {
	case r.Method == http.MethodGet && r.URL.Path == zone+"/servers/srv":
		if f.serverGone {
			http.NotFound(w, r)
			return
		}
		if f.getFails {
			http.Error(w, `{"type":"internal_error"}`, http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"server":{
			"volumes":{"0":{"id":"vol-sbs","volume_type":"sbs_volume"},"1":{"id":"vol-local","volume_type":"l_ssd"}},
			"public_ip":{"id":"ip-1"},
			"public_ips":[{"id":"ip-1"},{"id":"ip-2"}]}}`))
	case r.Method == http.MethodPost && r.URL.Path == zone+"/servers/srv/action":
		if f.terminateFails {
			http.Error(w, `{"type":"precondition_failed"}`, http.StatusBadRequest)
			return
		}
		f.serverGone = true
		f.localGone = true
		w.WriteHeader(http.StatusAccepted)
	case r.Method == http.MethodDelete && r.URL.Path == zone+"/servers/srv":
		f.serverGone = true
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodDelete && r.URL.Path == "/block/v1alpha1/zones/fr-par-1/volumes/vol-sbs":
		if f.volumeDenied {
			http.Error(w, `{"type":"permissions_denied"}`, http.StatusForbidden)
			return
		}
		if f.attachedPolls > 0 {
			f.attachedPolls--
			http.Error(w, `{"type":"precondition_failed","message":"volume is in_use"}`, http.StatusPreconditionFailed)
			return
		}
		f.deleted = append(f.deleted, "vol-sbs")
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodDelete && r.URL.Path == zone+"/volumes/vol-local":
		if f.localGone {
			http.NotFound(w, r)
			return
		}
		f.localGone = true
		f.deleted = append(f.deleted, "vol-local")
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodDelete && (r.URL.Path == zone+"/ips/ip-1" || r.URL.Path == zone+"/ips/ip-2"):
		f.deleted = append(f.deleted, r.URL.Path[len(zone+"/ips/"):])
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func TestTeardownServerDeletesVolumesAndIPs(t *testing.T) {
	defer func(d time.Duration) { volumeDetachPoll = d }(volumeDetachPoll)
	volumeDetachPoll = time.Millisecond

	for _, tc := range []struct {
		name           string
		terminateFails bool
		wantDelete     bool
		wantDeleted    []string
	}{
		{
			name: "terminate action", terminateFails: false, wantDelete: false,
			wantDeleted: []string{"ip-1", "ip-2", "vol-sbs"},
		},
		{
			name: "never booted, DELETE fallback", terminateFails: true, wantDelete: true,
			wantDeleted: []string{"ip-1", "ip-2", "vol-local", "vol-sbs"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeScaleway{terminateFails: tc.terminateFails, attachedPolls: 2}
			srv := httptest.NewServer(fake)
			defer srv.Close()

			c := &scwAPIClient{secretKey: "k", region: "fr-par", apiURL: srv.URL}
			c.teardownServer(context.Background(), "fr-par-1", "srv")

			if !fake.serverGone {
				t.Errorf("server not removed; calls: %v", fake.calls)
			}
			slices.Sort(fake.deleted)
			if !slices.Equal(fake.deleted, tc.wantDeleted) {
				t.Errorf("deleted %v, want %v; calls: %v", fake.deleted, tc.wantDeleted, fake.calls)
			}
			if !fake.localGone {
				t.Errorf("local volume left behind; calls: %v", fake.calls)
			}
			if got := slices.Contains(fake.calls, "DELETE /instance/v1/zones/fr-par-1/servers/srv"); got != tc.wantDelete {
				t.Errorf("DELETE fallback used = %v, want %v", got, tc.wantDelete)
			}
			if slices.Contains(fake.calls, "DELETE /block/v1alpha1/zones/fr-par-1/volumes/vol-local") {
				t.Error("tried to delete the local volume through the block API")
			}
		})
	}
}

func TestTeardownServerAlreadyGone(t *testing.T) {
	fake := &fakeScaleway{serverGone: true}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	c := &scwAPIClient{secretKey: "k", region: "fr-par", apiURL: srv.URL}
	c.teardownServer(context.Background(), "fr-par-1", "srv")

	if len(fake.deleted) != 0 {
		t.Errorf("deleted %v for a server that no longer exists", fake.deleted)
	}
}

func TestTeardownServerDoesNotWaitOnPermanentVolumeError(t *testing.T) {
	defer func(d time.Duration) { volumeDetachPoll = d }(volumeDetachPoll)
	volumeDetachPoll = time.Millisecond

	fake := &fakeScaleway{volumeDenied: true}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	c := &scwAPIClient{secretKey: "k", region: "fr-par", apiURL: srv.URL}
	err := c.deleteServerResources(context.Background(), "fr-par-1",
		serverResources{volumeIDs: []string{"vol-sbs"}})
	if err == nil {
		t.Fatal("expected the 403 to be reported")
	}
	n := 0
	for _, call := range fake.calls {
		if call == "DELETE /block/v1alpha1/zones/fr-par-1/volumes/vol-sbs" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("volume DELETE tried %d times, want 1 (a 403 is not retried)", n)
	}
}

func TestTeardownServerStillTerminatesWhenDiscoveryFails(t *testing.T) {
	defer func(d time.Duration) { cleanupRetryDelay = d }(cleanupRetryDelay)
	cleanupRetryDelay = time.Millisecond

	fake := &fakeScaleway{getFails: true}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	c := &scwAPIClient{secretKey: "k", region: "fr-par", apiURL: srv.URL}
	c.teardownServer(context.Background(), "fr-par-1", "srv")

	if !fake.serverGone {
		t.Errorf("server not removed after discovery failed; calls: %v", fake.calls)
	}
}
