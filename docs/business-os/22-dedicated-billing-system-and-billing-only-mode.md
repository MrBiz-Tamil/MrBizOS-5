# MrBiz-OS: Standalone Billing System & "Billing-Only Mode" Architecture

## 1. Overview & Business Rationale
Many small and medium retail shops, traders, consultants, and service businesses **only need a fast, streamlined billing and invoicing system** (similar to Zoho Invoice, Vyapar, Swipe, or QuickBooks Simple Start) rather than an entire multi-department ERP.

To solve this, MrBiz-OS now includes:
1. **Dedicated Express Billing Counter (`/billing`)**: An ultra-fast, 3-second billing counter with live GST calculations, dynamic UPI QR codes, 80mm thermal receipt printing, and 1-click WhatsApp bill sharing.
2. **"Billing-Only Mode" Toggle**: A single-click switch in the application sidebar and header that hides all non-billing ERP modules (Healthcare, Travel, Consulting, Real Estate, HRMS, BPO Call Center, Procurement) and transforms MrBiz-OS into a distraction-free, lightning-fast billing software.
3. **Backend `billing.v1.BillingService`**: A high-performance Go ConnectRPC gRPC service mounted as Service 16 in `grpcserver`, handling invoice creation, partial payment recording, aging receivables tracking, and invoice cancellation.

---

## 2. Full-Stack Implementation Details

### Backend: `billing.v1.BillingService` (Go ConnectRPC)
- **Proto Contract**: `proto/billing/v1/billing.proto`
- **Service Implementation**: `backend/internal/grpcservices/billing/service.go`
- **Server Registration**: Mounted at Service 16 in `backend/internal/grpcserver/mount.go`
- **RPC Endpoints**:
  1. `ListInvoices(ListInvoicesRequest) -> ListInvoicesResponse`: Returns invoices filtered by tenant and status (`PAID`, `PARTIAL`, `UNPAID`, `OVERDUE`).
  2. `GetInvoice(GetInvoiceRequest) -> GetInvoiceResponse`: Fetches full invoice with itemized line items and tax breakdown.
  3. `CreateInvoice(CreateInvoiceRequest) -> CreateInvoiceResponse`: Auto-calculates gross amount, line item discounts, taxable amount, and GST (Intra-state CGST + SGST vs Inter-state IGST).
  4. `RecordPayment(RecordPaymentRequest) -> RecordPaymentResponse`: Records full or partial payments via Cash, UPI, Card, NetBanking, or Cheque, auto-updating balance due and status.
  5. `GetBillingSummary(GetBillingSummaryRequest) -> GetBillingSummaryResponse`: Aggregates today's sales, this month's sales, total outstanding dues, and customer aging ledger.
  6. `CancelInvoice(CancelInvoiceRequest) -> CancelInvoiceResponse`: Marks invoice as cancelled with audit reason.
  7. `UpdateLogistics(UpdateLogisticsRequest) -> UpdateLogisticsResponse`: Updates vehicle number, E-way bill, transporter name, and delivery location.
  8. `GenerateInvoicePDF(GenerateInvoicePDFRequest) -> GenerateInvoicePDFResponse`: Generates print-ready PDF invoices.

### Frontend: Express Billing Page & Navigation Mode Switcher
- **Page Route**: `frontend/src/routes/billing/+page.svelte`
- **Client RPCs**: Added `billingGrpc` in `frontend/src/lib/grpc.ts`
- **Sidebar Integration**: `frontend/src/routes/+layout.svelte`

#### Core Features in `/billing`:
1. **Express Billing Counter (Tab 1)**:
   - **One-Click Walk-in**: Instant "🚶 Cash Walk-in Customer" button to bill in under 3 seconds.
   - **Smart GST Detection**: Automatically toggles between Intra-State (CGST + SGST) and Inter-State (IGST) based on the customer's state of supply.
   - **Dynamic Item Grid**: Real-time HSN/SAC, quantity, unit, price, discount %, and tax % (0%, 5%, 12%, 18%, 28%) with instant row calculations.
   - **Dynamic UPI QR Code**: Real-time generated UPI QR code displayed on screen for immediate customer scanning.
   - **Tendered Cash & Balance Due**: Automatic calculation of change to return or balance remaining.
   - **Thermal Receipt & WhatsApp**: 1-click 80mm thermal receipt print and formatted WhatsApp message with payment link.
2. **All Invoices & Instant Payment Recorder (Tab 2)**:
   - Filterable invoice table (`PAID`, `PARTIAL`, `UNPAID`, `OVERDUE`).
   - "💳 Record Pay" modal to log customer payments against any pending bill.
3. **Customer Receivables & Khata (Tab 3)**:
   - Customer aging ledger showing total billed, total paid, and balance due.
   - One-click "💬 Send WhatsApp Reminder" to nudge overdue customers with their pending balance.

---

## 3. "Billing-Only Mode" vs "Full Enterprise OS Mode"

| Feature / Navigation | Billing-Only Mode | Full Enterprise OS Mode |
| :--- | :--- | :--- |
| **Express Billing Counter (`/billing`)** | ✅ Primary Counter | ✅ Included |
| **Sales Invoices & Bills (`/invoices`)** | ✅ Included | ✅ Included |
| **Retail POS Terminal (`/pos`)** | ✅ Included | ✅ Included |
| **Quotations & Estimates (`/quotations`)** | ✅ Included | ✅ Included |
| **Customer Khata & Ledger (`/customers`)** | ✅ Included | ✅ Included |
| **Products & Pricing Catalog (`/products`)** | ✅ Included | ✅ Included |
| **Inventory Stock Tracking (`/inventory`)** | ✅ Included | ✅ Included |
| **Sales & Tax Accounting Reports (`/accounting`)** | ✅ Simplified View | ✅ Full Double-Entry GL |
| **Company Profile & UPI QR Settings (`/settings`)**| ✅ Included | ✅ Included |
| **Healthcare, Travel, Real Estate, Consulting** | 🚫 **Hidden** | ✅ Visible |
| **BPO Voice Call Center, Support Helpdesk** | 🚫 **Hidden** | ✅ Visible |
| **Projects, UTMS Tasks, Journeys, HRMS** | 🚫 **Hidden** | ✅ Visible |
| **1-Click Mode Toggle** | Available anytime in sidebar/header | Available anytime in sidebar/header |

---

## 4. Verification & Build Results
- **Go ConnectRPC Backend**: `backend/bin/grpc-server.exe` compiled cleanly with 16 mounted services.
- **SvelteKit Frontend**: `npm run check` passed with **0 errors and 0 warnings**.
