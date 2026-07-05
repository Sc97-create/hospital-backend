# Checkout → Webhook Flow

This document traces the complete journey from a cashier initiating a checkout to the inventory being decremented and the prescription being marked dispensed after payment confirmation.

---

## Overview

```
Cashier (frontend)
    │
    ▼
POST /billing/checkout
    │
    ▼
billing.Checkout (controller)
    │  parse request, build CheckoutReq DTO
    ▼
billing.InvoiceServ.CreatePaymentLink
    │
    ├─► [DB Tx 1]
    │       CreateInvoice         → inserts invoices row (status = unpaid)
    │       addInvoiceItems       → validates qty, inserts invoice_items rows
    │   [Commit Tx 1]
    │
    ├─► PatientServ.FindOne       → fetch patient info for SMS/email
    │
    └─► PaymentsService.StorePaymentandNotifyUser
            │
            ├─► RazorpayProvider.CreatePayment  → creates payment link via Razorpay API
            │
            └─► [DB Tx 2]
                    Create(payments)         → inserts payments row (invoice_id FK)
                    CreateAttempt            → inserts payment_attempts row (status = pending)
                [Commit Tx 2]

    Return: Razorpay short_url  →  frontend shows QR / link to patient

────────────── patient pays ──────────────

POST /payments/webhook  (Razorpay fires this)
    │
    ▼
payments.WebhookController → ProcessWebhook(payload, signature, provider)
    │
    ├─► PaymentFactory.GetProvider("razorpay")
    ├─► RazorpayProvider.VerifySignature       → HMAC check, abort if invalid
    ├─► RazorpayProvider.ParseWebhookEvent     → maps raw JSON to ParsedWebhookEvent DTO
    │
    ├─► PaymentAttempts.ClaimForProcessing(providerLinkID)
    │       UPDATE payment_attempts
    │         SET payment_status = 'processing'
    │         WHERE provider_link_id = ? AND payment_status = 'pending'
    │       → if 0 rows affected: duplicate webhook → ack and exit
    │
    ├─► WebhookRepository.CreateWebhookEvent   → raw payload stored for audit
    │
    ├─► [DB Tx 3] begin
    ├─► getInvoiceForUpdate(paymentAttempt.PaymentID)
    │       SELECT * FROM payments WHERE id = ?   → loads Payments row (has invoice_id, amount)
    │
    └─► switch dtowebhookevent.EventType ──────────────────────────────────────────────────
```

---

## Event Branches

### `payment_link.paid`

```
PaymentLinkPaid
    │
    ├─ Partial payment guard
    │       if AcceptPartial && AmountPaid < payments.Amount
    │           UPDATE payment_attempts SET status = 'partially_paid'
    │           Commit → return true (ack, no dispense)
    │
    ├─► updatePaymentAttempt(tx, ...)
    │       UPDATE payment_attempts SET
    │           payment_status, payment_link_status, amount_paid,
    │           amount_transferred, payer_account_type, payment_vpa,
    │           provider_order_id, provider_payment_id, paid_at, accept_partial
    │
    ├─► getMedInventoryForUpdate(invoiceInfo.InvoiceID)
    │       → CommonInterface.GetMedicineInventoryDetByInvoiceID(invoiceID)
    │         SELECT ii.*, mi.*, inv.cashier_id, inv.prescription_id,
    │                pi.quantity AS prescribed_qty,
    │                pi.balance_after_dispense AS already_dispensed
    │         FROM invoice_items ii
    │         JOIN medicine_inventories mi ON mi.id = ii.medicine_inventory_id
    │         JOIN invoices inv            ON inv.id = ii.invoice_id
    │         JOIN prescription_items pi  ON pi.id  = ii.prescription_item_id
    │         WHERE ii.invoice_id = ?
    │       returns []MedInvoiceItemResponse
    │
    ├─► for each MedInvoiceItemResponse item:
    │       │
    │       ├─ build MedicineStockMovements record (bulk insert later)
    │       │
    │       ├─► CommonInterface.UpdateMedInventoryStock(tx, inventoryID, dispensedQty)
    │       │       UPDATE medicine_inventories
    │       │         SET current_stock_units = current_stock_units - ?
    │       │         WHERE id = ? AND current_stock_units >= ?
    │       │       → atomic decrement, guard against negative stock
    │       │
    │       ├─► CommonInterface.UpdateDispenseItemQty(tx, prescriptionItemID, dispensedQty)
    │       │       UPDATE prescription_items
    │       │         SET balance_after_dispense = balance_after_dispense + ?
    │       │         WHERE id = ?
    │       │       → accumulates total dispensed so far
    │       │
    │       └─ item-level status:
    │               newBalance = already_dispensed + dispensed_qty
    │               if newBalance >= prescribed_qty
    │                   UpdateIPrescriptionStatus(tx, itemID, "fully_dispensed")
    │               else
    │                   allFullyDispensed = false
    │                   UpdateIPrescriptionStatus(tx, itemID, "partially_dispensed")
    │
    ├─► CommonInterface.CreateMedicineMvmt(tx, bulkMedicineMvmt)
    │       INSERT INTO medicine_stock_movements (bulk)
    │
    ├─► CommonInterface.UpdateExtPrescriptionStatus(tx, prescriptionID, status)
    │       if allFullyDispensed  → status = "fully_dispensed"
    │       else                  → status = "partially_dispensed"
    │       UPDATE prescriptions SET status = ? WHERE id = ?
    │
    ├─► CommonInterface.UpdateInvoiceStatus(tx, invoiceID, "paid")
    │       UPDATE invoices SET status = 'paid' WHERE id = ?
    │
    └─► Commit Tx 3 → return true
```

### `payment_link.cancelled`

```
PaymentLinkCancelled
    ├─► UpdatePaymentAttemptStatus → status = "cancelled"
    ├─► UpdateInvoiceStatus        → status = "unpaid"  (cashier can retry)
    └─► Commit → return true
```

### `payment_link.expired`

```
PaymentLinkExpired
    ├─► UpdatePaymentAttemptStatus → status = "expired"
    │   (invoice stays "unpaid" — cashier generates a new link)
    └─► Commit → return true
```

---

## Tables Written Per Flow

| Stage | Table | Operation |
|---|---|---|
| Checkout | `invoices` | INSERT (status = `unpaid`) |
| Checkout | `invoice_items` | INSERT (one row per medicine) |
| Checkout | `payments` | INSERT |
| Checkout | `payment_attempts` | INSERT (status = `pending`) |
| Webhook claim | `payment_attempts` | UPDATE status `pending → processing` |
| Webhook audit | `webhook_events` | INSERT (raw payload) |
| Webhook paid | `payment_attempts` | UPDATE (amounts, VPA, order/payment IDs) |
| Webhook paid | `medicine_inventories` | UPDATE (atomic stock decrement) |
| Webhook paid | `prescription_items` | UPDATE (`balance_after_dispense` increment) |
| Webhook paid | `prescription_items` | UPDATE (item status) |
| Webhook paid | `medicine_stock_movements` | INSERT (bulk audit trail) |
| Webhook paid | `prescriptions` | UPDATE (overall status) |
| Webhook paid | `invoices` | UPDATE (status = `paid`) |
| Webhook cancelled | `payment_attempts` | UPDATE (status = `cancelled`) |
| Webhook expired | `payment_attempts` | UPDATE (status = `expired`) |

---

## Concurrency Guards

| Risk | Guard |
|---|---|
| Two Razorpay webhook deliveries for the same `provider_link_id` arrive simultaneously | `ClaimForProcessing` atomically flips `pending → processing`; the second goroutine sees 0 rows affected and exits immediately |
| Two pharmacists dispense from the same batch at the same time | `UpdateMedInventoryStock` uses `SET current_stock_units = current_stock_units - ? WHERE current_stock_units >= ?`; stock never goes negative |

---

## Validation Checkpoints

| Where | Check |
|---|---|
| `addInvoiceItems` (checkout) | `dispensed_qty > current_stock_units` → error "insufficient stock in batch" |
| `addInvoiceItems` (checkout) | `dispensed_qty > (prescribed_qty - balance_after_dispense)` → error "exceeds remaining prescribed qty" |
| `ProcessWebhook` (paid event) | `AcceptPartial && AmountPaid < payments.Amount` → mark `partially_paid`, skip dispense |
| `UpdateMedInventoryStock` (webhook) | `current_stock_units < dispensed_qty` at DB level → GORM returns error, tx rolled back |

---

## Key Files

| File | Role |
|---|---|
| `internal/billing/controllers.go` | Parses HTTP request, calls `CreatePaymentLink` |
| `internal/billing/invoice.services.go` | Orchestrates invoice creation + payment link |
| `internal/billing/invoice-item.services.go` | Validates & inserts `invoice_items` |
| `internal/billing/inovice-item.repository.go` | Raw SQL for `invoice_items` + JOIN query |
| `internal/payments/payments.services.go` | Calls Razorpay, stores `payments` + `payment_attempts` |
| `internal/payments/payment-attempts.services.go` | `ClaimForProcessing`, `UpdatePaymentAttempt` |
| `internal/payments/payment-attempts.repository.go` | Atomic `pending → processing` UPDATE |
| `internal/payments/webhook_service.go` | Full webhook dispatch and dispense logic |
| `internal/payments/providers/razorpay/gateway.go` | `VerifySignature`, `ParseWebhookEvent` |
| `internal/medicine/medicine_inventory.repository.go` | Atomic stock decrement |
| `internal/prescription/prescriptions-items.repository.go` | `UpdateDispenseItemQty`, item status |
| `internal/prescription/prescriptions.repository.go` | Prescription-level status update |
| `pkg/constants/constant.go` | All shared status/event string constants |
