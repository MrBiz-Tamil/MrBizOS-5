# MrBiz-OS — Requirements Register

**Companion to** `02-PRD.md`. Every requirement has a unique ID, priority (P0–P3), status (from `01-current-state-inventory.md`), acceptance criteria, and verification method. Full per-requirement fields (actors, preconditions, main/alt/failure flows, permissions, data, dependencies, evidence) are summarized in-row; the long-form fields live with each requirement at implementation time. Statuses reflect discovery and must be re-confirmed at Phase 0 exit.

**Categories:** BUS (business), SHARED (platform capability), DOM (business-domain), SEC (security), DATA (data/persistence), API (contract), INT (integration), AI (agents), UX, NFR, TEST, OPS.
**Verification methods:** UT (unit), IT-PG (Postgres integration), CT (contract), E2E (browser), SEC-T (security test), LOAD, MANUAL, REVIEW.

---

## Business (BUS)

| ID | Name | Pri | Description / Acceptance criteria | Verify | Status |
| :-- | :--- | :-: | :--- | :-- | :--- |
| BUS-1 | Multi-business single tenant | P0 | One owner operates ≥2 BUs under one tenant; data isolated per tenant, tagged per BU; cross-tenant access impossible. AC: isolation tests pass; BU rollup reports correct. | IT-PG, SEC-T | MOCK |
| BUS-2 | Verified lead→cash chain | P0 | Lead→quote→booking/order→invoice→payment(verified)→accounting, each step verifiable; no step claims unverifiable success. AC: Scenario A & B pass. | E2E, IT-PG | MOCK |
| BUS-3 | Reliable financial records | P0 | Persistent, double-entry, idempotent, reconcilable; survive restart. AC: restart + reconciliation + duplicate-callback tests pass. | IT-PG | DISCONNECTED |
| BUS-4 | Server-side authz + audit | P0 | Every sensitive action permission-checked server-side and audit-logged. AC: authz matrix tests pass; audit rows present. | SEC-T | MISSING (handler authz) |
| BUS-5 | Propose vs execute for AI | P0 | High-impact actions require human approval + verified execution. AC: AI cannot complete a high-impact action without approval. | SEC-T, E2E | MISSING |
| BUS-6 | Consolidated cross-BU reporting | P1 | Owner sees consolidated + per-BU financials with permission. | IT-PG | MOCK |
| BUS-7 | Partner/customer self-service | P2 | Scoped portals for partners and customers. | E2E | PARTIAL |

---

## Shared platform (SHARED)

| ID | Capability | Pri | Acceptance criteria | Verify | Status |
| :-- | :--- | :-: | :--- | :-- | :--- |
| SHARED-IAM | Identity & access | P0 | Persisted users; hashed pw; scoped claims; MFA option; sessions expire/rotate. | IT-PG, SEC-T | MOCK |
| SHARED-ORG | Org/tenant/entity/BU/branch | P0 | Persisted hierarchy; settings, numbering, tax config per entity. | IT-PG | MOCK |
| SHARED-CUST | Customer master + 360 | P0 | Shared customer; cross-BU timeline excluding PHI. | IT-PG, SEC-T | PARTIAL |
| SHARED-LEAD | Unified lead/opportunity | P0 | One lead entity; pipeline stages across BUs. | IT-PG | MISSING |
| SHARED-CAT | Catalog (products/services) | P1 | Single catalog; variants; BU applicability. | IT-PG | PARTIAL/dup |
| SHARED-QUOTE | Pricing + quotation | P1 | Quote lifecycle + approval; pricing engine. | IT-PG | REAL (saas-billing)/MOCK (drayya) |
| SHARED-ORDER | Orders & bookings | P0 | Generic order/booking; reservation semantics. | IT-PG | PARTIAL |
| SHARED-INV | Inventory & stock | P1 | Stock, movements, reservations, adjustments transactional. | IT-PG | PARTIAL/dup |
| SHARED-PROC | Supplier & procurement | P1 | Supplier, PO, GRN, 3-way match. | IT-PG | MOCK/dup |
| SHARED-WF | Workflow & approval | P0 | Approval queue with limits; deterministic transitions. | IT-PG | PARTIAL |
| SHARED-BILL | Billing & invoicing | P0 | Persistent invoices; GST; sequences; real PDF. | IT-PG | REAL (saas-billing)/MOCK (drayya) |
| SHARED-PAY | Payments & refunds | P0 | Provider + verification + idempotency + refunds. | IT-PG, INT | BLOCKED |
| SHARED-ACCT | Accounting & reporting | P0 | Double-entry GL, AR/AP, P&L, BS, cost centers. | IT-PG | DISCONNECTED |
| SHARED-HR | Workforce & payroll | P1 | Employees; attendance; payroll state machine (calc≠approved≠submitted≠disbursed). | IT-PG | UNVERIFIED |
| SHARED-TASK | Task & project | P1 | Tasks/assignment/projects/milestones/timesheets persisted. | IT-PG | PARTIAL |
| SHARED-DOC | Document & file | P1 | Storage metadata; signed URLs; versioning. | IT-PG, SEC-T | UNVERIFIED |
| SHARED-NOTIF | Notifications & comms | P1 | Email/SMS/WA templates; delivery tracking. | INT | BLOCKED |
| SHARED-RPT | Reporting & analytics | P1 | Metrics with defined source/zone/refresh/permission. | IT-PG | MOCK |
| SHARED-AUDIT | Audit & observability | P0 | Audit trail; metrics; tracing; health. | REVIEW, IT-PG | PARTIAL |
| SHARED-AI | AI orchestration | P1 | Governed execution (see AI-*). | SEC-T | SIMULATED |
| SHARED-SUB | Subscription/platform billing | P1 | Plans; metering; proration. | IT-PG | REAL (saas-billing) |

---

## Business domains (DOM)

| ID | BU | Pri | Acceptance criteria (summary) | Status |
| :-- | :--- | :-: | :--- | :--- |
| DOM-TRV-1 | Travel | P0 | Package/itinerary catalog persisted; lead→quote→booking reserves capacity; payment verified before confirmation; invoice + accounting posted; customer notified. | PARTIAL |
| DOM-TRV-2 | Travel | P1 | Visa application + document checklist + status tracker; VFS integration marked BLOCKED until authorized. | PARTIAL/BLOCKED |
| DOM-TRV-3 | Travel | P2 | Supplier/vendor commission settlement posts to accounting. | MOCK |
| DOM-HC-1 | Healthcare | P0 | Patient reg, appointments, clinical records with PHI isolation; AI barred from autonomous clinical decisions. | UNVERIFIED |
| DOM-HC-2 | Healthcare | P1 | Pharmacy/lab/teleconsult; external health-system integration BLOCKED until authorized. | UNVERIFIED/BLOCKED |
| DOM-HR-1 | HR/Payroll | P1 | Payroll 4-state machine; payslips; journal posting; disbursement status verified. | UNVERIFIED |
| DOM-CON-1 | Consulting | P0 | SOW/contract→project→milestones→tasks→timesheets→milestone invoice→accounting→profitability. | PARTIAL |
| DOM-RE-1 | Real Estate | P1 | Property/unit inventory, leads, site visits, offers, deposits, contracts, commission; escrow only if licensed provider connected. | PARTIAL |
| DOM-EC-1 | E-commerce | P1 | Catalog→order→payment(verified)→fulfilment→shipment→return/refund; marketplace/POD/shipping BLOCKED until creds. | PARTIAL/BLOCKED |
| DOM-CRM-1 | CRM/Omnichannel | P1 | Leads/campaigns/segmentation; WA/IG/FB/Email/SMS with consent/opt-out, templates, delivery tracking, attribution. | MOCK/BLOCKED |

---

## Security (SEC)

| ID | Name | Pri | Acceptance criteria | Verify | Status |
| :-- | :--- | :-: | :--- | :-- | :--- |
| SEC-1 | Real authentication | P0 | No login without valid persisted user + correct hashed pw; mock Login removed. | SEC-T, IT-PG | MOCK (F-1) |
| SEC-2 | Per-permission authz | P0 | Every protected procedure checks `<bu>.<resource>.<action>` + membership. | SEC-T | MISSING |
| SEC-3 | Tenant isolation | P0 | RLS + tenant GUC + repo scoping; cross-tenant read/write impossible. | SEC-T, IT-PG | MISSING (F-4) |
| SEC-4 | No admin cross-tenant bypass | P0 | Replace blanket admin bypass with audited platform-admin scope. | SEC-T | GAP (F-7) |
| SEC-5 | Session security | P0 | Short-lived access; rotating refresh; revocation; HttpOnly cookie. | SEC-T | PARTIAL |
| SEC-6 | MFA | P1 | TOTP available; mandatory per DEC-07 roles. | SEC-T | CODE EXISTS (saas-billing) |
| SEC-7 | Webhook signature + idempotency | P0 | All financial callbacks verified + deduped. | SEC-T, IT-PG | MISSING |
| SEC-8 | Secrets hygiene | P0 | No creds/PHI/DB files in VCS (remove `cmd/temp_init_wallet/medihub.db`). | REVIEW | GAP |
| SEC-9 | File upload safety | P1 | Type/size validation; private buckets; signed URLs. | SEC-T | UNVERIFIED |
| SEC-10 | Audit-log integrity | P1 | Append-only; sensitive fields stripped; tamper-evident. | SEC-T | PARTIAL |
| SEC-11 | AI tool authorization | P0 | Agents limited to allow-listed APIs + scope. | SEC-T | MISSING |
| SEC-12 | Injection/IDOR | P0 | Parameterized queries; object ownership checks. | SEC-T | UNVERIFIED |

---

## Data (DATA)

| ID | Name | Pri | Acceptance criteria | Status |
| :-- | :--- | :-: | :--- | :--- |
| DATA-1 | No in-memory authority | P0 | No production business record stored only in memory. | VIOLATED (F-2) |
| DATA-2 | Tenant columns + constraints | P0 | `tenant_id NOT NULL` + FKs + unique constraints on every business entity. | PARTIAL |
| DATA-3 | Transaction boundaries | P0 | Booking/invoice/payment/GRN/payroll/journal/approval each single-tx + constraint-enforced. | MISSING |
| DATA-4 | Idempotency keys | P0 | Payment callbacks, webhooks, order create, jobs carry idempotency keys. | REAL (saas-billing)/MISSING (drayya) |
| DATA-5 | Append-only financials | P0 | Corrections via reversing entries; no edits/hard-deletes. | DISCONNECTED |
| DATA-6 | Retention/soft-delete | P1 | Soft-delete default; retention per DEC-08. | UNVERIFIED |
| DATA-7 | Single schema mechanism | P0 | One of SQL-migrations (recommended) or AutoMigrate; DEC-04. | AMBIGUOUS |

---

## API / Integration / AI / UX / NFR / OPS / TEST

| ID | Name | Pri | Acceptance criteria | Status |
| :-- | :--- | :-: | :--- | :--- |
| API-1 | ConnectRPC sole app API | P0 | All app traffic via proto contracts; ad-hoc BFF stubs retired. | PARTIAL |
| API-2 | Contract↔client sync | P1 | Generated clients match proto; CI check. | PARTIAL |
| INT-1 | Integration hub framework | P0 | Uniform config/creds/test/webhook/retry/idempotency/state/health per §9.1. | MISSING |
| INT-2 | Truthful integration states | P0 | UI shows real §9.2 state; success only after verified result. | MISSING |
| INT-3 | Provider onboarding (per DEC-02) | P1 | Each in-scope provider TEST→CONNECTED with sandbox tests. | BLOCKED |
| AI-1 | Agent governance envelope | P0 | Role/scope/tool-allow-list/autonomy/budget/approval/timeout/audit/stop per agent. | MISSING |
| AI-2 | Propose-only default | P0 | No autonomous high-impact action in v1. | MISSING |
| AI-3 | Result verification | P0 | Agent never claims unverified success. | MISSING |
| AI-4 | NL assistant pipeline | P1 | 11-step pipeline (§18) with permission scoping + audit. | MISSING |
| UX-1 | Permission-aware UI | P0 | Actions hidden/disabled by permission. | PARTIAL |
| UX-2 | Honest data-state badges | P0 | Real/test/stale/unavailable labelled. | MISSING |
| UX-3 | Loading/error/empty states | P1 | Every screen handles all states. | PARTIAL |
| UX-4 | Responsive + accessible | P2 | Mobile widths; WCAG AA for core flows. | UNVERIFIED |
| NFR-1 | Availability (DEC-10) | P1 | 99.5%→99.9%. | UNKNOWN |
| NFR-2 | Latency | P1 | p95 <400ms read/<800ms write excl. external, under load. | UNKNOWN |
| NFR-3 | Backup/restore | P0 | Automated backup + tested restore. | MISSING |
| NFR-4 | Observability | P1 | Structured logs/metrics/tracing/health. | PARTIAL |
| OPS-1 | Reproducible deploy/rollback | P1 | One-command; migration-safe; incl. consolidated finance engine. | PARTIAL |
| OPS-2 | Monitoring/alerting | P1 | Integration health, job backlog, DB health, security events. | MISSING |
| TEST-1 | PG integration for critical paths | P0 | Financial + isolation tested on Postgres, not SQLite. | PARTIAL |
| TEST-2 | Layered suite | P1 | §14 layers present with CI gates. | PARTIAL |
| TEST-3 | Failure-injection/recovery | P1 | Timeout/restart/duplicate/concurrency covered. | MISSING |

---

## Priority rollup
- **P0 (must fix/build before release):** BUS-1..5, SHARED-IAM/ORG/CUST/LEAD/ORDER/WF/BILL/PAY/ACCT/AUDIT, DOM-TRV-1/CON-1/HC-1, all SEC except SEC-6/9/10, DATA-1/2/3/4/5/7, API-1, INT-1/2, AI-1/2/3, UX-1/2, NFR-3, TEST-1.
- The P0 set is deliberately dominated by **security, data integrity, and verified money flow** — not features.
