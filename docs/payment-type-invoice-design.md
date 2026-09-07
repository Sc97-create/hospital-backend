# Payment Type on Invoice — Consultation vs Prescription

## Status: Design sketch (no code changed yet)

Decisions locked in for this sketch (confirmed with product/eng):

| Decision | Answer |
|---|---|
| When is a consultation invoice created? | **Not** auto-created inside appointment booking. A separate explicit call (existing checkout-style endpoint) creates it, made by frontend/reception right after "book appointment success". |
| `payment_type` values | `consultation`, `prescription` today — modeled so more values (`advance`, `package`, `refund`, …) can be added later without a redesign. |
| `Invoice.PrescriptionID` for consultation invoices | Made **nullable**. |
| `Invoice.AppointmentID` | **Added**, nullable, for tracking + validation (updated from the earlier "no appointment_id" sketch — see §2.1 and §2.2). |
| `payment_mode: "link"` for consultation invoices | **Out of scope for now.** Consultation checkout is cash/QR-in-person only; the Razorpay payment-link flow is not being designed/tested for consultation in this pass. Nothing in the code actively blocks `payment_mode: "link"` for a consultation invoice (it flows through the same generalized path), but it's an unsupported/untested combination — revisit if/when consultation needs remote payment links. |

---

## 1. Why this is needed (current state)

Today there is exactly **one** invoice-creation path, and it is prescription-shaped:

```5:19:internal/billing/models.go
type Invoice struct {
	ID             string    `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceCode    string    `json:"invoice_code" gorm:"type:text;not null"`
	PrescriptionID string    `json:"prescription_id" gorm:"type:uuid;uniqueIndex"`
	PatientID      string    `json:"patient_id" gorm:"type:uuid;not null"`
	Status         string    `json:"status" gorm:"default:unpaid"`
	CashierID      string    `json:"cashier_id" gorm:"type:uuid;not null"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	SubtotalAmount float64   `json:"sub_total_amount" gorm:"type:numeric(10,2);not null"`
	TaxAmount      float64   `json:"tax_amount" gorm:"type:numeric(10,2);not null"`
	TotalAmount    float64   `json:"total_amount" gorm:"type:numeric(10,2);not null"`
	DiscountAmount float64   `json:"discount_amount" gorm:"type:numeric(10,2);not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoCreateTime"`
}
```

* `PrescriptionID` is `uniqueIndex` and treated as required everywhere (`addInvoiceItems`, `GetInvoiceByPrescriptionID`, dispense logic).
* The only creation entry point is the cashier checkout, `POST /billing/create` → `InvoiceServ.CreateInvoice` (`internal/billing/invoice.services.go:35`), which always calls `addInvoiceItems` to validate/insert medicine lines against a prescription.
* Appointment booking (`AppointmentService.CreateApptmnt`, `internal/appointments/appointment.services.go:44`) creates **only** the appointment row — no invoice, no payment.
* There is no discriminator anywhere (Go code or DB) that says "this invoice is for a consultation fee" vs "this invoice is for dispensed medicines". A consultation invoice literally cannot exist today because `PrescriptionID` can't be empty.

So: to charge a consultation fee at booking time through the same billing/payment pipeline, we need a way to create an invoice **without** a prescription, and something on the invoice that says what it's for — hence `payment_type`.

---

## 2. Proposed design

### 2.1 `payment_type` on `Invoice`

Add a typed string column, following the same pattern already used for `appointments.Status`:

```go
package billing

type PaymentType string

const (
	PaymentTypeConsultation PaymentType = "consultation"
	PaymentTypePrescription PaymentType = "prescription"
)
```

```go
type Invoice struct {
	ID             string      `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceCode    string      `json:"invoice_code" gorm:"type:text;not null"`
	PaymentType    PaymentType `json:"payment_type" gorm:"type:varchar(30);not null;default:prescription"`
	PrescriptionID *string     `json:"prescription_id,omitempty" gorm:"type:uuid;uniqueIndex"`
	AppointmentID  *string     `json:"appointment_id,omitempty" gorm:"type:uuid;uniqueIndex"`
	PatientID      string      `json:"patient_id" gorm:"type:uuid;not null"`
	Status         string      `json:"status" gorm:"default:unpaid"`
	CashierID      string      `json:"cashier_id" gorm:"type:uuid;not null"`
	OrganisationID string      `json:"organisation_id" gorm:"type:uuid;not null"`
	SubtotalAmount float64     `json:"sub_total_amount" gorm:"type:numeric(10,2);not null"`
	TaxAmount      float64     `json:"tax_amount" gorm:"type:numeric(10,2);not null"`
	TotalAmount    float64     `json:"total_amount" gorm:"type:numeric(10,2);not null"`
	DiscountAmount float64     `json:"discount_amount" gorm:"type:numeric(10,2);not null"`
	CreatedAt      time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time   `json:"updated_at" gorm:"autoCreateTime"`
}
```

Why `*string` (nullable) for both FK columns instead of plain `string`:
Postgres unique indexes treat every `NULL` as distinct, but treat every `""` (empty string) as equal. If we kept `PrescriptionID string` and just left it blank for consultation invoices, the **second** consultation invoice ever created would fail the `uniqueIndex` with a duplicate-key error, because both rows would have `prescription_id = ''`. Switching to `*string` (nil when not applicable) makes Postgres store real `NULL`s, so each unique constraint only ever fires for genuine duplicates on that column — which is the actual intent of both indexes. The same reasoning applies to `AppointmentID`: prescription invoices will leave it `nil` (Postgres allows unlimited `NULL`s), while consultation invoices set it, and the `uniqueIndex` blocks a second consultation invoice from ever being created against the same appointment.

### Why `AppointmentID` earns its place (not just `PrescriptionID`)

* **Tracking**: consultation invoices otherwise have no FK back to what generated them — `patient_id`/`cashier_id`/`created_at` alone aren't enough to reliably answer "show me the invoice for appointment X" (multiple appointments per patient per day are common in a hospital). `AppointmentID` closes that gap directly.
* **Validation (new)**: `uniqueIndex` on `AppointmentID` means the DB itself rejects a second consultation invoice for an appointment that's already been billed — this is the double-charge guard, symmetric to how `PrescriptionID`'s `uniqueIndex` already stops double-billing a prescription today.
* **Cross-check for prescription invoices too**: prescriptions already carry their originating appointment (`prescription.appointment_id` per `docs/prescription-items.md`). When creating a *prescription* invoice, `CreateInvoice` can resolve the prescription's `appointment_id` and stamp it on the invoice as well (still optional/nullable — old rows won't have it) purely for reporting consistency; it does not change any existing prescription-invoice validation.

`payment_type` is a plain `varchar`, not a Postgres `ENUM`, specifically so adding `advance` / `package` / `refund` later is a data-only change (no migration to alter a DB enum type).

### 2.2 Validation rule going forward

Exactly one of these must hold per invoice:

| `payment_type` | `prescription_id` | `appointment_id` | `invoice_items` |
|---|---|---|---|
| `prescription` | required, non-null | optional — backfilled from `prescription.appointment_id` when available | one row per dispensed medicine (existing behavior) |
| `consultation` | must be `nil` | **required, non-null** | **zero rows** (no medicines) |

Additional checks enabled specifically by having `appointment_id` on the invoice, all enforced in `InvoiceServ.CreateInvoice` before the insert:

1. **Appointment must exist** — `CreateInvoice` calls `AppointmentService.GetAppntmentByID` (`internal/appointments/appointment.services.go:663`) and fails fast (`wrapError.ErrInvalidRequest` / a new `ErrAppointmentNotFound`) if the appointment doesn't exist, rather than only finding out later when nothing can be reconciled.
2. **No double-billing an appointment** — the DB `uniqueIndex` on `appointment_id` is the hard guarantee; the service layer can also pre-check via `GetInvoiceByAppointmentID` and return `wrapError.ErrInvoiceAlreadyExists` with a friendlier message before ever hitting the DB constraint.
3. **Tenant/patient consistency** — `appointment.organisation_id` and `appointment.patient_id` should match the request's `organisation_id`/`patient_id`; mismatch → `ErrInvalidRequest`. This catches a frontend bug (e.g. stale appointment ID from a previous session) before money changes hands.

None of this is enforced at the DB layer beyond the two `uniqueIndex`es (a DB `CHECK` constraint tying `payment_type` to which FK is set could be added later for belt-and-suspenders, but isn't required for correctness since the service layer owns it).

### 2.3 Where invoices get created (no change to *who* creates them, just *what* they can create)

Kept the checkout endpoint as the single creation path, generalized instead of forked, per your "clean slate" framing:

```
POST /billing/create
{
  "payment_type": "consultation" | "prescription",   // NEW field, defaults to "prescription" if omitted (back-compat)
  "prescription_id": "...",     // required only when payment_type = prescription
  "appointment_id": "...",      // NEW: required when payment_type = consultation, optional for prescription
  "patient_id": "...",
  "cashier_id": "...",
  "organisation_id": "...",
  "payment_mode": "cash" | "qr" | "link",
  "financials": { "sub_total_amount": ..., "tax_amount": ..., "discount_amount": ..., "total_amount": ... },
  "dispense_items": [ ... ]     // required only when payment_type = prescription, must be empty/omitted for consultation
}
```

* **Prescription checkout** (pharmacy counter): unchanged behavior, just now stamps `payment_type = "prescription"` explicitly instead of implicitly.
* **Consultation checkout** (reception counter, right after booking): frontend calls the same endpoint with `payment_type = "consultation"`, no `prescription_id`, no `dispense_items` — just the consultation fee in `financials`.

This avoids building a parallel "create consultation invoice" service/controller/route — one code path, branched on `payment_type`, which is what keeps this a "clean slate" instead of two half-duplicated billing flows.

### 2.4 Mapping your flow to backend calls

```
Book Appointment (frontend)
   │
   ▼
POST /appointment/create                     ← unchanged, still just creates the appointment
   │  (booking success)
   ▼
Frontend opens payment popup
   │  [ Cash ]  [ QR ]      ← "Cash" selected by default
   ▼
POST /billing/create
   payment_type = "consultation"
   appointment_id = <the just-booked appointment>
   payment_mode = "cash" (default) | "qr"
   → CreateInvoice first validates the appointment (exists, belongs to this patient/org, not already billed
     via appointment_id uniqueIndex) — see §2.2
   → creates Invoice(payment_type=consultation, appointment_id=..., prescription_id=nil, status=unpaid)
     + Payments(source=cash|qr, status=pending)
   → NOTE: for cash/qr, CreateInvoice calls PaymentServ.CreatePendingPayment — no gateway call,
     nothing to "show" (no QR image / link) until cashier actually collects money and confirms.
   │
   ▼
Cashier collects cash, or patient scans/pays QR, in person
   │
   ▼
POST /payment/confirm
   { invoice_id, organisation_id, payment_mode, txn_ref }
   → PaymentsService.ConfirmManualPayment
   → FulfillmentService.FulfillPaidInvoice(invoiceID)
        - queries invoice_items for this invoice → 0 rows for a consultation invoice
        - inventory/prescription-dispense loop simply doesn't execute (nothing to iterate)
        - invoice.status: unpaid → paid
   → NotifyPaymentReceived (SMS/email "payment received")
```

Important existing-code finding: **`FulfillmentService.FulfillPaidInvoice` (`internal/payments/fulfillment_service.go:29`) already degrades gracefully to a zero-item invoice.** `GetMedicineInventoryDetByInvoiceID` is an inner-join query keyed on `invoice_items.invoice_id`; a consultation invoice will simply return an empty slice, the dispense loop won't run, `prescriptionID` stays `""` so `ResolveAndUpdateParentPrescriptionStatus` is skipped, and it goes straight to marking the invoice `paid`. **No changes are required in the fulfillment/webhook/confirm code path** for this to work correctly — that's the main reason this design is low-risk.

Re: "how to show the payment for swipe" — read this as *"cash is the default toggle state in the popup, QR is the alternative, and once the cashier has physically taken the payment (cash handed over / QR scanned) the app just needs to mark it paid"* — that's exactly `POST /payment/confirm` above. If "swipe" instead means a **card/POS terminal** (not UPI QR), see the open question below — today `ConfirmManualPayment` only accepts `cash` or `qr` as the confirm mode; a card-swipe mode would need to be added explicitly, it does not fall out of QR/cash for free.

---

## 3. What all is affected

| # | File | Change |
|---|---|---|
| 1 | `internal/billing/models.go` | Add `PaymentType` field + type/consts; change `PrescriptionID` to `*string`; add `AppointmentID *string` (nullable, `uniqueIndex`) |
| 2 | `internal/billing/constants.go` | Add `PaymentTypeConsultation`, `PaymentTypePrescription` |
| 3 | `internal/billing/migrate.go` | `AutoMigrate` picks up both new/changed columns, **but** dropping the existing `NOT NULL` on `prescription_id` (if present in the live DB) needs an explicit `ALTER TABLE` — GORM `AutoMigrate` does not relax constraints on existing columns. See [Migration plan](#4-migration-plan). |
| 4 | `internal/billing/dto/request.go` (`CheckoutReq`) | Add `PaymentType string` and `AppointmentID string` fields |
| 5 | `internal/billing/controllers.go` (`Checkout`) | Parse `payment_type` and `appointment_id`; make `prescription_id`/`dispense_items` vs. `appointment_id` parsing conditional instead of always-required |
| 6 | `internal/billing/invoice.services.go` (`CreateInvoice`, `toInvoiceModel`) | Branch: only call `addInvoiceItems` when `payment_type == prescription`; validate appointment (exists, tenant/patient match, not already billed) when `payment_type == consultation`; for prescription invoices, resolve `prescription.appointment_id` and stamp it too; set description text per type; `toInvoiceModel` sets `PrescriptionID = nil` / `AppointmentID = nil` as appropriate |
| 7 | `internal/billing/invoice-item.services.go` | No logic change needed — just not called for consultation invoices |
| 8 | `internal/billing/inovice.repository.go` | Add `GetInvoiceByAppointmentID` (mirrors existing `GetInvoiceByPrescriptionID`) — used both for the double-billing pre-check and for consultation-invoice lookup; `GetInvoiceByID` already exists here but isn't exposed via a route — worth exposing too |
| 9 | `pkg/middleware/routers/functions.go` (`RegisterBillingRoutes`) | Add `billingGrp.Get("/getInvoiceByAppointmentID/:appointmentID", ...)` (consultation equivalent of `getInvoiceByPrescriptionID`); optionally also `getInvoiceByID/:invoiceID` |
| 10 | `internal/billing/invoice.services.go` → depends on `internal/appointments` | **New cross-package dependency**: `InvoiceServ` needs a narrow read-only dependency on `AppointmentService.GetAppntmentByID` (`internal/appointments/appointment.services.go:663`) to validate the appointment before creating a consultation invoice — same pattern as the existing `PatientServ` dependency it already has for patient lookup |
| 11 | `internal/payments/payments.services.go` (description text) | Description currently hardcoded `"please pay the amount to get prescribed medicine"` — needs a consultation-appropriate string |
| 12 | `internal/payments/fulfillment_service.go` | **No change required** (see finding above) |
| 13 | `internal/payments/webhook_service.go`, `payments.services.go::ConfirmManualPayment` | **No change required** — both call the same shared `FulfillPaidInvoice` |
| 14 | `internal/appointments/appointment.services.go` (`CreateApptmnt`) | **No change** — booking stays invoice-agnostic per the "separate call" decision; `GetAppntmentByID` is only *read*, not modified |
| 15 | `shared/error` (wrapError) | Add `ErrInvalidPaymentType`, `ErrAppointmentNotFound` (if not already present), `ErrAppointmentAlreadyBilled` |
| 16 | `docs/checkout-webhook-flow.md`, `docs/payment-confirm-webhook-test-scenarios.md` | Update once implemented — both currently describe only the prescription path |
| 17 | Existing data / rows | Backfill: every existing invoice row is implicitly a prescription invoice → `UPDATE invoices SET payment_type = 'prescription'`; `appointment_id` stays `NULL` for old rows (acceptable — see migration plan) |

Not affected (confirmed while tracing the code, worth stating explicitly so nothing gets missed):

* `internal/medicine/*` — inventory/stock code is only reached via `invoice_items`, which consultation invoices never populate.
* `internal/prescription/*` — same reasoning; `ResolveAndUpdateParentPrescriptionStatus` is skipped when there's no `prescription_id`.
* `internal/appointments/*` — only a new **read-only** call site from billing (`GetAppntmentByID`); no changes to appointment models, services, or the booking flow itself.
* `internal/payments/models.go` (`Payments` struct) — no new field needed; `payment_type`/`appointment_id` live on `Invoice`, reachable via `payments.invoice_id` join for reporting.
* `internal/payments/webhook_service.go` — Razorpay webhook path is **ignored for consultation for now** (decision, see decisions table above): booking-time consultation payment is cash/QR-in-person, not a payment link. It's not hard-blocked (`payment_mode: "link"` would still flow through the same generalized `CreateInvoice`, untouched, and land on this same webhook path), it's simply not a flow being designed/tested against right now.

---

## 4. Migration plan

This codebase runs `AutoMigrate` on every server boot (`shared/migration/functions.go` → `billing.Migrate` → `db.AutoMigrate(&Invoice{})`, `internal/billing/migrate.go:8`), so most of this happens automatically the moment the updated struct is deployed and the server restarts — no separate migration tool/script needed. The one exception is step 2, which `AutoMigrate` cannot do for you.

| # | Step | Automatic via `AutoMigrate` on restart? |
|---|---|---|
| 1 | Add column `payment_type varchar(30) NOT NULL DEFAULT 'prescription'` | **Yes** — new column, GORM adds it from the struct tag. Postgres applies the default to all existing rows as part of adding the column (fast, no table rewrite on PG 11+), so every pre-existing invoice reads back as `prescription` immediately — no manual backfill needed. |
| 2 | Relax constraint: `ALTER TABLE invoices ALTER COLUMN prescription_id DROP NOT NULL;` | **No** — `AutoMigrate` only adds *missing* columns/indexes, it never alters constraints on a column that already exists (by design, to avoid silently doing something destructive). If `prescription_id` is currently `NOT NULL` in the live DB, this needs a manual raw-SQL step run once, separately. (The Go struct tag today shows no `not null` on `PrescriptionID` — confirm against the actual live schema, since the tag and the DB can drift; this step may turn out to be a no-op.) |
| 3 | Add column `appointment_id uuid NULL` + its `uniqueIndex` | **Yes** — brand-new column and brand-new index, nothing pre-existing to conflict with, both created automatically from the struct tags. |
| 4 | (Optional) backfill `invoices.appointment_id` from `prescriptions.appointment_id` for old prescription invoices | Manual, one-off SQL — not required for correctness, purely improves historical reporting, safe to skip/defer. |
| 5 | `invoice_items` table | No change needed at all. |
| 6 | Deploy code that writes `payment_type`/`appointment_id` on every *new* invoice | This is the app-code change, not a DB step — without it, new rows would just silently keep using the column default. |

Net effect: for this specific change-set, only step 2 needs a person to run something by hand before/alongside the deploy; everything else is covered by the existing restart-time `AutoMigrate` call. Still worth doing step 2 against a staging DB first and confirming the live schema, rather than assuming the Go tag and the DB agree.

---

## 5. Open questions / follow-ups

1. ~~No `appointment_id` on `Invoice`~~ — **resolved by this update**: `AppointmentID *string` is now part of the design (§2.1), with a `uniqueIndex` doubling as the double-billing guard and a service-layer existence/ownership check (§2.2).
1a. ~~Payment link for consultation~~ — **resolved: out of scope for now**, see decisions table at the top. Only cash/QR (`ConfirmManualPayment`) is being supported/tested for consultation invoices in this pass.
2. **One appointment → one consultation invoice, forever?** The `uniqueIndex` on `appointment_id` assumes a consultation is billed at most once. If there's ever a legitimate reason to re-bill against the same appointment (e.g. a correction after a refund/void), the unique constraint will block it — decide now whether "void + recreate" or "amend in place" is the intended correction path, so it's not discovered under pressure later.
3. **"Swipe" ambiguity.** If this means a physical card/POS terminal (not UPI QR), `ConfirmManualPayment` (`internal/payments/payments.services.go:384`) currently hard-rejects any `payment_mode` other than `cash`/`qr`. There's an unused `PaymentExternal = "external"` constant already sitting in `pkg/constants/constant.go:47` that looks like it was reserved for exactly this — worth confirming intent before wiring it up.
4. **Exposing `GetInvoiceByID`.** The repo method already exists (`internal/billing/inovice.repository.go:54`, used internally by retry-payment-link) but there's no public route for it. Needed if frontend wants to poll/fetch a consultation invoice by ID after creating it (`getInvoiceByAppointmentID` from §3 covers the "look it up by appointment" case, but not "I already have the invoice ID").
5. **Payment description text / notification copy** for consultation vs. prescription — currently one hardcoded string in `payments.services.go`; needs a small copy decision (not just engineering).
6. **Default `payment_mode` when omitted.** You mentioned "keep cash by default" — confirm whether that default should be enforced server-side (fallback to `cash` if `payment_mode` missing) or is purely a frontend UI default with the field always sent explicitly. Current code (`CreateInvoice`) returns `ErrUnsupportedPaymentMode` if it's missing/unrecognized, i.e. no server-side default today.
