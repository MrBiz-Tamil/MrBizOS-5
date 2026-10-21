# Go Migration Verification Walkthrough

## Overview
This document guides you through verifying the migration of Core Logic, Wallet, Admin, and EMR services to Go.

## Prerequisites
1.  **Improper Go Installation**: Ensure Go is installed and in your PATH.
    - Path: `C:\Program Files\Go\bin\go.exe` (Verified)
2.  **Database Services**: PostgreSQL and Redis must be running locally.
    - Check with `docker ps` if using Docker.
    - Run `docker-compose up -d postgres redis` to start them.
    - Or ensure local services are running on ports 5432 and 6379.
3.  **Environment Variables**:
    - `apps/core-go/.env` has been created with local credentials.
    - Modify `DB_USER` and `DB_PASSWORD` in `apps/core-go/.env` if your local Postgres credentials differ from `medhub_prod_user`.

## Steps to Run Locally

### 1. Start the Go Server
Open a terminal in `d:\Project Files\medi-consultation-hub-dev\medi ai\apps\core-go` and run:

```powershell
& "C:\Program Files\Go\bin\go.exe" run cmd/server/main.go
```

**Functionality Verified**:
- Wallet (Ledger v2)
- Admin Dashboard & Settings
- EMR & Prescriptions

### 2. Run End-to-End Tests
In a separate terminal (keep server running), execute the Python E2E test script:

```powershell
python scripts/e2e_workflow_test.py
```

This script verifies:
- Wallet Balance & Top-up (`/api/wallet/*`)
- Admin Dashboard Stats (`/api/admin/dashboard/*`)
- Prescription Lifecycle (`/api/emr/*`)

## Troubleshooting

### Database Connection Error
If you see `failed to initialize database: ... connection refused`, it means PostgreSQL is not accessible.
- **Fix**: Ensure Postgres service is started. Update `.env` credentials if needed.

### Missing Dependencies
If you see variable validation errors or missing packages:
- Run: `& "C:\Program Files\Go\bin\go.exe" mod tidy`

### Port Conflicts
If port 8081 is busy, change `GO_PORT` in `.env`.
