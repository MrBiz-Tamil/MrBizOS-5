// Package roleconfig governs WHICH responsibilities a process instance owns,
// driven by the ROLE environment variable so the same binary can be deployed
// horizontally without double-acting.
//
//	ROLE=api      → N replicas: handle HTTP/WS only. NO schedulers, NO cron.
//	ROLE=worker   → exactly 1 replica: owns dispatch sweep, finance cron,
//	                compliance scanner, SLA monitor, notification outbox,
//	                anomaly scanner, daily digests.
//	ROLE=all      → the legacy single-process mode (everything in one).
//	                Default — preserves current dev / local docker-compose
//	                behaviour exactly.
//
// Rationale: at 2× horizontal scale, every scheduler that runs in-process on
// every instance double-acts. Splitting by role makes the dangerous part
// (schedulers) a singleton by deployment, not by Redis-lock gymnastics, while
// keeping HTTP horizontally trivial.
package roleconfig

import (
	"log"
	"os"
	"strings"
)

// Role is the deployment role of this process.
type Role string

const (
	RoleAll    Role = "all"    // legacy single-binary mode (default)
	RoleAPI    Role = "api"    // stateless request handler — no schedulers
	RoleWorker Role = "worker" // schedulers + cron singleton
)

// Current reads ROLE from the environment exactly once at startup.
func Current() Role {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ROLE"))) {
	case "api":
		return RoleAPI
	case "worker":
		return RoleWorker
	default:
		return RoleAll
	}
}

// ServesHTTP reports whether this process should bind the HTTP listener.
// True for api + all.
func ServesHTTP() bool {
	r := Current()
	return r == RoleAPI || r == RoleAll
}

// RunsSchedulers reports whether this process should start the background
// schedulers (dispatch sweep, cron, scanners, monitors, outbox worker).
// True for worker + all.
func RunsSchedulers() bool {
	r := Current()
	return r == RoleWorker || r == RoleAll
}

// LogRole logs the resolved role at boot so operators see which mode came up.
func LogRole() {
	r := Current()
	log.Printf("[role] running as ROLE=%s (http=%v schedulers=%v)", r, ServesHTTP(), RunsSchedulers())
}
