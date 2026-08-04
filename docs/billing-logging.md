# Billing Logging Plan

Review doc for what to log in the billing domain and where. Implementation follows the approved recommendations below.

Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for authentication, patient, appointments, and prescription. Request-level `request started` / `request completed` / `client error` / `request failed` already cover the HTTP envelope. This plan covers **domain billing events** on top of that.

Package: `internal/billing`  
Routes: `/api/v1/billing/*` (see `RegisterBillingRoutes`)

---

## 1. Scope

| Area | Endpoint / entry | Files |
|------|------------------|-------|
| Checkout (create invoice) | `POST /v1/billing/create` | `controllers.go`, `invoice.services.go`, `invoice-item.services.go`, repos |
| Get invoice by prescription | `GET /v1/billing/getInvoiceByPrescriptionID/:prescriptionID` | controllers + invoice service/repo |
| Retry payment link | `POST /v1/billing/invoices/:invoiceID/retry-payment-link` | controllers + invoice service → payments |
| Invoice items for fulfillment | `GetMedicineInventoryDetByInvoiceID` — payment fulfillment / webhook | `invoice-item.services.go` |
| Mark invoice paid | `UpdateInvoiceStatus` — fulfillment | `inovice.repository.go` (via adapter) |

**Auth note:** billing routes are **not** behind JWT today. RequestLogger still applies globally.

**Out of scope for this pass**

- Full payments / Razorpay / webhook domain logs (see `docs/payments-logging.md`) — only billing-side outcomes when calling payments
- Notification payload contents after pay
- Unused `supplier_id` product cleanup (note only)
- JWT on billing routes (follow-up)

---

## 2. Security / PII rules (non-negotiable)

**Never log**

- Full checkout body / `dispense_items` arrays / financial object dumps
- Patient name, email, phone (loaded for Razorpay customer)
- **`payment_url` / payment link URLs** — log `has_payment_url=true` only
- Idempotency key raw value — log `has_idempotency_key=true` (or hash later)
- Inventory pricing JSON / unit prices
- Batch numbers on INFO success

**Safe to log**

- `invoice_id`, `invoice_code`, `prescription_id`, `patient_id`, `cashier_id`, `organisation_id`
- `payment_mode` / channel (`link` / `cash` / `qr`)
- Invoice `status`
- Counts: `item_count`
- Booleans: `idempotency_replay`, `has_payment_url`, `has_idempotency_key`
- Stable `reason` enums

**Borderline**

- **Monetary amounts** (`total_amount`, tax, discount) — **omit on INFO** by default (commercial / health-adjacent); allow on ERROR only if ops need support (open decision)
- `medicine_id` — UUID OK; never medicine names
- `batch_no` — omit on success INFO

---

## 3. Propagation approach

```
RequestLogger (request_id, method, path, ip)
  → Controller: middleware.GetLogger(c)
  → InvoiceServ / InvoiceItemServ: *zap.Logger as first arg
  → Repository: log only on DB errors
```

Non-HTTP callers (fulfillment): pass request logger when available; else `logger.Log` / `ensureLog(nil)` via adapter (same pattern as prescription).

**Recommendation:** required `log *zap.Logger` first arg. Add `internal/billing/log.go` with `ensureLog`.

---

## 4. What to log — by flow

### 4.1 Checkout (`POST /billing/create`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| C1 | Controller | Missing `Idempotency-Key` / body parse / mandatory field | WARN | `invoice checkout request invalid` | `error`, `field` / `reason=missing_idempotency_key` |
| C2 | Controller | After parse OK | INFO | `invoice checkout attempt` | `prescription_id`, `patient_id`, `organisation_id`, `cashier_id`, `payment_mode`, `item_count`, `has_idempotency_key=true` |
| C3 | Service | Idempotency replay (existing payment) | INFO | `invoice checkout success` | `invoice_id`, `reason=idempotency_replay`, `has_payment_url`, `payment_mode` |
| C4 | Service | Unique invoice for prescription | WARN | `invoice checkout failed` | `prescription_id`, `reason=invoice_already_exists` |
| C5 | Item service | Validation fail | WARN | `invoice checkout failed` | `prescription_id`, `reason=medicine_not_in_prescription` / `qty_exceeds_remaining` / `insufficient_stock`, `medicine_id` if known |
| C6 | Service / Repo | DB create invoice/items fail | ERROR | `invoice checkout failed` | `prescription_id`, `reason=db_create` / `db_add_items`, `error` |
| C7 | Service | Patient lookup fail | WARN | `invoice checkout failed` | `patient_id`, `reason=patient_not_found` / `patient_lookup`, `error` |
| C8 | Service | Unsupported `payment_mode` | WARN | `invoice checkout failed` | `payment_mode`, `reason=unsupported_payment_mode` |
| C9 | Service | Payment create fail (invoice may already be committed) | ERROR | `invoice checkout failed` | `invoice_id`, `prescription_id`, `payment_mode`, `reason=payment_create`, `error` |
| C10 | Service | Success | INFO | `invoice checkout success` | `invoice_id`, `invoice_code`, `prescription_id`, `patient_id`, `organisation_id`, `payment_mode`, `item_count`, `has_payment_url` |
| C11 | Repository | DB errors | ERROR | `billing repo error` | `op`, `error` |

**Mandatory fields:** `Idempotency-Key` header; body `prescription_id`, `patient_id`, `cashier_id`, `payment_mode`, `organisation_id`, `financials` (+ sub/tax/total), `dispense_items` array key.  
**Optional:** `financials.discount_amount`; per-item stock/qty/price fields currently ignored on parse error → `0`.  
**Unused but required today:** `supplier_id` — still WARN on missing until product removes it.

**Notes**

- Invoice+items commit **before** payment create — C9 is a first-class ops signal (orphan unpaid invoice).
- Do not log amounts or payment URL.

### 4.1.1 Frontend-facing checkout errors

| Case | Internal `reason` | HTTP (proposed) | API `error` |
|------|-------------------|-----------------|-------------|
| Invalid / missing fields / idempotency | request invalid | `400` | `invalid request` |
| Invoice already exists | `invoice_already_exists` | `409` | `invoice already exists for this prescription` |
| Patient not found | `patient_not_found` | `404` | `patient not found` |
| Stock / qty validation | stock/qty reasons | `400` | short stable message |
| Unsupported mode | `unsupported_payment_mode` | `400` | `invalid request` |
| Payment create / DB | `payment_create` / `db_*` | `500` | `failed to create invoice` |

**Decision:** today most failures return **409**. Recommend mapping as above when implementing.

---

### 4.2 Get invoice (`GET .../getInvoiceByPrescriptionID/:prescriptionID`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| G1 | Controller | Missing param | WARN | `invoice get request invalid` | `reason=missing_prescription_id` |
| G2 | Controller | Entered OK | INFO | `invoice get attempt` | `prescription_id` |
| G3 | Service | Not found | WARN | `invoice get failed` | `prescription_id`, `reason=not_found` |
| G4 | Service / Repo | DB error | ERROR | `invoice get failed` | `prescription_id`, `reason=db_read`, `error` |
| G5 | Service | Success | INFO | `invoice get success` | `invoice_id`, `prescription_id`, `status`, `payment_mode` |

**Note:** join on `payments` — invoice without payment row looks like not found.

### 4.2.1 Frontend-facing get errors

| Case | HTTP | API `error` |
|------|------|-------------|
| Missing ID | `400` | `invalid request` |
| Not found | `404` | `invoice not found` |
| DB | `500` | `failed to fetch invoice` |

---

### 4.3 Retry payment link (`POST .../invoices/:invoiceID/retry-payment-link`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| R1 | Controller | Missing `invoiceID` / `Idempotency-Key` | WARN | `invoice retry payment link request invalid` | `reason` |
| R2 | Controller | Entered OK | INFO | `invoice retry payment link attempt` | `invoice_id`, `has_idempotency_key=true` |
| R3 | Service | Not found | WARN | `invoice retry payment link failed` | `invoice_id`, `reason=not_found` |
| R4 | Service | Not unpaid | WARN | `invoice retry payment link failed` | `invoice_id`, `status`, `reason=invoice_not_unpaid` |
| R5 | Service | Patient / payment retry fail | ERROR/WARN | `invoice retry payment link failed` | `invoice_id`, `reason=patient_lookup` / `payment_retry` / `unsupported_payment_mode`, `error` |
| R6 | Service | Success / replay | INFO | `invoice retry payment link success` | `invoice_id`, `has_payment_url`, optional `reason=idempotency_replay` / `reuse_pending_link` |
| R7 | Repository | DB errors | ERROR | `billing repo error` | `op`, `error` |

### 4.3.1 Frontend-facing retry errors

| Case | HTTP (proposed) | API `error` |
|------|-----------------|-------------|
| Invalid / missing | `400` | `invalid request` |
| Not found | `404` | `invoice not found` |
| Not unpaid | `400` | `invoice is not unpaid` |
| Gateway / DB | `500` | `failed to retry payment link` |

**Decision:** today many failures map to **400**. Recommend 404/500 as above.

---

### 4.4 Invoice items for fulfillment (internal)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| F1 | Item service | Load fail | ERROR | `invoice items for fulfillment failed` | `invoice_id`, `reason=db_read`, `error` |
| F2 | Item service | Success | DEBUG | `invoice items for fulfillment success` | `invoice_id`, `item_count` |

---

### 4.5 Mark invoice paid (internal)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| P1 | Repo / adapter | Already paid / 0 rows | WARN | `invoice status update failed` | `invoice_id`, `reason=already_paid` |
| P2 | Repo | DB error | ERROR | `invoice status update failed` | `invoice_id`, `reason=db_update`, `error` |
| P3 | Success | INFO | `invoice status update success` | `invoice_id`, `status=paid` |

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/billing/controllers.go` | Entry / invalid request / HTTP mapping |
| `internal/billing/invoice.services.go` | Checkout / get / retry outcomes |
| `internal/billing/invoice-item.services.go` | Item validation + fulfillment load |
| `internal/billing/inovice.repository.go` | DB errors + paid transition |
| `internal/billing/inovice-item.repository.go` | DB errors |
| `internal/billing/log.go` | `ensureLog` (new) |
| `shared/error/structures.go` | Optional: `ErrInvoiceNotUnpaid`, `ErrUnsupportedPaymentMode`, fetch/create failed |
| `appinit/payment_fulfillment.go` | Pass logger into billing when signatures change |

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | RequestLogger |
| `invoice_id`, `invoice_code` | After create / get / retry |
| `prescription_id`, `patient_id`, `organisation_id`, `cashier_id` | Checkout |
| `payment_mode`, `status` | Checkout / get / status |
| `item_count` | Checkout / fulfillment |
| `has_payment_url`, `has_idempotency_key`, `idempotency_replay` | Flags |
| `medicine_id` | Validation failures only |
| `reason`, `field`, `op`, `error` | Failures / repo |

Example checkout success:

```json
{
  "level": "info",
  "msg": "invoice checkout success",
  "request_id": "...",
  "invoice_id": "...",
  "invoice_code": "INV...",
  "prescription_id": "...",
  "patient_id": "...",
  "organisation_id": "...",
  "payment_mode": "link",
  "item_count": 3,
  "has_payment_url": true
}
```

---

## 7. What we will **not** add in this pass

- Logging payment URLs, amounts (unless decided), patient contact, full item payloads
- Full Razorpay / webhook instrumentation (payments doc)
- JWT on billing routes
- Removing unused `supplier_id` (product follow-up)

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on billing HTTP routes | Done (global) |
| Billing controller domain logs | Done |
| Invoice / item service domain logs | Done |
| Billing repo DB error logs | Done |
| Client-safe HTTP status mapping | Done |
| Fulfillment billing-side logs | Done (adapter uses `logger.Log`) |
| JWT on billing routes | Not present (follow-up) |

---

## 9. Implementation order (after approval)

1. `log.go` + optional sentinels  
2. Thread logger controller → service → repo  
3. Checkout (C1–C11) + HTTP mapping  
4. Get invoice + retry  
5. Fulfillment inventory load + status paid  
6. Optional JWT follow-up  

---

## 10. Open decisions for review

1. **Omit amounts from INFO logs?** — **Recommend: yes**
2. **Never log `payment_url` / idempotency key value?** — **Recommend: yes**
3. **Map HTTP statuses** (400/404/409/500) instead of blanket 409/400? — **Recommend: yes**
4. **Orphan invoice after `payment_create` fail:** ERROR with `invoice_id` as first-class signal? — **Recommend: yes**
5. **Empty `dispense_items`:** allow + `item_count=0` (current) or reject? — **Recommend: allow this pass**
6. **New sentinels** (`ErrInvoiceNotUnpaid`, etc.)? — **Recommend: yes**
7. **JWT this pass?** — **Recommend: follow-up**
8. **Logger style:** required `log *zap.Logger` first arg — **yes**

---

## 11. Approval

Once this list looks right, implement in §9 order without expanding into full payments webhook docs beyond billing-side fulfillment hooks.
