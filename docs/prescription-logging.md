# Prescription Logging Plan

Review doc for what to log in the prescription domain and where. Implementation follows the approved recommendations below; adjust remaining open decisions if needed.

Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for authentication, patient, and appointments. Request-level `request started` / `request completed` / `client error` / `request failed` already cover the HTTP envelope. This plan covers **domain prescription events** on top of that.

Package: `internal/prescription`  
Routes: `/api/v1/prescription/*` (see `RegisterPrescriptionRoutes`)

---

## 1. Scope

| Area | Endpoint / entry | Files |
|------|------------------|-------|
| Create prescription | `POST /v1/prescription/create` | `controllers.go`, `prescriptions.services.go`, `prescriptions-items.services.go`, repos |
| List (org) | `GET /v1/prescription/get` | `controllers.go`, `prescriptions.services.go`, `prescriptions.repository.go` |
| List by status | `GET /v1/prescription/getByStatus` | same |
| Add items | `PATCH /v1/prescription/updatePrescriptions` | controllers + item service/repo |
| Update one item | `PATCH /v1/prescription/updatePrescriptionItem` | controllers + item service/repo |
| Items by prescription ID | `GET /v1/prescription/getprescriptionbyPid` | item service/repo |
| By appointment | `POST /v1/prescription/getPrescriptionByAppointmentID` | prescriptions service/repo |
| By patient | `GET /v1/prescription/getPrescriptionByPatientID` | prescriptions service/repo |
| Update status (manual) | `PATCH /v1/prescription/updateStatus` | prescriptions service (+ appointment status + notification on `sent`) |
| Medicine info (billing UI) | `GET /v1/prescription/getMedicineInfo/:prescription_id` | item service/repo |
| External status update | `UpdateExtPrescriptionStatus` — payments / checkout | `prescriptions.services.go` |
| Resolve parent status after dispense | `ResolveAndUpdateParentStatus` — payment fulfillment | same |
| Qty map for invoice | `GetqtyByMedicine` — billing invoice items | `prescriptions-items.services.go` |
| Dispense qty / item status | `UpdateDispenseItemQty`, `UpdateIPrescriptionStatus` — fulfillment | item service/repo |

**Auth note:** prescription routes are **not** behind JWT `Authenticate` today. RequestLogger still applies globally. JWT is a follow-up (same as appointments).

**Partial today:** `CreatePrescription` already accepts `*zap.Logger` and passes it into `GetAppntmentByID`. No prescription domain log lines yet.

**Out of scope for this pass**

- Notification worker / email template internals — only enqueue handoff from status → `sent`
- Full billing / payment domain logs (only prescription-side methods they call)
- Delete prescription (unused / stub)
- Dispense HTTP API (`DispensePayload` DTO exists; no dedicated controller route yet)
- Medicine name / dosage / food_instruction values in log fields

---

## 2. Security / PII rules (non-negotiable)

Prescriptions join patient, doctor, and medicine data. Prefer IDs over PII / clinical detail.

**Never log**

- Full request bodies / medicine arrays (frequency, duration, food instructions, dosages)
- Patient name, email, doctor username
- Notification payload map (contains patient name/email + medicine list)
- Medicine names / strengths / forms on success paths
- Raw `search` string (log `has_search` only)

**Safe to log**

- `prescription_id`, `prescription_code`, `prescription_item_id`
- `organisation_id`, `patient_id`, `appointment_id`, `prescribed_by` / `created_by` (`user_id`)
- `medicine_id` (UUID only)
- `status` / filter status enums
- Counts: `item_count`, `count`, `total`, `limit`, `page_no` / `offset`
- Booleans: `notification_enqueued`, `has_search`, `has_appointment_id`
- Dispense: `dispensed_qty`, `prescription_item_id`

**Borderline**

- Frequency / duration values — **omit** on INFO; allow `field=` name only on validation WARN
- Empty `medicine_array` — log `item_count=0`; do not dump contents

---

## 3. Propagation approach

Same as auth / patient / appointments:

```
RequestLogger (request_id, method, path, ip)
  → Controller: middleware.GetLogger(c)
  → Service: accept *zap.Logger as first arg
  → Item service: accept *zap.Logger as first arg
  → Repository: log only on DB errors
```

For non-HTTP callers (`UpdateExtPrescriptionStatus`, dispense helpers from payment fulfillment):

| Caller | How to pass logger |
|--------|-------------------|
| Payment fulfillment (has request logger) | Pass through from webhook/confirm path |
| Payment fulfillment (no request logger yet) | `logger.Log` or `ensureLog(nil)` |
| Billing invoice item build | `GetqtyByMedicine(logger.Log, id)` until billing logging lands |
| Future HTTP path | `middleware.GetLogger(c)` |

**Recommendation:** required `log *zap.Logger` first arg (HTTP-agnostic, testable). No variadic.

---

## 4. What to log — by flow

### 4.1 Create prescription (`POST /prescription/create`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| C1 | Controller | Payload / mandatory field parse failure | WARN | `prescription create request invalid` | `error`, `field` if known (`appointment_id` / `organisation_id` / `prescribed_by` / `medicine_array`) |
| C2 | Controller | After parse OK | INFO | `prescription create attempt` | `organisation_id`, `appointment_id`, `prescribed_by`, `item_count` |
| C3 | Service | Appointment lookup failed | WARN | `prescription create failed` | `appointment_id`, `organisation_id`, `reason=appointment_not_found` (or `appointment_lookup`), `error` |
| C4 | Service | DB create failed | ERROR | `prescription create failed` | `organisation_id`, `appointment_id`, `reason=db_create`, `error` |
| C5 | Service / Item | Add items failed | ERROR | `prescription create failed` | `prescription_id`, `organisation_id`, `reason=db_add_items` (or `duplicate_medicine`), `error` |
| C6 | Service | Tx committed | INFO | `prescription create success` | `prescription_id`, `prescription_code`, `organisation_id`, `patient_id`, `appointment_id`, `prescribed_by`, `item_count`, `status=draft` |
| C7 | Repository | `CreatePrescription` DB error | ERROR | `prescription repo error` | `op=CreatePrescription`, `error` |
| C8 | Item repository | `AddItems` / `GetMedicineIDsByPrescriptionID` DB error | ERROR | `prescription item repo error` | `op`, `error` |

**Notes**

- Create starts as `draft`. Notification is **not** sent on create (only on status → `sent`).
- Do not log medicine names, dosages, or food instructions.
- `toMedicineArray` currently ignores per-field parse errors (`_`). Logging empty `medicine_id` as validation is enough; tightening item validation is **out of scope** unless bundled.
- On item failure after prescription insert, tx rolls back — log failure only, no success line.

### 4.1.1 Frontend-facing create errors

| Case | Internal log `reason` | HTTP (proposed) | API `error` | UI copy (suggested) |
|------|----------------------|-----------------|-------------|---------------------|
| Missing / invalid body fields | `prescription create request invalid` | `400` | `invalid request` | “Please check prescription details and try again.” |
| Appointment not found | `appointment_not_found` | `404` | `appointment not found` | “Appointment not found.” |
| Duplicate medicine | `duplicate_medicine` | `409` | `medicine already present in prescription` | “This medicine is already on the prescription.” |
| DB / unexpected | `db_create` / `db_add_items` | `500` | `failed to create prescription` | “Something went wrong. Please try again.” |

**Decision to confirm:** today create returns `409` with raw `err.Error()`. Recommend mapping as above.

---

### 4.2 List prescriptions (`GET /prescription/get`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| L1 | Controller | Query parse failure | WARN | `prescription list request invalid` | `error` |
| L2 | Controller | Missing `organisation_id` | WARN | `prescription list request invalid` | `reason=missing_organisation_id` |
| L3 | Controller | Request entered | INFO | `prescription list attempt` | `organisation_id`, `limit`, `offset`, `has_search` |
| L4 | Service / Repo | FindMany or Count failed | ERROR | `prescription list failed` | `organisation_id`, `reason=db_read` / `db_count`, `error` |
| L5 | Service | Success | INFO | `prescription list success` | `organisation_id`, `count`, `total` |
| L6 | Repository | DB errors on `FindMany` / `Count` | ERROR | `prescription repo error` | `op`, `error` |

**Notes**

- Do not log individual prescription rows.
- Empty list (`total=0`) is still **success** (INFO), not WARN.
- Search is by prescription `code` only today; still omit raw `search` text.

### 4.2.1 Frontend-facing list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing org / bad query | `400` | `invalid request` | Show validation |
| DB / unexpected | `500` | `failed to fetch prescriptions` | Retry / toast |

---

### 4.3 List by status (`GET /prescription/getByStatus`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| FS1 | Controller | Query parse failure | WARN | `prescription status list request invalid` | `error` |
| FS2 | Controller | Missing `organisation_id` or `status` | WARN | `prescription status list request invalid` | `reason=missing_organisation_id` / `missing_status` |
| FS3 | Controller | Request entered | INFO | `prescription status list attempt` | `organisation_id`, `status`, `limit`, `offset` |
| FS4 | Service | Invalid status string | WARN | `prescription status list failed` | `organisation_id`, `status`, `reason=invalid_status` |
| FS5 | Service / Repo | FindByStatus or CountByStatus failed | ERROR | `prescription status list failed` | `organisation_id`, `status`, `reason=db_read` / `db_count`, `error` |
| FS6 | Service | Success | INFO | `prescription status list success` | `organisation_id`, `status`, `count`, `total` |
| FS7 | Repository | DB errors | ERROR | `prescription repo error` | `op=FindByStatus` / `CountByStatus`, `error` |

### 4.3.1 Frontend-facing status-list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Invalid / missing params or status | `400` | `invalid request` | Show validation |
| DB / unexpected | `500` | `failed to fetch prescriptions` | Retry |

---

### 4.4 Add items (`PATCH /prescription/updatePrescriptions`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| A1 | Controller | Payload / mandatory field failure | WARN | `prescription items add request invalid` | `error`, `field` if known |
| A2 | Controller | After parse OK | INFO | `prescription items add attempt` | `prescription_id`, `prescribed_by`, `item_count` |
| A3 | Item service | Duplicate medicine | WARN | `prescription items add failed` | `prescription_id`, `medicine_id`, `reason=duplicate_medicine` |
| A4 | Item service / Repo | DB failure | ERROR | `prescription items add failed` | `prescription_id`, `reason=db_read_existing` / `db_add_items`, `error` |
| A5 | Service | Success | INFO | `prescription items add success` | `prescription_id`, `item_count`, `prescribed_by` |
| A6 | Item repository | DB errors | ERROR | `prescription item repo error` | `op=AddItems` / `GetMedicineIDsByPrescriptionID`, `error` |

### 4.4.1 Frontend-facing add-items errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing / invalid body | `400` | `invalid request` | Show validation |
| Duplicate medicine | `409` | `medicine already present in prescription` | Show conflict |
| DB / unexpected | `500` | `failed to update prescription` | Retry |

---

### 4.5 Update one item (`PATCH /prescription/updatePrescriptionItem`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| UI1 | Controller | Mandatory field failure | WARN | `prescription item update request invalid` | `error`, `field` |
| UI2 | Controller | After parse OK | INFO | `prescription item update attempt` | `prescription_item_id`, `medicine_id` |
| UI3 | Item service | Not found | WARN | `prescription item update failed` | `prescription_item_id`, `reason=not_found` |
| UI4 | Item service | Already dispensed | WARN | `prescription item update failed` | `prescription_item_id`, `reason=already_dispensed`, `status` |
| UI5 | Item service / Repo | DB update failed | ERROR | `prescription item update failed` | `prescription_item_id`, `reason=db_update`, `error` |
| UI6 | Item service | Success | INFO | `prescription item update success` | `prescription_item_id`, `medicine_id` |
| UI7 | Item repository | DB errors | ERROR | `prescription item repo error` | `op=GetPrescriptionItemByID` / `UpdatePrescriptionItem`, `error` |

### 4.5.1 Frontend-facing item-update errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Invalid body | `400` | `invalid request` | Show validation |
| Not found | `404` | `prescription item not found` | Refresh |
| Already dispensed | `409` | `cannot edit dispensed item` | Disable edit |
| DB / unexpected | `500` | `failed to update prescription item` | Retry |

---

### 4.6 Get items by prescription ID (`GET /prescription/getprescriptionbyPid`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| G1 | Controller | Missing `prescription_id` | WARN | `prescription items get request invalid` | `reason=missing_prescription_id` |
| G2 | Controller | Request entered | INFO | `prescription items get attempt` | `prescription_id`, `limit`, `offset` |
| G3 | Item service / Repo | Read or Count failed | ERROR | `prescription items get failed` | `prescription_id`, `reason=db_read` / `db_count`, `error` |
| G4 | Item service | Success | INFO | `prescription items get success` | `prescription_id`, `count`, `total` |
| G5 | Item repository | DB errors | ERROR | `prescription item repo error` | `op`, `error` |

### 4.6.1 Frontend-facing get-items errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing ID | `400` | `invalid request` | — |
| DB / unexpected | `500` | `failed to fetch prescription items` | Retry |

---

### 4.7 Get by appointment (`POST /prescription/getPrescriptionByAppointmentID`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| GA1 | Controller | Mandatory field failure | WARN | `prescription appointment list request invalid` | `error`, `field` |
| GA2 | Controller | After parse OK | INFO | `prescription appointment list attempt` | `appointment_id`, `organisation_id`, `limit`, `page_no` |
| GA3 | Service / Repo | Read or Count failed | ERROR | `prescription appointment list failed` | `appointment_id`, `organisation_id`, `reason=db_read` / `db_count`, `error` |
| GA4 | Service | Success | INFO | `prescription appointment list success` | `appointment_id`, `organisation_id`, `count`, `total` |
| GA5 | Repository | DB errors | ERROR | `prescription repo error` | `op`, `error` |

### 4.7.1 Frontend-facing appointment-list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Invalid body | `400` | `invalid request` | Show validation |
| DB / unexpected | `500` | `failed to fetch prescriptions` | Retry |

---

### 4.8 Get by patient (`GET /prescription/getPrescriptionByPatientID`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| GP1 | Controller | Query parse failure | WARN | `prescription patient list request invalid` | `error` |
| GP2 | Controller | Missing `patient_id` | WARN | `prescription patient list request invalid` | `reason=missing_patient_id` |
| GP3 | Controller | Request entered | INFO | `prescription patient list attempt` | `patient_id`, `limit`, `page_no` |
| GP4 | Service / Repo | Read or Count failed | ERROR | `prescription patient list failed` | `patient_id`, `reason=db_read` / `db_count`, `error` |
| GP5 | Service | Success | INFO | `prescription patient list success` | `patient_id`, `count`, `total` |
| GP6 | Repository | DB errors | ERROR | `prescription repo error` | `op`, `error` |

### 4.8.1 Frontend-facing patient-list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing patient / bad query | `400` | `invalid request` | Show validation |
| DB / unexpected | `500` | `failed to fetch prescriptions` | Retry |

---

### 4.9 Update status (`PATCH /prescription/updateStatus`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| US1 | Controller | Payload / mandatory field failure | WARN | `prescription status update request invalid` | `error`, `field` |
| US2 | Controller | After parse OK | INFO | `prescription status update attempt` | `prescription_id`, `status`, `has_appointment_id` |
| US3 | Service | Invalid status string | WARN | `prescription status update failed` | `prescription_id`, `status`, `reason=invalid_status` |
| US4 | Service | `sent` but `appointment_id` empty | WARN | `prescription status update failed` | `prescription_id`, `status`, `reason=missing_appointment_id` |
| US5 | Service | Appointment status update failed | ERROR | `prescription status update failed` | `prescription_id`, `appointment_id`, `reason=appointment_status`, `error` |
| US6 | Service | Prescription status DB update failed | ERROR | `prescription status update failed` | `prescription_id`, `status`, `reason=db_update`, `error` |
| US7 | Service | Notify payload build failed (after commit, status=`sent`) | ERROR | `prescription status notify failed` | `prescription_id`, `reason=notification_payload`, `error` |
| US8 | Service | Success | INFO | `prescription status update success` | `prescription_id`, `status`, `appointment_id` (if any), `notification_enqueued` |
| US9 | Repository | `UpdateStatus` DB error | ERROR | `prescription repo error` | `op=UpdateStatus`, `error` |

**Notes**

- When status is `sent`, appointment is marked completed and a notification is enqueued.
- **Recommend:** if notify payload fails after DB commit, still return HTTP 200 and log ERROR (do not fail the client after a successful write).
- Pass request logger into appointment `UpdateStatus` (today passes `nil`).

### 4.9.1 Frontend-facing status errors

| Case | Internal log `reason` | HTTP | API `error` | Frontend |
|------|----------------------|------|-------------|----------|
| Missing / invalid body | request invalid | `400` | `invalid request` | Show validation |
| Invalid status | `invalid_status` | `400` | `invalid request` | Show validation |
| `sent` without appointment | `missing_appointment_id` | `400` | `invalid request` | Require appointment |
| DB / appointment update | `db_update` / `appointment_status` | `500` | `failed to update prescription` | Retry |
| Notify payload fail (post-commit) | `notification_payload` | `200` | success | Treat as success; ops use logs |

---

### 4.10 Medicine info (`GET /prescription/getMedicineInfo/:prescription_id`)

Used by billing UI to show medicines + available batches.

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| MI1 | Controller | Missing `prescription_id` param | WARN | `prescription medicine info request invalid` | `reason=missing_prescription_id` |
| MI2 | Controller | Request entered | INFO | `prescription medicine info attempt` | `prescription_id` |
| MI3 | Item service / Repo | Read or Count failed | ERROR | `prescription medicine info failed` | `prescription_id`, `reason=db_read` / `db_count`, `error` |
| MI4 | Item service | Success | INFO | `prescription medicine info success` | `prescription_id`, `count`, `total` |
| MI5 | Controller | Response encode / current bare `return` on err | ERROR | `prescription medicine info response failed` | `prescription_id`, `error` |
| MI6 | Item repository | DB errors | ERROR | `prescription item repo error` | `op=FindMedicineInfoByPID` / `GetTotalCountByPrescID`, `error` |

**Fix when implementing:** controller currently does `return` without wrapping on error — always wrap and return a client-safe error.

### 4.10.1 Frontend-facing medicine-info errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing ID | `400` | `invalid request` | — |
| DB / unexpected | `500` | `failed to fetch medicine info` | Retry |

---

### 4.11 External status (`UpdateExtPrescriptionStatus`)

Used by payments (checkout, webhook, confirm) via fulfillment adapter — not a public prescription route.

**Logger API:** required `*zap.Logger` as first argument (after we change the fulfillment interface, or pass logger separately — see decisions).

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| EX1 | Service | Invalid status string | WARN | `prescription external status failed` | `prescription_id`, `status`, `reason=invalid_status` |
| EX2 | Service / Repo | DB update failed | ERROR | `prescription external status failed` | `prescription_id`, `status`, `reason=db_update`, `error` |
| EX3 | Service | Success | INFO | `prescription external status success` | `prescription_id`, `status` |
| EX4 | Repository | `UpdateStatus` DB error | ERROR | `prescription repo error` | `op=UpdateStatus`, `error` |

**Call sites to update:** `payments.services.go`, `webhook_service.go`, `appinit/payment_fulfillment.go` (adapter), and any interface in `common-interface.go`.

**Interface impact:** `IPaymentFulfillment` / `PrescriptionStatus` currently have no logger arg. Options:

1. Add `log *zap.Logger` to interface methods (touches payments + fulfillment)  
2. Keep interface as-is; adapter uses `logger.Log` when calling prescription service  

**Recommendation:** option 2 for this pass (smaller blast radius); option 1 when payments logging is done.

---

### 4.12 Resolve parent status (`ResolveAndUpdateParentStatus`)

Used by payment fulfillment after dispense.

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| RP1 | Service | Computed parent status | INFO | `prescription parent status resolved` | `prescription_id`, `status` (`completed`\|`tentative`), `partial_item_count` |
| RP2 | Service | Update failed | ERROR | (delegates to EX2) | — |

---

### 4.13 Qty map (`GetqtyByMedicine`)

Used by billing when building invoice items.

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| Q1 | Item service | Lookup failed | ERROR | `prescription qty map failed` | `prescription_id`, `reason=db_read`, `error` |
| Q2 | Item service | Success | DEBUG | `prescription qty map success` | `prescription_id`, `medicine_count` |

---

### 4.14 Dispense qty / item status (`UpdateDispenseItemQty`, `UpdateIPrescriptionStatus`)

Used by payment fulfillment after payment.

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| DI1 | Item service | Qty update failed | ERROR | `prescription item dispense failed` | `prescription_item_id`, `dispensed_qty`, `reason=db_update`, `error` |
| DI2 | Item service | Qty update success | DEBUG | `prescription item dispense success` | `prescription_item_id`, `dispensed_qty` |
| DI3 | Item service | Status update failed | ERROR | `prescription item status failed` | `prescription_item_id`, `status`, `reason=db_update`, `error` |
| DI4 | Item service | Status update success | INFO | `prescription item status success` | `prescription_item_id`, `status` |

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/prescription/controllers.go` | Entry / invalid request logs; pass logger; fix `FindMedicineDetInfo` error wrap |
| `internal/prescription/prescriptions.services.go` | Business outcomes (create, lists, status, external status) |
| `internal/prescription/prescriptions-items.services.go` | Item add/update/get/medicine-info/dispense outcomes |
| `internal/prescription/prescriptions.repository.go` | DB errors only |
| `internal/prescription/prescriptions-items.repository.go` | DB errors only |
| `internal/prescription/log.go` | `ensureLog` helper (new) |
| `shared/error/structures.go` | Optional: `ErrPrescriptionNotFound`, `ErrPrescriptionCreateFailed`, `ErrPrescriptionFetchFailed`, `ErrPrescriptionUpdateFailed`, `ErrMedicineAlreadyPresent` (move from package-local), etc. |
| `appinit/payment_fulfillment.go` | Pass logger into prescription methods (or use `logger.Log`) |
| `internal/billing/invoice-item.services.go` | Pass logger into `GetqtyByMedicine` when signature changes |
| `internal/payments/*` | Only if we change fulfillment interface (see decisions) |

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | From RequestLogger child logger |
| `prescription_id` | After create / on get / status / items |
| `prescription_code` | Create success |
| `prescription_item_id` | Item update / dispense |
| `organisation_id` | Create / org-scoped lists |
| `patient_id` | Create success, patient list |
| `appointment_id` | Create, appointment list, status=`sent` |
| `prescribed_by` | Create / add items |
| `medicine_id` | Item ops / duplicate |
| `status` | Status update / filter |
| `item_count`, `count`, `total` | Create / lists |
| `limit`, `offset`, `page_no` | Pagination |
| `has_search`, `has_appointment_id` | Optional filters present |
| `dispensed_qty` | Dispense |
| `notification_enqueued` | Status `sent` success |
| `reason` | Stable enum for failures |
| `field` | Validation / missing body field name |
| `op` | Repository operation |
| `error` | `zap.Error(err)` on WARN/ERROR with an error |

---

## 7. What we will **not** add in this pass

- Logging full medicine arrays or clinical fields (frequency, duration text, food instructions)
- Logging patient name / email / doctor name
- Notification delivery provider logs
- JWT middleware on prescription routes
- Dispense HTTP endpoint (not implemented)
- Expanding billing/payment logging beyond prescription method signatures / adapter pass-through

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on prescription HTTP routes | Done (global middleware) |
| `CreatePrescription` accepts logger + passes to appointment | Done |
| Prescription controller domain logs | Done |
| Prescription service domain logs | Done |
| Prescription item service domain logs | Done |
| Prescription / item repo DB error logs | Done |
| Client-safe / consistent HTTP status mapping | Done |
| `FindMedicineDetInfo` error wrap | Done |
| External / dispense logger threading | Done (adapter uses `logger.Log`) |
| JWT on prescription routes | Not present (follow-up) |

---

## 9. Implementation order (after approval)

1. Add `log.go` (`ensureLog`); add shared prescription sentinels  
2. Thread `*zap.Logger` controller → service → item service → repo  
3. Create prescription (C1–C7) + client error mapping  
4. Update status (St1–St7) + notify behavior  
5. Add items (A1–A5) + update item (U1–U6)  
6. Read paths: list, status list, by appointment, by patient, items by PID, medicine info  
7. Internal: `UpdateExtPrescriptionStatus`, `ResolveAndUpdateParentStatus`, dispense helpers — update call sites  
8. Optional: JWT on prescription group  

---

## 10. Open decisions for review

Please mark yes/no or alternatives:

1. **Omit patient/doctor names and medicine clinical fields from all prescription logs?** — **Recommend: yes**
2. **Pass `*zap.Logger` into prescription + item service methods?** — **Recommend: yes**
3. **Map HTTP statuses** (400 / 404 / 409 / 500) instead of mixed raw errors? — **Recommend: yes** (with shared sentinels)
4. **Allow empty `medicine_array` on create** (current) and only log `item_count`? — **Recommend: yes** for this pass
5. **Require `appointment_id` when status=`sent`?** — **Recommend: yes**
6. **Notify failure after `sent` commit: return 200 + ERROR log?** — **Recommend: yes**
7. **Reject unknown status on manual update** (don’t write arbitrary strings)? — **Recommend: yes**
8. **Duplicate medicine → 409** with stable message? — **Recommend: yes**
9. **Include external/dispense helpers in this pass?** — **Recommend: yes**
10. **Org list: `has_search` only, never raw search text?** — **Recommend: yes**
11. **Add JWT `Authenticate` to prescription routes in this pass?** — **Recommend: follow-up**
12. **Logger style:** required `log *zap.Logger` first arg (no variadic) — **yes**

---

## 11. Approval

Once this list looks right, next step is implementing in the order in §9 without expanding into billing/payment domain docs or JWT.
