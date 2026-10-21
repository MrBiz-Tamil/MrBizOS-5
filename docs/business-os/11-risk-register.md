# 11 - Risk Register & Mitigation Strategy

## Risk Assessment Matrix

| Risk ID | Category | Risk Description | Likelihood | Impact | Severity | Mitigation Strategy |
| :--- | :--- | :--- | :---: | :---: | :---: | :--- |
| **RSK-01** | **Security / Compliance** | Healthcare clinical records (PHI) exposed to unauthorized Business OS users via Customer 360 view. | Low | High | **CRITICAL** | Enforce strict RBAC check (`healthcare.clinical.read`) in Customer 360 handler. Only display relationship summary, not medical notes. |
| **RSK-02** | **Performance** | Multi-brand hostname resolution causing dynamic DNS lookup delays or cache misses in Next.js middleware. | Medium | Medium | **HIGH** | Cache site configurations in Redis and in-memory LRU cache inside Next.js middleware. |
| **RSK-03** | **Data Integrity** | Schema migration failure during horizontal scale out of API servers. | Low | High | **HIGH** | Use Postgres advisory locks (`pg_advisory_lock`) so only one process runs migrations at a time. |
| **RSK-04** | **Architecture** | Unintentional duplicate creation of task or workflow engines per business unit. | Medium | High | **HIGH** | Strict anti-duplication policy enforcing extension of `dispatch` Journey Engine and central `tasks` domain. |
| **RSK-05** | **Third-Party** | Etsy API rate limits or POD webhook failures blocking order fulfillment. | High | Medium | **HIGH** | Implement transactional outbox pattern with exponential backoff retries and dead-letter queues. |
| **RSK-06** | **Performance** | N+1 database queries on unified Customer 360 timeline aggregation. | Medium | Medium | **MEDIUM** | Use GORM preloading and indexed partial joins for timeline events. |
