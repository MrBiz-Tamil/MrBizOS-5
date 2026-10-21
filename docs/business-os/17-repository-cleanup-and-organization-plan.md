# 17 - Repository Cleanup & Organization Plan

**Date:** 2026-10-21  
**Objective:** Streamline the repository from 6 fragmented, cluttered directories into a clean, unified, enterprise-grade architecture.  
**Immediate Impact:** Reclaims **~1.2 GB of disk space**, eliminates 469 clutter files, removes patient PHI databases, and establishes a single intuitive folder structure.

---

## 1. Current State vs. Clean Target State

### Current Fragmented Structure (~1.5 GB, 6 Competing Directories)
```
MrBiz-OS/
├── Multi_Business_Group_Business_OS_Master_PRD.docx
├── SaaS Billing Application/          (Spaced name; contains SvelteKit frontend + Go backend + .tmp_server.exe)
├── Smart Hospital Managment Engine/   (Spaced name; Python FastAPI + old doctor app + Medi-consultation-Hub.zip)
├── api/                               (Python trading agents + loose trade_ai.db + 20+ debug txt/logs)
├── client/                            (Next.js 14 frontend; 687 files — OBSOLETE)
├── core-go/                           (Go Fiber v2 + our new gRPC engine; contains compiled .exe, PHI databases, logs)
├── docs/business-os/                  (16 documentation files)
└── web/                               (Next.js 16 frontend; 963 MB — OBSOLETE)
```

---

### Clean Target Structure (Organized, Modern, Production-Ready)
```
MrBiz-OS/
├── proto/                             # 100% Contract-First Protocol Buffers (13 Canonical Services)
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   ├── iam/v1/iam.proto
│   ├── billing/v1/billing.proto
│   ├── accounting/v1/accounting.proto
│   ├── healthcare/v1/healthcare.proto
│   ├── travel/v1/travel.proto
│   ├── realestate/v1/realestate.proto
│   ├── ecommerce/v1/ecommerce.proto
│   ├── consulting/v1/consulting.proto
│   ├── workflow/v1/workflow.proto
│   └── ai/v1/ai.proto
│
├── backend/                           # Unified Go Modular Monolith (100% gRPC / ConnectRPC)
│   ├── cmd/
│   │   ├── grpc-server/               # Main production server entrypoint
│   │   └── seed/                      # Clean database seeder (safe demo data)
│   ├── gen/                           # Generated Go gRPC & Connect stubs (buf output)
│   ├── internal/
│   │   ├── domain/                    # Pure domain logic (5 BUs + Shared Core)
│   │   ├── grpcserver/                # H2C server, strict tenant & auth interceptors
│   │   ├── grpcservices/              # gRPC service handlers
│   │   ├── database/                  # PostgreSQL 16 (pgxpool) & Redis connection
│   │   ├── audit/                     # Immutable write-interception audit log
│   │   └── tokens/                    # JWT & session verification
│   ├── go.mod
│   └── go.sum
│
├── frontend/                          # Unified SvelteKit (Svelte 5) Web Application
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/            # UnitSwitcher, templates, modals
│   │   │   ├── grpc.ts                # Type-safe ConnectRPC client
│   │   │   └── stores/                # Svelte 5 runes stores ($state, $derived)
│   │   ├── routes/
│   │   │   ├── +layout.svelte         # Master navigation shell + UnitSwitcher
│   │   │   ├── +page.svelte           # Executive Multi-Unit KPI Suite
│   │   │   ├── healthcare/            # Smart Healthcare & Telehealth
│   │   │   ├── travel/                # Travel, Tours & Visa OS
│   │   │   ├── realestate/            # Real Estate & Deal Room
│   │   │   ├── billing/               # Invoicing & GST Compliance
│   │   │   ├── accounting/            # Double-Entry Ledger & P&L
│   │   │   └── workflow/              # Human-in-the-Loop Approvals & Tasks
│   │   └── hooks.server.ts            # SvelteKit BFF session & security headers
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.ts
│
├── ai-workers/                        # Python Autonomous Mission & Trading Agents
│   ├── agents/                        # Sourcing, negotiation, marketing agents (from api/)
│   ├── worker.py                      # Background worker listening to Redis Streams
│   ├── missions/                      # Autonomous sector mission definitions
│   └── requirements.txt
│
├── docs/                              # Architecture Specs & Operational Manuals
│   ├── business-os/                   # Phase 1 audit & roadmap documentation
│   └── Multi_Business_Group_Business_OS_Master_PRD.docx
│
├── deploy/                            # Single-Command Production Deployment
│   ├── docker-compose.yml             # Orchestrates Postgres 16, Redis, Backend, Frontend, Worker
│   └── Caddyfile                      # Automated HTTPS / TLS reverse proxy
│
└── .gitignore                         # Hardened gitignore blocking DBs, binaries, logs, secrets
```

---

## 2. Waste & Clutter Breakdown to be Removed

| Category | Specific Files / Directories | Size Reclaimed | Security & Performance Rationale |
| :--- | :--- | :--- | :--- |
| **Obsolete React/Next.js Frontends** | `web/` (963.7 MB)<br>`client/` (18.3 MB) | **~982 MB** | Standardized 100% on SvelteKit (Svelte 5). Next.js code is dead weight that causes confusion and bloat. |
| **Compiled Executables & Binaries** | `core-go/bin/*.exe`<br>`core-go/*.exe`<br>`SaaS Billing Application/backend/*.exe`<br>`SaaS Billing Application/backend/server_linux`<br>`SaaS Billing Application/.tmp_server.exe` | **~180 MB** | Binaries should never be tracked in source control; they can be compiled on demand. |
| **Committed Databases (PHI Risk)** | `core-go/Dr.Ayya.db*` (~9 MB)<br>`core-go/medihub_e2e_test.db`<br>`Smart Hospital Managment Engine/backend/mediconsult.db`<br>`api/trade_ai.db`<br>`api/trading_ai.db`<br>`api/huey.db` | **~25 MB** | Patient PHI and production data must never sit in git history. Fresh data will come from seeders only. |
| **Debug Logs & Output Dumps** | `api/output_*.txt`<br>`api/*.log`<br>`api/seed_log.txt`<br>`core-go/*.log`<br>`core-go/tests/integration/storage/prescriptions/*.pdf` | **~10 MB** | Over 100 generated logs and test prescription PDFs cluttering git status. |
| **Zip Archives & Temp Dirs** | `Smart Hospital Managment Engine/Medi-consultation-Hub.zip`<br>`SaaS Billing Application/.tmp/` | **~15 MB** | Redundant compressed copies of files already extracted in the tree. |
| **Total Reclaimed Space** | | **~1.21 GB** | **469 clutter files completely eliminated.** |

---

## 3. Step-by-Step Execution Plan

### Step 1: Promote the SvelteKit Frontend
1. Move `SaaS Billing Application/frontend` $\to$ `frontend/` at the workspace root.
2. Verify `frontend/package.json` and run `npm run check` to ensure all links and imports resolve cleanly.

### Step 2: Consolidate the Go Backend
1. Rename `core-go/` $\to$ `backend/` at the workspace root.
2. Merge any missing financial models from `SaaS Billing Application/backend/internal/accounting` into `backend/internal/domain/finance/`.
3. Verify that `backend/bin/grpc-server.exe` builds cleanly from `backend/cmd/grpc-server/main.go`.

### Step 3: Organize AI Mission Agents
1. Create `ai-workers/` directory.
2. Move active Python agent logic from `api/agents/` and mission definitions from `api/mission_*.py` into `ai-workers/`.
3. Provide clean `ai-workers/requirements.txt` with only the needed dependencies.

### Step 4: Purge Clutter & Obsolete Folders
1. Permanently delete:
   - `web/` (Next.js 16)
   - `client/` (Next.js 14)
   - `SaaS Billing Application/` (now that `frontend/` is promoted)
   - `Smart Hospital Managment Engine/` (now that domains are unified in Go)
   - `api/` (now that active agent code is in `ai-workers/`)
2. Remove all loose `.db`, `.log`, `.exe`, `.zip`, and `.txt` debug files.

### Step 5: Update Root `.gitignore`
Add strict ignore rules so clutter never returns:
```gitignore
# Databases
*.db
*.db-shm
*.db-wal
*.sqlite
*.sqlite3

# Binaries & Builds
*.exe
*.dll
*.so
*.dylib
bin/
dist/
build/
server_linux

# Node & Frontend
node_modules/
.svelte-kit/
.next/

# Python
__pycache__/
*.pyc
.venv/
venv/

# Logs & Secrets
*.log
.env
.env.*.local
!*.example
```

### Step 6: Create Unified Docker Compose (`deploy/docker-compose.yml`)
Provide a single command to run the whole stack:
- **`postgres`**: PostgreSQL 16 on port 5432
- **`redis`**: Redis 7 on port 6379
- **`backend`**: Go gRPC modular monolith on port 8080
- **`frontend`**: SvelteKit BFF on port 3000
- **`ai-workers`**: Python autonomous mission worker
