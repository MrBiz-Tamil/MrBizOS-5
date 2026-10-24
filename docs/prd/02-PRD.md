# MrBiz-OS — Master Product Requirements Document

**Version:** 2.0 (definitive, implementation-ready draft for owner approval)
**Date:** 2026-10-24 · **Status:** PRD only — no implementation authorized.
**Companions:** `01-current-state-inventory.md` (evidence), `03-requirements-matrix.md` (requirement register + traceability), `04-acceptance-matrix.md` (workflow/acceptance), `05-phase0-evidence.md` (build/test baseline + audit verification), `06-matrices-and-backlog.md` (module-dependency, role-permission, integration-status, ADR register, component verdicts, prioritized backlog + first 10 tickets).

**Evidence convention:** **[F]** = verified fact (code/test with `path:line`); **[O]** = repository observation; **[A]** = assumption; **[R]** = recommendation; **[D]** = decision requiring owner approval (`DEC-xx`). Nothing here claims a capability works without evidence, and no test is reported passed without execution proof (`05`).

---

## 1. Executive Summary

MrBiz-OS aims to be one multi-tenant Business Operating System running several businesses (Travel, Healthcare, CRM, HRMS, Consulting, Real Estate, E-commerce, Pharmacy, Finance, Procurement, Marketing, Support, and more) on a **shared business engine** with one authoritative data layer, consistent UI, configurable per-business rules, deterministic financial processing, real integrations, and governed AI assistance.

**[F] Verified current reality (Phase 0, `05`):** both Go backends compile cleanly on Go 1.26.0; the deployed `drayya-core` backend exposes 16 ConnectRPC services of which **13 are mock/in-memory and 3 are partial** (none fully real); application-layer authentication is bypassed (`iam.Login` issues super-admin to any email); invoices do not post accounting; payroll and reconciliation are simulated; the one mature, genuinely-tested billing/accounting engine (`saas-billing`) is **not deployed**; its real tenant-isolation and accounting tests pass logically but are **blocked pending a Postgres instance**. All 10 audit findings were checked: 7 VERIFIED, 1 partially, 1 verified+blocked, 1 (frontend type/test) not yet run.

**[R] The core recommendation is not a rebuild.** It is to **repair the foundation and integrate the asset you already have**: fix auth + tenant isolation, make the organization/data layer persistent, and adopt `saas-billing` as the single authoritative billing/accounting engine (DEC-01) instead of the mock. Then prove two end-to-end workflows (Travel, Consulting) on real Postgres before expanding. Component-level Keep/Repair/Integrate/Replace/Defer verdicts are in `06`.

---

## 2. Product Vision
One unified application and navigation; one authoritative source of business data; secure multi-tenancy; multiple business units with controlled sharing; reusable business services; consistent ConnectRPC contracts; persistent DB-backed operations; reliable double-entry accounting; **real** integrations (never simulated success); centralized workflow orchestration; human approval for sensitive actions; governed AI assistance; auditability/observability; clear error handling/recovery; automated tests for complete workflows. A user never switches between disconnected apps to run one business process.

## 3. Business Objectives
- **BO-1** Run ≥2 businesses from one platform on isolated, audited tenant data. (Pilot: Travel + Consulting.)
- **BO-2** Make the money path trustworthy end-to-end: lead→quote→booking→invoice→verified payment→posted accounting, idempotent and reconcilable.
- **BO-3** Stop presenting simulated output as real (auth, payroll, reconciliation, "sent" messages).
- **BO-4** Reuse one shared engine across businesses; configure, don't fork, per-business rules.
- **BO-5** Add AI that reduces operating effort without becoming an ungoverned shadow system.
- **BO-6** Reach a measurable production-readiness gate (§30) before go-live.

## 4. User Personas & Roles
| Persona | Needs | Representative permissions | Approvals they trigger/own |
| :--- | :--- | :--- | :--- |
| Business Owner | Cross-BU visibility, approvals. | `*.read`, approve≤limit. | Owns high-value approvals. |
| Tenant Admin | Users, roles, settings, integrations. | `iam.*`,`integration.*`. | Standing rules, credential changes. |
| BU Manager | Run one BU. | `<bu>.*`. | Above-threshold ops. |
| Finance Manager | Invoices, payments, ledger, reconciliation. | `finance.*`,`billing.*`. | Journal reversals, supplier pay. |
| Sales/CRM Employee | Leads→quotes→bookings. | `crm.*`,`<bu>.booking.write`. | Discounts above limit → manager. |
| Operations Employee | Fulfil bookings/orders, stock. | `<bu>.ops.*`,`inventory.*`. | Stock write-offs. |
| HR Manager | Employees, attendance, payroll. | `hr.*` (salary-restricted). | Payroll approve + disburse. |
| Project Manager | Projects, tasks, milestones, billing. | `project.*`. | Milestone invoice. |
| Partner/Agent | Submit leads, see commissions. | Scoped partner portal. | — |
| Customer | Self-service bookings/invoices/payments. | Own records only. | — |
| AI Supervisor | Configure/approve agents. | `ai.*`. | Enable autonomy, raise budgets. |
Full role→permission grid: `06` (role-permission matrix).

## 5. Product Scope & Non-Goals
**In scope (v1):** shared engine (§9) + Travel and Consulting pilots (§10), with Finance/Accounting, IAM, Org model, CRM/Lead, Catalog, Quotation, Order/Booking, Invoice, Payment, Approval, Task, Document, Notification, Audit, and platform Subscription as the shared substrate.
**Phased-in (v1.1+):** Healthcare, HRMS/Payroll, E-commerce, Real Estate, Procurement, Omnichannel marketing, Workflow designer, AI employees, BPO voice.
**Non-goals (v1):** regulated escrow money-handling; licensed investment/financial advice; autonomous clinical decisions; any integration lacking credentials + provider authorization; a second billing engine (explicitly prohibited — consolidate per DEC-01); microservices split (ADR-1).

## 6. Current-State Audit Findings
Full verification table with evidence + method is `05-phase0-evidence.md` §2; inventory is `01`. Summary: **[F]** 13/16 services mock, 3 partial; auth bypassed (`iam/service.go:22`); no invoice→ledger posting; payroll/reconciliation simulated; `saas-billing` real but undeployed; integrations present-but-not-connected; AI simulated; drayya-core test result pending (`05` §1.1); frontend type/test (finding #9) not yet run. Doc-vs-code contradictions (`business-os/*` describes a different Next.js/Fiber stack) are recorded in `01` §8.

## 7. Target Operating Model
**One modular monolith** (`drayya-core`) exposes ConnectRPC as the sole app API to a SvelteKit frontend/BFF. A **shared business engine** owns cross-cutting domains; **BU modules** add only business-specific entities/rules and reuse the engine. `saas-billing` is folded in as the authoritative finance module (ADR-3). PostgreSQL (with RLS) is the single source of truth; Redis handles async/events; an **integration hub** mediates every external provider with explicit states; a **governed AI layer** calls the same authorized APIs as humans, in propose-then-approve mode. Legacy `domain/*`/`services/*` are deliberately wired or retired (DEC-11) to remove dead-code ambiguity.

## 8. Multi-Tenant Organization Model
Hierarchy (not interchangeable): **Platform → Tenant (hard isolation boundary) → Business Group → Legal Entity (owns books + numbering + tax identity) → Business Unit (tags operational records) → Branch → Department → Team → Employee**; plus shared masters **Customer/Supplier/Partner** and **Role/Permission**; plus **Subscription/Plan** at platform level.
Rules: every business row carries `tenant_id` (NOT NULL); entity-scoped rows add `legal_entity_id`; BU-scoped rows add `business_unit_id`. Customers/suppliers are shared at tenant (or entity — DEC-06) scope by permission; **clinical/PHI is never shared into cross-BU views**. Consolidated reporting rolls up within a tenant with permission. **[F] The server must never trust client-supplied tenant/BU/role/admin values** — resolve from the signed session and verify membership per call (interceptor does tenant/BU identity today but not per-permission authz, and admin bypasses cross-tenant — must fix: SEC-2/SEC-4). Data model detail: §14 + `03` (DATA-*). Decisions: DEC-05 (entity first-class in v1?), DEC-06 (sharing scope).

## 9. Shared Business Engine
22 shared domains, each specified in `03` with purpose/users/entities/ownership/rules/APIs/permissions/persistence/events/integration/failure/acceptance/dependencies. Summary + current status + build verdict (full verdicts in `06`):

| Domain group | Domains | Current status | Verdict |
| :--- | :--- | :--- | :--- |
| Org & Identity | tenant, group, legal-entity, BU, branch, dept/team, user, role/permission, auth/session, subscription | IAM MOCK; subscription REAL (saas-billing) | **Repair** IAM (reuse `saas-billing/internal/auth`); **Integrate** subscription |
| Customer & Sales | customer+contact, lead, opportunity, pipeline, activity, quotation, product/service, pricing, order/booking, comms | lead MISSING; quotation/pricing REAL (saas-billing); customer PARTIAL | **Build** lead/pipeline; **Integrate** quotation/pricing/catalog |
| Operational | task, project, inventory, procurement, supplier, PO, GRN, asset/doc, approval, workflow | task/project/approval PARTIAL/MOCK; supplier/purchase REAL (saas-billing); inventory dup | **Build** approval/workflow; **Integrate** supplier/purchase/inventory; **Repair** task/project |
| Financial | billing/invoice, AR, AP, payment/refund, tax, GL, COA, double-entry, bank-rec, reporting, BU profitability, consolidation | REAL in saas-billing + domain/finance; MOCK in drayya-core; disconnected | **Integrate** `saas-billing` as authoritative (DEC-01); **Replace** mock billing; reconcile `domain/finance` |
| Workforce | employee, attendance/leave, payroll, timesheet, assignment, expense, performance, approvals | MOCK | **Build** (payroll 4-state) |
| Integration & Intelligence | external API, webhooks, events, scheduled jobs, notifications, reporting/analytics, AI orchestration, audit, monitoring | framework MISSING; AI simulated | **Build** integration hub (§12) + AI governance (§10) |

Rule: identify duplicated responsibilities and pick **one** authoritative implementation (billing/accounting is the key duplicate — DEC-01). Do not merge unrelated logic just to cut module count.

## 10. Business-Unit Requirements
MVP vs later-phase split per BU; detail in `03` (DOM-*). Pilots first (§15).
- **Travel & Tours (MVP pilot):** CRM/lead (+social capture later), package/itinerary catalog, pricing/quotation, booking/reservation with availability, vendor/supplier + hotels/transport, visa processing + document mgmt, payment/refund + commission/vendor settlement, travel accounting, customer/traveler portal, notifications, profitability. Journey: Enquiry→Lead→Assignment→Qualification→Package→Quotation→Approval→Booking→Vendor Confirmation→Payment Verification→Invoice→Accounting→Operations→Completion→Follow-up. Edge cases (availability change, vendor rejection, payment failure, cancellation, refund) specified in `04`.
- **IT Project Mgmt & Consulting (MVP pilot):** customer/contract(SOW), project+templates, tasks/subtasks, Kanban, milestones, assignment, timesheets, utilization, expenses/budgets, deliverables, client approvals, progress reports, milestone billing→invoice→accounting, profitability.
- **Healthcare (later):** patient, appointments, clinical records, prescriptions, pharmacy, consultation, billing, telehealth — strict clinical RBAC + PHI privacy; **AI barred from autonomous high-risk clinical actions**.
- **HRMS & Payroll (later):** employees, attendance, leave, payroll calc, statutory config, approvals, payslips, payment submission, payroll accounting — **distinguish calculated / approved / submitted / confirmed-paid**.
- **Real Estate (later):** properties, units, owners, listings, enquiries, site visits, agreements, bookings, commissions, payments, documents, reporting; escrow only if a licensed provider is connected.
- **E-commerce & Retail (later):** products, inventory, orders, checkout, payments, POS, shipping, returns/refunds, marketplace sync.
- **CRM & Omnichannel (later):** pipelines, campaigns, segmentation, WhatsApp/IG/FB/email/SMS with consent, templates, delivery tracking, conversion reporting — only integrations the providers' real APIs/permissions allow.
- **Finance & Accounting (MVP, shared):** invoices, receipts, supplier bills, credit notes, refunds, payments, journals, GL, bank reconciliation, taxes, statements — every workflow names its authoritative records + reconciliation rules.
- **Workflow Automation (later):** triggers/conditions/branching/approvals/retries/timeouts/idempotency/history/versioning/recovery.
- **AI Employees (later):** NL requests, planning, tool use, delegation, persistent state, approval policies, budgets, cancellation, audit, result verification (§10 governance, §11 assistant).

## 11. Domain Ownership & Boundaries
Each shared domain has exactly one owning module and one authoritative table set; BU modules reference shared entities by ID and never duplicate them. Finance owns all monetary postings; no BU writes ledgers directly. Customer master is tenant-owned; BU interactions attach to it. The module-dependency matrix (`06`) enumerates allowed dependencies (BU→shared only; no BU→BU coupling except via events).

## 12. Architecture Decision Records
Full register in `06`. Key ADRs: **ADR-1** keep modular monolith (no microservices); **ADR-2** ConnectRPC as sole app API (retire ad-hoc BFF stubs); **ADR-3 adopt `saas-billing` as authoritative billing/accounting (DEC-01)**; **ADR-4** one schema mechanism — versioned SQL migrations over GORM AutoMigrate (DEC-04); **ADR-5** tenant isolation via RLS + tenant GUC + repository scoping; **ADR-6** explicit constructor DI; **ADR-7** Redis Streams for async only, money/stock flows stay synchronous+transactional; **ADR-8** AI calls approved APIs only, never writes business tables. Each carries current-state/problem/options/decision/rationale/trade-offs/migration/acceptance in `06`.

## 13. System Architecture
SvelteKit (frontend + BFF, session cookie) → ConnectRPC `drayya-core` (H2C) with an auth interceptor (tenant/BU + per-permission authz) → shared engine + BU modules → PostgreSQL(RLS) + Redis + object storage. Integration hub mediates external providers. Python AI workers call ConnectRPC APIs in propose-then-approve mode. Deployment via docker-compose + Caddy, extended to include the consolidated finance engine, backups, and monitoring. Diagram: `01`/earlier §16; interfaces, DI, error handling, migrations, background processing, event delivery, adapters, logging/tracing, secrets, backup/restore, deploy/rollback are specified as requirements in `03` (OPS-*, NFR-*).

## 14. Data Model
Cross-cutting rules (every business entity): UUID PK; `tenant_id` NOT NULL (+ `legal_entity_id`/`business_unit_id` where scoped); FK + unique constraints (e.g., invoice number per sequence per entity — see `saas-billing/internal/invoice/sequence.go`); lifecycle/state column; `created_at`/`updated_at`; audit (`created_by`/`updated_by`); soft-delete default (hard-delete prohibited for financial/clinical); optimistic-concurrency version where contended; indexes on `(tenant_id, …)`. **[F] DATA-1 (hard): no in-memory map may be the authoritative store** — directly fixes finding #2. Ledger integrity enforced by DB constraint (`backend/migrations/014_ledger_balance_constraint.sql`). Retention/residency per DEC-08. Entity list + ownership + sharing + consolidation detail in `03` (DATA-*) and §8.

## 15. API & Protobuf Requirements
ConnectRPC is authoritative (`proto/*`, 21 packages). Requirements: contracts versioned (`v1`); generated Go handlers + the SvelteKit client (`frontend/src/lib/grpc.ts`) kept in sync by a CI buf-generate + diff check (API-2); every protected procedure enforces authn + per-permission authz server-side (SEC-2); errors use typed Connect codes (already partly done, `grpc.ts:13`); no procedure returns fabricated success (finding #7 — e.g. remove `omnichannel` fake "sent" id, `healthcare` fake tokens). Proto changes follow a compatibility policy (no breaking field renumber).

## 16. Business Workflow Specifications
The two pilot workflows (Travel, Consulting) are fully specified with stages, state transitions, edge cases, and acceptance in `04-acceptance-matrix.md` §3 (Scenarios A, B) and tested for success + invalid input + unauthorized + duplicate + concurrency + DB failure + provider timeout + rejected approval + retry + restart + cross-tenant. Each stage names the authoritative record it writes and the verification that gates the next stage (e.g., booking is confirmed only after payment is verified, not on API 200).

## 17. Accounting & Transaction Requirements
**[F] Financial correctness is non-negotiable.** Transaction-safe (single DB tx + constraints + idempotency) for: invoice creation, payment confirmation, refunds, booking/inventory reservation, PO receipt, supplier invoice, payroll posting, journal entries, bank reconciliation. Require: atomic state changes; unique transaction IDs; idempotency keys (present in `saas-billing/internal/api/idempotency.go`); duplicate-webhook protection; double-entry (debits=credits constraint); reconciliation; reversal/correction via reversing entries (no edits/hard-deletes); audit trail; explicit status transitions. **A 200 response must not imply money received / settled / posted / disbursed** — each outcome is verified against the provider/ledger before the UI claims it (fixes findings #5, #6). Authoritative records: invoices, payments, journals owned by the consolidated finance engine (DEC-01).

## 18. Integration Catalog
Uniform framework per provider (§12 requirements): config, credentials (encrypted, rotatable), connection test, capability/permission checks, webhooks + signature verification, rate-limit handling, retries w/ backoff, idempotency, error classification, observability, test & production modes, health, disconnection behavior, audit. Explicit states: `NOT_CONFIGURED → DISCONNECTED → TEST_MODE → CONNECTED → DEGRADED → ERROR` — the UI shows the true state and never reports a simulated response as a real transaction. Provider status table (Razorpay, Stripe, WhatsApp Cloud, Email, SMS, LiveKit, courier, banking feeds, marketplaces, external accounting) in `06` (integration-status matrix); all currently NOT_CONFIGURED/BLOCKED pending DEC-02 credentials + authorization.

## 19. AI Agent Governance
One central orchestrator + a small set of specialized agents (Business Manager, Sales/CRM, Travel Ops, Finance Assistant, Project, Workflow Builder, Integration Monitor). **[R] Start with the minimum set and propose-only autonomy; justify each additional agent (rule 14).** Every agent carries: explicit responsibilities; tenant+BU scope; authorized-tool allow-list (tools = approved business APIs only); autonomy level; budget/usage limits; approval requirements; execution timeout; retry policy; persistent task status; audit records; cancellation/revocation. **Agents use the same authorized services as humans and never bypass permissions or write authoritative tables directly** (ADR-8). High-impact classes (payments, payroll, clinical, external comms, publishing) always require human approval. Current state: SIMULATED (`05` §2 finding #8) — governance is MISSING and must precede any autonomy.

## 20. Natural-Language Business Assistant
For each request ("today's sales across businesses", "assign new travel leads", "overdue invoices + draft reminders", "weekly project report", "low-stock → draft POs", "pending approvals", "explain profit change", "prepare campaign for review", "investigate failed integrations + retry eligible") the pipeline is: intent → user context → BU resolution → data retrieval (permission-scoped) → permission validation → plan → risk classification → approval determination → execution via authorized APIs → outcome verification → audit → user-facing result. The assistant distinguishes **propose / prepare / execute / verify** and **never fabricates records or reports simulated results as real** (fixes the pattern behind findings #6/#7/#8). Depends on §19 governance + real backing services; status MISSING.

## 21. Security & Privacy
Threat model (each with a testable mitigation + acceptance in `03` SEC-*): authentication bypass (finding #3), cross-tenant leakage, BU privilege escalation, broken access control, exposed credentials, session compromise, unsafe file uploads, malicious webhooks, SQL injection, unauthorized AI tool execution, duplicate payment callbacks, audit-log tampering, sensitive data in logs/AI context. Controls: real auth (Argon2id/bcrypt — code in `saas-billing/internal/auth`), short-lived access + rotating refresh + revocation, optional MFA/TOTP (present), per-permission server-side authz (MISSING — build), RLS tenant isolation, remove admin cross-tenant bypass (`interceptor.go:110`), signed short-lived file URLs, webhook signature + idempotency, secrets via env/secret-manager, stricter controls for healthcare/payroll/financial data. **[O] Remove the committed `backend/cmd/temp_init_wallet/medihub.db` and confirm it holds no real data.** Compliance tied to actual jurisdiction — **[A]** India/GST (GSTIN/HSN/CGST-SGST-IGST evidence); confirm DEC-09. No secrets/PHI in docs.

## 22. Reporting & Analytics
Every metric declares calculation, source of truth, time zone, refresh policy, permission model, and a data-state (real/test/stale/unavailable). Dashboards: revenue/expenses, outstanding invoices, payments/settlements, leads/conversions, bookings/fulfilment, inventory, employee/project performance, workflow success/failure, integration health (§18 states), AI task execution, approval queues, API latency/errors, job backlog, DB health, security events. Current dashboards read mock data → **not acceptable as reporting** until on real data.

## 23. Non-Functional Requirements
Baselines (approve via DEC-10; `03` NFR-*): availability 99.5%→99.9%; API p95 <400ms read / <800ms write excl. external, measured under load (`saas-billing` has a loadtest harness); concurrency baseline 200 users/tenant **[A]**; zero tolerance for lost financial records (verified by restart + reconciliation tests); automated backup + tested restore (**MISSING today**); audit retention ≥7y for financial (DEC-09); structured logs/metrics/tracing/health; reproducible deploy + rollback with migration safety.

## 24. Testing & Acceptance Criteria
Layered strategy (`04`): unit → domain service → **PostgreSQL integration** → API contract → authn/authz → tenant-isolation → transaction integrity → workflow → external integration (sandbox) → frontend → browser E2E → failure/recovery → AI tool authorization → performance → backup/restore. **[F] SQLite is for fast local feedback only — not a substitute for Postgres on financial/isolation behavior** (the real `saas-billing` isolation tests already require Postgres, `05` §1.2). Every acceptance test carries: Test ID, Requirement ID, preconditions, data, actions, expected result, DB verification, permission assertions, failure behavior, evidence, pass/fail.

## 25. Implementation Roadmap
Phases with dependencies/deliverables/risks/tests/exit-criteria/blockers (expanded in `06` backlog):
- **Phase 0 — Current-state verification:** reproduce findings; build/test baselines; classify persistent/mock/disconnected/missing; record risks. **Exit:** `05` complete (drayya-core test recorded, Postgres isolation run done, frontend check run). *~80% done.*
- **Phase 1 — Security & foundation:** real auth (replace `iam.Login`); per-permission authz; remove admin cross-tenant bypass; persistent org model; chosen schema mechanism; RLS + tenant GUC; audit trail; Postgres test infra. **Exit:** SEC + isolation acceptance tests pass; no mock identity path; restart-safe.
- **Phase 2 — Shared engine:** resolve DEC-01 + integrate authoritative billing/accounting; customer + unified lead; catalog; quotation; orders/bookings; invoice; payment; accounting posting; approvals — persistent, transactional, idempotent, audited. **Exit:** balanced double-entry; idempotent payment; Scenario A/B finance legs pass on Postgres.
- **Phase 3 — Pilots:** complete Travel + Consulting; DB-integrity + financial + failure/recovery tests. **Exit:** Scenarios A & B pass incl. all edge cases.
- **Phase 4 — Additional modules:** Healthcare, HRMS, E-commerce, Real Estate, Procurement by value/risk; each meets DoD (§28).
- **Phase 5 — Integrations & automation:** hub + chosen providers TEST→CONNECTED; webhooks/retry/recovery; workflow execution; monitoring.
- **Phase 6 — AI employees:** governance; assistant; authorized tools; approval workflows; persistent state; budgets; audit.
- **Phase 7 — Production readiness:** security review; performance; backup/restore; monitoring; ops docs; deploy/rollback validation; add consolidated finance engine to deploy. **Exit:** §30 Go/No-Go green.

## 26. Risk Register
Top risks (full register in `06`): R1 shipping mock-as-real erodes trust + data integrity (active); R2 two billing systems double cost + diverge (DEC-01); R3 ungoverned AI acting on money/comms (defer autonomy); R4 no backups/monitoring in deploy; R5 frontend type/test status unknown (finding #9 not run); R6 legacy `domain/*`/`services/*` dead-code ambiguity (DEC-11); R7 integrations blocked on credentials/authorization (DEC-02).

## 27. Open Business Decisions
`DEC-01` adopt `saas-billing` as authoritative billing/accounting (recommended) vs rebuild. `DEC-02` which providers in v1 + who supplies credentials/authorization. `DEC-03` which BUs truly in v1 (recommend Travel + Consulting). `DEC-04` schema mechanism (SQL migrations recommended). `DEC-05` legal entity first-class in v1? `DEC-06` customer/supplier sharing scope. `DEC-07` MFA mandatory for which roles. `DEC-08` retention/residency per data class. `DEC-09` jurisdictions for statutory/tax commitments (India/GST assumed). `DEC-10` NFR targets sign-off. `DEC-11` disposition of legacy `domain/*`+`services/*` (keep/wire/retire).

## 28. Definition of Done (per capability)
Done only when all hold: (1) UI↔intended backend connected; (2) authn + per-permission authz enforced; (3) business rules implemented; (4) data persisted in authoritative DB; (5) DB constraints + tx boundaries correct; (6) integration implemented **or** explicitly marked unavailable (§18 state); (7) errors returned + displayed; (8) idempotent duplicate handling; (9) audit events recorded; (10) unit + integration tests pass; (11) relevant E2E passes; (12) restart/recovery verified (no in-memory authority); (13) docs match implementation; (14) no simulated success presented as real.

## 29. Production-Readiness Checklist
Real auth, no mock identity path · per-permission authz on every protected proc · verified cross-tenant isolation (RLS + tests on Postgres) · single authoritative billing/accounting, balanced double-entry, idempotent payments · no in-memory authoritative stores · pilots A & B pass incl. failure/duplicate/concurrency/restart/cross-tenant · integration states truthful · backups + tested restore · monitoring + audit · no secrets/PHI in VCS · NFR baselines met under load · frontend type-check + tests green.

## 30. Go/No-Go Criteria
**No-Go** unless every item in §29 is true with evidence (`04` evidence standard). The decisive gates: (a) no path issues a session without a verified user; (b) cross-tenant access is impossible and proven by tests on Postgres; (c) the money path is double-entry, idempotent, reconcilable, and restart-safe; (d) pilots A & B pass end-to-end; (e) nothing simulated is presented as real.

---

## Final recommendation (Keep / Repair / Integrate / Replace / Defer)
Component-by-component verdicts with evidence are in `06-matrices-and-backlog.md`. Headline:
- **Keep:** Go modular monolith, ConnectRPC contracts, SvelteKit frontend shell, fail-closed interceptor, PostgreSQL/Redis/Caddy topology.
- **Repair:** IAM (real auth + authz), tenant isolation (RLS + remove admin bypass), the 3 partial services (tenant-scope them), schema mechanism.
- **Integrate:** `saas-billing` (billing/accounting/quotation/tax/trade/subscription) as the authoritative finance engine; reconcile `domain/finance`.
- **Replace:** the 13 mock drayya-core services with real, persistent, tenant-scoped implementations (starting with the pilot-critical ones).
- **Defer:** Healthcare/HRMS/E-com/RealEstate/Procurement BUs, external integrations (until DEC-02), AI autonomy (until §19 governance), BPO voice.

**First ten implementation tickets** (dependencies, acceptance, tests) are in `06` §7. **Deliver stops here — awaiting owner approval and DEC-01..DEC-11 before any implementation.**
