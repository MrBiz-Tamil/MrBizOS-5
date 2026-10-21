# 14 - Security, Compliance & Data Isolation Review

## Security Standards & Hierarchy

```
USER
 └── ORGANIZATION MEMBERSHIP
      └── BUSINESS UNIT MEMBERSHIP
           └── ROLE
                └── PERMISSIONS (e.g. travel.lead.read, healthcare.clinical.read)
                     └── RESOURCE SCOPE (Filtered by BusinessUnitID & OrganizationID)
```

---

## Key Security Controls

### 1. Healthcare Clinical Isolation (PHI & HIPAA Compliance)
- **Requirement**: Clinical details (prescriptions, diagnostic reports, EMR records) must NEVER be exposed to non-healthcare personnel via Customer 360 or general CRM dashboards.
- **Implementation**: Handled by `emr.BackfillEncryptedFields` and strict RBAC permission `healthcare.clinical.read`. Customer 360 displays only non-clinical relationship meta (e.g. `Patient ID: 1042`, `Last Appointment: 2026-07-20`).

### 2. Authentication & JWT Security
- Standardized JWT claims with short expiration (15 minutes) and HTTP-only refresh tokens.
- Password hashing using Argon2id / bcrypt.
- Passwordless OTP login with Redis rate-limiting (max 3 OTP requests per 10 minutes).

### 3. Signed File Access & Document Security
- Files uploaded to Object Storage (S3 / MinIO) remain in private buckets.
- Download links generated via short-lived (15-minute) HMAC signed URLs requiring authorization header verification.

### 4. Audit Logging & Anomaly Detection
- Intercepted by `internal/audit` GORM plugin on every DB insert/update/delete.
- Sensitive fields (`password_hash`, `otp`, `card_number`, `token`) are automatically stripped before audit logging.
- Anomaly detector triggers alerts on rapid role escalation or bulk data download events.
