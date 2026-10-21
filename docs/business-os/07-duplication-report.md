# 07 - Duplication Report & Anti-Duplication Policy

## Executive Summary
To prevent technical debt, code sprawl, and conflicting sources of truth, this report identifies existing functional overlaps and sets strict rules for code reuse across all backend and frontend modules.

---

## Overlap & Consolidation Audit

| Overlapping Modules | Existing Packages | Root Cause | Consolidation Decision |
| :--- | :--- | :--- | :--- |
| **Billing vs. Finance** | `domain/billing`, `domain/finance`, `handlers/payments` | `billing` handles point-of-sale invoice creation while `finance` handles accounting ledgers. | **Consolidate under `finance`**: Retain `billing` for customer invoicing APIs, but feed all financial records directly into `finance.LedgerEntry`. |
| **User vs. Patient Profile** | `models/user.go`, `models/patient.go`, `domain/identity` | `user` represents identity/credentials, `patient` holds clinical profile. | **Maintain Decoupled Association**: Keep `User` as the root identity. Link `PatientProfile`, `DoctorProfile`, `EmployeeRecord`, and `CustomerProfile` via 1-to-1 foreign keys on `user_id`. |
| **Chat vs. Communication** | `handlers/chat`, `domain/communication`, `services/notification` | `chat` handles live web sockets; `communication` handles WhatsApp & SMS logs. | **Consolidate Gateway**: Wrap both under `domain/communication` service layer. Chat threads become a channel inside the communication engine. |
| **Pharmacy vs. Store** | `domain/pharmacy`, `domain/store` | `pharmacy` handles medical SKUs & courier dispatch; `store` handles generic catalog. | **Unify SKU Engine**: Generalize `pharmacy` inventory into core SKU & Inventory Management engine. Pharmacy becomes a specialized health catalog category. |
| **HR Tasks vs. Central Tasks** | `domain/hr` (shifts), `/tasks` (frontend) | Staff shift assignments created local tasks without a backing DB entity. | **Unify under `domain/task`**: Create a central task engine. HR shift assignments generate standard `Task` records. |

---

## Anti-Duplication Directives

1. **Do NOT Create Second Pipeline Engine**: Extend existing `dispatch` state machine for Journey Engine. Do not create duplicate workflow tables for Travel, Real Estate, or Healthcare.
2. **Do NOT Create Duplicate Task Tables**: Use one central `tasks` table supporting `organization_id`, `business_unit_id`, `entity_type`, and `entity_id`.
3. **Do NOT Create Duplicate Document Repositories**: Use central `document` domain for file attachments across all business units.
4. **Do NOT Create Second Customer Aggregation Engine**: Extend existing Customer 360 data aggregator rather than building separate customer summaries per domain.
