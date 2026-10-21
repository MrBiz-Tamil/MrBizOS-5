# 15 - Group Business OS Comprehensive Module Map

## Business Unit Module Mapping Matrix

| Business Unit | Specialized Domain Entities | Shared Core Platform Services Reused | Backend Go Domain Packages | Next.js Frontend Routes |
| :--- | :--- | :--- | :--- | :--- |
| **1. Travel, Tours & Visa** | Destinations, Tour Packages, Visa Checklists, Immigration Applications, Travellers | CRM, Customer 360, Journey, Tasks, Documents, Communication, Finance, CMS | `domain/career`, `domain/document`, `domain/finance`, `domain/communication` + `domain/travel` [NEW] | `/admin/travel`, `/admin/visa`, `/tours`, `/visa-status` |
| **2. Healthcare** | Patients, Doctors, Clinics, Encounters, EMR Records, Prescriptions, Lab Tests, Pharmacy SKUs | CRM, Customer 360, Tasks, Documents, Communication, Finance, CMS, AI | `domain/identity`, `domain/clinical`, `domain/emr`, `domain/hospital`, `domain/pharmacy`, `domain/lab` | `/admin/healthcare`, `/doctor`, `/patient`, `/emr`, `/pharmacy` |
| **3. Real Estate** | Property Inventory, Units, Site Visits, Buyers, Sellers, Property Deals | CRM, Customer 360, Journey, Tasks, Documents, Finance, CMS | `domain/corporate`, `domain/document`, `domain/finance`, `domain/task` + `domain/realestate` [NEW] | `/admin/realestate`, `/properties`, `/site-visits` |
| **4. E-Commerce & POD** | Products, POD Suppliers, Etsy Sync, Orders, Contribution Margin | Tasks, Documents, Finance, Communication, Shipping Logistics | `domain/store`, `domain/pharmacy`, `domain/shipping`, `domain/finance` | `/admin/ecommerce`, `/store`, `/orders`, `/shipping` |
| **5. Software Consulting** | Accounts, Proposals, SOW Contracts, Projects, Sprints, Timesheets, AMC | CRM, Customer 360, Tasks, Documents, Finance, Subscriptions | `domain/corporate`, `domain/subscription`, `domain/support`, `domain/hr` + `domain/consulting` [NEW] | `/admin/software`, `/projects`, `/timesheets`, `/support` |
