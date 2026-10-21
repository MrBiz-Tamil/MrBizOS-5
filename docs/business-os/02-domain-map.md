# 02 - Target & Existing Domain Map

## Architecture Overview

```
                                    ┌─────────────────────────────────────────────────────────────┐
                                    │                     GROUP BUSINESS OS                       │
                                    └──────────────────────────────┬──────────────────────────────┘
                                                                   │
    ┌──────────────────────────────────────────────────────────────┴──────────────────────────────────────────────────────────────┐
    │                                                     SHARED CORE ENGINE                                                      │
    ├─────────────────────────────┬──────────────────────────────┬──────────────────────────────┬─────────────────────────────────┤
    │ Identity & RBAC             │ Shared CRM & Lead Pipeline   │ Customer 360 Aggregator      │ Journey & Workflow Engine       │
    │ (existing: identity, auth)  │ (existing: user, referral)   │ (existing: patient, user)    │ (existing: dispatch, queue)     │
    ├─────────────────────────────┼──────────────────────────────┼──────────────────────────────┼─────────────────────────────────┤
    │ Task Management Engine      │ Document Storage Engine      │ Communication Gateway        │ Finance & Accounting Engine     │
    │ (existing: tasks, hr)       │ (existing: document)         │ (existing: communication)    │ (existing: finance, billing)    │
    ├─────────────────────────────┼──────────────────────────────┼──────────────────────────────┼─────────────────────────────────┤
    │ Multi-Brand CMS Engine      │ Marketing & Attribution      │ Group Intelligence & Audit   │ AI Assistant Service (Devi)     │
    │ (existing: content, blog)   │ (existing: referral, rating) │ (existing: analytics, audit) │ (existing: devi, embedding)     │
    └─────────────────────────────┴──────────────────────────────┴──────────────────────────────┴─────────────────────────────────┘
                                                                   │
    ┌─────────────────────────┬───────────────────┬────────────────┴───────────┬─────────────────────────┬────────────────────────┐
    │                         │                   │                            │                         │                        │
    ▼                         ▼                   ▼                            ▼                         ▼                        ▼
┌──────────────────────┐  ┌───────────────┐  ┌──────────────────────────┐  ┌─────────────────────────┐  ┌───────────────────────┐
│ TRAVEL, TOURS & VISA │  │  HEALTHCARE   │  │       REAL ESTATE        │  │ E-COMMERCE & POD        │  │  SOFTWARE CONSULTING  │
├──────────────────────┤  ├───────────────┤  ├──────────────────────────┤  ├─────────────────────────┤  ├───────────────────────┤
│ • Destinations       │  │ • Patients    │  │ • Property Inventory     │  │ • Product Catalog       │  │ • Client Accounts     │
│ • Tour Packages      │  │ • Doctors     │  │ • Units & Pricing        │  │ • Marketplaces (Etsy)   │  │ • Project & SOW       │
│ • Visa Checklists    │  │ • Clinics     │  │ • Buyers & Sellers       │  │ • POD Supplier Adapters │  │ • Sprints & Milestones│
│ • Immigration Workflows││ • EMR/PHI     │  │ • Site Visits & Match    │  │ • Fulfillment & Courier │  │ • Timesheets & Support│
│ • Supplier Bookings  │  │ • Diagnostics │  │ • Deals & Commissions    │  │ • Contribution Margin   │  │ • Software AMC        │
└──────────────────────┘  └───────────────┘  └──────────────────────────┘  └─────────────────────────┘  └───────────────────────┘
```

---

## Domain Alignment Matrix

| Shared Platform Layer | Existing Domain Codebase (`core-go/internal/domain/`) | Extension Strategy |
| :--- | :--- | :--- |
| **Organization & BU Boundary** | `identity`, `corporate`, `hospital` | Add `OrganizationID`, `BusinessUnitID`, `SiteID` context to session and database scoping middleware. |
| **CRM & Leads** | `identity`, `user`, `referral`, `corporate` | Generalize User entity to support Contacts & Leads across all 5 Business Units with unified pipeline tracking. |
| **Customer 360** | `patient`, `user`, `emr`, `finance`, `document` | Extend aggregation to stitch timeline of interactions, documents, financial transactions, and BU-specific relationships without breaching clinical PHI boundaries. |
| **Journey Engine** | `dispatch`, `cron`, `queue`, `appointment` | Evolve existing `dispatch` state machine into a generic `Journey Engine` supporting custom stage definitions, state transitions, conditions, and automated actions. |
| **Task Management** | `tasks` (frontend), `hr` (shifts/tasks) | Standardize backend task model linking Tasks to `OrganizationID`, `BusinessUnitID`, `CustomerID`, `JourneyID`, `OwnerID`, `AssigneeID`, `Status`, `Priority`, `DueDate`. |
| **Finance Engine** | `finance`, `billing`, `budget`, `payout`, `ledger` | Unify Ledger, Multi-currency, Tax (GST/TDS), Wallet, and Cost-Center Profitability by tagging every revenue/expense with `BusinessUnitID`. |
| **Documents** | `document` | Centralize storage metadata with private S3 buckets, pre-signed URLs, strict RBAC, and document classification tags (`Passport`, `ClinicalReport`, `Contract`, `PropertyDeed`). |
| **Communication Gateway**| `communication`, `services/email`, `services/notification`, `whatsapp_webhook` | Abstract vendor adapters (Resend, Twilio, WhatsApp Cloud API) behind a single provider-agnostic messaging interface. |
| **Multi-Site CMS** | `content`, `blog` | Upgrade content model to support `SiteID` scoping, dynamic section JSON payloads, custom domain mapping, SEO metadata, and form builders. |
| **AI Assistant** | `devi`, `embedding`, `services/ocr` | Connect Devi AI engine to central search and RAG knowledge base for internal CEO metrics and document analysis. |
