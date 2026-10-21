# 16 - Master Architecture Blueprint: 100% gRPC Backend + SvelteKit Frontend

**Date:** 2026-10-21  
**Target:** Single Unified Multi-Tenant SaaS Business OS  
**Strategic Mandate:**  
1. **API Protocol:** **100% gRPC / ConnectRPC** (Protocol Buffers) — Complete sunset and elimination of REST APIs.  
2. **Frontend:** **100% SvelteKit (Svelte 5 + TypeScript + Vite)** — Complete consolidation of all frontends into a single, high-performance SvelteKit application.  
3. **Backend Core:** **Go Modular Monolith** with gRPC Service Registry, `pgxpool`, and PostgreSQL 16 row-level tenant isolation.

---

## 1. Why gRPC + SvelteKit is the Superior Stack for MrBiz-OS

```
┌────────────────────────────────────────────────────────────────────────┐
│                   UNIFIED SVELTEKIT APPLICATION (Svelte 5)             │
│  - Modern Reactive UI (Svelte Runes: $state, $derived, $effect)        │
│  - SvelteKit BFF (hooks.server.ts with Secure HTTP-Only Cookies)       │
│  - Executive Multi-Unit Switcher + 5 Domain Workspaces                 │
│  - Connect-ES / gRPC-Web Client (100% Type-Safe CodeGen)               │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ gRPC / ConnectRPC (HTTP/2 / Protobuf)
                                    │ Strict Metadata: tenant_id, auth_token
┌───────────────────────────────────▼────────────────────────────────────┐
│                  GO MODULAR MONOLITH (gRPC Engine)                     │
│  - gRPC Auth & Tenant Interceptor (Enforces Strict Isolation)          │
│  - Concurrency Sequences (INV-YYYY-XXXX with SELECT ... FOR UPDATE)    │
│  - Decimal Engine (shopspring/decimal for 100% GST/Finance Accuracy)   │
│  - 10 Core Domain Services in Go (Pure gRPC, Zero REST Handlers)       │
└──────────────────┬─────────────────────────────────┬───────────────────┘
                   │ Redis Streams (Events)          │ PostgreSQL 16 (pgxpool)
┌──────────────────▼───────────────┐  ┌──────────────▼───────────────────┐
│     AI Worker Agents (api/)      │  │        PostgreSQL 16 DB          │
│ (Python autonomous missions)     │  │  - Row-Level Tenant Isolation    │
│ Triggers approvals via gRPC      │  │  - Immutable Double-Entry Ledger │
└──────────────────────────────────┘  └──────────────────────────────────┘
```

### 1.1 SvelteKit (Svelte 5) Advantages Over Next.js:
- **Ultra-Low Memory & CPU:** Runs comfortably in < 60MB RAM (vs 350MB+ for Next.js Node SSR), satisfying the budget VPS mandate ($\le$ ₹11,000/year).
- **Zero Virtual DOM Overhead:** Svelte 5 compiles directly to surgical DOM mutations using runes (`$state`, `$derived`), giving sub-30ms client-side route transitions.
- **Unified Svelte Foundation Already in Repo:** `SaaS Billing Application/frontend` is already built in Svelte 5 with full accounting, invoicing, settings, and auth routes.
- **SvelteKit BFF (Backend-For-Frontend):** `hooks.server.ts` handles session cookie validation and talks directly to the Go gRPC backend over fast local TCP/in-process buffers.

### 1.2 gRPC / ConnectRPC Advantages Over REST:
- **Contract-First Rigor:** Every model and endpoint is defined in `.proto` files. Zero ambiguous JSON dictionaries or `: Promise<any>` contract drift.
- **End-to-End Type Safety:** `buf generate` compiles Proto definitions simultaneously into:
  - Strongly typed Go server interfaces (`gen/go/...`)
  - Strongly typed TypeScript clients for SvelteKit (`gen/ts/...`)
- **Strict Tenant Enforcement:** Security headers (`tenant_id`, `business_unit_id`) are validated by gRPC interceptors on *every single RPC*, making tenant leakage technically impossible.
- **Native Streaming:** Built-in server streaming for live financial ledger feeds, real-time doctor appointment queues, and AI agent chatter.

---

## 2. Complete REST Sunset & Proto Service Architecture

All ~1,846 REST endpoints in `core-go` and FastAPI routes will be retired and mapped into **10 Canonical Protocol Buffer Services**:

```
proto/
├── iam/v1/iam.proto                   (Auth, Tenants, Organizations, Roles, Users)
├── billing/v1/billing.proto           (Invoices, GST, Quotations, Subscriptions)
├── accounting/v1/accounting.proto     (Double-Entry Ledger, Financial Reports, Taxes)
├── healthcare/v1/healthcare.proto     (Patients, Doctors, Clinics, Encounters, EMR, Prescriptions)
├── travel/v1/travel.proto             (Tours, Packages, Visa Checklists, Applications)
├── realestate/v1/realestate.proto     (Properties, Units, Site Visits, Deals, Escrow)
├── ecommerce/v1/ecommerce.proto       (Products, Inventory, Orders, Shipping, POD)
├── consulting/v1/consulting.proto     (Proposals, SOW Contracts, Sprints, Timesheets)
├── workflow/v1/workflow.proto         (Journey State Machine, Tasks, Approval Queue)
└── ai/v1/ai.proto                     (CEO AI Prompts, Agent Missions, Negotiation Logs)
```

---

## 3. Frontend Architecture: The Unified SvelteKit Platform

We expand `SaaS Billing Application/frontend` to become the master **`mrbiz-os-svelte`** client.

### Route Hierarchy in SvelteKit
```
src/routes/
├── +layout.svelte               (Global Shell: Topbar, Sidebar, Unit Switcher, Notifications)
├── +page.svelte                 (Executive Multi-Unit Dashboard: Global KPIs, P&L, Active Missions)
│
├── (auth)/                      (Public Pages)
│   ├── login/+page.svelte
│   ├── register/+page.svelte
│   └── reset-password/+page.svelte
│
├── admin/                       (Organization & System Administration)
│   ├── tenants/+page.svelte
│   ├── users/+page.svelte
│   └── roles/+page.svelte
│
├── billing/                     (From SaaS Billing Application)
│   ├── invoices/+page.svelte
│   ├── quotations/+page.svelte
│   ├── payments/+page.svelte
│   └── accounting/+page.svelte
│
├── healthcare/                  (From client/ and Smart Hospital)
│   ├── appointments/+page.svelte
│   ├── doctor-console/+page.svelte
│   ├── emr/+page.svelte
│   └── prescriptions/+page.svelte
│
├── travel/                      (From client/ travel domain)
│   ├── packages/+page.svelte
│   ├── visa-pipeline/+page.svelte
│   └── travellers/+page.svelte
│
├── realestate/                  (From client/ real estate domain)
│   ├── properties/+page.svelte
│   ├── site-visits/+page.svelte
│   └── deals/+page.svelte
│
├── ecommerce/                   (From client/ store & shipping)
│   ├── inventory/+page.svelte
│   ├── orders/+page.svelte
│   └── fulfillment/+page.svelte
│
├── consulting/                  (From client/ corporate & consulting)
│   ├── projects/+page.svelte
│   ├── sprints/+page.svelte
│   └── timesheets/+page.svelte
│
└── workflow/                    (Tasks, Human-in-the-Loop AI Approvals)
    ├── tasks/+page.svelte
    └── approvals/+page.svelte   (Approval queue for AI trading & operations)
```

---

## 4. Master Step-by-Step Implementation Plan

### Step 1: Protocol Buffer Schema Consolidation
1. Consolidate proto definitions from `SaaS Billing Application/backend/proto/` (`iam`, `billing`, `accounting`, `customer`, `catalog`, `payment`).
2. Add proto definitions for `healthcare.v1`, `travel.v1`, `realestate.v1`, `ecommerce.v1`, `consulting.v1`, and `workflow.v1`.
3. Run `buf generate` to compile Go server stubs and TypeScript Connect-ES clients for SvelteKit.

### Step 2: Go Backend gRPC Server Refactoring
1. Upgrade `core-go` to initialize the ConnectRPC / gRPC listener.
2. Implement strict tenant auth interceptors on all RPC calls.
3. Wire existing domain services to gRPC methods.
4. Turn off and decommission Fiber REST routers.

### Step 3: SvelteKit Master Client Development
1. Elevate `SaaS Billing Application/frontend` into the single primary web application.
2. Port the Executive Suite views (`UnitSwitcher`, `ApprovalQueue`, `ExecutiveDashboard`) into Svelte 5 components.
3. Port operational screens for Healthcare, Travel, Real Estate, E-Commerce, and Consulting into SvelteKit route groups.
4. Hook all data fetching to the generated Connect-ES gRPC client.

### Step 4: Python AI Agents as gRPC Clients
1. Update `api/` Python agents to call Go gRPC services directly.
2. Direct all autonomous proposals into the SvelteKit `ApprovalQueue`.

### Step 5: Clean-up and Decommissioning
1. Delete redundant Next.js directories (`client/` and `web/`).
2. Finalize single Docker Compose containing PostgreSQL, Redis, Go gRPC backend, SvelteKit BFF, and Python AI Worker.
