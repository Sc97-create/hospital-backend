# Payment Type on Invoice — Implementation Log

Companion to the design doc: `docs/payment-type-invoice-design.md`. That doc has the *why*
and the flow diagrams; this doc is the *what changed* — every function touched, per file,
plus what deliberately deviated from the design sketch and what's still open.

## Status: Implemented (backend), not yet deployed

Verified locally: `go build ./...`, `go vet` (billing/payments/appinit/routers/shared — clean),
`gofmt` clean on all touched files, no new linter errors. No unit tests existed for
`internal/billing`/`internal/payments` before this change, so none were run/added here.

---

## 1. `internal/billing/models.go` + `internal/billing/constants.go`

| Change | Detail |
|---|---|
| **Added** `Invoice.PaymentType PaymentType` | `gorm:"type:varchar(30);not null;default:prescription"` — new column, existing rows read back as `prescription` automatically (see §7). `PaymentType` type + `PaymentTypeConsultation`/`PaymentTypePrescription` consts live in `internal/billing/constants.go`, alongside `StatusUnpaid`/`StatusPaid`/etc. |
| **Changed** `Invoice.PrescriptionID` from `string` to `*string` | Still `uniqueIndex`. Nullable so consultation invoices don't collide on `''` (Postgres treats every `NULL` as distinct, every `''` as equal). |
| **Added** `Invoice.AppointmentID *string` | `uniqueIndex`, nullable. Required for consultation invoices; the double-billing guard for "one consultation invoice per appointment." |

`InvoiceItem` — unchanged.

---

## 2. `internal/billing/dto/request.go` / `response.go`

| File | Change |
|---|---|
| `request.go` | `CheckoutReq` gained `PaymentType string` (`json:"payment_type"`, optional) and `AppointmentID string` (`json:"appointment_id"`, required only for consultation). |
| `response.go` | `InvoiceByPrescriptionResponse` gained `PaymentType string` and `AppointmentID string` (both `omitempty` where relevant). This struct is now reused by both the prescription and the appointment lookup — see §5. |

---

## 3. `internal/billing/invoice.services.go` — the bulk of the change

`CreateInvoice` was already ~148 LOC before this change (over the repo's 60-LOC-per-function
cap in `docs/coding-standards.md`). Rather than add more branching on top of an
already-oversized function, it was split into single-purpose helpers. Net result: `CreateInvoice`
is now ~48 LOC and reads as a straight-line list of steps; each step's error handling lives in
its own helper.

| Function | Status | What it does |
|---|---|---|
| `NewInvoiceServ(...)` | **Changed** | Added 6th param `appointmentServ *appointments.AppointmentService`. New field `InvoiceServ.AppointmentS`. |
| `CreateInvoice` | **Rewritten** | Now: idempotency replay → resolve+validate `payment_type` → build model → persist (tx) → commit → patient lookup → create payment → log+return. Each arrow is a helper call, not inline logic. |
| `replayIfIdempotent` | **New** | Extracted from `CreateInvoice`. Same idempotency-key replay logic as before, unchanged behavior. |
| `resolvePaymentType` | **New** | `""`/omitted → `PaymentTypePrescription` (back-compat with clients that predate this field); anything else passed through for `validatePaymentTypeInputs` to accept/reject. |
| `validatePaymentTypeInputs` | **New** | Switches on resolved type: consultation → `validateConsultationAppointment`; prescription → requires non-blank `prescription_id`; anything else → `ErrInvalidPaymentType`. |
| `validateConsultationAppointment` | **New** | For consultation checkouts only: requires non-blank `appointment_id`; loads the appointment via `AppointmentS.GetAppntmentByID` (propagates `ErrAppointmentNotFound`/`ErrAppointmentFetchFailed` as-is); rejects org/patient mismatch (`ErrAppointmentMismatch`); rejects if an invoice already exists for that appointment (`ErrAppointmentAlreadyBilled`), via `GetInvoiceByAppointmentID`. |
| `toInvoiceModel` | **Changed** | Added `paymentType PaymentType` param. Sets `invoice.PaymentType`; sets exactly one of `AppointmentID`/`PrescriptionID` via `nonEmptyPtr`, never both. |
| `nonEmptyPtr` | **New** | `string → *string`, blank → `nil`. Keeps the two nullable FK columns storing real SQL `NULL` instead of `''`. |
| `derefString` | **New** | `*string → string`, `nil` → `""`. Used everywhere a nullable model field needs to go back into a plain-string DTO/log field. |
| `persistInvoiceAndItems` | **New** | Extracted DB-write half of `CreateInvoice`: inserts the invoice row, then — only when `paymentType == PaymentTypePrescription` — calls the existing `addInvoiceItems`. Owns `tx.Rollback()` on any failure; caller owns `tx.Commit()`. Consultation invoices skip `addInvoiceItems` entirely (no medicines to validate/insert). |
| `mapInvoiceCreateErr` | **New** | Extracted from the old inline unique-violation handling. Now type-aware: a unique-constraint hit on a consultation invoice maps to `ErrAppointmentAlreadyBilled` instead of the generic `ErrInvoiceAlreadyExists`. |
| `lookupPatientOrFail` | **New** | Extracted patient-lookup-and-map-error block, unchanged behavior. |
| `toPaymentlinkModel` | **Changed** | Added `paymentType PaymentType` param; `Description` now comes from `paymentDescription(paymentType)` instead of a hardcoded string. |
| `paymentDescription` | **New** | `"please pay the consultation fee"` for consultation, else the original `"please pay the amount to get prescribed medicine"`. Used by both `toPaymentlinkModel` and `RetryPaymentLink`. |
| `createPaymentForMode` | **New** | Extracted the `link`/`cash`/`qr`/default switch from `CreateInvoice`, unchanged branch behavior, just isolated + reusing `invoiceID` instead of a full `Invoice` struct. |
| `logCheckoutSuccess` | **New** | Extracted the final success-log call; now also logs `payment_type` and `appointment_id`. |
| `GetInvoiceByPrescriptionID` | **Changed (internal only)** | Now calls the renamed `toInvoiceDetailResponse` instead of `toInvoiceByPrescriptionResponse`. Behavior/signature unchanged. |
| `GetInvoiceByAppointmentID` | **New** | Consultation-invoice lookup — same shape as `GetInvoiceByPrescriptionID` but `WHERE invoices.appointment_id = ?`. Needed because consultation invoices have no `prescription_id` to look up by. |
| `RetryPaymentLink` | **Changed** | `cmd.PrescriptionID` now `derefString(invoice.PrescriptionID)` (was a direct field, now nullable); `cmd.Description` now `paymentDescription(invoice.PaymentType)` instead of the hardcoded string. |
| `toInvoiceByPrescriptionResponse` | **Renamed → `toInvoiceDetailResponse`** | Same body, plus new `PaymentType`/`AppointmentID` fields (both via `derefString`/`string()` cast), used by both lookup paths now. |
| `isUniqueViolation`, `createCode`, `updateInvoiceStatus` | **Unchanged** | |

---

## 4. `internal/billing/invoice-item.services.go`

**No changes.** `addInvoiceItems` is simply not called for consultation invoices (see
`persistInvoiceAndItems` above) — nothing inside it needed to change.

---

## 5. `internal/billing/inovice.repository.go`

| Function | Status | What it does |
|---|---|---|
| `InvoiceRepo` interface | **Changed** | Added `GetInvoiceByAppointmentID(log, query, args...) (InvoiceWithPayment, error)`. |
| `(*DB).GetInvoiceByAppointmentID` | **New** | Same shape as the existing `GetInvoiceByPrescriptionID` — raw query + scan, `gorm.ErrRecordNotFound` when `row.ID == ""`. |

---

## 6. `internal/billing/controllers.go`

| Function | Status | What it does |
|---|---|---|
| `Checkout` | **Changed** | Parses `payment_type`, `supplier_id`, `appointment_id` as **optional** now (`_` discards the "missing" error) instead of hard-required. Replaced the always-required `prescription_id`/`dispense_items` parsing with a call to `parsePrescriptionFields`. Added `payment_type`/`appointment_id` to the attempt log line. |
| `parsePrescriptionFields` | **New** | If `PaymentType == "consultation"`, no-ops (nothing to parse). Otherwise parses `prescription_id` + `dispense_items` exactly as `Checkout` used to inline. |
| `wrapCheckoutError` | **Changed** | Added mappings: `ErrAppointmentAlreadyBilled` → 409 (alongside `ErrInvoiceAlreadyExists`), `ErrAppointmentNotFound` → 404 (alongside `ErrPatientNotFound`), `ErrInvalidPaymentType`/`ErrAppointmentMismatch` → 400 (alongside `ErrInvalidRequest`). |
| `GetInvoiceByAppointmentID` | **New** | Mirrors `GetInvoiceByPrescriptionID`: reads `:appointmentID` path param, calls the new service method, same not-found/invalid/500 mapping. |
| `BillingHandler` interface | **Changed** | Added `GetInvoiceByAppointmentID(c *fiber.Ctx) error`. |
| `GetInvoiceByPrescriptionID`, `RetryPaymentLink`, `tofinancialMap`, `toDispenseItems` | **Unchanged** | |

**Deliberate behavior change worth flagging:** `supplier_id` is now optional at the HTTP layer.
It was previously hard-required by `Checkout` even though nothing downstream in
`toInvoiceModel`/`CreateInvoice` ever reads `CheckoutReq.SupplierID` — it was dead weight for
every existing prescription checkout too. Making it optional unblocks consultation checkout
(which has no supplier concept) without inventing a dummy value on the frontend. If
`supplier_id` does matter for some out-of-band consumer of this payload, flag it and it can be
made conditionally required the same way `prescription_id` is.

---

## 7. `pkg/middleware/routers/functions.go`

Added one route in `RegisterBillingRoutes`:

```go
billingGrp.Get("/getInvoiceByAppointmentID/:appointmentID", billingController.GetInvoiceByAppointmentID)
```

---

## 8. `appinit/app.go`

One-line change: `billing.NewInvoiceServ(...)` now passes `appointmentSrv.Appointmentservice`
as the 6th argument. `appointmentSrv` already existed earlier in `NewContainer` (used by
`prescriptionService` already), so this was a rewire, not a new construction.

---

## 9. `shared/error/structures.go`

Added three sentinel errors:

```go
ErrInvalidPaymentType       = errors.New("invalid or unsupported payment type")
ErrAppointmentAlreadyBilled = errors.New("appointment already has a consultation invoice")
ErrAppointmentMismatch      = errors.New("appointment does not belong to this patient/organisation")
```

`ErrAppointmentNotFound` and `ErrAppointmentFetchFailed` already existed (from the appointments
domain) and are reused as-is by `validateConsultationAppointment`.

---

## 10. New request/response contract

```
POST /api/v1/billing/create
{
  "payment_type": "consultation" | "prescription",  // optional, defaults to "prescription"
  "prescription_id": "...",   // required + parsed only when payment_type = prescription
  "appointment_id": "...",    // required only when payment_type = consultation
  "patient_id": "...",
  "cashier_id": "...",
  "organisation_id": "...",
  "payment_mode": "cash" | "qr" | "link",
  "financials": { "sub_total_amount": ..., "tax_amount": ..., "discount_amount": ..., "total_amount": ... },
  "dispense_items": [ ... ]   // required + parsed only when payment_type = prescription
}

GET /api/v1/billing/getInvoiceByAppointmentID/:appointmentID   // NEW — consultation invoice lookup
GET /api/v1/billing/getInvoiceByPrescriptionID/:prescriptionID // unchanged
```

Existing prescription-checkout callers that don't send `payment_type` at all are unaffected —
`resolvePaymentType` defaults blank to `prescription`, and `parsePrescriptionFields` treats
anything other than exactly `"consultation"` as prescription-shaped.

---

## 11. Deviations from `docs/payment-type-invoice-design.md`

| Design doc said | Implementation did | Why |
|---|---|---|
| Backfill `appointment_id` onto **prescription** invoices from `prescription.appointment_id` (optional/"reporting consistency") | **Deferred, not implemented** | Would require wiring a new `PrescriptionService`/repo dependency into `InvoiceServ` purely for a cosmetic backfill the design doc itself flagged as optional. Kept this pass scoped to what's required for consultation invoices to work correctly. `AppointmentID` stays `nil` on prescription invoices for now — no functional impact, just an easy follow-up later. |
| New `ErrAppointmentAlreadyBilled` used only as a "friendlier pre-check" message | Also wired as the fallback when the DB `uniqueIndex` itself is hit (`mapInvoiceCreateErr`) | Belt-and-suspenders: if two consultation checkouts race past the pre-check, the DB constraint still fires, and now it maps to the correct error instead of the generic prescription-flavored `ErrInvoiceAlreadyExists`. |
| `supplier_id` required, per current controller | Made optional | See §6 note — it's unread downstream, and required-but-meaningless for consultation checkout. |

---

## 12. What happens on next server restart (recap)

`payment_type` and `appointment_id` (+ its unique index) are added automatically by the
existing `AutoMigrate` call chain (`shared/migration/functions.go` → `billing.Migrate` →
`db.AutoMigrate(&Invoice{})`). The one thing that is **not** automatic: if the live `invoices.prescription_id`
column is currently `NOT NULL` at the DB level, that constraint needs a manual
`ALTER TABLE invoices ALTER COLUMN prescription_id DROP NOT NULL;` — confirm against the actual
schema before deploying. Full detail in `docs/payment-type-invoice-design.md` §4.

---

## 13. Still open (unchanged from the design doc, not addressed in this pass)

1. "Swipe" / card-POS payment mode — `ConfirmManualPayment` still only accepts `cash`/`qr`.
2. Exposing a generic `getInvoiceByID/:invoiceID` route (the repo method already exists, still unrouted).
3. Whether `payment_mode` should default server-side to `cash` when omitted (currently: `ErrUnsupportedPaymentMode` if missing).
4. One-appointment-forever unique constraint — no "void + recreate" story defined yet for consultation-invoice corrections.
5. `docs/checkout-webhook-flow.md` / `docs/payment-confirm-webhook-test-scenarios.md` still describe only the prescription path — not updated in this pass.
6. **Decision (deferred, not a gap):** `payment_mode: "link"` (Razorpay payment link / webhook path) for consultation invoices is explicitly out of scope for now. Consultation checkout is cash/QR-in-person only for this pass; nothing in the code hard-blocks `link` mode for a consultation invoice (it would flow through the same `createPaymentForMode`/`webhook_service.go` untouched), it's simply not being designed or tested against.
