# 08 - Target Architecture Design

## Architecture Philosophy
> **ONE GROUP → MULTIPLE BUSINESSES → MULTIPLE BRANDS → SHARED CORE → SPECIALIZED BUSINESS MODULES**

The target system is a modular monolith in Go (`core-go`) coupled with a multi-tenant Next.js application (`client`) that dynamically resolves multi-brand public websites and provides a unified administrative Business OS console.

---

## Technical Specifications

### 1. Multi-Business Scoping Model
Every request passing through Fiber middleware is decorated with context:
- `X-Organization-Id`: Identifies the top-level group or subsidiary organization.
- `X-Business-Unit-Id`: Identifies the active business unit (`1: Travel`, `2: Healthcare`, `3: Real Estate`, `4: E-commerce`, `5: Software`).
- `X-Site-Id`: Identifies the active public website domain context.

```go
type BusinessContext struct {
    OrganizationID uint
    BusinessUnitID uint
    SiteID         uint
    UserID         uint
    UserRole       string
    Permissions    []string
}
```

### 2. Unified Customer 360 Architecture
Customer 360 acts as a central aggregator that listens to domain events (`lead.created`, `booking.completed`, `payment.received`, `appointment.created`) via Redis Streams and compiles a unified timeline.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              CUSTOMER 360 AGGREGATOR                            │
├─────────────────────────────────────────────────────────────────────────────────┤
│ • Basic Profile (Name, Email, Phone, Addresses, Associated Contacts)            │
│ • Cross-BU Relationships (Travel, Healthcare, Real Estate, E-comm, Consulting)  │
│ • Unified Interaction Timeline (Calls, Emails, WhatsApp, Web Visits)             │
│ • Consolidated Financial Summary (Invoices, Paid, Outstanding Balances)         │
│ • Marketing Attribution (First Touch, Last Touch, Campaign, Channel)            │
├─────────────────────────────────────────────────────────────────────────────────┤
│ 🔒 Clinical Security Guard: Clinical PHI requires `healthcare.clinical.read`    │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 3. Journey Engine (State Machine Architecture)
Generic workflow orchestration layer evolving the existing `dispatch` engine.

```
JOURNEY TEMPLATE (e.g. "Visa Processing", "Real Estate Deal", "Software SOW")
   │
   ├──> JOURNEY STAGE (e.g. Lead → Requirement → Quote → Documents → Complete)
   │       │
   │       ├──> TRANSITION RULES (Conditions: BU, Service, Amount, Role)
   │       │
   │       └──> AUTOMATED ACTIONS (Assign Task, Send Email/WA, Change Stage, Event)
   │
   └──> JOURNEY INSTANCE (Tracked record attached to CustomerID & BusinessUnitID)
```

### 4. Multi-Brand Public Website Engine
Next.js middleware detects incoming request hostname and maps domain to site configuration:

```typescript
// client/middleware.ts
export async function middleware(req: NextRequest) {
  const hostname = req.headers.get("host");
  const siteConfig = await getSiteConfigByDomain(hostname);
  
  const response = NextResponse.next();
  response.headers.set("x-site-id", siteConfig.id);
  response.headers.set("x-business-unit", siteConfig.businessUnit);
  return response;
}
```

Dynamic branding tokens (Logo, Colors, Fonts, Layouts, Content) are rendered seamlessly per domain without duplicating code.
