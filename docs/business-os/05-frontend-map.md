# 05 - Frontend Map & Page Audit

## Frontend Overview
The frontend is a single-repo Next.js 14 application (`client/app`) serving both public portal interfaces and administrative dashboards.

---

## Route Classification Matrix

| Route Path | Associated Feature / Module | Backend API Link | Current Status | Recommended Action |
| :--- | :--- | :--- | :--- | :--- |
| `/admin` | Master Administrative Console | `/api/v1/admin/*` | **Working** | Extend into Group Business OS Navigation Shell |
| `/dashboard` | User Operations Center | `/api/v1/users/me` | **Working** | Adapt per Business Unit role |
| `/doctor` & `/doctors` | Doctor Portal & Directory | `/api/v1/doctors/*` | **Working** | Retain under Healthcare BU |
| `/patient` & `/emr` | Medical Records & Patient Portal | `/api/v1/emr/*` | **Working** | Retain under Healthcare BU |
| `/appointments` | Appointment Booking | `/api/v1/appointments/*` | **Working** | Generalize for Travel & Real Estate Bookings |
| `/billing` & `/finance` | Billing & Ledger Management | `/api/v1/billing/*` | **Working** | Connect to Business Unit Cost Centers |
| `/pharmacy` | Pharmacy Catalog & Checkout | `/api/v1/pharmacy/*` | **Working** | Adapt for E-commerce & POD Catalog |
| `/tasks` | Task List & Workflows | Disconnected / Local State | **Partial Data** | Connect to new Central Task Engine API |
| `/hr` & `/staff` | HR & Staff Management | `/api/v1/hr/*` | **Working** | Extend with Commission & Business Unit filters |
| `/bpo` & `/corporate` | Corporate Accounts | `/api/v1/corporate/*` | **Working** | Extend into B2B Consulting Account Portal |
| `/blog` & `/press` & `/faq` | CMS Public Content | `/api/v1/blog/*`, `/content/*` | **Working** | Connect to Multi-Brand Domain Resolver |
| `/chat` & `/inbox` | Real-time Communication Hub | `/api/v1/chat/*` | **Working** | Extend for Lead & Customer WhatsApp/Email |
| `/support` & `/qa` | Support Desk & Help Center | `/api/v1/support/*` | **Working** | Multi-BU Support Ticketing |
| `/loyalty` & `/referrals` | Loyalty & Referral Program | `/api/v1/loyalty/*` | **Working** | Group-wide Attribution Engine |

---

## Frontend Layout & UI/UX Audit

1. **Design System & Styling**: Uses Tailwind CSS with a dark/light mode toggle. UI components are clean, responsive, and accessible.
2. **State Management**: Uses React Query for server state caching and Zustand for local application state.
3. **Multi-Domain Website Engine**:
   - Currently, hostnames are handled globally.
   - Requires middleware-based Site Resolution (`middleware.ts`) to detect domain (e.g. `travel-brand.com`, `health-brand.com`) and serve corresponding site branding tokens (logo, palette, typography, nav).
