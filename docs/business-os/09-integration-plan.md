# 09 - System Integration Plan

## Integration Principles
All internal domain communication must use in-process Go interfaces or Redis Streams domain events. External third-party APIs must be encapsulated behind provider-independent adapter interfaces.

---

## 1. Internal Platform Integrations

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  SHARED CRM  │ ──► │JOURNEY ENGINE│ ──► │ TASKS ENGINE │ ──► │  DOCUMENTS   │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │                    │
       ▼                    ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    REDIS STREAMS DOMAIN EVENT BUS                           │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                ┌──────────────────────┼──────────────────────┐
                ▼                      ▼                      ▼
        ┌───────────────┐      ┌───────────────┐      ┌───────────────┐
        │GROUP FINANCE  │      │ COMMUNICATIONS│      │ CUSTOMER 360  │
        └───────────────┘      └───────────────┘      └───────────────┘
```

- **CRM → Journey**: Converting a lead automatically instantiates a Journey for the target Business Unit.
- **Journey → Tasks**: Advancing a stage automatically triggers task creation assigned to team roles.
- **Tasks → Communication**: Task escalation triggers SMS/WhatsApp alerts via the communication gateway.
- **Finance → Ledger**: Payments fire `payment.received` events to update cost-center profitability.

---

## 2. External Third-Party Adapters

| Third-Party System | Functional Scope | Adapter Interface | Isolation & Error Fallback |
| :--- | :--- | :--- | :--- |
| **Etsy API** | E-commerce product listing & order retrieval | `integrations/etsy` | Circuit breaker, background polling worker, Redis rate-limiting. |
| **POD Providers** | Print-on-demand fulfillment & shipping updates | `integrations/pod` | Outbox pattern for order submission; webhook handler for tracking. |
| **Payment Gateways** | Razorpay & Stripe integration | `services/stripe`, `services/razorpay` | Unified `PaymentProvider` interface with automatic retry logic. |
| **WhatsApp Cloud API**| Interactive WhatsApp messages & lead intake | `handlers/whatsapp_webhook` | Outbox queue pattern with status callbacks. |
| **Resend Email API** | Transactional & marketing emails | `services/email` | Asynchronous worker dispatch with DNS fallback checks. |
| **LiveKit Infrastructure**| Real-time video consultations | `internal/livekit` | Webhook quality ingestion and room token generation. |
