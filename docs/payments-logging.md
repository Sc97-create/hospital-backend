# Payments Logging Plan

Implemented. Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for other domains. Request-level HTTP envelope logs already exist. This plan covers **domain payment events** (create link / pending, webhook, confirm, fulfillment).

## Layer logging (service + repo)

| Layer | What we log |
|-------|-------------|
| **Controller** | Entry / invalid request; HTTP status mapping; pass `*zap.Logger` |
| **Service** | Business outcomes (`attempt` / `success` / `failed` + stable `reason`) |
| **Repo** | **DB errors only** (`op` + `error`) — no success INFO spam |

Non-HTTP callers (billing) pass the request logger into payment service methods.

Package: `internal/payments` (+ Razorpay providers, `appinit/payment_fulfillment.go`)  
Routes: `/api/v1/payment/*` (see `RegisterPaymentRoutes`)  
Related entry (billing): checkout / retry-payment-link — see `docs/billing-logging.md`

---

## 1. Scope

| Area | Endpoint / entry | Files |
|------|------------------|-------|
| Razorpay webhook | `POST /v1/payment/webhook` | `controllers.go`, `webhook_service.go`, gateway, attempts, fulfillment |
| Manual confirm (cash/QR) | `POST /v1/payment/confirm` | `controllers.go`, `payments.services.go`, fulfillment |
| Create link payment | Called from billing checkout | `payments.services.go`, Razorpay client, attempts repo |
| Create pending payment | Cash/QR checkout | `payments.services.go` |
| Retry link payment | Billing retry | `payments.services.go` |
| Fulfill paid invoice | Webhook paid + confirm | `fulfillment_service.go` + adapter |
| Notify payment received | After fulfill commit | `fulfillment_service.go` |

**Auth note:** payment routes are **not** behind JWT. Webhook is provider-authenticated via signature. Confirm mutating inventory with no auth is a product risk (follow-up).

**Out of scope for this pass**

- Full billing checkout logs (billing doc owns attempt/success at invoice layer)
- Refunds model (unused)
- JWT on confirm (follow-up)
- Changing Razorpay HTTP client beyond sanitized errors

---

## 2. Security / PII rules (non-negotiable)

**Never log**

- Razorpay `ApiKey`, `ApiSecret`, `WebhookSecret`
- `X-Razorpay-Signature` or computed HMAC
- Full webhook raw body / `provider_response` JSON
- Payment link URLs (`payment_url` / `short_url`)
- Customer name / email / mobile (create payload / provider_request)
- UPI VPA / card method payloads
- Notification `Data` maps
- Raw gateway HTTP response bodies embedded in `error` strings (sanitize before `zap.Error`)

**Safe to log**

- `payment_id`, `payment_attempt_id`, `invoice_id`, `prescription_id`, `patient_id`
- `provider_link_id`, `provider_order_id`, `provider_payment_id`
- `event_type`, `provider` (`razorpay`)
- Status enums: payment / attempt / invoice / source / channel
- `attempt_no`, `idempotency_key` (useful for support)
- Counts: `item_count`, `movement_count`
- Booleans: `claimed`, `idempotent_replay`, `notification_enqueued`, `accept_partial`

**Borderline**

- **`amount` / `amount_paid` / currency** — open decision; `docs/logging.md` sketch logs `amount_paid`. Prefer omit on INFO or DEBUG only unless finance support requires INFO.

---

## 3. Propagation approach

```
RequestLogger
  → Controller: middleware.GetLogger(c)
  → PaymentsService / WebhookService / FulfillmentService: *zap.Logger first arg
  → Attempts / payment / webhook repos: DB errors only
  → Gateway: verify/parse outcomes without secrets
```

Fulfillment adapter already uses `logger.Log` for prescription/patient. When payments gains request loggers, prefer passing them through; until then `logger.Log` / `ensureLog(nil)` is OK.

**Recommendation:** required `log *zap.Logger` first arg on payment service methods. Add `internal/payments/log.go` with `ensureLog`. Prefer **not** expanding `IPaymentFulfillment` with logger in this pass (adapter injects `logger.Log`) — same as prescription option 2.

---

## 4. What to log — by flow

### 4.1 Webhook (`POST /payment/webhook`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| W1 | Controller | Request entered | INFO | `payment webhook received` | `provider=razorpay` (no body) |
| W2 | Gateway / Service | Missing / invalid signature | WARN | `payment webhook rejected` | `reason=missing_signature` / `signature_invalid` |
| W3 | Service | Parse fail | ERROR | `payment webhook failed` | `reason=parse`, `error` (sanitized) |
| W4 | Service | Event parsed | INFO | `payment webhook event` | `event_type`, `provider_link_id` |
| W5 | Attempts | Already claimed (duplicate) | WARN | `payment webhook duplicate skipped` | `reason=already_claimed`, `provider_link_id` / `payment_attempt_id` |
| W6 | Service | Persist webhook event fail | ERROR | `payment webhook failed` | `reason=persist_event`, `error` |
| W7 | Service | Partial payment skipped | WARN | `payment webhook partial skipped` | `reason=partial_amount`, ids |
| W8 | Service | Load payment fail | ERROR | `payment webhook failed` | `reason=load_payment`, `error` |
| W9 | Service | Fulfill fail | ERROR | `payment webhook failed` | `reason=fulfill`, `invoice_id`, `error` |
| W10 | Service | Paid success | INFO | `payment webhook paid success` | `event_type`, `invoice_id`, `payment_id`, `prescription_id` |
| W11 | Service | Cancelled / expired | WARN | `payment webhook link closed` | `event_type`, `provider_link_id`, `status` |
| W12 | Service | Unhandled event type | WARN | `payment webhook unhandled event` | `reason=unhandled_event_type`, `event_type` |
| W13 | Service | Notify fail after commit | ERROR | `payment notification failed` | `invoice_id`, `reason=notification`, `error` |
| W14 | Service | Notify enqueued | INFO | `payment notification enqueued` | `invoice_id`, `notification_enqueued=true` |

### 4.1.1 Webhook HTTP / behavior quirks (fix when implementing)

| Issue today | Recommended behavior |
|-------------|----------------------|
| Duplicate claim → service error → **500** (Razorpay retries) | Treat duplicate as success ack: **200** + WARN `already_claimed` |
| Unknown `EventType` → `(false, nil)` → controller **401** | **200** + WARN `unhandled_event_type` |
| Notify fail after fulfill commit → **500** | **200** + ERROR log (money already applied) |
| Controller returns raw `err.Error()` on 500 | Stable client message; detail in logs only |

Signature fail → **401** (keep).

---

### 4.2 Manual confirm (`POST /payment/confirm`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| M1 | Controller | Missing `invoice_id` / `organisation_id` / `payment_mode` | WARN | `payment confirm request invalid` | `field` |
| M2 | Controller | Entered OK | INFO | `payment confirm attempt` | `invoice_id`, `organisation_id`, `payment_mode` |
| M3 | Service | Source mismatch / not cash-qr | WARN | `payment confirm failed` | `invoice_id`, `organisation_id`, `payment_mode`, `reason=source_mismatch` / `unsupported_payment_mode` |
| M4 | Service | Payment / invoice not found (incl. wrong org) | WARN | `payment confirm failed` | `invoice_id`, `organisation_id`, `reason=not_found` |
| M5 | Service | Fulfill fail | ERROR | `payment confirm failed` | `invoice_id`, `organisation_id`, `reason=fulfillment_failed`, `error` |
| M6 | Service | Notify fail after commit | ERROR | `payment notification failed` | `invoice_id`, `reason=notification` |
| M7 | Service | Success | INFO | `payment confirm success` | `invoice_id`, `organisation_id`, `payment_id`, `payment_mode`, `notification_enqueued` |

**Mandatory:** `invoice_id`, `payment_mode` (`cash`/`qr`).  
**Optional ignored:** `transaction_reference`.

### 4.2.1 Frontend-facing confirm errors

| Case | HTTP (proposed) | API `error` |
|------|-----------------|-------------|
| Invalid body | `400` | `invalid request` |
| Not found | `404` | `payment not found` / `invoice not found` |
| Mode mismatch | `400` | `invalid request` |
| Fulfillment / DB | `500` | `payment confirm failed` |

**Decision:** today almost everything is **409**. Recommend mapping as above.

---

### 4.3 Create link payment (billing caller)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| L1 | Service | Missing idempotency | WARN | `payment link create failed` | `reason=idempotency_required` |
| L2 | Service | Idempotent replay | INFO | `payment link create success` | `payment_id`, `invoice_id`, `idempotent_replay=true`, `has_payment_url` |
| L3 | Gateway | Create fail | ERROR | `payment link create failed` | `invoice_id`, `reason=gateway_create`, sanitized `error` |
| L4 | Service | DB payment / attempt / Rx status fail | ERROR | `payment link create failed` | `invoice_id`, `reason=db_create` / `db_attempt` / `prescription_status`, `error` |
| L5 | Service | Success | INFO | `payment link create success` | `payment_id`, `invoice_id`, `prescription_id`, `provider_link_id`, `has_payment_url` |

Never log customer PII or URL value.

---

### 4.4 Create pending payment (cash/QR)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| N1 | Service | Fail | ERROR | `payment pending create failed` | `invoice_id`, `payment_mode`, `reason`, `error` |
| N2 | Service | Success | INFO | `payment pending create success` | `payment_id`, `invoice_id`, `payment_mode` |

---

### 4.5 Retry link payment

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| T1 | Service | Not link source / missing IDs | WARN | `payment link retry failed` | `invoice_id`, `reason=unsupported_payment_mode` / `missing_ids` |
| T2 | Service | Reuse pending attempt | INFO | `payment link retry success` | `invoice_id`, `reason=reuse_pending_attempt`, `has_payment_url` |
| T3 | Service | Gateway / DB fail | ERROR | `payment link retry failed` | `invoice_id`, `reason=gateway_create` / `db_*`, `error` |
| T4 | Service | New link success | INFO | `payment link retry success` | `payment_id`, `invoice_id`, `provider_link_id`, `has_payment_url` |

---

### 4.6 Fulfillment (`FulfillPaidInvoice`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| F1 | Service | Inventory / stock / dispense / parent / invoice fail | ERROR | `payment fulfillment failed` | `invoice_id`, `reason=inventory` / `stock_mvmt` / `dispense` / `parent_status` / `invoice_status`, `error` |
| F2 | Service | Success | INFO | `payment fulfillment success` | `invoice_id`, `prescription_id`, `item_count`, `movement_count` |

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/payments/controllers.go` | Webhook/confirm entry; no body/signature logs; HTTP mapping |
| `internal/payments/payments.services.go` | Create/retry/pending/confirm |
| `internal/payments/webhook_service.go` | Verify → claim → event branches → fulfill/notify |
| `internal/payments/fulfillment_service.go` | Dispense/stock/invoice paid + notify handoff |
| `internal/payments/payment-attempts.services.go` (+ repo) | Claim/duplicate |
| `internal/payments/payments.repository.go` | DB errors |
| `internal/payments/webhook_repository.go` | Persist event errors |
| `internal/payments/providers/razorpay/gateway.go` | Verify/parse (no secrets) |
| `internal/payments/providers/razorpay/client.go` | Sanitized gateway errors only |
| `internal/payments/log.go` | `ensureLog` (new) |
| `appinit/payment_fulfillment.go` | Keep injecting logger into deps |
| `shared/error/structures.go` | Optional payment sentinels |

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | RequestLogger |
| `payment_id`, `payment_attempt_id` | Create / webhook / confirm |
| `invoice_id`, `prescription_id`, `patient_id` | When known |
| `provider`, `provider_link_id`, `provider_payment_id`, `provider_order_id` | Gateway correlation |
| `event_type` | Webhook |
| `payment_mode` / `source` / `channel` | Create / confirm |
| `idempotency_key` | Create / retry |
| `has_payment_url`, `idempotent_replay`, `notification_enqueued` | Flags |
| `item_count`, `movement_count` | Fulfillment |
| `reason`, `op`, `error` | Failures |

Example webhook paid:

```json
{
  "level": "info",
  "msg": "payment webhook paid success",
  "request_id": "...",
  "provider": "razorpay",
  "event_type": "payment_link.paid",
  "invoice_id": "...",
  "payment_id": "...",
  "prescription_id": "..."
}
```

---

## 7. What we will **not** add in this pass

- Logging secrets, signatures, webhook bodies, payment URLs, customer PII
- Billing checkout domain logs (billing doc)
- JWT on confirm
- Refunds API

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on payment routes | Done (global) |
| Payments controller domain logs | Done |
| Create/retry/pending/confirm service logs | Done |
| Webhook service domain logs | Done |
| Fulfillment / notify logs | Done |
| Duplicate webhook → 200 ack | Done (`ClaimForProcessing` returns `claimed=false, nil`) |
| Unhandled event → 200 ack | Done |
| Notify fail after paid commit → 200 + ERROR | Done |
| Client-safe HTTP mapping on confirm | Done (400/404/500) |
| JWT on confirm | Not present (follow-up) |

---

## 9. Implementation order (after approval)

1. `log.go` + optional sentinels  
2. Fix webhook duplicate / unhandled / post-commit notify HTTP behavior alongside logs  
3. Webhook path (highest risk)  
4. Confirm + fulfillment + notify  
5. Create link / pending / retry (called from billing)  
6. Sanitize gateway errors for logging  
7. Optional: JWT on confirm  

---

## 10. Open decisions for review

1. **Log amounts on INFO?** — **Recommend: omit** (or DEBUG only)
2. **Never log webhook body / signature / payment URL?** — **Recommend: yes**
3. **Duplicate webhook: 200 + WARN?** — **Recommend: yes** (fix claim handling)
4. **Unhandled event: 200 + WARN?** — **Recommend: yes**
5. **Notify fail after paid commit: 200 + ERROR?** — **Recommend: yes**
6. **Confirm HTTP mapping 400/404/500?** — **Recommend: yes**
7. **Log `idempotency_key`?** — **Recommend: yes** (support)
8. **Expand `IPaymentFulfillment` with logger?** — **Recommend: no this pass** (adapter `logger.Log`)
9. **JWT on confirm this pass?** — **Recommend: follow-up**
10. **Logger style:** required `log *zap.Logger` first arg — **yes**

---

## 11. Approval

Once this list looks right, implement in §9 order. Coordinate with `docs/billing-logging.md` so checkout vs payment-link create logs don’t double-noise (billing = invoice attempt/success; payments = gateway/payment row outcomes).
