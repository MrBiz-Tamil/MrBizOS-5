# MrBiz-OS — Current-State Source-of-Truth Inventory

**Status:** Read-only discovery. No source code was modified.
**Date:** 2026-10-24
**Scope:** Evidence-based inventory of what exists in the repository *right now*, classified by verified status. This document is the factual foundation for the PRD (see `02-PRD.md`). Where evidence was insufficient, the limitation is stated explicitly.

> **Prime directive applied:** Successful compilation, an attractive screen, a registered route, a generated Protobuf contract, or a successful mock response is **not** treated as proof of a working business capability.

---

## 0. Status Legend

| Code | Meaning |
| :--- | :--- |
| **VERIFIED** | Confirmed working by reading code + supporting tests. |
| **UNVERIFIED** | Code exists and looks plausible; behavior not established (not run). |
| **PARTIAL** | Some required behavior exists; material gaps remain. |
| **MOCK** | Returns generated/hardcoded responses without the intended real operation. |
| **DISCONNECTED** | Real component exists but is not wired into the running/operational path. |
| **MISSING** | No adequate implementation found. |
| **BLOCKED** | Depends on credentials/external systems/unresolved decisions. |
| **UNKNOWN** | Insufficient evidence. |

Confidence is noted where it is not high. "Evidence" cites `path:line` where possible.

---

## 1. Headline Findings (read these first)

1. **There are two parallel, overlapping backends, and the better one is not deployed.**
   - `backend/` = `drayya-core`, a ConnectRPC "modular engine" (entry `backend/cmd/grpc-server/main.go`). This is what the main `frontend/` talks to and what `deploy/docker-compose.yml` runs.
   - `SaaS billing/backend/` = `saas-billing`, a mature DDD modular monolith (pgxpool, SQL migration runner, idempotency, write-gating, RLS, GST e-invoice/GSTR-1/Tally exports, subscription state machine, 38 test files, load test, authz scanner). **It is not referenced by `deploy/docker-compose.yml`.**
   - **Consequence:** the most production-ready billing/accounting code in the repository is orphaned from the running product, while the deployed billing path is an in-memory mock. (Confidence: high.)

2. **The deployed backend (`drayya-core`) is predominantly mock / in-memory.** Of 16 mounted ConnectRPC services, most return hardcoded literals or store state in process-memory maps that are lost on restart. Evidence in §4.

3. **Application-layer authentication is effectively absent in `drayya-core`.** The network interceptor now *fails closed* (good — recent fix `c87b7e2`), but `IAMService.Login` issues a **super-admin token to any email with no password check and no database lookup**. Evidence: `backend/internal/grpcservices/iam/service.go:22`.

4. **The genuinely engineered double-entry finance domain is disconnected.** `backend/internal/domain/finance/` has a GORM-backed double-entry ledger with integration tests, but it is **not mounted** in `backend/internal/grpcserver/mount.go`.

5. **The AI "autonomous workers" are a simulation.** `ai-workers/worker.py` runs 3 hardcoded handlers on a 15-second loop (falling back to *simulated* events when Redis is absent) and posts to the mock Workflow service. The rich agent classes (`ai-workers/agents/*.py`) and `missions/*.py` are **not imported or dispatched** by the worker loop.

6. **The planning docs contradict the code.** `docs/business-os/*` describe a *Next.js 14 + Go/Fiber REST "core-go" hospital-derived* platform. The actual repository is *SvelteKit + ConnectRPC `drayya-core` + a separate `saas-billing`*. Treat `docs/business-os/*` as **stale** planning artifacts, not as a description of the current system. (See §8.)

7. **Two schema mechanisms coexist with ambiguous ownership.** `drayya-core` creates tables via GORM `AutoMigrate` (`backend/internal/automigrate`). A separate set of **26 hand-written SQL migrations** lives in `backend/migrations/` (tenant template lock, row-level security, ledger balance constraint, idempotency, metering dedup, quotations, recurring invoices, trade, weighment) whose concepts match `saas-billing`, not the mounted `drayya-core` services. `drayya-core`'s entrypoint does **not** run these SQL files. (Confidence: high on the mismatch; the intended owner is a decision — see PRD DEC-04.)

---

## 2. Repository Topology (verified)

```
MrBiz-OS/
├── backend/                 drayya-core — ConnectRPC "modular engine" (DEPLOYED)
│   ├── cmd/grpc-server/      entrypoint :8080, H2C, "REST DECOMMISSIONED"
│   ├── internal/grpcserver/  mux, auth interceptor, service mount (16 services)
│   ├── internal/grpcservices/<16 services>  ConnectRPC handlers (mostly mock)
│   ├── internal/domain/<40+ domains>  37k LOC GORM models/services (mostly legacy/healthcare; largely NOT mounted)
│   ├── internal/services/<40+ services>  legacy integration clients (razorpay, stripe, gst, tds, abdm, livekit…)
│   ├── internal/automigrate/ GORM AutoMigrate across domain models
│   └── migrations/           26 hand-written SQL migrations (NOT run by entrypoint)
├── SaaS billing/            saas-billing — mature DDD monolith (NOT DEPLOYED)
│   ├── backend/              pgxpool, SQL migrate runner, 38 tests, loadtest, authzscan
│   └── frontend/             separate SvelteKit app
├── frontend/                Main SvelteKit app (~88k LOC) → talks to drayya-core :8080
├── ai-workers/              Python daemon + agent/mission classes (simulation)
├── proto/<21 packages>      ConnectRPC contracts (iam, billing, travel, realestate, …)
├── deploy/                  docker-compose (postgres16, redis7, backend, frontend, ai-workers) + Caddy
└── docs/
    ├── business-os/         STALE planning docs (describe a different Next.js/Fiber stack)
    └── hospital-reference/  Reference material from a prior hospital product
```

**Verified line counts (excluding node_modules):** proto ≈ 2.7k lines; `drayya-core` grpcservices ≈ 4.2k; `drayya-core` domain ≈ 37.3k; main frontend ≈ 88.3k. `saas-billing` backend has 38 `*_test.go` files.

---

## 3. Platform / Infrastructure Components

| Component | Evidence | Status | Notes |
| :--- | :--- | :--- | :--- |
| ConnectRPC server (H2C, Connect-JSON) | `backend/cmd/grpc-server/main.go`, `grpcserver/server.go`, `mount.go` | **VERIFIED (runs)** | 16 services mounted; REST "decommissioned". Not confirmed healthy at runtime (not executed). |
| Auth interceptor (fail-closed) | `grpcserver/interceptor.go:46` | **VERIFIED** | Requires signed JWT; rejects missing/invalid token + missing tenant unless `MRBIZ_DEV_AUTH` on. Enforces tenant match & BU allow-list; **admin/superadmin bypass the cross-tenant check** (`interceptor.go:110`, `:128`). |
| JWT issuance/validation | `backend/internal/tokens/jwt.go` | **PARTIAL** | HS256; `JWT_SECRET` required (panics if unset — good). 15-min access, 7-day refresh. `CreateAccessToken` hardcodes `tenant-1`/healthcare (`jwt.go:32`). No key rotation, no `kid`, no revocation/denylist, no refresh-rotation seen. |
| Database connection | `backend/internal/database/database.go` | **UNVERIFIED** | GORM. Supports Postgres + (per schema_test) SQLite for tests. Connection not exercised here. |
| Row-Level Security | `backend/migrations/015_row_level_security.sql` + `saas-billing` tenantctx | **DISCONNECTED (for drayya-core)** | RLS defined in SQL migrations that `drayya-core` does not run; `drayya-core` services do not set a per-request tenant GUC. Enforced in `saas-billing` (not deployed). |
| Redis / event bus | `ai-workers/worker.py`, `backend/internal/events`, `queue` | **UNVERIFIED / PARTIAL** | Redis in compose; worker falls back to simulation when absent. drayya-core event emission on business actions not verified. |
| Audit logging | `backend/internal/audit` (GORM plugin), `saas-billing/internal/audit` | **UNVERIFIED (drayya-core) / PRESENT (saas-billing)** | drayya-core audit plugin present but its coverage over the mounted (mock) services is unverified. |
| Deployment | `deploy/docker-compose.yml`, `deploy/Caddyfile` | **PARTIAL** | Runs postgres/redis/backend/frontend/ai-workers. **No `saas-billing`, no backups, no monitoring/observability stack, secrets via env.** |

---

## 4. Deployed Backend — ConnectRPC Services (`drayya-core`, the operational path)

All 16 are mounted with a `*gorm.DB` and the auth interceptor (`mount.go`). **Update (Phase 0):** every service below has since been read in full and its build/test baseline captured — see `05-phase0-evidence.md` §3 for the final confirmed classification (**13 MOCK, 3 PARTIAL**). The *(sampled)* caveats below are superseded: project, journey, omnichannel, support, consulting, ecommerce, bpo, hrms, healthcare are all **confirmed MOCK** (in-memory maps or hardcoded literals; `db` injected but unused).

| # | Service | Persistence reality | Status | Evidence |
| :-- | :--- | :--- | :--- | :--- |
| 1 | **IAM** (`iam`) | Hardcoded; DB injected but unused | **MOCK / CRITICAL** | `iam/service.go:22` Login → super-admin for any email, no password, no DB. `GetMe`/`ValidateSession`/`ListTenants`/`CreateTenant` return constants; `CreateTenant` not persisted (`:197`). |
| 16 | **Billing/GST** (`billing`) | In-memory `map` + mutex | **MOCK** | `billing/service.go:16-33` map store; `seedDefaults` 3 demo invoices; `CreateInvoice`/`RecordPayment`/`Cancel` mutate memory (lost on restart); `GenerateInvoicePDF:464` returns a fake 1-line PDF; GST split computed but **no accounting posting**; inter-state logic hardcoded to Maharashtra (`:257`). |
| 3 | **Travel** (`travel`) | GORM Find/Create, seed-on-empty | **PARTIAL** | `travel/service.go:27` reads `TourPackage`; seeds demo rows when empty; hardcodes `tenant-1` (`:59`); no tenant filter on reads. Visa flow persistence unverified *(sampled)*. |
| 4 | **Real Estate** (`realestate`) | GORM Find/Create, seed-on-empty | **PARTIAL** | `realestate/service.go:27` reads `Property`; seeds demo rows; no tenant scoping. Deals/site-visits persistence unverified *(sampled)*. |
| 5 | **Workflow/Tasks/Approvals** (`workflow`) | GORM Find/Create (task domain), seed-on-empty | **PARTIAL** | `workflow/service.go:27` reads `task.Task`; seeds demo tasks; `OrganizationID` hardcoded to 1 (`:32`). Approval queue / approve / reject persistence + permission enforcement unverified *(sampled)*. |
| 2 | **Healthcare** (`healthcare`) | DB injected | **UNVERIFIED** *(sampled)* | Thin service (140 lines); relationship to large `domain/emr`,`domain/clinical` not confirmed wired. |
| 6 | **Procurement** (`procurement`) | Literal arrays | **MOCK** | `procurement/service.go:24` `ListPurchaseOrders` returns hardcoded POs; 3-way match/receive-goods persistence not present in sampled code. |
| 7 | **Banking/Reconciliation** (`banking`) | Parses CSV, returns ephemeral | **MOCK** | `banking/service.go:24` parses uploaded CSV to transient structs; no persistence; reconciliation against real ledger not present. |
| 8 | **HRMS/Payroll** (`hrms`) | DB injected | **UNVERIFIED → likely MOCK** *(sampled)* | Payroll calc vs approval vs disbursement distinction not established. |
| 9 | **Project/UTMS** (`project`) | Mixed (11 db refs, also in-memory patterns) | **PARTIAL / UNVERIFIED** *(sampled)* | Largest service (343 lines); needs full read to confirm projects/tasks/milestones/timesheets persistence. |
| 10 | **Journey** (`journey`) | Mixed | **UNVERIFIED** *(sampled)* | Generic stage engine intent; persistence + transition rules unverified. |
| 11 | **Omnichannel** (`omnichannel`) | Mixed | **UNVERIFIED → likely MOCK** *(sampled)* | `SuggestAIReply` referenced by AI worker; no external WhatsApp/IG/FB API integration found in sampled code. |
| 12 | **Support/Helpdesk** (`support`) | Mixed | **UNVERIFIED** *(sampled)* | Ticketing/SLA persistence unverified. |
| 13 | **BPO Voice** (`bpo`) | Literal/sampled | **MOCK → BLOCKED** *(sampled)* | Voice/telephony requires LiveKit/provider credentials; not configured. |
| 14 | **E-commerce/POD** (`ecommerce`) | Mixed | **UNVERIFIED → likely MOCK** *(sampled)* | `SyncEtsyOrders` present as method; no Etsy credentials/integration found. |
| 15 | **Consulting/SOW** (`consulting`) | Mixed | **UNVERIFIED** *(sampled)* | SOW/timesheet persistence unverified. |

**Pattern across the layer:** where a DB is used at all, services (a) **seed demo rows when a table is empty**, (b) **hardcode `tenant-1`**, and (c) **do not filter reads by tenant/business-unit**, so cross-tenant isolation is not enforced at the data layer even when persistence exists. No service performs cross-domain transactional business logic (e.g., invoice → ledger posting, order → inventory decrement, payment → verification).

> **Verification limit:** services marked *(sampled)* were classified from imports, constructor signatures, and 1–2 handlers. Before Phase 0 exit, each must be read in full and exercised with `go test ./...` and a live smoke test. Do not treat the *(sampled)* statuses as final.

---

## 5. Disconnected but Real Code (assets worth preserving)

| Asset | Evidence | Status | Why it matters |
| :--- | :--- | :--- | :--- |
| Double-entry finance domain | `backend/internal/domain/finance/service.go`, `repository.go`, `*_test.go` (integration, reconcile, split, wallet guard) | **DISCONNECTED (real)** | GORM-backed ledger accounts, transactions with entries, revenue posting, tax calc, reconciliation. Not in `mount.go`. Candidate source of truth for accounting. |
| `saas-billing` invoicing/accounting | `SaaS billing/backend/internal/{invoice,accounting,quotation,creditnote,tax,trade,metering,recurring,subscription}` + 38 tests | **DISCONNECTED (mature)** | Real persistence, idempotency (`api/idempotency.go`), write-gating (`api/writegate.go`), RLS/tenantctx, GST exports (`gstexport/{einvoice,gstr1,tally}.go`), proration + state machine, sequence generation, advisory locks. **Strongest code in the repo; not deployed.** |
| Legacy integration clients | `backend/internal/services/{razorpay,stripe,gst,tds,pan,abdm,livekit,email,storage,ocr,fx}` | **UNVERIFIED / BLOCKED** | SDKs/clients present; real connectivity depends on credentials; wiring into mounted services not established. |
| AI agent + mission library | `ai-workers/agents/*.py`, `ai-workers/missions/*.py` | **DISCONNECTED** | ~17 agent classes + 9 mission scripts exist but are not driven by `worker.py`. |

---

## 6. Frontend (main `frontend/`, SvelteKit)

| Area | Evidence | Status |
| :--- | :--- | :--- |
| Transport to backend | `frontend/src/lib/grpc.ts` | **VERIFIED** — Connect-JSON over `fetch` to `:8080`; auth via HttpOnly `saas_session` cookie; no more forged superadmin header (fixed in `c87b7e2`). |
| Route surface | `frontend/src/routes/*` | **PARTIAL** — ~50 route groups (billing, invoices, quotations, pos, travel/*, realestate, ecommerce, healthcare, hrms, consulting, procurement, trade/*, workflow, settings/*, websites, marketing, inbox). Screens exist; most are backed by MOCK services per §4. |
| BFF `/api` routes | `frontend/src/routes/api/*`, `lib/server/devmode.ts` | **PARTIAL** — several server routes now fail closed unless `MRBIZ_DEV_AUTH` (fixed in `c87b7e2`). |

> **Verification limit:** screens were inventoried by route path, not individually loaded/tested. A visually complete screen is not evidence its workflow works (per prime directive).

---

## 7. AI Layer

| Element | Evidence | Status |
| :--- | :--- | :--- |
| Worker loop | `ai-workers/worker.py` | **MOCK/SIMULATED** — 3 hardcoded handlers (low-stock, overdue-invoice, inbound-social); simulated events every 15s when Redis absent; posts `workflow.v1.WorkflowService/CreateTask` (a mock service). |
| Agent classes | `ai-workers/agents/*.py` (crm, finance, sales, sourcing, negotiation, marketing, compliance, logistics, collections, executive_pa, supervisor, …) | **DISCONNECTED** — defined but not dispatched by the worker. |
| Mission scripts | `ai-workers/missions/mission_*.py` (9) | **DISCONNECTED** |
| Governance (budgets, tool permissions, approval gates, autonomy levels, audit) | — | **MISSING** — no governance model found binding agents to tenant/BU scope, tool allow-lists, budgets, or human-approval gates. |

---

## 8. Documentation vs Code — Explicit Contradictions (not silently resolved)

| # | Document claim | Code reality | Resolution |
| :-- | :--- | :--- | :--- |
| C-1 | `docs/business-os/01,08` — "Go microservices `core-go` + Next.js 14 `client`, Fiber REST, `/api/v1/*`". | Repo is ConnectRPC `drayya-core` (REST "decommissioned") + SvelteKit; no Next.js `client`. | `business-os/*` is **STALE**. The PRD supersedes it. |
| C-2 | `business-os/06` — Healthcare "90% coverage, comprehensive EMR/appointments/pharmacy/lab built". | Those live in `domain/*` (legacy) and are **not confirmed mounted**; the mounted `healthcare` service is a 140-line thin wrapper. | Coverage claims unverified against the operational path. |
| C-3 | `business-os/14` — security controls (Argon2id, signed URLs, anomaly detection) "implemented". | Implemented where? `drayya-core` IAM has no password check at all; signed-URL/anomaly code not verified in operational path. | Security claims **not verified**; treat as aspirational. |
| C-4 | README — long feature list (POS, e-commerce, omnichannel, EMR, double-entry accounting, 1-click payroll). | Most are MOCK or DISCONNECTED per §4–§5. | README is a **vision statement**, not current capability. |
| C-5 | 26 SQL migrations under `backend/migrations` imply `drayya-core` uses them. | `drayya-core` entrypoint runs GORM AutoMigrate only; the SQL set matches `saas-billing`. | Schema ownership is an open **decision** (PRD DEC-04). |

---

## 9. What Is Actually Safe to Rely On Today

- **Fail-closed network auth** in `drayya-core` (interceptor) — real, but undermined by a no-op `Login`.
- **`saas-billing`** invoicing/accounting/quotation/tax/trade logic + its test suite — real, but not deployed and not integrated with `drayya-core` or the main frontend.
- **`domain/finance`** double-entry ledger — real, but disconnected.
- **Proto contracts** for 21 packages — real interface definitions; the frontend client (`lib/grpc.ts`) is generated against them.

Everything else in the operational path should be treated as **not yet a business capability** until re-verified by the Phase 0 evidence work in the PRD.

---

## 10. Verification Backlog (to close UNKNOWN/UNVERIFIED before Phase 1)

1. `cd backend && go build ./... && go test ./...` — capture pass/fail per package.
2. `cd "SaaS billing/backend" && go build ./... && go test ./...` — confirm the 38 tests pass; record coverage of accounting/idempotency/RLS.
3. Read each *(sampled)* service in §4 fully; confirm persist-vs-mock and tenant filtering.
4. Boot `deploy/docker-compose.yml`; confirm DB connects, automigrate runs, a login→list→create round-trip; confirm what survives a backend restart (expected: in-memory services lose data).
5. Confirm whether any `domain/*` or `services/*` code is reachable from the mounted services, or is dead code.
6. Decide schema ownership (GORM AutoMigrate vs the SQL migration set) — DEC-04.
