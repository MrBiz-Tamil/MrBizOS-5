# 01 - Existing System Inventory

## Executive Overview
This inventory maps the current state of the production-scale platform before starting the Group Business OS expansion. The repository contains a high-throughput Go microservices backend (`core-go`) and a comprehensive Next.js 14 frontend (`client`).

---

## Backend Go Domains Inventory (`core-go/internal/domain/`)

| Domain | Purpose | Key Entities / Models | Main APIs | Status & Reusability |
| :--- | :--- | :--- | :--- | :--- |
| **identity** | User auth, RBAC, doctor & staff profiles, OAuth/JWT | `User`, `DoctorProfile`, `Role`, `Permission`, `Specialty` | `/api/v1/auth/*`, `/api/v1/doctors/*`, `/api/v1/users/*` | **B — REUSE + EXTEND** (Add multi-org & business-unit scoping) |
| **appointment** | Appointment scheduling, slot booking, queue status | `Appointment`, `TimeSlot`, `QueueItem` | `/api/v1/appointments/*` | **B — REUSE + EXTEND** (Extend for general appointments & site visits) |
| **billing** | Invoicing, payments, line items, checkout | `Invoice`, `BillingItem`, `Payment` | `/api/v1/billing/*` | **B — REUSE + EXTEND** (Generalize into Group Finance domain) |
| **budget** | Departmental budgets, expense tracking, cost centers | `Budget`, `CostCenter`, `BudgetItem` | `/api/v1/budget/*` | **B — REUSE + EXTEND** (Connect to Business Unit cost centers) |
| **career** | Job listings, candidate applications, recruitment | `JobPosting`, `Applicant`, `Resume` | `/api/v1/careers/*` | **B — REUSE + EXTEND** (Extend for group HR recruitment & visa candidates) |
| **clinical** | AYUSH pods, clinical notes, doctor onboarding, diagnostics | `ClinicalNote`, `DiagnosticReport`, `AYUSHPod` | `/api/v1/clinical/*` | **A — REUSE AS-IS** (Strict HIPAA/data isolation retained) |
| **communication** | Multi-channel chat, WhatsApp webhooks, SMS, Email logs | `Message`, `ChatSession`, `WaLog`, `NotificationLog` | `/api/v1/chat/*`, `/api/v1/communication/*` | **B — REUSE + EXTEND** (Central group communication gateway) |
| **content** | CMS posts, blogs, FAQs, media, testimonials | `BlogPost`, `FAQ`, `Testimonial`, `PageContent` | `/api/v1/blog/*`, `/api/v1/content/*` | **B — REUSE + EXTEND** (Extend into Multi-Site Multi-Brand CMS) |
| **corporate** | B2B accounts, enterprise subscriptions, corporate contracts | `CorporateAccount`, `CorporateSubscription` | `/api/v1/corporate/*` | **B — REUSE + EXTEND** (Extend into B2B client management for IT Consulting) |
| **devi** | AI clinical assistant, diagnostic engine, RAG service | `DeviSession`, `ClinicalPrompt`, `Embedding` | `/api/v1/devi/*` | **B — REUSE + EXTEND** (Evolve into Group AI Assistant backend) |
| **document** | Document storage metadata, signed URLs, file upload | `Document`, `Attachment`, `FileMeta` | `/api/v1/documents/*` | **A — REUSE AS-IS** (Central document repository for all BUs) |
| **emr** | Electronic Medical Records, encrypted PHI, prescriptions | `Prescription`, `EMRRecord`, `HealthMetric` | `/api/v1/emr/*` | **A — REUSE AS-IS** (Retain strict clinical authorization) |
| **finance** | Ledger, multi-currency, bank reconciliation, GST/TDS | `LedgerEntry`, `BankStatement`, `TaxRule`, `Wallet` | `/api/v1/finance/*` | **B — REUSE + EXTEND** (Central Group Financial Engine) |
| **franchise** | Franchise outlets, royalty tracking, store licenses | `FranchiseOutlet`, `RoyaltyFee` | `/api/v1/franchise/*` | **B — REUSE + EXTEND** (Generalize into Multi-Unit Partner Management) |
| **hospital** | Facilities, clinics, wards, beds, departments | `Hospital`, `Clinic`, `Department`, `Bed` | `/api/v1/hospital/*` | **A — REUSE AS-IS** (Healthcare BU physical infrastructure) |
| **hr** | Employees, payroll, shifts, attendance, employment types | `Employee`, `PayrollRun`, `Shift`, `Attendance` | `/api/v1/hr/*` | **B — REUSE + EXTEND** (Extend for Group-wide HR & commissions) |
| **insurance** | Claims, policy verification, third-party admins | `Claim`, `Policy`, `Coverage` | `/api/v1/insurance/*` | **A — REUSE AS-IS** (Healthcare insurance billing) |
| **lab** | Lab test catalog, packages, sample collection | `LabTest`, `TestPackage`, `Sample` | `/api/v1/lab/*` | **A — REUSE AS-IS** (Healthcare diagnostic lab ops) |
| **loyalty** | Point rewards, tiering, referral bonuses | `LoyaltyPoint`, `RewardTier` | `/api/v1/loyalty/*` | **B — REUSE + EXTEND** (Cross-business group loyalty engine) |
| **payout** | Provider settlements, commission calculations, payouts | `PayoutRun`, `CommissionSplit` | `/api/v1/payout/*` | **B — REUSE + EXTEND** (Extend for Real Estate & E-comm payouts) |
| **pharmacy** | SKUs, medicine inventory, courier dispatch, orders | `PharmacyItem`, `InventorySKU`, `Order` | `/api/v1/pharmacy/*` | **B — REUSE + EXTEND** (E-commerce / POD fulfillment foundation) |
| **questionnaire** | Intake surveys, feedback forms, diagnostic forms | `SurveyTemplate`, `FormResponse` | `/api/v1/questionnaire/*` | **A — REUSE AS-IS** (Dynamic lead & patient intake forms) |
| **rating** | Service reviews, doctor ratings, feedback | `Rating`, `Review` | `/api/v1/ratings/*` | **A — REUSE AS-IS** (Cross-site rating and social proof) |
| **referral** | Attribution, referral links, coupon management | `ReferralCode`, `AttributionLog` | `/api/v1/referrals/*` | **B — REUSE + EXTEND** (Group Marketing Attribution engine) |
| **refill** | Automated recurring subscription order schedules | `RefillSchedule`, `RefillOrder` | `/api/v1/refills/*` | **B — REUSE + EXTEND** (E-commerce & SaaS recurring orders) |
| **store** | Products, categories, cart, e-commerce orders | `Product`, `Category`, `Cart`, `StoreOrder` | `/api/v1/store/*` | **B — REUSE + EXTEND** (E-commerce & POD marketplace baseline) |
| **subscription** | Member plans, recurring billing | `SubscriptionPlan`, `UserSubscription` | `/api/v1/subscription/*` | **B — REUSE + EXTEND** (Software AMC & membership billing) |
| **support** | Support tickets, SLA monitoring, resolution tracking | `SupportTicket`, `TicketComment`, `SLA` | `/api/v1/support/*` | **B — REUSE + EXTEND** (Central Group Support Desk) |
| **zone** | Geofencing, delivery zones, coverage maps | `Zone`, `CoveragePolygon` | `/api/v1/zones/*` | **A — REUSE AS-IS** (Service delivery geofencing) |
| **analytics** | Event telemetry, audit reporting, anomaly alerts | `TelemetryEvent`, `AuditLog`, `Anomaly` | `/api/v1/analytics/*` | **B — REUSE + EXTEND** (Group Business Intelligence engine) |

---

## Key Backend Infrastructure Components

1. **Audit & Anomaly (`internal/audit`)**: GORM plugin intercepting all DB writes for audit trails.
2. **Event Bus (`internal/events`)**: Redis Streams based asynchronous domain event dispatching.
3. **Queue & Workers (`internal/dispatch`, `internal/cron`)**: SLA monitoring, batch optimization, background jobs.
4. **Middleware (`internal/middleware`)**: JWT authentication, RBAC authorization, rate limiting, Sentry error logging.

---

## Frontend Next.js Inventory (`client/app/`)

- **Total App Routes**: 70+ route directories (Admin, Doctor, Patient, Pharmacy, EMR, Billing, HR, Support, CMS, Marketing).
- **Architecture**: Next.js 14 App Router with Tailwind CSS, Lucide icons, React Query, Zustand state management.
- **Key Modules Identified**:
  - Administrative Console (`/admin`)
  - Operational Dashboards (`/dashboard`, `/doctor`, `/staff`, `/patient`)
  - EMR & Medical Records (`/emr`, `/medical-records`, `/prescriptions`)
  - Financial Portals (`/billing`, `/finance`, `/wallet`, `/payout`)
  - Customer Service & Chat (`/chat`, `/inbox`, `/support`, `/qa`)
  - CMS Content Management (`/blog`, `/careers`, `/press`, `/faq`)
  - Task Execution (`/tasks`)
