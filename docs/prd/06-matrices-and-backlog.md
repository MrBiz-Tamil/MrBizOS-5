# MrBiz-OS — Matrices, Component Verdicts & Implementation Backlog

Companion to `02-PRD.md`. Contains the seven required matrices/registers, the component-level Keep/Repair/Integrate/Replace/Defer verdicts, the prioritized backlog, and the first ten implementation tickets. Statuses are from `01` + `05` (Phase 0 evidence).

---

## 1. Architecture Decision Register

| ADR | Current state | Problem | Options | Decision [R] | Trade-offs | Migration | Acceptance |
| :-- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| ADR-1 Monolith | Go modular monolith (`drayya-core`) | Many domains tempt a microservices split | (a) keep monolith (b) microservices | **Keep monolith** | Simpler ops; must enforce module boundaries in-code | None | Module boundaries documented + linted |
| ADR-2 App API | ConnectRPC + some ad-hoc BFF stubs | Two ways to reach backend | (a) ConnectRPC only (b) REST BFF | **ConnectRPC sole API; retire BFF stubs** | Frontend uses generated client | Remove stub routes | No business logic in BFF; all via proto |
| ADR-3 Billing/Accounting | `saas-billing` (real, undeployed) + drayya-core billing (mock) + `domain/finance` (real, disconnected) | Duplicate engines; money path is mock | (a) adopt `saas-billing` (b) rebuild in drayya-core (c) wire `domain/finance` | **Adopt `saas-billing` as authoritative (DEC-01)** | Integration effort; must choose in-process module vs internal service | Fold module in; migrate schema; retire mock | Scenario A/B finance legs pass on Postgres; double-entry balanced |
| ADR-4 Schema | GORM AutoMigrate (drayya-core) + 26 SQL migrations (saas-billing-style) | Two mechanisms, ambiguous ownership | (a) SQL migrations (b) AutoMigrate | **Versioned SQL migrations (DEC-04)** | Explicit, ordered, safe; more discipline | Consolidate to one runner | `schema_migrations` authoritative; AutoMigrate removed for prod |
| ADR-5 Tenant isolation | App-only tenant header; RLS defined but not run by drayya-core | Cross-tenant leakage risk | (a) RLS + GUC + repo scoping (b) app-only | **RLS + tenant GUC + repo scoping** | Defense in depth | Set GUC per request; enable RLS | Isolation tests pass on Postgres |
| ADR-6 DI | Constructor injection in `mount.go` | — | keep | **Explicit constructor DI** | — | None | Services testable w/ fakes |
| ADR-7 Async/events | Redis in compose; worker simulates | Eventual consistency on money is unsafe | (a) sync money flows (b) all async | **Async for notifications/AI; money/stock sync+tx** | — | Define which events are async | No financial invariant depends on async |
| ADR-8 AI execution | Simulated worker | Ungoverned AI risk | (a) approved-APIs-only (b) direct DB | **AI calls approved APIs only; propose-then-approve** | Slower AI; safe | Build governance first | AI cannot complete high-impact action w/o approval |

---

## 2. Component Verdicts (Keep / Repair / Integrate / Replace / Defer)

| Component | Evidence | Verdict | Rationale |
| :--- | :--- | :--- | :--- |
| Go modular monolith | `backend/`, builds clean | **Keep** | Fit for purpose; no split justified (ADR-1). |
| ConnectRPC contracts (`proto/*`) | 21 packages, client generated | **Keep** | Real interface layer; reuse. |
| Auth interceptor (fail-closed) | `grpcserver/interceptor.go:46` | **Keep + Repair** | Keep fail-closed; add per-permission authz; remove admin cross-tenant bypass (`:110`). |
| `iam` service | `grpcservices/iam/service.go:22` | **Replace** | No real auth; rebuild on persisted users (reuse `saas-billing/internal/auth`). |
| `billing` service | `grpcservices/billing` in-memory | **Replace** | Mock; superseded by `saas-billing` (DEC-01). |
| `travel`/`realestate`/`workflow` | partial GORM | **Repair** | Add tenant scoping, remove demo-seed, real logic. |
| `project`/`journey`/`omnichannel`/`support`/`consulting`/`ecommerce`/`bpo`/`procurement`/`banking`/`hrms`/`healthcare` | mock | **Replace** (pilot-critical first) / **Defer** (non-pilot) | Rebuild persistent + tenant-scoped; defer non-pilot BUs. |
| `saas-billing` backend | real, tested, undeployed | **Integrate** | Authoritative finance engine (ADR-3). |
| `domain/finance` | real, disconnected | **Integrate/reconcile** | Reconcile against `saas-billing`; avoid a 3rd ledger. |
| `domain/*` (healthcare/emr/etc.), `services/*` | legacy, mostly unwired | **Defer + decide (DEC-11)** | Wire deliberately or retire; remove dead-code ambiguity. |
| AI workers (`worker.py`, agents) | simulated | **Defer + rebuild under governance** | No autonomy until §19 governance. |
| SvelteKit frontend | ~88k LOC, mock-backed | **Keep shell + Repair** | Keep UI; rewire to real services; add data-state badges. |
| `SaaS billing/frontend` | separate app | **Decide (DEC-11)** | Merge useful screens or retire. |
| deploy (compose/Caddy) | no backups/monitoring/saas-billing | **Repair** | Add finance engine, backups, monitoring. |
| `docs/business-os/*` | stale (different stack) | **Replace** | Superseded by this PRD. |

---

## 3. Requirement Traceability Matrix (representative)

| Requirement | Pilot workflow | Acceptance (in `04`) | Test layer | Ticket |
| :--- | :--- | :--- | :--- | :--- |
| SEC-1 real auth | A,B preconditions | SEC-1 rows | authn + IT-PG | T1 |
| SEC-2 per-permission authz | A,B | BUS-4 row | SEC-T | T2 |
| SEC-3 tenant isolation | A,B | cross-tenant rows | SEC-T + IT-PG | T3 |
| DATA-1 no in-memory authority | A,B | restart row | IT-PG | T4 |
| SHARED-BILL/ACCT (DEC-01) | A,B | invoice→journal rows | IT-PG | T5 |
| SHARED-PAY idempotent | A,C | duplicate-callback row | IT-PG + INT | T6 |
| SHARED-LEAD/CUST | A | lead→quote rows | IT-PG | T7 |
| DOM-TRV-1 booking gates on payment | A | Scenario A | E2E | T8 |
| DOM-CON-1 milestone billing | B | Scenario B | E2E | T9 |
| INT-2 truthful integration states | A,C | UX-2 row | SEC-T + MANUAL | T10 |
Full register: `03-requirements-matrix.md`.

---

## 4. Module Dependency Matrix

Rule: BU modules depend only on shared modules (down the stack); no BU→BU direct coupling (communicate via events).

| Module | Depends on (allowed) |
| :--- | :--- |
| IAM/Org | — (foundation) |
| Customer/Lead | IAM/Org, Audit |
| Catalog/Pricing | IAM/Org |
| Quotation | Customer, Catalog/Pricing, Approval |
| Order/Booking | Customer, Catalog, Inventory, Quotation |
| Invoice | Order/Booking, Customer, Tax |
| Payment | Invoice, Integration Hub |
| Accounting | Invoice, Payment, Payroll, Procurement (postings) |
| Approval/Workflow | IAM/Org, (any domain it gates) |
| Travel (BU) | Customer, Lead, Quotation, Order/Booking, Invoice, Payment, Accounting, Task, Document, Notification, Journey |
| Consulting (BU) | Customer, Contract, Task/Project, Document, Invoice, Accounting, Approval |
| AI layer | all shared APIs (via authorized tools only) |

---

## 5. Role–Permission Matrix (core)

Permission form: `<bu|shared>.<resource>.<action>`. R=read, W=write, A=approve.

| Resource | Owner | Tenant Admin | BU Manager | Finance Mgr | Sales | Ops | HR Mgr | PM | Partner | Customer |
| :--- | :-: | :-: | :-: | :-: | :-: | :-: | :-: | :-: | :-: | :-: |
| iam.user | R | RWA | R | – | – | – | R | – | – | – |
| crm.lead | R | R | RW | – | RW | – | – | – | W(own) | – |
| sales.quotation | R | R | RWA | R | RW | – | – | – | R(own) | R(own) |
| booking/order | R | R | RWA | R | RW | RW | – | – | R(own) | RW(own) |
| finance.invoice | R | R | R | RWA | R | – | – | R | R(own) | R(own) |
| finance.payment | R | R | R | RWA | – | – | – | – | – | W(own) |
| finance.journal | R | – | – | RWA | – | – | – | – | – | – |
| hr.payroll | R | R | – | R | – | – | RWA | – | – | – |
| project.* | R | R | RW | R | – | – | – | RWA | – | – |
| ai.agent | RA | RWA | R | – | – | – | – | – | – | – |
| integration.* | R | RWA | R | R | – | – | – | – | – | – |
High-impact actions (payment, payroll disbursement, journal reversal, publish, AI autonomy) require A within a configured limit; above limit escalates to Owner.

---

## 6. Integration Status Matrix

| Provider | Purpose | Required creds | State today | Blocker |
| :--- | :--- | :--- | :--- | :--- |
| Razorpay | Payments/refunds | key/secret/webhook secret | NOT_CONFIGURED | DEC-02 |
| Stripe | Payments/refunds | key/secret/webhook | NOT_CONFIGURED | DEC-02 |
| WhatsApp Cloud | Messaging | Meta app, number, templates | NOT_CONFIGURED | DEC-02 + Meta approval |
| Email | Notifications | SMTP/API key | NOT_CONFIGURED | DEC-02 |
| SMS | Notifications | provider key | NOT_CONFIGURED | DEC-02 |
| LiveKit | Tele/voice | API key/secret | NOT_CONFIGURED | DEC-02 |
| Courier/shipping | Fulfilment | provider acct | NOT_CONFIGURED | provider choice |
| Banking feeds | Reconciliation | feed/provider | NOT_CONFIGURED (CSV only) | DEC-02 |
| Marketplaces (Etsy…) | E-com sync | API keys + authorization | NOT_CONFIGURED | DEC-02 + authorization |
| External accounting | Export | per system | NOT_CONFIGURED | optional |
No provider is CONNECTED. All must pass TEST_MODE with sandbox acceptance before CONNECTED.

---

## 7. Prioritized Backlog & First Ten Implementation Tickets

Backlog order follows the roadmap (`02` §25): Phase 0 closure → Phase 1 security/foundation → Phase 2 shared engine → Phase 3 pilots. Each ticket: dependencies, acceptance criteria, test requirements. **No ticket starts before owner approval + the relevant DEC.**

> **Pre-tickets (Phase 0 closure, read-only, no code):** P0a record drayya-core `go test` result (`05` §1.1); P0b run `saas-billing` integration/isolation tests on Postgres; P0c run frontend `npm run check` + tests (finding #9); P0d boot→create→restart→get smoke test (finding #2). These gate T-series start.

| # | Ticket | Depends on | Acceptance criteria | Test requirements |
| :-- | :--- | :--- | :--- | :--- |
| **T1** | Replace mock `iam.Login` with real authentication against persisted users (hashed pw; issue scoped claims) | DEC-04, Phase 0 closure; reuse `saas-billing/internal/auth` | Login fails on wrong/unknown credentials; no super-admin issued to arbitrary email; claims carry real tenant/BU/roles; session expires/rotates | authn unit + IT-PG (user seeded); SEC-T (no mock path) |
| **T2** | Per-permission server-side authorization on every protected procedure | T1 | Each proc checks `<resource>.<action>` + membership; unauthorized → PermissionDenied + audit row | SEC-T authz matrix; CT |
| **T3** | Tenant isolation: RLS + per-request tenant GUC + repository scoping; remove admin cross-tenant bypass | T1, DEC-04 | Tenant B cannot read/write tenant A rows (proven on Postgres); admin bypass removed; `interceptor.go:110` path gone | SEC-T + IT-PG (adapt `saas-billing/tests/TestTenantIsolation`) |
| **T4** | Persistent organization model (tenant/entity/BU/branch/dept/team) replacing hardcoded IAM responses | T1, DEC-05 | CreateTenant/BU persist + survive restart; `ListTenants`/`GetCompany` read DB | IT-PG; restart test |
| **T5** | Integrate `saas-billing` as authoritative billing/accounting; retire mock `grpcservices/billing`; invoice posts balanced double-entry journal | DEC-01, DEC-04, T3 | CreateInvoice persists + posts balanced journal in one tx; mock billing removed; PDF is real | IT-PG (debits=credits); contract; restart |
| **T6** | Payment path with provider abstraction, verification, idempotency keys, duplicate-webhook protection | T5, DEC-02 (sandbox) | Payment confirmed only after verified result; duplicate callback is a no-op; refund reverses via journal | IT-PG + INT sandbox; duplicate/concurrency |
| **T7** | Unified Customer master + Lead/pipeline (shared) | T3 | Lead persists, tenant-scoped; converts to customer + quote; cross-BU timeline excludes PHI | IT-PG; SEC-T (PHI) |
| **T8** | Travel pilot wiring to shared engine (lead→quote→booking→payment→invoice→accounting→notification) | T5,T6,T7 | Scenario A passes incl. availability change, vendor rejection, payment failure, cancellation, refund, restart, cross-tenant | E2E + IT-PG + failure-injection (`04` §3A, §4) |
| **T9** | Consulting pilot wiring (customer→contract→project→tasks→timesheet→expenses→milestone→invoice→accounting) | T5,T7 | Scenario B passes incl. same persistence/security/financial/recovery checks | E2E + IT-PG (`04` §3B) |
| **T10** | Integration hub framework + truthful states (NOT_CONFIGURED…ERROR) with health + audit; UI data-state badges | T2 | UI shows real state; no simulated response reported as real (remove `omnichannel`/`healthcare` fabrications); webhook signatures verified | SEC-T; MANUAL state review; INT sandbox |

**Sequencing:** T1→T2→T3 (security foundation) must land before T5+ (money). T4 parallels T2/T3. T5 gated by DEC-01. T8/T9 are the Phase 3 exit. T10 enables real payments in T6/T8 (sandbox first).

---

## 8. Risk Register (full)

| ID | Risk | Likelihood | Impact | Mitigation |
| :-- | :--- | :-: | :-: | :--- |
| R1 | Mock-as-real shipped (auth, payroll, reconciliation, "sent" msgs) | High | Critical | Phase 1 fixes + §28 DoD #14 + data-state badges |
| R2 | Two billing engines diverge / double cost | High | High | DEC-01 consolidation (ADR-3) |
| R3 | Ungoverned AI acts on money/comms | Med | Critical | Propose-only; §19 governance before autonomy |
| R4 | No backups/monitoring in deploy | High | High | Phase 7 + NFR-3/OPS-2 |
| R5 | Frontend type/test status unknown (#9) | Med | Med | Phase 0 P0c run |
| R6 | Legacy dead-code ambiguity | Med | Med | DEC-11 keep/wire/retire |
| R7 | Integrations blocked on creds/authorization | High | Med | DEC-02; mark unavailable honestly |
| R8 | Postgres-only tests skipped → isolation regressions | Med | High | CI requires Postgres for financial/isolation (TEST-1) |
