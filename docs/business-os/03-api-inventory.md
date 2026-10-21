# 03 - API Inventory & Endpoint Analysis

## API Overview
The backend API is powered by Fiber v2 (`drayya-core`) exposed under the `/api/v1` base path. A comprehensive analysis of `core-go/routes.json` reveals over 250 unique route handlers.

---

## Core API Endpoint Mapping

### 1. Identity & Auth (`/api/v1/auth`, `/api/v1/users`, `/api/v1/doctors`)
- `POST /api/v1/auth/register` — User registration
- `POST /api/v1/auth/login` — Password authentication & JWT generation
- `POST /api/v1/auth/refresh` — Token refresh
- `GET /api/v1/auth/me` — Active session profile
- `GET /api/v1/doctors/list` — Doctor profiles listing
- `GET /api/v1/doctors/stats/overview` — Doctor operational statistics

### 2. Clinical & EMR (`/api/v1/emr`, `/api/v1/clinical`)
- `GET /api/v1/emr/records/:patient_id` — Patient EMR history
- `POST /api/v1/emr/prescriptions` — Create digital prescription
- `GET /api/v1/clinical/pods` — AYUSH clinical pods
- `POST /api/v1/clinical/consultations` — Record consultation encounter

### 3. Finance, Billing & Wallet (`/api/v1/billing`, `/api/v1/finance`, `/api/v1/wallet`, `/api/v1/payouts`)
- `GET /api/v1/billing/invoices` — List invoices
- `POST /api/v1/billing/invoices` — Create invoice
- `GET /api/v1/finance/ledger` — Accounting ledger entries
- `POST /api/v1/finance/reconcile` — Bank reconciliation trigger
- `GET /api/v1/wallet/balance` — User wallet balance
- `POST /api/v1/payouts/process` — Execute provider payouts

### 4. Communication & Notifications (`/api/v1/chat`, `/api/v1/communication`, `/api/v1/notifications`)
- `GET /api/v1/chat/threads` — List active chat sessions
- `POST /api/v1/chat/messages` — Send real-time chat message
- `POST /api/v1/communication/whatsapp/webhook` — Incoming WhatsApp webhook
- `GET /api/v1/notifications` — In-app notification feed

### 5. Document Management (`/api/v1/documents`)
- `POST /api/v1/documents/upload` — Upload document metadata & get pre-signed URL
- `GET /api/v1/documents/:id` — Get document details & download URL
- `DELETE /api/v1/documents/:id` — Soft-delete document

### 6. CMS & Content (`/api/v1/blog`, `/api/v1/content`, `/api/v1/faq`)
- `GET /api/v1/blog/posts` — Published blog articles
- `GET /api/v1/content/pages/:slug` — Dynamic page content
- `GET /api/v1/faq` — Frequently asked questions

### 7. Operations & Logistics (`/api/v1/pharmacy`, `/api/v1/shipping`, `/api/v1/zones`)
- `GET /api/v1/pharmacy/catalog` — SKU inventory catalog
- `POST /api/v1/shipping/dispatch` — Automated courier dispatch
- `GET /api/v1/zones/coverage` — Geofence coverage areas

---

## API Hygiene & Gap Identification

1. **Inconsistent Response Payloads**: Legacy endpoints return raw struct maps, while newer endpoints use standard `response.Success(c, data)` wrapper. Standardize all API responses.
2. **Missing Scoping Parameters**: Most endpoints currently assume a single tenant (`Dr.Ayya`). Must inject `X-Organization-Id` and `X-Business-Unit-Id` headers via Fiber middleware.
3. **N+1 Query Bottlenecks**: Identified N+1 query patterns in `/api/v1/doctors/list` and `/api/v1/billing/invoices` when fetching associated relations. Preload GORM associations.
