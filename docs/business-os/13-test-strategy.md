# 13 - Comprehensive Test Strategy

## Testing Philosophy
No feature is complete without automated verification. Testing must validate both individual domain logic and end-to-end multi-domain workflows across all 5 Business Units.

---

## Test Hierarchy & Existing Test Suite Audit

```
┌─────────────────────────────────────────────────────────────┐
│                 E2E WORKFLOW TESTS                          │
│ (master_30_domains_e2e_test.go, client/e2e)                 │
├─────────────────────────────────────────────────────────────┤
│              API INTEGRATION & PERMISSION TESTS             │
│ (core-go/tests/integration/*_test.go)                      │
├─────────────────────────────────────────────────────────────┤
│             DOMAIN SERVICE & REPOSITORY TESTS               │
│ (core-go/internal/domain/*/*_test.go)                       │
├─────────────────────────────────────────────────────────────┤
│                UNIT TESTS & SCHEMA VALIDATION               │
│ (database/schema_test.go, middleware/auth_test.go)         │
└─────────────────────────────────────────────────────────────┘
```

### Existing Backend Test Suites:
1. `master_30_domains_e2e_test.go`: End-to-end verification covering all 30 Go domains.
2. `finance_integration_test.go`: Ledger split, bank reconciliation, and wallet guard tests.
3. `uheid_integration_test.go`: User identity & doctor verification tests.

---

## Business Unit E2E Test Verification Workflows

1. **Travel & Visa**:
   - `Lead Created → Requirement Collected → Quotation → Visa Document Upload → Payment → Booking Confirmed`
2. **Healthcare**:
   - `Patient Registration → Appointment Booked → Consultation Encounter → EMR Prescription → Pharmacy Dispatch`
3. **Real Estate**:
   - `Buyer Enquiry → Property Match → Site Visit Task → Offer & Booking → Document Upload → Closed Transaction`
4. **E-Commerce / POD**:
   - `Order Received → Contribution Margin Calculated → POD Supplier Dispatched → Courier Tracking → Fulfillment`
5. **Software Consulting**:
   - `Opportunity → Proposal/SOW → Project Created → Milestone Task → Timesheet → Invoiced → Paid`
