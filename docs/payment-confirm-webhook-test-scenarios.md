# Payment Confirm & Webhook — Test Scenarios

Manual QA / sign-off for:


| API                     | Method | Path                      | Handler                 |
| ----------------------- | ------ | ------------------------- | ----------------------- |
| Update payment manually | `POST` | `/api/v1/payment/confirm` | `UpdatePaymentManually` |
| Razorpay webhook        | `POST` | `/api/v1/payment/webhook` | `RazorPayWebhook`       |


Both paths share fulfillment (`FulfillPaidInvoice`): mark invoice **paid**, decrement stock (per batch line), write stock movements, update prescription / item dispense status (including multi-batch `remHashing`), resolve parent status, then enqueue payment-received notification.

**Doc map**


| Part | Focus                                                                                |
| ---- | ------------------------------------------------------------------------------------ |
| A    | Confirm API validation / happy / tenant guard                                        |
| B    | Webhook signature / events + multi-batch webhook (B.2b)                              |
| D    | Deep `FulfillPaidInvoice` fixtures (multi-batch, OOS, sequential invoices, rollback) |
| C    | Combined release sign-off                                                            |


**Auth note:** these payment routes are **not** behind JWT. Webhook is authenticated via `X-Razorpay-Signature`. Confirm has no auth today.

---



## Shared prerequisites



### Environment


| Variable                | Example                 | Notes                                                        |
| ----------------------- | ----------------------- | ------------------------------------------------------------ |
| `base_url`              | `http://localhost:9069` | From local deploy                                            |
| `organisation_id`       | *(from checkout org)*   | Must match the invoice’s organisation                        |
| `other_organisation_id` | *(different org UUID)*  | For cross-tenant negative tests                              |
| `invoice_id_cash`       | *(from checkout)*       | Unpaid invoice created with `payment_mode: cash`             |
| `invoice_id_qr`         | *(from checkout)*       | Unpaid invoice created with `payment_mode: qr`               |
| `invoice_id_link`       | *(from checkout)*       | Unpaid invoice created with `payment_mode: link`             |
| `provider_link_id`      | `plink_xxx`             | From `payment_attempts.provider_link_id` after link checkout |
| `webhook_secret`        | *(env)*                 | Razorpay webhook secret used by the server                   |




### Data setup (before each happy path)

1. Prescription with at least one line item and available inventory stock.
2. Checkout via `POST /api/v1/billing/create`:
  - **Confirm tests:** `payment_mode` = `cash` or `qr` → creates unpaid invoice + pending payment (no payment attempt / link).
  - **Webhook tests:** `payment_mode` = `link` → creates unpaid invoice + payment + pending payment attempt with Razorpay `provider_link_id`.
3. Capture `invoice_id`, stock quantities, prescription / item statuses, and (for webhook) `provider_link_id`.



### DB / side-effect checks (use after success cases)


| Check                        | Expected after successful pay                                                     |
| ---------------------------- | --------------------------------------------------------------------------------- |
| `invoices.status`            | `paid`                                                                            |
| `medicine_inventories` stock | Decremented by dispensed qty                                                      |
| Stock movements              | Dispense rows for dispensed items                                                 |
| Prescription item statuses   | `fully_dispensed` / `partially_dispensed` as applicable                           |
| Parent prescription status   | Resolved from item states                                                         |
| Notification                 | Payment-received event enqueued (log: `notification_enqueued=true` or equivalent) |


---



# Part A — UpdatePaymentManually (`POST /api/v1/payment/confirm`)



## A.0 Target API


| Item         | Value                                 |
| ------------ | ------------------------------------- |
| Method       | `POST`                                |
| URL          | `{{base_url}}/api/v1/payment/confirm` |
| Content-Type | `application/json`                    |
| Auth         | None                                  |




### Request fields


| Field                   | Required | Notes                                                                 |
| ----------------------- | -------- | --------------------------------------------------------------------- |
| `invoice_id`            | Yes      | Invoice UUID from cash/QR checkout                                    |
| `organisation_id`       | Yes      | Must match `invoices.organisation_id` for that invoice (tenant scope) |
| `payment_mode`          | Yes      | Must be `cash` or `qr`                                                |
| `transaction_reference` | No       | Accepted but currently unused by service                              |




### Allowed modes

- Request `payment_mode` must be `cash` or `qr`.
- Existing payment row `source` must also be `cash` or `qr` (link payments cannot be confirmed here).
- If request mode differs from stored source (e.g. checkout was `cash`, confirm as `qr`), source/channel are updated then fulfilled.
- Payment is loaded with `invoice_id` **and** `organisation_id` (join on `invoices`). Wrong org → `404` payment not found (no cross-tenant confirm).



### Sample request

```json
{
  "invoice_id": "{{invoice_id_cash}}",
  "organisation_id": "{{organisation_id}}",
  "payment_mode": "cash",
  "transaction_reference": "RCPT-1001"
}
```



### Success response (200)

```json
{
  "message": "payment confirmed"
}
```

---



## A.1 Scenario checklist


| ID  | Scenario                                                | Expected HTTP |
| --- | ------------------------------------------------------- | ------------- |
| C1  | Confirm cash payment — happy path                       | 200           |
| C2  | Confirm QR payment — happy path                         | 200           |
| C3  | Confirm with mode switch cash → qr                      | 200           |
| C4  | Confirm with mode switch qr → cash                      | 200           |
| C5  | Missing `invoice_id`                                    | 400           |
| C6  | Empty `invoice_id`                                      | 400           |
| C7  | Missing `payment_mode`                                  | 400           |
| C8  | Empty `payment_mode`                                    | 400           |
| C9  | Unsupported `payment_mode` (e.g. `link`, `upi`, `card`) | 400           |
| C10 | Unknown `invoice_id` (no payment row)                   | 404           |
| C11 | Invoice from **link** checkout (source = `link`)        | 400           |
| C12 | Double confirm on already paid invoice                  | 500           |
| C13 | Malformed JSON body                                     | 400           |
| C14 | Optional `transaction_reference` omitted                | 200           |
| C15 | Missing `organisation_id`                               | 400           |
| C16 | Empty `organisation_id`                                 | 400           |
| C17 | Valid invoice + **wrong** `organisation_id`             | 404           |


Exact status depends on fulfillment failure when invoice/stock is already settled; treat as **must not double-dispense**. Prefer assert: second call fails and stock is unchanged from after first confirm.

---



## A.2 Detailed scenarios



### C1 — Confirm cash (happy path)

**Given:** Unpaid invoice + pending payment with `source=cash`, stock available.

**Request**

```json
{
  "invoice_id": "{{invoice_id_cash}}",
  "organisation_id": "{{organisation_id}}",
  "payment_mode": "cash"
}
```

**Expect**

- HTTP `200`, `"payment confirmed"`
- Invoice → `paid`
- Stock decremented; movements created; prescription statuses updated
- Notification enqueued (confirm success even if notify fails — payment must stay committed)

**Pass criteria:** All shared DB checks pass; no duplicate movements on a single confirm.

---



### C2 — Confirm QR (happy path)

Same as C1 with `payment_mode: "qr"` and `invoice_id_qr`.

---



### C3 / C4 — Mode switch on confirm

**Given:** Checkout created with `cash`, confirm with `qr` (and reverse).

**Expect**

- HTTP `200`
- `payments.source` updated to requested mode (`qr` → channel `upi`; `cash` → channel `cash`)
- Fulfillment completes as in C1

---



### C5–C8 — Validation


| ID  | Body                 | Expect              |
| --- | -------------------- | ------------------- |
| C5  | omit `invoice_id`    | 400 invalid request |
| C6  | `"invoice_id": ""`   | 400                 |
| C7  | omit `payment_mode`  | 400                 |
| C8  | `"payment_mode": ""` | 400                 |


---



### C9 — Unsupported mode

```json
{
  "invoice_id": "{{invoice_id_cash}}",
  "organisation_id": "{{organisation_id}}",
  "payment_mode": "link"
}
```

**Expect:** `400` — unsupported payment mode.

---



### C10 — Payment not found

```json
{
  "invoice_id": "00000000-0000-0000-0000-000000000000",
  "organisation_id": "{{organisation_id}}",
  "payment_mode": "cash"
}
```

**Expect:** `404` — payment not found.

---



### C11 — Link invoice cannot use confirm

**Given:** Invoice from `payment_mode: link` checkout.

```json
{
  "invoice_id": "{{invoice_id_link}}",
  "organisation_id": "{{organisation_id}}",
  "payment_mode": "cash"
}
```

**Expect:** `400` (source mismatch / invalid request). Invoice remains unpaid; stock unchanged.

---



### C12 — Idempotency / double confirm

1. Run C1 successfully.
2. Call confirm again with same `invoice_id`, `organisation_id`, + `cash`.

**Expect**

- Second call **fails** (non-200; typically 500 payment confirm failed)
- Stock / movements **not** applied a second time
- Invoice remains `paid`

---



### C13 — Malformed JSON

Send body `{ invoice_id:` → **Expect:** `400`.

---



### C14 — Optional txn ref omitted

Omit `transaction_reference` on a valid cash confirm (include `organisation_id`) → **Expect:** `200` (same as C1).

---



### C15 / C16 — Organisation required


| ID  | Body change             | Expect |
| --- | ----------------------- | ------ |
| C15 | omit `organisation_id`  | 400    |
| C16 | `"organisation_id": ""` | 400    |


---



### C17 — Wrong organisation (cross-tenant)

**Given:** Valid cash invoice belonging to `organisation_id`.

```json
{
  "invoice_id": "{{invoice_id_cash}}",
  "organisation_id": "{{other_organisation_id}}",
  "payment_mode": "cash"
}
```

**Expect:** `404` payment not found; invoice remains unpaid; stock unchanged.

## A.3 Sign-off — UpdatePaymentManually


| ID  | Result (Pass / Fail / N/A) | Tester | Date       | Notes                        |
| --- | -------------------------- | ------ | ---------- | ---------------------------- |
| C1  | P                          | Sachin | 05-08-2026 | checked and its working fine |
| C2  | P                          | Sachin |            |                              |
| C3  | P                          | Sachin |            |                              |
| C4  | P                          | Sachin |            |                              |
| C5  | P                          | Sachin |            |                              |
| C6  | P                          | Sachin |            |                              |
| C7  | P                          | Sachin |            |                              |
| C8  | P                          | Sachin |            |                              |
| C9  | P                          | Sachin |            |                              |
| C10 | P                          | Sachin |            |                              |
| C11 | P                          | Sachin |            |                              |
| C12 | P                          | Sachin |            |                              |
| C13 | -                          |        |            |                              |
| C14 | P                          | Sachin |            |                              |
| C15 | P                          | Sachin |            |                              |
| C16 | P                          | Sachin |            |                              |
| C17 | P                          | Sachin |            |                              |


**Overall — UpdatePaymentManually**


| Field                 | Value                                |
| --------------------- | ------------------------------------ |
| Build / commit        | -                                    |
| Environment           | local                                |
| Tester name           | Sachin                               |
| Sign-off date         | 05-08-2026                           |
| Overall result        | ☑ Pass &nbsp;&nbsp; ☐ Fail &nbsp;&nbsp; ☐ Pass with exceptions |
| Exceptions / blockers | C13 (malformed JSON) not run — N/A   |
| Signature / initials  | sachin chate                         |


---



# Part B — Razorpay webhook (`POST /api/v1/payment/webhook`)



## B.0 Target API


| Item         | Value                                                                                |
| ------------ | ------------------------------------------------------------------------------------ |
| Method       | `POST`                                                                               |
| URL          | `{{base_url}}/api/v1/payment/webhook`                                                |
| Content-Type | `application/json`                                                                   |
| Auth         | Header `X-Razorpay-Signature` = HMAC-SHA256 hex of **raw body** using webhook secret |




### Signature helper (local)

```bash
# macOS / Linux — BODY_FILE is exact bytes posted
openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" "$BODY_FILE" | awk '{print $2}'
```

Use that hex string as `X-Razorpay-Signature`.

### Handled event types


| Event                    | Behavior                                                                                |
| ------------------------ | --------------------------------------------------------------------------------------- |
| `payment_link.paid`      | Claim attempt → persist webhook event → update attempt → **fulfill invoice** → notify   |
| `payment_link.cancelled` | Mark attempt cancelled; prescription stays payment-pending style update; **no fulfill** |
| `payment_link.expired`   | Mark attempt expired; same as cancelled for prescription; **no fulfill**                |
| Other / unhandled        | Ack `200` after claim path considerations; no fulfill                                   |




### Claim / duplicate behavior

`ClaimForProcessing` transitions attempt `pending` → `processing` atomically.

- First webhook for that `provider_link_id` claims and processes.
- Duplicate (already claimed / not pending) → **200** ack, no second fulfill.



### Minimal paid payload shape

Razorpay-shaped JSON (amounts in **paise**). Adjust IDs to match your DB attempt:

```json
{
  "event": "payment_link.paid",
  "account_id": "acc_test",
  "payload": {
    "payment": {
      "entity": {
        "id": "pay_test_001",
        "amount": 50000,
        "amount_transferred": 0,
        "status": "captured",
        "created_at": 1710000000,
        "upi": {
          "vpa": "patient@upi",
          "payer_account_type": "bank_account"
        }
      }
    },
    "payment_link": {
      "entity": {
        "id": "{{provider_link_id}}",
        "status": "paid",
        "reference_id": "INV-1234",
        "accept_partial": false
      }
    },
    "order": {
      "entity": {
        "id": "order_test_001"
      }
    }
  }
}
```

**Important:** `payload.payment_link.entity.id` must equal `payment_attempts.provider_link_id` for the checkout under test. Amount for full pay should match invoice/payment amount (rupees × 100).

### Success response (200)

```json
{
  "message": "Webhook processed successfully"
}
```



### Unauthorized (401)

```json
{
  "message": "Unauthorized"
}
```

---



## B.1 Scenario checklist


| ID      | Scenario                                                    | Expected HTTP                                |
| ------- | ----------------------------------------------------------- | -------------------------------------------- |
| W1      | `payment_link.paid` — full amount happy path                | 200                                          |
| W2      | Duplicate paid webhook (same link)                          | 200 (no double fulfill)                      |
| W3      | Missing `X-Razorpay-Signature`                              | 401                                          |
| W4      | Invalid signature                                           | 401                                          |
| W5      | `payment_link.cancelled`                                    | 200 (no stock change)                        |
| W6      | `payment_link.expired`                                      | 200 (no stock change)                        |
| W7      | Unhandled event type (e.g. `payment_link.created`)          | 200 (no fulfill)                             |
| W8      | Partial pay when `accept_partial=true` and amount < invoice | 200 (attempt partially_paid; **no fulfill**) |
| W9      | Unknown `provider_link_id` (no matching attempt)            | 401 or 500                                   |
| W10     | Webhook for already manually paid invoice (edge)            | Document actual behavior                     |
| W11     | Notification failure after paid fulfill                     | 200 (fulfill stays committed)                |
| WF1–WF6 | Multi-batch / complex fulfill via webhook                   | See Part B.2b / Part D                       |


If claim/lookup fails after a valid signature, controller may return `500` Failed to process webhook, or `401` if verification path returns unverified — record actual status during test.

---



## B.2 Detailed scenarios



### W1 — Paid happy path

**Given:** Link checkout unpaid; attempt `pending` with known `provider_link_id`; stock available.

**Steps**

1. Build paid payload with matching link id and full amount (paise).
2. Sign raw body; `POST` with `X-Razorpay-Signature`.

**Expect**

- HTTP `200`
- `payment_attempts` updated (paid / link paid, provider payment ids, amount_paid)
- `webhook_events` row stored
- Shared fulfillment checks (invoice paid, stock, movements, prescription)
- Notification enqueued

---



### W2 — Duplicate webhook

Replay **exact same** W1 request (or second paid event for same `provider_link_id` after claim).

**Expect**

- HTTP `200`
- Log reason `already_claimed` / duplicate skipped
- **No** second stock decrement / movements

---



### W3 — Missing signature

Omit header → **Expect:** `401` Unauthorized.

---



### W4 — Invalid signature

Valid body, wrong signature hex → **Expect:** `401`.

---



### W5 — Cancelled

Event `payment_link.cancelled`, valid signature, pending attempt.

**Expect**

- HTTP `200`
- Attempt status cancelled
- Invoice still unpaid; stock unchanged

---



### W6 — Expired

Same as W5 with `payment_link.expired` → attempt expired; no fulfill.

---



### W7 — Unhandled event

e.g. `"event": "payment_link.created"` with valid signature and claimable/pending attempt if applicable.

**Expect:** `200`; no invoice paid / no stock change. (If event is claimed first, note attempt may leave `processing` — record observed status.)

---



### W8 — Partial amount

**Given:** Payload with `accept_partial: true` and `amount` (paise) less than invoice amount × 100.

**Expect**

- HTTP `200`
- Attempt `partially_paid`
- **No** fulfillment (invoice unpaid, stock unchanged)

---



### W9 — Unknown provider link

Valid signature, `payment_link.entity.id` that does not exist.

**Expect:** Non-success processing; record status + that no invoice was paid.

---



### W10 — Already fulfilled via confirm (edge)

If product allows mixed paths: confirm cash somehow vs link — normally link invoices cannot confirm. Skip if N/A.

Optional: force invoice paid then send paid webhook → must not double-dispense.

---



### W11 — Notify failure after commit

Simulate notification dependency failure if possible (bad patient email path, etc.).

**Expect:** HTTP `200` for webhook; invoice/stock still fulfilled; ERROR log for notification only.

---



## B.2b Webhook + fulfillment (multi-batch / complex)

These reuse Part D data setups. Checkout with `payment_mode: link`, then fire signed `payment_link.paid`.


| ID  | Scenario (see Part D)                                  | Expected                                                            |
| --- | ------------------------------------------------------ | ------------------------------------------------------------------- |
| WF1 | Same as F1 via webhook                                 | Full fulfill; parent `completed`                                    |
| WF2 | Same as F2 (two batches, full cover) via webhook       | 2 movements; item `fully_dispensed`; parent `completed`             |
| WF3 | Same as F3 (two batches, partial + OOS) via webhook    | Item `partially_dispensed`; `out_of_stock=true`; parent `tentative` |
| WF4 | Same as F5 (multi-medicine mixed) via webhook          | Mixed item statuses; parent `tentative`                             |
| WF5 | Duplicate paid after WF2                               | 200; no second stock/movement                                       |
| WF6 | Cancelled link after multi-batch checkout (before pay) | No fulfill; stocks unchanged                                        |


Same parent-status caveat as F2 / F-NOTE-1.

---



## B.3 Sign-off — Razorpay webhook


| ID  | Result (Pass / Fail / N/A) | Tester | Date | Notes |
| --- | -------------------------- | ------ | ---- | ----- |
| W1  |                            |        |      |       |
| W2  |                            |        |      |       |
| W3  |                            |        |      |       |
| W4  |                            |        |      |       |
| W5  |                            |        |      |       |
| W6  |                            |        |      |       |
| W7  |                            |        |      |       |
| W8  |                            |        |      |       |
| W9  |                            |        |      |       |
| W10 |                            |        |      |       |
| W11 |                            |        |      |       |
| WF1 |                            |        |      |       |
| WF2 |                            |        |      |       |
| WF3 |                            |        |      |       |
| WF4 |                            |        |      |       |
| WF5 |                            |        |      |       |
| WF6 |                            |        |      |       |


**Overall — Razorpay webhook**


| Field                 | Value                                |
| --------------------- | ------------------------------------ |
| Build / commit        |                                      |
| Environment           |                                      |
| Tester name           |                                      |
| Sign-off date         |                                      |
| Overall result        | ☐ Pass ☐ Fail ☐ Pass with exceptions |
| Exceptions / blockers |                                      |
| Signature / initials  |                                      |


---



# Part D — `FulfillPaidInvoice` deep scenarios (confirm + shared logic)

Entry points that call the same fulfillment:


| Path    | API                            | When                     |
| ------- | ------------------------------ | ------------------------ |
| Confirm | `POST /api/v1/payment/confirm` | cash / qr after checkout |
| Webhook | `POST /api/v1/payment/webhook` | `payment_link.paid`      |


Fulfillment side effects (in order):

1. Load invoice lines + inventory + prescription item qty (`GetMedicineInventoryDetByInvoiceID`)
2. Per line: stock decrement + dispense movement (if `dispensed_qty > 0`)
3. Per line: `balance_after_dispense -= dispensed_qty`
4. Per line: item status via **running remaining** (`remHashing` keyed by `prescription_item_id`)
5. Bulk insert stock movements
6. Resolve parent prescription (`completed` vs `tentative`)
7. Mark invoice `paid`
8. (After commit) notify payment received



### Remaining / OOS rules (important)

For each invoice line in order:

```
if first line for prescription_item_id:
  remaining = prescribed_qty - dispensed_qty
else:
  remaining = previous_remaining - dispensed_qty

if remaining != 0:
  status = partially_dispensed
  out_of_stock = (dispensed_qty > 0
                  AND dispensed_qty >= current_stock_unit
                  AND current_stock_unit_after_dispense == 0)
else:
  status = fully_dispensed
  out_of_stock = false
```

Parent:

- all items on the fulfillment slice look `fully_dispensed` → parent `completed`
- any non-fully → parent `tentative`

**F-NOTE-1 (multi-batch parent):** fixed in code — parent resolve now collapses to one final status per `prescription_item_id` before setting parent. Full multi-batch cover should yield parent `completed`.

**F-NOTE-2 (checkout remaining):** checkout validates each dispensed line against the **same** `balance_after_dispense` without reducing across lines in one request. Sum of multi-batch qtys must be ≤ remaining, or fulfillment can drive balance negative. Prefer fixtures where sum(qty) ≤ remaining.

---



## D.0 Fixture patterns

Capture before/after for every scenario:


| Snapshot field | Tables / columns                                                       |
| -------------- | ---------------------------------------------------------------------- |
| Batch stocks   | `medicine_inventories.current_stock_units` per `medicine_inventory_id` |
| Item balance   | `prescription_items.balance_after_dispense`, `status`, `out_of_stock`  |
| Parent status  | `prescriptions.status`                                                 |
| Invoice        | `invoices.status`                                                      |
| Movements      | count + `qty_changed` + `medicine_inventory_id` for dispense rows      |




### Fixture A — single medicine, one batch (baseline)


| Field                  | Value                            |
| ---------------------- | -------------------------------- |
| Medicine M1 prescribed | 10                               |
| Batch B1 stock         | 50                               |
| Checkout dispense      | B1 × 10                          |
| Payment                | cash (confirm) or link (webhook) |




### Fixture B — single medicine, two batches, full cover


| Field                  | Value                                           |
| ---------------------- | ----------------------------------------------- |
| Medicine M1 prescribed | 20                                              |
| Batch B1 stock         | 12                                              |
| Batch B2 stock         | 30                                              |
| Checkout lines         | B1 × 12 + B2 × 8 (same `prescription_item_id`)  |
| Expected after pay     | B1=0, B2=22; balance=0; item fully; 2 movements |




### Fixture C — single medicine, two batches, partial + OOS


| Field                  | Value                                                                                       |
| ---------------------- | ------------------------------------------------------------------------------------------- |
| Medicine M1 prescribed | 30                                                                                          |
| Batch B1 stock         | 10                                                                                          |
| Batch B2 stock         | 5                                                                                           |
| Checkout lines         | B1 × 10 + B2 × 5                                                                            |
| Expected after pay     | B1=0, B2=0; balance=15; item partially; `out_of_stock=true` on last write; parent tentative |




### Fixture D — partial take, stock remains (no OOS)


| Field                  | Value                                                                |
| ---------------------- | -------------------------------------------------------------------- |
| Medicine M1 prescribed | 20                                                                   |
| Batch B1 stock         | 50                                                                   |
| Checkout               | B1 × 8 only                                                          |
| Expected               | B1=42; balance=12; partially; `out_of_stock=false`; parent tentative |




### Fixture E — multi-medicine mixed


| Field            | Value                                                 |
| ---------------- | ----------------------------------------------------- |
| M1 prescribed 10 | one batch, dispense 10 → fully                        |
| M2 prescribed 15 | one batch stock 40, dispense 6 → partially, oos=false |
| Expected parent  | `tentative`                                           |




### Fixture F — multi-medicine all full


| Field                                                          | Value       |
| -------------------------------------------------------------- | ----------- |
| M1 × 5 full, M2 × 7 full (possibly multi-batch on one of them) |             |
| Expected parent                                                | `completed` |




### Fixture G — second visit after partial (sequential invoices)

1. Run Fixture D via confirm → invoice1 paid, balance=12
2. New checkout for remaining 12 (or less) on same prescription → invoice2
3. Confirm invoice2

---



## D.1 Scenario checklist — confirm path


| ID  | Fixture / case                                                      | Payment       | Expected HTTP                          | Core asserts                                                      |
| --- | ------------------------------------------------------------------- | ------------- | -------------------------------------- | ----------------------------------------------------------------- |
| F1  | A — single batch full                                               | confirm cash  | 200                                    | item fully; parent completed; 1 movement; invoice paid            |
| F2  | B — two batches full cover                                          | confirm cash  | 200                                    | both stocks; balance 0; item fully; 2 movements; parent completed |
| F3  | C — two batches partial OOS                                         | confirm cash  | 200                                    | stocks 0; balance 15; partially; oos true; parent tentative       |
| F4  | D — under-prescribe take                                            | confirm qr    | 200                                    | stock left; oos false; parent tentative                           |
| F5  | E — multi-medicine mixed                                            | confirm cash  | 200                                    | M1 fully, M2 partially; parent tentative                          |
| F6  | F — multi-medicine all full                                         | confirm cash  | 200                                    | all fully; parent completed                                       |
| F7  | G step1 then step2                                                  | confirm ×2    | 200, 200                               | balances/stocks cumulative; no double-count on invoice1           |
| F8  | B via confirm, then confirm again                                   | 200 then fail | no second movements                    |                                                                   |
| F9  | Zero-qty line skipped at checkout                                   | confirm       | 200                                    | only positive qty lines in `invoice_items` / movements            |
| F10 | Three batches same item full cover                                  | confirm       | 200                                    | 3 movements; balance 0; item fully; parent completed              |
| F11 | Two batches same item, wrong org on confirm                         | 404           | no fulfill                             |                                                                   |
| F12 | Link checkout then confirm                                          | 400           | no fulfill                             |                                                                   |
| F13 | Partial first invoice, second invoice exceeds remaining at checkout | checkout fail | `qty_exceeds_remaining` before confirm |                                                                   |
| F14 | Notify fail after F1                                                | 200           | fulfill committed; notify error logged |                                                                   |


See F-NOTE-1.

---



## D.2 Detailed scenarios



### F1 — Single batch, full dispense (confirm)

**Setup:** Fixture A → `POST /billing/create` cash → capture `invoice_id`.

**Confirm**

```json
{
  "invoice_id": "{{invoice_id}}",
  "organisation_id": "{{organisation_id}}",
  "payment_mode": "cash"
}
```

**Assert**


| Check                    | Expected                              |
| ------------------------ | ------------------------------------- |
| HTTP                     | 200                                   |
| `invoices.status`        | `paid`                                |
| B1 stock                 | 40 (50−10)                            |
| `balance_after_dispense` | 0                                     |
| item `status`            | `fully_dispensed`                     |
| item `out_of_stock`      | false                                 |
| parent `status`          | `completed`                           |
| movements                | 1 row, `qty_changed=10`, inventory=B1 |


---



### F2 — Two batches, same prescription item, full cover (confirm)

**Setup:** Fixture B. Checkout body must include **two** `dispensed_items` rows sharing the same `prescription_item_id`, different `medicine_inventory_id` / `batch_no`.

**Fulfillment walk-through**


| Order | Line    | remHashing remaining | Item status write     | OOS                  |
| ----- | ------- | -------------------- | --------------------- | -------------------- |
| 1     | B1 × 12 | 20−12 = 8            | `partially_dispensed` | true (batch emptied) |
| 2     | B2 × 8  | 8−8 = 0              | `fully_dispensed`     | false                |


**Assert**


| Check            | Expected                                     |
| ---------------- | -------------------------------------------- |
| B1 / B2 stock    | 0 / 22                                       |
| balance          | 0                                            |
| item status (DB) | `fully_dispensed` (last write wins)          |
| movements        | 2                                            |
| parent           | `completed` (after per-item status collapse) |
| invoice          | `paid`                                       |


---



### F3 — Two batches, still short of prescribed (OOS)

**Setup:** Fixture C.

**Walk-through**


| Order | Line    | remaining | Status    | OOS  |
| ----- | ------- | --------- | --------- | ---- |
| 1     | B1 × 10 | 20        | partially | true |
| 2     | B2 × 5  | 15        | partially | true |


**Assert:** stocks 0/0; balance 15; `partially_dispensed`; `out_of_stock=true`; parent `tentative`; 2 movements.

---



### F4 — Partial take with stock left (no OOS)

**Setup:** Fixture D, confirm `qr`.

**Assert:** stock 42; balance 12; partially; `out_of_stock=false`; parent `tentative`; 1 movement.

---



### F5 — Multi-medicine mixed statuses

**Setup:** Fixture E.

**Assert**

- M1: fully, oos false
- M2: partially, oos false
- Parent: `tentative`
- Movements: 2 (one per medicine line)
- Invoice: paid

---



### F6 — Multi-medicine all fully dispensed

**Setup:** Fixture F (optionally one medicine split across 2 batches).

**Assert:** every item fully; parent `completed` ; movements = number of positive invoice lines.

---



### F7 — Sequential invoices on same prescription

**Step 1:** Fixture D → confirm → balance 12, invoice1 paid.

**Step 2:** Checkout remaining 12 from same or new batch → confirm invoice2.

**Assert**

- After step2: balance 0; item fully; parent completed
- Invoice1 and invoice2 both `paid`
- Stocks reflect both dispenses
- Movements from both invoices present (no overwrite)

---



### F8 — Double confirm (same invoice)

After F2 success, confirm again with same ids.

**Assert:** non-200; stocks/movements/balance unchanged from post-F2 snapshot.

---



### F9 — Zero quantity lines

At checkout, omit zero-qty medicines (service skips `QuantitySoldUnits == 0`). Confirm remaining positive lines only.

**Assert:** no movement with qty 0; invoice paid for included lines only.

---



### F10 — Three batches, one prescription item

Prescribed 25; batches 10 + 10 + 20; dispense 10+10+5.

**Assert:** 3 movements; balance 0; last status fully; stocks 0/0/15; parent completed.

---



### F11 / F12 — Tenant and mode guards (still hit before fulfill)


| ID  | Action                                             | Expect               |
| --- | -------------------------------------------------- | -------------------- |
| F11 | Valid multi-batch invoice, wrong `organisation_id` | 404; no stock change |
| F12 | Link-mode invoice + confirm cash                   | 400; no stock change |


---



### F13 — Second checkout exceeds remaining

After F4 (balance 12), attempt checkout with qty 13.

**Expect:** checkout fails (`qty_exceeds_remaining`); no new invoice/payment; confirm N/A.

---



### F14 — Notification failure does not roll back fulfill

Force notify failure if possible after F1.

**Expect:** confirm 200; invoice paid; stock moved; ERROR on notification only.

---



## D.3 Failure / rollback oriented (fulfill internals)

Hard to force without faults; when possible:


| ID  | Injection point                                     | Expect                                                                              |
| --- | --------------------------------------------------- | ----------------------------------------------------------------------------------- |
| FR1 | Fail `UpdateMedInventoryStock` mid multi-batch      | Confirm/webhook 500; **no** invoice paid; **no** partial stock commit (tx rollback) |
| FR2 | Fail `CreateMedicineMvmt` after stock updates in tx | Rollback all stock + dispense qty                                                   |
| FR3 | Fail `UpdateInvoiceStatus`                          | Rollback; invoice stays unpaid                                                      |
| FR4 | Empty invoice_items for invoice                     | Fulfill fails; confirm/webhook error; nothing paid                                  |


---



## D.4 Sign-off — FulfillPaidInvoice / multi-batch


| ID  | Result (Pass / Fail / N/A) | Tester | Date | Notes (incl. F-NOTE-1) |
| --- | -------------------------- | ------ | ---- | ---------------------- |
| F1  |                            |        |      |                        |
| F2  |                            |        |      |                        |
| F3  |                            |        |      |                        |
| F4  |                            |        |      |                        |
| F5  |                            |        |      |                        |
| F6  |                            |        |      |                        |
| F7  |                            |        |      |                        |
| F8  |                            |        |      |                        |
| F9  |                            |        |      |                        |
| F10 |                            |        |      |                        |
| F11 |                            |        |      |                        |
| F12 |                            |        |      |                        |
| F13 |                            |        |      |                        |
| F14 |                            |        |      |                        |
| FR1 |                            |        |      |                        |
| FR2 |                            |        |      |                        |
| FR3 |                            |        |      |                        |
| FR4 |                            |        |      |                        |
| WF1 |                            |        |      |                        |
| WF2 |                            |        |      |                        |
| WF3 |                            |        |      |                        |
| WF4 |                            |        |      |                        |
| WF5 |                            |        |      |                        |
| WF6 |                            |        |      |                        |


**Overall — Fulfillment deep suite**


| Field                 | Value                                                                |
| --------------------- | -------------------------------------------------------------------- |
| Build / commit        |                                                                      |
| Environment           | local (via confirm API)                                              |
| Tester name           | Sachin                                                               |
| Sign-off date         | 05-08-2026                                                           |
| Overall result        | ☑ Pass with exceptions &nbsp;&nbsp; ☐ Pass &nbsp;&nbsp; ☐ Fail      |
| F-NOTE-1 observed?    | ☐ No (parent completed on full multi-batch) ☐ Yes (regression) ☑ N/A |
| Exceptions / blockers | Covered via confirm sharing `FulfillPaidInvoice`. Per-ID F/WF/FR rows not individually filled; FR* not fault-injected. |
| Signature / initials  | sachin chate                                                         |


---



# Part C — Combined release sign-off

Use after Part A, Part B, and Part D are complete.

**Note (05-08-2026):** Confirm API fully exercised locally (Part A). Webhook paid path uses the same `FulfillPaidInvoice`, so stock / movements / prescription / invoice side effects are covered via confirm. Webhook-only envelope cases (signature, claim/duplicate, cancelled/expired/partial, unknown link) remain optional if not exercised separately.


| Area                                                  | Overall                    | Signed by     | Date       |
| ----------------------------------------------------- | -------------------------- | ------------- | ---------- |
| UpdatePaymentManually (`/payment/confirm`)            | ☑ Pass ☐ Fail ☐ Exceptions | Sachin        | 05-08-2026 |
| Razorpay webhook (`/payment/webhook`)                 | ☐ Pass ☐ Fail ☐ Exceptions |               |            |
| FulfillPaidInvoice multi-batch / complex (Part D)     | ☑ Pass* ☐ Fail ☐ Exceptions | Sachin       | 05-08-2026 |
| Cross-check: no double dispense (C12 + W2 + F8 + WF5) | ☑ Pass* (C12) ☐ Fail       | Sachin        | 05-08-2026 |
| Notifications observed (confirm + webhook)            | ☑ Pass* (confirm) ☐ Fail ☐ N/A | Sachin   | 05-08-2026 |

\*Fulfillment + notify + double-confirm covered via confirm path (`FulfillPaidInvoice` shared). W2 / WF5 not separately run.


| Field               | Value                                                                 |
| ------------------- | --------------------------------------------------------------------- |
| Release / build     | local                                                                 |
| QA lead             | Sachin                                                                |
| Final sign-off date | 05-08-2026                                                            |
| Ship decision       | ☑ Ready &nbsp;&nbsp; ☐ Blocked                                        |
| Blockers            | None for confirm + shared fulfill. Webhook envelope tests still open. |


