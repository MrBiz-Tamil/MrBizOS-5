# MrBiz-OS — Phase 0 Evidence & Audit-Finding Verification

**Mode:** Read-only. No source code modified. Builds/tests run; nothing deployed.
**Date:** 2026-10-24 · **Toolchain:** Go 1.26.0 (windows/amd64).
**Purpose:** Establish the honest build/test baseline and verify the audit's reported findings against the actual repository, with evidence and verification method per finding.

---

## 1. Build & Test Baseline (executed this session)

| Target | Command | Result | Evidence |
| :--- | :--- | :--- | :--- |
| drayya-core (`backend/`) | `go build ./...` | **PASS (exit 0)**, no warnings | Clean compile on Go 1.26.0. |
| drayya-core (`backend/`) | `go test ./... -count=1` | **FAIL overall** (see §1.1) | One package (`domain/pharmacy`) hung ~607s then failed; **all 16 grpcservices have no test files**; several infra/legacy packages pass. |
| saas-billing (`SaaS billing/backend/`) | `go build ./...` | **PASS (exit 0)** | Clean compile. |
| saas-billing (`SaaS billing/backend/`) | `go test ./... -count=1` | **PASS for all unit/domain packages; integration suite BLOCKED (no Postgres)** | See §1.2. |
| frontend (`frontend/`) | `npm run check` (svelte-check) | **PASS — 0 errors**, 153 warnings (a11y/CSS), 512 files, exit 0 | Node 22.17. Type-check is clean. There is **no `test` script** (scripts: dev/build/preview/prepare/check) → no frontend unit/E2E suite exists. |

> **Compilation is not correctness.** A clean `go build` across both backends only proves the code type-checks; it says nothing about whether a handler persists data or performs a real operation (see `01-current-state-inventory.md`).

### 1.1 drayya-core test result (recorded)
**Overall: FAIL.** Key observations from the run:
- **`drayya-core/internal/domain/pharmacy` — FAIL after 606.9s.** The output shows a goroutine stuck in `database/sql.(*Tx).awaitDone` — i.e. a test opened a DB transaction and **hung for ~10 minutes** before the package failed. This is a real test-health defect (a hang/deadlock or a test assuming a live DB that never responds), not a simple assertion failure. It also makes `go test ./...` effectively unusable in CI until fixed/skipped.
- **All 16 mounted ConnectRPC services report `[no test files]`** (`grpcservices/{iam,billing,travel,realestate,workflow,procurement,banking,hrms,project,journey,omnichannel,support,bpo,ecommerce,consulting,healthcare}`). → **Audit finding #10 VERIFIED**: there is zero service-level test coverage on the operational path.
- **Passing packages** (infra/legacy): `internal/money`, `internal/roleconfig`, `internal/shipping`, `internal/utils/privacy`, `internal/domain/store`, and several `internal/services/*` (`automation`, `disbursement`, `fx`, `notification`, `pan`, `storage`, `stripe`).
- **Targeted re-run captured (this session):**
  - `drayya-core/internal/domain/finance` — **PASS (3.77s)**. The disconnected double-entry ledger engine's own tests pass → corroborates that `domain/finance` is genuinely engineered (not mock), reinforcing it as a real asset to reconcile (not rebuild).
  - `drayya-core/internal/database` — **FAIL** on `TestSchemaSnapshot`: *"Database schema has drifted! If this is intentional, run tests with UPDATE_SNAPSHOT=1."* The GORM-generated schema (SQLite snapshot) no longer matches the committed snapshot — i.e. models changed without updating the snapshot. A real, if low-severity, drift signal; also a hint that schema is managed by GORM here (not the SQL migrations), consistent with the DEC-04 ambiguity.

**Net:** drayya-core's suite does not pass today; the failure is a hanging pharmacy test plus the complete absence of tests on the 16 services that actually serve the product. (`exit code 0` in the raw log is the trailing `tail` pipe's status, not the test's — the test run itself FAILED.)

### 1.2 saas-billing test result (verbatim summary)
**Unit/domain packages — PASS:** `billing` (4.9s), `contexts/iam/delivery/grpc` (6.7s), `creditnote`, `database`, `grpcerr`, `grpcserver`, `invoice`, `metadata`, `pdf`, `pdfrender`, `pricing`, `realtime`, `recurring`, `subscription`, `tax`, `trade`, `user`, `weighment`. (~18 packages green.)

**Integration suite (`saas-billing/tests`) — FAIL, cause = no database:** every failure is `dial tcp 127.0.0.1:5432 ... actively refused` (Postgres not running), not a logic failure. Affected tests are exactly the high-value ones:
- `TestTenantIsolation`, `TestCompanyMasterDataProtection` (cross-tenant isolation)
- `TestConcurrentInvoiceNumbering` (concurrency/sequence safety)
- `TestMasterEndToEndLifecycle` (full lifecycle)
- `TestAccountingPaymentSafety`, `TestAuditedAccountingEngine` (accounting integrity)

**Interpretation:** these DB-backed tests are real and meaningful; their status is **BLOCKED-pending-Postgres**, not failing. Running `deploy/docker-compose.yml` postgres (or any local Postgres with the test DSN) would exercise them. This is the single highest-value remaining Phase 0 verification.

---

## 2. Audit Findings — Verification (the 10 reported findings)

Classification per finding: **VERIFIED / PARTIALLY VERIFIED / NOT REPRODUCED / NOT VERIFIED / OUTDATED / BLOCKED**. Evidence cites `path:line`; verification method stated.

| # | Reported finding | Verdict | Evidence | Verification method |
| :-- | :--- | :--- | :--- | :--- |
| 1 | ConnectRPC services use static data / in-memory maps. | **VERIFIED** | 13 of 16 mounted services are in-memory/hardcoded: `grpcservices/billing/service.go:16-33` (map), `project/service.go:15-39`, `journey/service.go:15-34`, `omnichannel/service.go:15-32`, `support/service.go:15-32`, `consulting/service.go:15-34`, `ecommerce/service.go:15-32`, `bpo/service.go:16-31` (maps + `seedDefaults`); `iam/service.go:22`, `healthcare/service.go:22`, `hrms/service.go:22`, `procurement/service.go:24`, `banking/service.go:24` (hardcoded literals / ephemeral). | Full read of all 16 service files. |
| 2 | Business records disappear after server restart. | **VERIFIED (by construction)** | The in-memory services above hold state only in process maps with no DB writes (`db *gorm.DB` injected but unused). A restart re-runs `seedDefaults()` and loses all created records. | Code analysis; confirmable by a boot→create→restart→get smoke test (listed in `01` §10 item 4). |
| 3 | Authentication & tenant-isolation vulnerabilities. | **VERIFIED (auth) / PARTIALLY VERIFIED (isolation)** | `iam/service.go:22` — `Login` returns a signed **super-admin** token for any email, no password, no DB lookup → auth is effectively bypassed at the app layer. Interceptor now fails closed (`interceptor.go:46`) **but** admin/superadmin bypass the cross-tenant check (`interceptor.go:110`,`:128`), and persisting services hardcode `tenant-1` with no tenant filter on reads (e.g. `travel/service.go:59`, `realestate`, `workflow`). | Code read. The real isolation tests exist in `saas-billing/tests` but are BLOCKED-pending-Postgres (§1.2). |
| 4 | Standalone billing engine disconnected from the primary server. | **VERIFIED** | `SaaS billing/backend` (`saas-billing`) is a separate module/monolith and is **absent from `deploy/docker-compose.yml`** (only postgres/redis/backend/frontend/ai-workers). The deployed `drayya-core` billing is the in-memory mock (`grpcservices/billing`). | `deploy/docker-compose.yml` read; module graph. |
| 5 | Invoice operations not connected to accounting entries. | **VERIFIED** | `grpcservices/billing` computes GST splits but performs **no** ledger posting; the real double-entry engine (`domain/finance/service.go`) is **not mounted** (`grpcserver/mount.go` registers 16 services; finance is not one). | Code read of billing service + mount.go. |
| 6 | Simulated bank reconciliation and payroll results. | **VERIFIED** | Payroll: `hrms/service.go:101` `RunMonthlyPayroll` returns payslips hardcoded as `PaymentStatus:"DISBURSED"`/`DisbursedAt:now` with no calc, persistence, or bank call. Reconciliation: `banking/service.go:24` parses an uploaded CSV into transient structs; no persisted ledger match. | Code read. |
| 7 | External integrations present in code but not connected. | **VERIFIED / BLOCKED** | SDK clients exist (`backend/internal/services/{razorpay,stripe,gst,tds,pan,livekit,email,storage}`) but no mounted service calls a live provider; `healthcare` returns fake LiveKit tokens (`healthcare/service.go:78,95`); `omnichannel.SendMessage` returns a fabricated `ExternalMessageId` without sending (`omnichannel/service.go:322-325`); `ecommerce.SyncEtsyOrders` has no Etsy client. No credentials configured. | Code read; no integration framework (§ `02` §9). |
| 8 | AI workflow proposals may not persist/execute after approval. | **VERIFIED** | `ai-workers/worker.py` posts `CreateTask` to the **mock** workflow service; `workflow` approval/approve/reject act on in-memory/seeded tasks; there is no execution engine that runs an approved proposal against real services. Agent classes (`ai-workers/agents/*.py`) are not dispatched by `worker.py`. | Code read of worker.py + workflow service. |
| 9 | Frontend type-check and automated-test failures. | **NOT REPRODUCED (type-check) / N/A (tests)** | `npm run check` → **0 errors**, 153 a11y/CSS warnings, exit 0 (512 files). Type-checking passes. There is **no frontend test script**, so "automated-test failures" cannot occur — no frontend tests exist (a gap, not a failure; relates to #10). | Ran `npm run check`; inspected `package.json` scripts. |
| 10 | Missing service-level and end-to-end verification. | **VERIFIED** | drayya-core has finance-domain unit tests but **no** service/E2E tests for the mounted services; no browser E2E; the only real DB-backed integration/E2E suite (`saas-billing/tests`) is not run in CI/deploy and needs Postgres. | Test inventory across both modules. |

**Net:** 7 findings VERIFIED, 1 PARTIALLY VERIFIED (isolation leg), 1 VERIFIED+BLOCKED (integrations), 1 NOT REPRODUCED (frontend type-check #9 — it passes; frontend has no test suite). The audit's backend picture is accurate; its frontend type/test claim does not reproduce.

---

## 3. Confirmed Service Classification (supersedes the *(sampled)* statuses in `01`/`03`)

All 16 mounted services were read in full. Final classification:

| Service | Verdict | Persistence | Note |
| :--- | :--- | :--- | :--- |
| iam | **MOCK** | none | super-admin for any email; `CreateTenant` not persisted. |
| billing | **MOCK** | in-memory map | GST computed; no accounting; fake PDF. |
| hrms | **MOCK** | none | fabricates `DISBURSED` payroll. |
| healthcare | **MOCK** | none | fake LiveKit tokens; empty medical record. |
| procurement | **MOCK** | none | literal POs; no 3-way match persistence. |
| banking | **MOCK** | none | CSV→transient; no reconciliation persistence. |
| project | **MOCK** | in-memory maps | db injected, unused. |
| journey | **MOCK** | in-memory maps | db injected, unused. |
| omnichannel | **MOCK** | in-memory maps | fabricates "sent" ids; hardcoded "AI" replies. |
| support | **MOCK** | in-memory maps | db injected, unused. |
| consulting | **MOCK** | in-memory maps | db injected, unused. |
| ecommerce | **MOCK** | in-memory maps | db injected, unused. |
| bpo | **MOCK** | in-memory maps | db injected, unused. |
| travel | **PARTIAL** | GORM find/create | seeds demo rows; hardcodes tenant-1; no tenant filter. |
| realestate | **PARTIAL** | GORM find/create | as travel. |
| workflow | **PARTIAL** | GORM find/create (task) | seeds; `OrganizationID` hardcoded 1. |

**Tally: 13 MOCK, 3 PARTIAL, 0 fully real.** No mounted service enforces tenant isolation at the data layer or performs cross-domain transactional business logic.

---

## 4. Phase 0 Exit Checklist (status)

| Item | Status |
| :-- | :--- |
| drayya-core build baseline | ✅ PASS |
| saas-billing build baseline | ✅ PASS |
| saas-billing unit/domain tests | ✅ PASS |
| drayya-core test result recorded | ✅ recorded — **FAIL** (pharmacy hang; 0 service tests) (§1.1) |
| drayya-core finance targeted re-run | ✅ **PASS** (double-entry engine tested) |
| drayya-core database schema snapshot | ✅ recorded — **FAIL** (schema drift) |
| drayya-core pharmacy test | ⚠ **FAIL/hang** (~607s on DB tx) — needs fix/skip before CI |
| saas-billing integration/isolation tests on Postgres | ⬜ BLOCKED (needs DB) — highest-value next step |
| Frontend `npm run check` (finding #9) | ✅ **PASS** — 0 errors, 153 a11y/CSS warnings; no test suite exists |
| All 16 services classified | ✅ done (§3) |
| Audit findings verified | ✅ done (§2) |
| Boot → create → restart → get smoke test (finding #2) | ⬜ recommended |

**Phase 0 is ~95% complete.** Build + test baselines are captured for both backends, the frontend type-check passes (0 errors), and all 16 services (111 RPC procedures) are classified. **Remaining — BLOCKED on infrastructure, not code:** run the `saas-billing` isolation/accounting suite on a live Postgres. **`docker` is not installed/on PATH in this environment**, so `deploy/docker-compose.yml` could not be started; the suite needs a reachable Postgres (`DATABASE_URL`, default `postgres://saas_user:saas_password@127.0.0.1:5432/saas_billing`). Exact next step: install Docker Desktop (or provide any Postgres 16 endpoint) and run `cd "SaaS billing/backend" && DATABASE_URL=... go test ./tests/... -run 'TenantIsolation|Accounting|Concurrency|RLS|E2E'` — the tests self-migrate. A boot→create→restart→get smoke test (finding #2) similarly needs Postgres. Two real test defects surfaced for Phase 1 triage: the `domain/pharmacy` hanging test and the `database` schema-snapshot drift.
