package roleconfig

import (
	"os"
	"testing"
)

// Note: Current() reads the env var on every call (no startup memoisation), so
// these tests can exercise each role mode without process restarts.

func TestRolesMatchSchedulerOwnership(t *testing.T) {
	cases := []struct {
		env          string
		wantRole     Role
		wantHTTP     bool
		wantSchedule bool
	}{
		{"", RoleAll, true, true},          // default = legacy single-binary
		{"all", RoleAll, true, true},       // explicit legacy
		{"api", RoleAPI, true, false},      // N replicas: serve HTTP, NO schedulers
		{"worker", RoleWorker, false, true}, // 1 replica: schedulers, no HTTP
		{"API", RoleAPI, true, false},      // case-insensitive
		{" worker ", RoleWorker, false, true}, // whitespace-tolerant
		{"garbage", RoleAll, true, true},   // unknown → legacy mode (safe default)
	}
	for _, c := range cases {
		t.Run(c.env, func(t *testing.T) {
			t.Setenv("ROLE", c.env)
			if got := Current(); got != c.wantRole {
				t.Errorf("Current() = %q, want %q", got, c.wantRole)
			}
			if got := ServesHTTP(); got != c.wantHTTP {
				t.Errorf("ServesHTTP() = %v, want %v", got, c.wantHTTP)
			}
			if got := RunsSchedulers(); got != c.wantSchedule {
				t.Errorf("RunsSchedulers() = %v, want %v", got, c.wantSchedule)
			}
		})
	}
}

// TestSafetyInvariant is the property that makes horizontal scaling safe: at
// any deployment, AT LEAST ONE process must own schedulers. ROLE=api everywhere
// would silently leave the system with no cron — the test pins the invariant
// so a future "tighten the default" PR can't accidentally do that.
func TestApiRoleNeverOwnsSchedulers(t *testing.T) {
	t.Setenv("ROLE", "api")
	if RunsSchedulers() {
		t.Fatal("ROLE=api must NOT run schedulers — that's the whole point of the split")
	}
}

func TestWorkerRoleOwnsSchedulersButNotHTTP(t *testing.T) {
	t.Setenv("ROLE", "worker")
	if !RunsSchedulers() || ServesHTTP() {
		t.Fatalf("worker should be schedulers-only: schedulers=%v http=%v", RunsSchedulers(), ServesHTTP())
	}
}

// Sanity: clearing ROLE must restore legacy behaviour, never silently leave
// stale state from a prior test.
func TestDefaultRoleIsAllForLegacyCompat(t *testing.T) {
	os.Unsetenv("ROLE")
	if Current() != RoleAll {
		t.Fatalf("empty ROLE must default to RoleAll for backwards compat, got %q", Current())
	}
}
