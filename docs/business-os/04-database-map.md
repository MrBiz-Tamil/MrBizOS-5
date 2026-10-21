# 04 - Database Map & Schema Audit

## Database Architecture
- **Primary RDBMS**: PostgreSQL (System of Record with GORM ORM)
- **Vector DB Extension**: `pgvector` enabled for Devi AI document embeddings & RAG
- **In-Memory Cache & Event Bus**: Redis (Sessions, Rate limiting, Redis Streams `EventBus`)
- **Local Dev / Testing DB**: SQLite / Embedded SQLite (`Dr.Ayya.db`, `medihub.db`)

---

## Core Schema Audit (26 Primary Models)

```
                            ┌──────────────────────────────────┐
                            │               USER               │
                            │ (id, email, role, business_units)│
                            └────────────────┬─────────────────┘
                                             │
      ┌────────────────────────┬─────────────┼───────────────┬────────────────────────┐
      │                        │             │               │                        │
      ▼                        ▼             ▼               ▼                        ▼
┌──────────────┐      ┌─────────────────┐ ┌─────┐   ┌─────────────────┐      ┌─────────────────┐
│DOCTOR_PROFILE│      │ PATIENT_PROFILE │ │STAFF│   │CUSTOMER_PROFILE │      │ EMPLOYEE_RECORD │
└──────┬───────┘      └────────┬────────┘ └─────┘   └────────┬────────┘      └────────┬────────┘
       │                       │                             │                        │
       ▼                       ▼                             ▼                        ▼
┌──────────────┐      ┌─────────────────┐           ┌─────────────────┐      ┌─────────────────┐
│ APPOINTMENTS │      │  EMR & CLINICAL │           │ LEADS & JOURNEY │      │ PAYROLL & COMM. │
└──────────────┘      └─────────────────┘           └─────────────────┘      └─────────────────┘
```

### Table Categories

1. **Identity & Authorization**:
   - `user`: Primary identity record (id, email, password_hash, role_id, phone, status)
   - `roles` & `permissions`: RBAC matrix
   - `doctor_profiles`, `staff_profiles`: Specialty metadata, registration numbers

2. **Healthcare Operations**:
   - `appointments`: Booking records, slot status, cancellation reasons
   - `emr_records`, `prescriptions`: Encrypted medical encounters and prescriptions
   - `hospitals`, `clinics`, `wards`, `beds`: Facility inventory

3. **Finance & Accounting**:
   - `invoices`, `billing_items`: Financial transactions
   - `ledger_entries`: Double-entry accounting ledger
   - `wallets`, `wallet_transactions`: Stored value balance
   - `bank_statements`, `reconciliations`: Automated bank matching

4. **Documents & Communication**:
   - `documents`: File attachment metadata, S3 paths, encryption status
   - `chat_sessions`, `messages`: Real-time chat threads
   - `activity_logs`, `audit_logs`: Operational & security audit trail

---

## Database Extension Strategy for Group Business OS

1. **Organizational & Business Unit Scoping**:
   - Add nullable `organization_id` (UUID/uint) and `business_unit_id` (uint) to top-level records (`user`, `invoices`, `leads`, `tasks`, `documents`, `audit_logs`).
   - Retain un-scoped records where data is globally shared across BUs (e.g. unified master contacts).

2. **Indexing Strategy**:
   - Create composite index `idx_bu_org` on `(business_unit_id, organization_id)` for high-frequency filter queries.
   - Create partial indexes on active records (e.g. `WHERE deleted_at IS NULL`).

3. **Schema Migration Policy**:
   - Execute GORM `AutoMigrate` inside non-blocking background goroutines during startup.
   - Use Postgres advisory locks to prevent concurrent migration race conditions across horizontal API replicas.
   - Never issue destructive `DROP COLUMN` or `ALTER COLUMN TYPE` without explicit backward-compatibility backfills.
