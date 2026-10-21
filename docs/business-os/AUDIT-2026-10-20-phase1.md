# MrBiz-OS — Phase 1 Audit: Findings & Implementation Plan

**Date:** 2026-10-20
**Scope:** Evidence-based repository audit (Phase 1 of the Shared Business Engine engagement).
**Method:** Direct inspection of source, schema, middleware, tests; `go build ./...`, targeted `go test` runs. Claims below cite files/lines. Nothing is marked "works" because a route or table merely exists.

> **Status of prior docs:** The existing `docs/business-os/0X` files (esp. `06-feature-gap-analysis.md`) are **stale**. They describe the new business units at 10–35%. The code has moved well past that — those domains, tenant middleware, and integration tests now exist. This report supersedes them where they conflict.

---

## 0. Correction to the engagement brief

The brief assumes a stack of **gRPC/Connect + SvelteKit**. The repository does not use these. Actual stack (evidence: [`core-go/go.mod`](../../core-go/go.mod), [`client/lib/api/base.ts`](../../client/lib/api/base.ts)):

| Brief assumed | Actual |
| :-- | :-- |
| gRPC / Connect APIs | **REST** over Go **Fiber v2** (`github.com/gofiber/fiber/v2`); ~1,846 routes in `routes.json`. gRPC present only as an *indirect* dep of LiveKit. |
| SvelteKit frontend | **Next.js 14 App Router** (React), Axios, 687 `.ts/.tsx` files, 70+ route groups. |
| PostgreSQL | ✅ Postgres (GORM), with a SQLite fallback for dev/test. |
| Redis | ✅ `redis/go-redis/v9` (event bus, queue). |

**Implication:** The "REST compatibility boundary" the brief mentions is moot — REST *is* the primary API. No gRPC/Connect migration is warranted. Recommend keeping the modular-monolith-in-Fiber design and **not** introducing Connect or rewriting the frontend in SvelteKit.

---

## 1. Current architecture & domain map

**Shape:** Modular monolith. `core-go` (~108k LOC Go, 620 files) + `client` (Next.js).

**Backend layers:**
- `internal/domain/<name>/` — 42 domains, each typically `model.go` / `repository.go` / `service.go` / `handler.go`. This is the modern, correct layering.
- `internal/handlers/` — 66 older handler groups (pre-domain style). **Two coexisting styles** (see §6).
- `internal/models/` — 26 legacy GORM models (duplicates of some `domain/*/model.go`; see §6).
- Infra: `middleware` (JWT, RBAC, rate-limit, business context, Sentry), `events` (Redis Streams), `audit` (GORM write-interception), `cron`/`dispatch`/`queue` (workers), `crypto`/`utils/privacy` (AES-256-GCM PHI encryption), `money`, `pdf`, `ws`, `livekit`, `tokens`.

**The five business units — all now present as domains:**

| BU | Domain package(s) | Integration test | Reality |
| :-- | :-- | :-- | :-- |
| Healthcare | `identity`, `clinical`, `emr`, `hospital`, `pharmacy`, `lab`, `appointment` | `emr_grant_test`, `pharmacy_*_test` | **Mature** — the built-out core. |
| Travel & Visa | `domain/travel` | `travel_visa_test.go` | Scaffolded + tested; depth unverified. |
| Real Estate | `domain/realestate` | `real_estate_test.go` | Scaffolded + tested; depth unverified. |
| E-commerce/POD | `domain/ecommerce`, `store`, `shipping` | `ecommerce_pod_test.go` | Scaffolded + tested; depth unverified. |
| Software Consulting | `domain/consulting`, `corporate`, `subscription` | `consulting_test.go` | Scaffolded + tested; depth unverified. |

**Shared engine services present:** `customer` (Customer 360), `journey` (workflow state machine), `task`, `finance` (double-entry ledger), `communication`, `cms` (multi-site), `middleware/business_context.go` (org/BU/site scoping), `ceo_ai`.

---

## 2. Shared-engine vs business-specific matrix

| Capability | Shared service exists? | Evidence | Enforced consistently? |
| :-- | :-- | :-- | :-- |
| Org / BU / Site context | ✅ `internal/middleware/business_context.go` | header/query → Fiber locals | ❌ **Not authorized** (P0-1) |
| Identity / Auth / RBAC | ✅ `domain/identity`, `middleware`, `tokens/jwt.go` | JWT, roles | Partial |
| Customer 360 | ✅ `domain/customer` | has `OrganizationID`/BU fields | Unverified cross-BU |
| CRM / Leads | ⚠️ partial | `client/components/crm/LeadCaptureForm.tsx` exists | backend depth unverified |
| Workflow / Journey | ✅ `domain/journey` | `journey/service.go` + `workflow_test.go` | Unverified |
| Tasks | ✅ `domain/task` | scopes by `BusinessUnitID` (`task/repository.go:60`) | ✅ at repo level |
| Finance / ledger | ✅ `domain/finance` | double-entry, BU-scoped (`finance/repository.go:48-79`) | ✅ at repo level |
| Multi-currency / tax | ⚠️ partial | `components/finance/TaxCenter.tsx` | unverified |
| Notifications / Email / WA | ✅ | `handlers/whatsapp_webhook.go`, Resend, FCM | present |
| Products / inventory | ✅ | `store`, `pharmacy` | present |
| Analytics / dashboards | ✅ | `domain/analytics` | present |
| Subscriptions / SaaS billing | ✅ | `domain/subscription` | present |

**Verdict:** The shared-engine vision is **genuinely implemented in structure** — this is *not* five copy-pasted apps. The gap is not "missing engine"; it is **enforcement and verification** (tenant auth, consistent scoping, green tests).

---

## 3. Feature inventory: implemented / partial / missing / broken

**Implemented & verified:**
- Server compiles cleanly: `go build ./cmd/server/` → **exit 0**.
- PHI field encryption (AES-256-GCM) with a correct prod safety gate (`internal/utils/privacy/crypto.go:28-53`): refuses to boot in prod without `DB_ENCRYPTION_KEY`.
- Finance double-entry ledger, BU-scoped queries.
- Frontend genuinely wired to backend (not mocks) in spot-checks: `app/appointments/page.tsx` calls `appointmentsApi`; `lib/api-stubs.ts` (misnamed) issues real `api.get/post` calls.

**Partial:**
- The four non-healthcare BUs: domains + tests exist, but end-to-end depth (full CRUD, business rules, UI journeys) not yet traced.
- CRM/leads, multi-currency/tax: UI present, backend depth unconfirmed.

**Broken:**
- `cmd/seed` **fails to build**: `cmd/seed/seed_finance.go:104` references removed field `PostedAt` on `finance.LedgerTransaction`. → fresh-environment data seeding is broken (P1).
- Integration tests **failing**: `tests/integration/architecture_test.go` `TestHubFallbackRouting` → got 404 then 400, and stock not deducted (expected 8, got 10). Tests execute on SQLite but are **not green** (P1).

**Missing / not in repo:**
- Client `package.json`, `next.config.*`, `tsconfig.json`, lockfile — **the frontend is not buildable as checked in** (P2).
- No `.git` — the project is **not under version control** (P0-4).

---

## 4. Critical security & tenant-isolation findings

### P0-1 — Tenant context is unauthenticated (client-controlled)
`internal/middleware/business_context.go:20-61`: `business_unit_id`, `organization_id`, `site_id` are read from the **`X-Business-Unit-Id` header / `?business_unit_id=` query**, with **no check that the authenticated user belongs to that org/BU/site**. Missing values default to Healthcare (BU 2) / org 1.
- **Business impact:** Any authenticated user can switch BU/org by changing a header and read/write another business's data — including BU-scoped **finance/ledger** rows (`finance/repository.go:48`). This is exactly the "never rely on client-supplied tenant identifiers" rule, violated.
- **Fix:** Derive org/BU membership from the JWT/user record server-side; validate the requested BU against the user's allowed set; reject (403) otherwise. Remove the silent Healthcare default.
- **Verify:** Add a negative test: user of BU 3 sets `X-Business-Unit-Id: 2` → expect 403; assert no BU-2 rows returned.

### P0-2 — Secrets committed in `core-go/.env`
Real-looking live secrets: `LIVEKIT_API_SECRET`, `RESEND_API_KEY` (`re_LRt2…`), `DB_ENCRYPTION_KEY` (the PHI key), `FINANCE_SECRET_SALT`. `JWT_SECRET` is a weak placeholder (`your_super_secret_jwt_key`).
- **Impact:** Anyone with the tree can decrypt PHI, forge finance integrity hashes, send mail/video as the org.
- **Fix:** Rotate all four secrets, remove `.env` from the tree, provide `.env.example` with placeholders, load real values from a secret store/CI.
- **Verify:** `git` history clean of `.env`; app reads from env/secret manager; secret-scanner in CI.

### P0-3 — Patient PHI databases committed
`core-go/Dr.Ayya.db` (+ `-wal`/`-shm`, ~9 MB) and `medihub_e2e_test.db` sit in the repo.
- **Impact:** Patient data at rest in source control — privacy/compliance breach.
- **Fix:** Remove from tree; `.gitignore` all `*.db*`; regenerate test data via seeders. **Never use production patient data for testing.**
- **Verify:** No `*.db` tracked; test env built from seed only.

### P0-4 — Not under version control + inadequate `.gitignore`
No `.git`; `core-go/.gitignore` lists only `main`/`seed`/`admin`. If committed as-is, P0-2/3 and all binaries/logs land in history.
- **Fix:** `git init` with a proper `.gitignore` (`.env`, `*.db*`, `*.exe`, `*.log`, build artifacts) **before** the first commit; purge the artifacts first.

### P1 — PHI encryption prod-gate depends solely on `APP_ENV`
`crypto.go:30` treats prod as `APP_ENV in {production,prod}`. Deploy without `APP_ENV` set → it **silently uses the dev default key** `medi_hub_default_secret_key_32bytes` for PHI.
- **Fix:** Fail-closed — require an explicit non-dev key unless a dedicated dev flag is set; log loudly; add a startup assertion.

---

## 5. Frontend-to-backend integration findings

- **Positive (spot-checked):** `app/appointments/page.tsx` loads via `appointmentsApi` with real loading/empty states; `lib/api-stubs.ts` and `lib/api/base.ts` issue real Axios calls to `/api/v1/*` with an idempotency-key interceptor. No `const mockData`/hardcoded `id:` arrays found in a client-wide scan. The earlier "mock/dummy/placeholder" keyword count (609 hits) is **mostly noise** (HTML `placeholder=` attributes, `TODO` comments) — not evidence of fake data.
- **Caveat:** Full journey tracing (the brief's 8-step trace × 70 routes) is **not done** in Phase 1. The `: Promise<any>` typing across the API layer hides contract drift; worth type-tightening.
- **Action for Phase 3+:** Trace the high-value journeys per BU (lead→invoice→payment; booking; order→fulfillment) end-to-end and assert DB writes, not HTTP 200.

---

## 6. Duplicate logic & unnecessary complexity

- **Two backend layering styles:** modern `internal/domain/*` vs legacy `internal/handlers/*` (66 groups) + `internal/models/*` (26 models). Several entities exist in both → drift risk. Converge on `domain/*`; treat `handlers/*`+`models/*` as the migration backlog.
- **Frontend API-layer sprawl in `client/lib/`:** `api.ts`, `api.ts_append`, `api-stubs.ts` (misnamed — real calls), `stubs-extended`, `generated/`, plus ~40 per-domain modules. Consolidate; delete `api.ts_append`; rename away from "stub".
- **Repo bloat (not code, but risk):** ~16 compiled binaries (40–70 MB each), ~180 `.log`/`.txt` build/test artifacts (`integration_debug_v1..v14`, `final_verification_v2..v6`, `test_full_v2..v18`), `routes.json` (135 KB), loose `.js` scripts. All should be gitignored/removed.

---

## 7. Prioritized implementation plan (with dependencies)

**Phase 2 — Stop the bleeding (P0, do first):**
1. `git init` + proper `.gitignore`; purge binaries/logs/DBs/`.env` **before** any commit. *(P0-4; blocks everything)*
2. Rotate + externalize secrets; add `.env.example`. *(P0-2; depends on 1)*
3. Remove committed PHI DBs; seed-only test data. *(P0-3; depends on 1)*
4. Fix `cmd/seed` build (`PostedAt`) so a clean env can be seeded. *(P1; needed for 6)*

**Phase 3 — Tenant trust (P0-1, the core of the brief):**
5. Server-side org/BU/site resolution from JWT + membership check in `business_context.go`; 403 on mismatch; remove defaults.
6. Audit every `domain/*` repository: ensure BU/org appears in `WHERE` for all tenant-scoped reads/writes (today only `finance` and `task` do). Add a shared scoping helper so it's applied once, not 40 times.
7. Negative cross-tenant tests (the brief's requirement).

**Phase 4 — Green baseline & shared services:**
8. Stand up a controlled test env (Postgres + Redis, or documented SQLite path); run the full suite; fix failing tests (`TestHubFallbackRouting`, …) to a **known-green baseline**. *(depends on 4)*
9. PHI prod-gate fail-closed (P1).
10. Verify Customer 360 cross-BU aggregation respects the clinical-read permission gate.

**Phase 5 — Per-BU depth:** trace & complete one BU at a time (Healthcare = reference), using the shared engine, keeping each BU's own rules.

**Phase 6 — Full test matrix:** unit + integration + browser + security + regression per BU; "change to one BU doesn't break another" regression gate.

**Phase 7 — Production-readiness verdict** with remaining blockers.

---

## 8. Code changes completed in this pass
**None.** Phase 1 is read-only audit + verification runs (`go build`, `go test`) as the brief directs ("establish the system's real condition before any rewrite"). No source modified. The failing build/tests above are the **pre-existing** state.

## 9. Tests executed & real results
- `go build ./...` → **FAIL**, single cause: `cmd/seed/seed_finance.go:104` (`PostedAt`).
- `go build ./cmd/server/` → **PASS (exit 0)**. `cmd/admin`, `cmd/seed_production`, `cmd/seed_frontend` → **PASS**.
- `go test ./tests/integration -run 'TestArchitecture…'` (SQLite, `JWT_SECRET=test-secret`) → **FAIL**: `TestArchitectureSuite/TestHubFallbackRouting` (404/400; stock not deducted). Tests do execute; suite is not green.
- Full suite not run (needs Postgres/Redis for some packages) — Phase 4 task.

## 10. Remaining blockers & production-readiness verdict
**NOT production-ready.** Blockers, in order:
1. **P0** Unauthenticated tenant context — cross-business data access. *(§4 P0-1)*
2. **P0** Committed secrets + PHI databases, no VCS hygiene. *(§4 P0-2/3/4)*
3. **P1** Broken seeder; red integration tests → no trustworthy baseline.
4. **P1** Tenant scoping applied in only 2 of ~40 domains.

**Honest summary:** The shared-engine architecture is real and more complete than the old docs claim — this is a genuine multi-business platform, not five apps. But it fails the brief's own acceptance test on **tenant isolation enforcement** and **verified-by-evidence readiness**. Fix P0s and establish a green baseline before building more BU depth.

---
*Classification: P0 critical blocker · P1 launch-essential · P2 important · P3 optimization.*
