# MrBiz-OS PRD — Index

Definitive, implementation-ready PRD produced from a **read-only** inspection of this repository. No source code was modified. It is grounded in actual code (`path:line`) and in build/test runs executed this session; mock/disconnected/blocked capabilities are labelled honestly, and no test is reported passed without execution evidence.

## Documents
1. **[01-current-state-inventory.md](01-current-state-inventory.md)** — Source-of-truth inventory: what exists, classified (VERIFIED/UNVERIFIED/PARTIAL/MOCK/DISCONNECTED/MISSING/BLOCKED), with evidence and doc-vs-code contradictions.
2. **[02-PRD.md](02-PRD.md)** — The master PRD, all 30 required sections (Executive Summary → Go/No-Go), ending with component Keep/Repair/Integrate/Replace/Defer verdicts and the first-ten-tickets pointer.
3. **[03-requirements-matrix.md](03-requirements-matrix.md)** — Requirement register (BUS/SHARED/DOM/SEC/DATA/API/INT/AI/UX/NFR/OPS/TEST) with IDs, priority, acceptance, verification, status.
4. **[04-acceptance-matrix.md](04-acceptance-matrix.md)** — Acceptance rows + the six end-to-end scenarios (A–F) + failure-injection matrix. Pilots A (Travel) and B (Consulting) are the Phase 3 gate.
5. **[05-phase0-evidence.md](05-phase0-evidence.md)** — Build/test baseline (executed) + verification of the audit's 10 findings + confirmed 16-service classification.
6. **[06-matrices-and-backlog.md](06-matrices-and-backlog.md)** — ADR register, component verdicts, requirement-traceability / module-dependency / role-permission / integration-status matrices, risk register, prioritized backlog + **first 10 implementation tickets**.

> `docs/business-os/*` is **superseded** by this PRD (it describes a different Next.js + Go/Fiber stack that does not match the code — see `01` §8).

## Phase 0 evidence headline (verified this session)
- Both Go backends **build clean** (Go 1.26.0).
- **drayya-core `go test ./...` FAILS**: `domain/pharmacy` hangs ~607s on a DB tx; **all 16 mounted services have no tests**; `database` schema-snapshot drift; `domain/finance` (the disconnected double-entry engine) **passes**.
- **saas-billing**: ~18 unit/domain packages **pass**; its real tenant-isolation/accounting/concurrency suite is **blocked only by no Postgres** (connection refused), not logic.
- All 16 wired services confirmed: **13 MOCK, 3 PARTIAL, 0 fully real.**
- Audit findings: 7 VERIFIED, 1 partially, 1 verified+blocked, 1 (frontend type/test) not yet run.

## The eight questions, answered
1. **What should MrBiz-OS do?** 02 §2–§5, §10; `03`.
2. **How do businesses share one engine?** 02 §9, §11; `06` module-dependency matrix.
3. **How do DB/services/accounting/integrations work together?** 02 §13–§18; `03`.
4. **What must be fixed/verified first?** 02 §6 + Phase 0/1 of §25; `05`.
5. **How do AI agents operate safely?** 02 §19–§20.
6. **What's implemented first?** 02 §25 + `06` §7 (first 10 tickets).
7. **How is each capability tested?** 02 §24, §28; `04`.
8. **What proves production-readiness?** 02 §29–§30.

## Final recommendation (component-level verdicts in `06` §2)
**Keep** the Go modular monolith, ConnectRPC, SvelteKit shell, fail-closed interceptor, Postgres/Redis/Caddy. **Repair** IAM (real auth+authz), tenant isolation (RLS + remove admin bypass), the 3 partial services, schema mechanism, the pharmacy/schema test defects. **Integrate** `saas-billing` as the authoritative billing/accounting engine (DEC-01) and reconcile `domain/finance`. **Replace** the 13 mock services with persistent, tenant-scoped implementations (pilot-critical first). **Defer** non-pilot BUs, external integrations (DEC-02), and all AI autonomy (until §19 governance).

**Decisions needed from the owner:** DEC-01..DEC-11 (02 §27) — the pivotal one is **DEC-01: consolidate onto `saas-billing`**.

**Next step is a decision, not code:** approve the PRD and resolve DEC-01..DEC-04 so Phase 1 can begin. Implementation stops here pending approval.
