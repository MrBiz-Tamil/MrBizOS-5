# MrBiz-OS — Acceptance & End-to-End Test Matrix

**Companion to** `02-PRD.md` / `03-requirements-matrix.md`. Acceptance rows are the phase gates: a capability advances only when its row passes with the required evidence. All financial and tenant-isolation rows run against **PostgreSQL** (SQLite is for fast local feedback only).

## 1. Acceptance row template
Each row carries: Requirement ID · Scenario · Preconditions · Test data · Action · Expected result · **DB verification** · **Security assertion** · Failure handling · Evidence required · Pass/Fail.

## 2. Representative acceptance rows

| Req | Scenario | Preconditions | Action | Expected result | DB verification | Security assertion | Evidence |
| :-- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| SEC-1 | Login rejects bad credentials | User seeded, hashed pw | POST Login wrong pw | Unauthenticated; no token | No session row created | No claims issued | Test log + DB query |
| SEC-1 | Login no longer mocks | No matching user | POST Login any email | Unauthenticated | — | No super-admin issued | Replaces `iam/service.go:22` behavior |
| SEC-3 | Cross-tenant read blocked | Tenants A,B; invoice in A | B's token lists/gets A's invoice | NotFound/PermissionDenied | RLS prevents row visibility | No A rows in B response | SEC-T output |
| SEC-7 | Duplicate payment callback | Invoice unpaid | Same webhook twice (same idempotency key) | One payment; second is no-op | One payment row; balance correct | Signature verified | IT-PG + log |
| DATA-1 | Restart survives | Invoice created | Restart backend; GetInvoice | Invoice still present | Row persisted in Postgres | — | Before/after restart |
| DATA-3 | Invoice posts balanced journal | COA seeded | CreateInvoice | Invoice + balanced journal in one tx | debits=credits (constraint `014`) | Authorized role only | IT-PG |
| SHARED-PAY | Payment verified before confirm | Order pending | Pay via provider TEST mode | Order confirmed only after provider success | Payment row = verified | Idempotent | INT sandbox |
| BUS-4 | Authz on protected proc | Sales role | Call finance.PostJournal | PermissionDenied | No journal row | Audit row for denial | SEC-T |
| UX-2 | Honest data badge | Integration NOT_CONFIGURED | Open integrations screen | Shows NOT_CONFIGURED, not "connected" | — | — | Screenshot |
| AI-2 | AI cannot auto-send | Agent propose-only | Agent drafts WhatsApp blast | Draft queued for approval; nothing sent | No message dispatched | Human approval required | SEC-T + provider log |

## 3. Required End-to-End Scenarios

Each scenario is tested for all of: **successful completion, invalid input, unauthorized access, duplicate requests, concurrent requests, database failure, external-provider timeout, rejected approval, retry/recovery, server restart, cross-tenant access attempt.**

### Scenario A — Travel sales (pilot #1)
`Lead → assignment → quotation → approval → availability check → booking → payment verification → invoice → accounting → customer notification.`
- Success: booking confirmed **only** after payment verified; invoice persisted; balanced journal posted; notification sent only after verified send; all rows tenant/BU scoped.
- Invalid input: malformed lead/quote rejected with field errors.
- Unauthorized: sales role cannot approve above its discount/credit limit; cross-BU read blocked.
- Duplicate: duplicate booking/payment callback deduped (idempotency).
- Concurrency: two agents booking last unit → exactly one succeeds (reservation lock).
- DB failure: transaction rolls back; no partial booking/invoice.
- Provider timeout: payment marked pending, not confirmed; ret/recovery reconciles.
- Rejected approval: quote stays unapproved; no booking/invoice.
- Restart: in-flight booking state recovers from DB.
- Cross-tenant: tenant B cannot see A's booking/invoice.
- Evidence: Postgres rows, journal balance, provider sandbox logs, notification provider receipt, test transcript.

### Scenario B — IT project management (pilot #2)
`Customer contract(SOW) → project → tasks → employee assignment → timesheets → expenses → milestone completion → invoice → accounting.`
- Milestone invoice posts balanced journal; timesheets/expenses persisted and attributed; profitability computed from real rows; approvals enforced; same failure matrix as A.

### Scenario C — E-commerce
`Product → inventory → order → payment verification → fulfilment → shipment → delivery → refund/return.`
- Inventory decrements transactionally on confirmed+paid order; refund reverses accounting; marketplace/shipping run in provider TEST mode or are marked BLOCKED.

### Scenario D — HR & payroll
`Employee → attendance/payroll inputs → calculation → approval → payment submission → verified payment status → payroll accounting.`
- Four distinct states (calc/approved/submitted/disbursed); disbursement status reflects verified provider/bank result, never assumed; payroll journal posted on confirmation.

### Scenario E — Procurement
`Purchase request → approval → PO → goods receipt → invoice matching (3-way) → supplier payment → inventory + accounting updates.`
- 3-way match blocks payment on discrepancy; GRN updates inventory transactionally; supplier payment idempotent.

### Scenario F — AI-controlled workflow
`NL request → plan → permission check → proposal → human approval → authorized execution → result verification → audit trail.`
- Agent confined to allow-listed APIs + tenant/BU scope; high-impact action blocked without approval; agent reports only verified results; full audit chain present; stop/revoke works mid-execution.

## 4. Failure-injection matrix (applies to A–F)

| Fault | Injection | Expected behavior |
| :--- | :--- | :--- |
| Duplicate request | Replay with same idempotency key | Single effect |
| Concurrency | Parallel writes to same resource | Serialized; invariants hold |
| DB failure | Kill/timeout mid-transaction | Rollback; no partial state |
| Provider timeout | Sandbox delay/error | Pending, not success; recoverable |
| Rejected approval | Approver declines | No downstream effect |
| Restart | Kill backend mid-flow | State recovered from DB |
| Cross-tenant | Foreign tenant token | Denied; audited |

## 5. Evidence standard
A row passes only with: the automated test name + result, a DB query proving the persisted state, the security assertion result, and (for integrations) the provider sandbox log. "It worked in the UI" is not evidence.
