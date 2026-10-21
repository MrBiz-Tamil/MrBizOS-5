# 06 - Feature Gap Analysis

## Overview
This document evaluates the delta between the platform's current state and the target Group Business OS capabilities.

---

## Core Platform Capability Gaps

| Required Capability | Existing Implementation | Coverage % | Identified Gap | Recommended Action | Priority |
| :--- | :--- | :---: | :--- | :--- | :---: |
| **Multi-Org & Business Units** | Corporate domain handles B2B entities, but single-tenant DB model dominates. | 35% | Lack of explicit `BusinessUnitID` and `OrganizationID` context in session and API context. | Add Business Unit scoping middleware, session context, and database schema fields. | **P0** |
| **Shared CRM & Leads** | User profiles and doctor/patient models exist. | 50% | No unified `Lead` entity or sales pipeline stage tracking across BUs. | Extend CRM model with Lead stages, opportunity values, and multi-BU pipeline views. | **P0** |
| **Customer 360** | Patient profile aggregates medical records, appointments, and billing. | 60% | Aggregates only Healthcare data. Lacks unified cross-BU interaction timeline. | Create unified `Customer360` view aggregating all interactions while strictly isolating PHI. | **P0** |
| **Journey & Workflow** | `dispatch` state machine & background queue exist. | 40% | Configured for courier/delivery pipelines; lacks generic stage engine. | Abstract `dispatch` into a generic `Journey Engine` supporting custom stage definitions. | **P0** |
| **Task Management** | Frontend `/tasks` page exists with local state. `hr` shifts handle staff tasks. | 30% | No backend task persistent table or task assignment engine. | Build backend Task entity & repository supporting owner, assignee, due date, status. | **P0** |
| **Group Finance & Ledger** | `finance`, `billing`, `budget`, `payout` exist with double-entry ledger. | 75% | Invoices and ledger entries lack `BusinessUnitID` cost-center tags. | Tag all financial entries with `BusinessUnitID` and create Group Profitability view. | **P1** |
| **Multi-Site CMS** | `content` & `blog` domains exist with Markdown/HTML content. | 50% | Content is single-site; lacks `SiteID` domain mapping and section builder. | Add `SiteID` scoping to content tables and create site resolution middleware. | **P1** |
| **Communication Gateway** | Email, WhatsApp webhooks, and real-time WebSocket chat exist. | 80% | Communication logs are siloed; lack central customer timeline link. | Link all incoming/outgoing messages to central Customer 360 timeline. | **P1** |

---

## Business Unit Specific Gaps

### 1. Travel, Tours & Visa
- **Coverage**: 10% (Shared Auth, Billing, Documents reusable)
- **Gaps**: Missing Destination catalog, Tour package builder, Visa document checklist, Immigration status tracker.
- **Action**: Build Travel & Visa domain modules plugging into shared Journey, Task, Document, and Billing engines.

### 2. Healthcare
- **Coverage**: 90% (Comprehensive EMR, Appointments, Pharmacy, Lab, Doctors, Clinics already built)
- **Gaps**: Integration into unified Group Customer 360 with permission-gated clinical views.
- **Action**: Retain existing Healthcare engine; expose summarized relationship status to Customer 360.

### 3. Real Estate
- **Coverage**: 25% (Shared CRM, Finance, Documents, Tasks reusable)
- **Gaps**: Property inventory catalog, Unit availability tracker, Buyer/Seller matching, Site visit scheduling.
- **Action**: Build Real Estate property domain plugging into shared CRM, Journey, and Task engines.

### 4. E-Commerce & POD / Etsy
- **Coverage**: 50% (Pharmacy inventory, SKUs, Store cart, Shipping dispatch exist)
- **Gaps**: Multi-marketplace (Etsy) sync adapters, POD provider fulfillment adapters, net profit calculator.
- **Action**: Extend `store` & `pharmacy` inventory with POD supplier adapters and marketplace fee calculators.

### 5. Software Consulting
- **Coverage**: 35% (Corporate B2B, Budgeting, HR, Tickets exist)
- **Gaps**: SOW/Contract tracking, Project milestones, Sprint tasks, Timesheets, AMC subscription billing.
- **Action**: Build Software Project management domain reusing central Tasks, Documents, and Subscriptions.
