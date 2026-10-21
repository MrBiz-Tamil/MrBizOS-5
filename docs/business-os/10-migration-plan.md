# 10 - Migration Strategy & Backward Compatibility Plan

## Migration Guiding Principles
1. **Zero Downtime**: Database migrations run in background goroutines guarded by PostgreSQL advisory locks.
2. **Backward Compatibility**: Existing database tables, APIs, and frontend pages must continue functioning uninterrupted.
3. **Additive Schema Updates**: Only add columns, tables, and indexes. Never drop or rename active columns in production without deprecation phase.

---

## Migration Execution Phases

### Phase A: Schema Preparation & Additive Columns
- Add `business_unit_id` and `organization_id` as nullable columns to existing tables (`user`, `invoices`, `activity_logs`, `documents`).
- Create standard indexes on newly added columns.
- Backfill default values (`business_unit_id = 2` for Healthcare records, `organization_id = 1` for default Group).

### Phase B: Dual Write & Event Bridging
- Configure Fiber middleware to populate business unit context headers.
- Enable dual-writing to legacy and updated models where necessary.
- Verify data consistency via integration test suites (`master_30_domains_e2e_test.go`).

### Phase C: API Deprecation & Clean Cutover
- Mark old un-scoped endpoints with `Deprecation: true` HTTP headers.
- Route frontend applications to updated multi-scoped endpoints.
- Archive legacy data structures after validation window.

---

## Safety & Rollback Verification

```sql
-- Postgres Advisory Lock Guard Pattern (internal/server/app.go)
SELECT pg_advisory_lock(123456789);
-- Run schema AutoMigrate safely across horizontal container replicas
SELECT pg_advisory_unlock(123456789);
```

If a migration encounters an error during startup, the application logs a detailed trace and fails safely without corrupting production data.
