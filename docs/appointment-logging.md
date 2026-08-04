# Appointment Logging Plan

Review doc for what to log in the appointments domain and where. No implementation yet — approve / adjust this list, then we instrument.

Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for authentication and patient. Request-level `request started` / `request completed` / `client error` / `request failed` already cover the HTTP envelope. This plan covers **domain appointment events** on top of that.

Package: `internal/appointments`  
Routes: `/api/v1/appointment/*` (see `RegisterAppointments`)

---

## 1. Scope

| Area | Endpoint / entry | Files |
|------|------------------|-------|
| Create appointment | `POST /v1/appointment/create` | `appointment.controllers.go`, `appointment.services.go`, `appointment.repository.go` |
| Get time slots | `GET /v1/appointment/getTimeSlots` | same |
| List by organisation | `POST /v1/appointment/getappointmentbyOrgID` | same |
| Appointment preview | `GET /v1/appointment/getAppointmentsPreview` | same |
| Update status | `PATCH /v1/appointment/updateStatus` | same |
| List by patient | `POST /v1/appointment/getappointmentByPatientID` | same |
| Get by ID (internal) | `GetAppntmentByID` — called from prescription create | `appointment.services.go`, `appointment.repository.go` |
| Notification details (internal) | `GetNotificationDetails` — used after create | `appointment.services.go`, `appointment.repository.go` |

**Auth note:** appointment routes are **not** behind JWT `Authenticate` today (unlike patient). RequestLogger still applies globally. Auth rejection logs do not apply unless we add JWT in this pass (see open decisions).

**Out of scope for this pass**

- Appointment series (`appointment-series.*` — controllers/services are empty stubs)
- Delete appointment (commented as future work)
- Double-booking / slot-conflict enforcement beyond what exists today (slots mark `Allow=false`; create does not re-check occupancy)
- Notification provider / worker internals — only the fire-and-forget handoff from create
- Prescription / billing logs that only *reference* an `appointment_id`

---

## 2. Security / PII rules (non-negotiable)

Appointment flows join patient and doctor data. Prefer IDs over PII in logs.

**Never log**

- Full create / update request bodies
- `notes` / free-text clinical content
- Patient name, mobile, email, age, gender
- Doctor display name / username (use `doctor_id`)
- Notification payload map (contains `patient_name`, `patient_email_id`, etc.)
- List / preview row contents

**Safe to log**

- `appointment_id`, `appointment_code`, `organisation_id`, `patient_id`, `doctor_id`, `created_by` (`user_id`)
- `schedule_id`, `series_id` (if present)
- `visit_type`, `status` / `new_status` / `old_status` (enums)
- Scheduling metadata: `appointment_date`, slot `date` query (date only, not clock times unless needed for support)
- Pagination / filters: `limit`, `page_no`, `total`, `count`, filter enums (`date` preset, `status`, `visit_type`) — **not** the raw `search` string (often a patient name)
- Boolean outcomes: `notification_enqueued=true`, `has_series_id=true`

**Borderline**

- `start_time` / `end_time` — **omit** on INFO success by default; allow on WARN validation failures as `field=start_time` (name only) or truncated ISO if debugging slot bugs
- `search` filter text — **omit**; log `has_search=true` only

---

## 3. Propagation approach

Same as auth / patient:

```
RequestLogger (request_id, method, path, ip)
  → Controller: middleware.GetLogger(c)
  → Service: accept *zap.Logger as first arg
  → Repository: log only on DB errors
```

For non-HTTP callers (`GetAppntmentByID` from prescription):

| Caller | How to pass logger |
|--------|-------------------|
| Prescription create (has request logger) | `GetAppntmentByID(reqLogger, id)` |
| Future / missing logger | `ensureLog(nil)` → `zap.NewNop()` |

**Recommendation:** required `log *zap.Logger` as first arg on all appointment service methods (HTTP-agnostic, testable). Add `internal/appointments/log.go` with `ensureLog` (same as patient).

---

## 4. What to log — by flow

### 4.1 Create appointment (`POST /appointment/create`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| C1 | Controller | Payload / field parse failure | WARN | `appointment create request invalid` | `error`, `field` if known (`patient_id`, `user_id`, `organisation_id`, `doctor_id`, `start_time`, `end_time`, `appointment_date`, `visit_type`) |
| C2 | Controller | After parse OK | INFO | `appointment create attempt` | `organisation_id`, `patient_id`, `doctor_id`, `created_by`, `appointment_date`, `visit_type`, `has_series_id` |
| C3 | Service | Org schedule lookup failed | WARN | `appointment create failed` | `organisation_id`, `reason=org_schedule_not_found` (or `org_schedule_lookup`), `error` |
| C4 | Service | Validation failed | WARN | `appointment create failed` | `organisation_id`, `patient_id`, `doctor_id`, `reason=validation`, `field` (`patient_id` / `doctor_id` / `start_time` / `end_time` / `appointment_date` / `slot_duration`) |
| C5 | Service | DB create failed | ERROR | `appointment create failed` | `organisation_id`, `patient_id`, `doctor_id`, `reason=db_create`, `error` |
| C6 | Service | Notification details lookup failed | ERROR | `appointment create failed` | `appointment_id`, `organisation_id`, `reason=notification_payload`, `error` |
| C7 | Service | Saved; notification enqueued (fire-and-forget) | INFO | `appointment create success` | `appointment_id`, `appointment_code`, `organisation_id`, `patient_id`, `doctor_id`, `created_by`, `visit_type`, `appointment_date`, `schedule_id`, `notification_enqueued=true` |
| C8 | Repository | `Create` / `GetNotificationsDetails` DB error | ERROR | `appointment repo error` | `op=Create` / `op=GetNotificationsDetails`, `error` |

**Notes**

- Today `NotificationServ.Create` is fire-and-forget with **no error checked** (same as patient). Do **not** fail HTTP create if notify fails later; C7 covers enqueue intent.
- Do not log `notes` or notification map contents.
- `validateAppointmentFields` currently ignores some parse side-effects in `toApptmntModel` (`Parse` errors discarded). Logging `field=*` is enough; fixing parse behavior is **out of scope** unless bundled.
- Create does **not** check slot occupancy again after `GetSlots`. If double-book happens, it looks like a normal create; optional later: `reason=slot_conflict`.

### 4.1.1 Frontend-facing create errors

| Case | Internal log `reason` | HTTP (proposed) | API `error` | UI copy (suggested) |
|------|----------------------|-----------------|-------------|---------------------|
| Missing / invalid body fields | `appointment create request invalid` | `400` | `invalid request` | “Please check appointment details and try again.” |
| Validation (IDs, times, past date) | `validation` | `400` | short message **or** `invalid request` | Field-specific if product wants it |
| Org schedule missing | `org_schedule_not_found` | `404` | `organisation schedule not found` | “Clinic schedule is not configured.” |
| DB / unexpected | `db_create` / other | `500` | `failed to create appointment` | “Something went wrong. Please try again.” |
| Notification payload after save | `notification_payload` | `500` *or* still return 200 with appointment | TBD — **recommend:** return create success + ERROR log only (appointment already exists) |

**Decision to confirm:** today everything returns `409` with raw `err.Error()`. Recommend mapping as above. For C6 (notify details fail after insert), prefer not rolling back silently without a log — either return 200 + ERROR, or wrap in a transaction in a later pass.

---

### 4.2 Get time slots (`GET /appointment/getTimeSlots`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| S1 | Controller | Missing `doctor_id` / `organisation_id` | WARN | `appointment slots request invalid` | `reason=missing_doctor_id` / `missing_organisation_id` |
| S2 | Controller | Request entered (IDs OK) | INFO | `appointment slots attempt` | `doctor_id`, `organisation_id`, `date` |
| S3 | Service | Existing appointments DB read failed | ERROR | `appointment slots failed` | `doctor_id`, `organisation_id`, `date`, `reason=db_read`, `error` |
| S4 | Service | Org schedule missing / empty | WARN | `appointment slots failed` | `organisation_id`, `reason=org_schedule_not_found` |
| S5 | Service | Slot generation / processing failed | ERROR | `appointment slots failed` | `organisation_id`, `date`, `reason=slot_processing`, `error` |
| S6 | Service | Success | INFO | `appointment slots success` | `doctor_id`, `organisation_id`, `date`, `slot_count`, `available_count` (optional) |
| S7 | Repository | `GetAppointmentsByIDs` DB error | ERROR | `appointment repo error` | `op=GetAppointmentsByIDs`, `error` |

**Notes**

- Empty slots for “today after hours” (`[]`) is still **success** (INFO).
- Do not log every slot’s start/end times at INFO.

### 4.2.1 Frontend-facing slots errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing query params | `400` | `invalid request` | Show validation |
| Org schedule missing | `404` | `organisation schedule not found` | “No schedule configured” |
| DB / processing | `500` | `failed to fetch slots` | Retry |

---

### 4.3 List by organisation (`POST /appointment/getappointmentbyOrgID`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| L1 | Controller | Payload parse failure | WARN | `appointment list request invalid` | `error`, `field` |
| L2 | Controller | Missing `organisation_id` | WARN | `appointment list request invalid` | `reason=missing_organisation_id` |
| L3 | Controller | After parse OK | INFO | `appointment list attempt` | `organisation_id`, `limit`, `page_no`, `doctor_id` (if set), `date`, `status`, `visit_type`, `has_search` |
| L4 | Service / Repo | FindMany or Count failed | ERROR | `appointment list failed` | `organisation_id`, `reason=db_read` / `db_count`, `error` |
| L5 | Service | Success | INFO | `appointment list success` | `organisation_id`, `count`, `total` |
| L6 | Repository | DB errors | ERROR | `appointment repo error` | `op=FindManyByOrganisationID` / `GetTotalAppointmentsByOrgID`, `error` |

**Notes**

- Do not log individual appointment rows or `search` text.
- Empty list (`total=0`) is still **success**.

### 4.3.1 Frontend-facing list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Invalid / missing fields | `400` | `invalid request` | Show validation |
| DB / unexpected | `500` | `failed to fetch appointments` | Retry |

---

### 4.4 Appointment preview (`GET /appointment/getAppointmentsPreview`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| P1 | Controller | Missing `organisation_id` / `appointment_id` | WARN | `appointment preview request invalid` | `reason=missing_organisation_id` / `missing_appointment_id` |
| P2 | Controller | Request entered | INFO | `appointment preview attempt` | `organisation_id`, `appointment_id` |
| P3 | Service | Not found / empty row | WARN | `appointment preview failed` | `organisation_id`, `appointment_id`, `reason=not_found` |
| P4 | Service / Repo | DB error | ERROR | `appointment preview failed` | `organisation_id`, `appointment_id`, `reason=db_read`, `error` |
| P5 | Service | Success | INFO | `appointment preview success` | `organisation_id`, `appointment_id`, `appointment_code`, `status`, `visit_type` |
| P6 | Repository | `GetAppointmentsPreview` DB error | ERROR | `appointment repo error` | `op=GetAppointmentsPreview`, `error` |

**Notes**

- Do not log patient name / mobile / age / gender from the join.
- Today empty `Scan` may return zero value without `gorm.ErrRecordNotFound` — when implementing, detect empty `appointment_id` / code and map to `not_found`.

### 4.4.1 Frontend-facing preview errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing query | `400` | `invalid request` | — |
| Not found | `404` | `appointment not found` | Empty / redirect |
| DB | `500` | `failed to fetch appointment` | Retry |

---

### 4.5 Update status (`PATCH /appointment/updateStatus`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| U1 | Controller | Payload parse failure | WARN | `appointment status update request invalid` | `error`, `field` |
| U2 | Controller | After parse OK | INFO | `appointment status update attempt` | `appointment_id`, `status` |
| U3 | Service | Unknown status string (today defaults to `scheduled`) | WARN | `appointment status update failed` | `appointment_id`, `status`, `reason=invalid_status` — **only if we harden**; today silent default |
| U4 | Service / Repo | Update failed | ERROR | `appointment status update failed` | `appointment_id`, `status`, `reason=db_update`, `error` |
| U5 | Service | Success | INFO | `appointment status update success` | `appointment_id`, `status` |
| U6 | Repository | `UpdateStatus` DB error | ERROR | `appointment repo error` | `op=UpdateStatus`, `error` |

**Notes**

- `SelectStatus` maps unknown values to `StatusScheduled` without error — recommend WARN + reject (`invalid_status`) when implementing logs, or keep current behavior and only log the resolved status.
- No notification on status change today — out of scope.
- Optional later: load previous status and log `old_status` → `new_status`.

### 4.5.1 Frontend-facing status update errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Invalid body / status | `400` | `invalid request` | Show validation |
| Not found (if we check rows affected) | `404` | `appointment not found` | Refresh list |
| DB | `500` | `failed to update appointment` | Retry |

---

### 4.6 List by patient (`POST /appointment/getappointmentByPatientID`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| Pat1 | Controller | Payload parse failure | WARN | `appointment patient list request invalid` | `error`, `field` |
| Pat2 | Controller | After parse OK | INFO | `appointment patient list attempt` | `patient_id`, `organisation_id`, `limit`, `page_no`, `status` (if set) |
| Pat3 | Service / Repo | Read or Count failed | ERROR | `appointment patient list failed` | `patient_id`, `organisation_id`, `reason=db_read` / `db_count`, `error` |
| Pat4 | Service | Success | INFO | `appointment patient list success` | `patient_id`, `organisation_id`, `count`, `total` |
| Pat5 | Repository | DB errors | ERROR | `appointment repo error` | `op=GetAppointmentByPatientID` / `GetAppointmentByPatientIDCount`, `error` |

### 4.6.1 Frontend-facing patient list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Invalid body | `400` | `invalid request` | Show validation |
| DB | `500` | `failed to fetch appointments` | Retry |

---

### 4.7 Get by ID — internal (`GetAppntmentByID`)

Used by prescription create — not a public appointment route.

```go
func (s *AppointmentService) GetAppntmentByID(log *zap.Logger, appointmentID string) (Appointment, error) {
    log = ensureLog(log)
    // ...
}
```

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| G1 | Service | Lookup started | DEBUG | `appointment get by id` | `appointment_id` |
| G2 | Service | Not found / empty | WARN | `appointment get by id failed` | `appointment_id`, `reason=not_found` |
| G3 | Service / Repo | DB error | ERROR | `appointment get by id failed` | `appointment_id`, `reason=db_read`, `error` |
| G4 | Service | Success | DEBUG | `appointment get by id success` | `appointment_id`, `organisation_id`, `patient_id`, `status` |
| G5 | Repository | `GetAppointmentByID` DB error | ERROR | `appointment repo error` | `op=GetAppointmentByID`, `error` |

**Recommendation:** DEBUG on happy path (called from other domains); WARN/ERROR on failure. Update prescription call site to pass `*zap.Logger`.

---

### 4.8 Notification details — internal (`GetNotificationDetails`)

Only used after create today. Prefer logging at create layer (C6/C7); keep repo ERROR only unless called elsewhere later.

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| N1 | Service | Lookup failed | (covered by C6) | — | — |
| N2 | Repository | DB error | ERROR | `appointment repo error` | `op=GetNotificationsDetails`, `error` |

Do **not** log the returned map (PII).

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/appointments/appointment.controllers.go` | Entry logs, invalid request, pass logger into service |
| `internal/appointments/appointment.services.go` | Business outcomes for all flows above |
| `internal/appointments/appointment.repository.go` | DB errors only (`op` + `error`) |
| `internal/appointments/log.go` | `ensureLog` helper (new) |
| `internal/prescription/prescriptions.services.go` | Pass logger into `GetAppntmentByID` when that method gains `*zap.Logger` |
| `shared/error/structures.go` | Optional: appointment client sentinels (`ErrAppointmentNotFound`, `ErrAppointmentCreateFailed`, …) |
| `pkg/middleware/routers/functions.go` | Optional: wrap appointment group with JWT (see decisions) |

No changes required to `RequestLogger` for this pass.

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | From RequestLogger child logger |
| `appointment_id` | After create / preview / status / get-by-id |
| `appointment_code` | Create success, preview success |
| `organisation_id` | All org-scoped flows |
| `patient_id` | Create, patient list, get-by-id success |
| `doctor_id` | Create, slots |
| `created_by` | Create (`user_id`) |
| `schedule_id` | Create success |
| `series_id` / `has_series_id` | Create |
| `visit_type`, `status` | Create / preview / status update |
| `appointment_date`, `date` | Create / slots / list filters |
| `limit`, `page_no`, `count`, `total` | List flows |
| `slot_count`, `available_count` | Slots success (optional) |
| `has_search` | Org list when search present |
| `reason` | Stable enum for failures |
| `field` | Validation / missing body field name |
| `op` | Repository operation |
| `error` | `zap.Error(err)` on WARN/ERROR with an error |
| `notification_enqueued` | Create success |

Example create success:

```json
{
  "level": "info",
  "msg": "appointment create success",
  "request_id": "...",
  "appointment_id": "...",
  "appointment_code": "APT-20260725-a1b",
  "organisation_id": "...",
  "patient_id": "...",
  "doctor_id": "...",
  "created_by": "...",
  "visit_type": "follow_up",
  "appointment_date": "2026-07-26",
  "schedule_id": "...",
  "notification_enqueued": true
}
```

Example create failure:

```json
{
  "level": "warn",
  "msg": "appointment create failed",
  "request_id": "...",
  "organisation_id": "...",
  "patient_id": "...",
  "doctor_id": "...",
  "reason": "validation",
  "field": "appointment_date"
}
```

Example slots success:

```json
{
  "level": "info",
  "msg": "appointment slots success",
  "request_id": "...",
  "doctor_id": "...",
  "organisation_id": "...",
  "date": "2026-07-26",
  "slot_count": 24,
  "available_count": 18
}
```

---

## 7. What we will **not** add in this pass

- Logging patient / doctor PII or `notes`
- INFO log per row in list / preview responses
- Logging full notification payloads
- Appointment series instrumentation (stubs only)
- Slot double-book conflict detection (new product behavior)
- Status-change notifications
- Audit table / DB audit trail
- Duplicate HTTP envelope logs already emitted by RequestLogger

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on appointment HTTP routes | Done (global middleware) |
| Appointment controller domain logs | Done |
| Appointment service domain logs | Done |
| Appointment repo DB error logs | Done |
| Logger passed into appointment service | Done |
| Client-safe / consistent HTTP status mapping | Done |
| JWT on appointment routes | Not present (follow-up) |
| `GetAppntmentByID` logs + call-site logger | Done |
| Appointment series logs | N/A (empty stubs) |

---

## 9. Implementation order (after approval)

1. Add `log.go` (`ensureLog`); thread `*zap.Logger` controller → service → repo  
2. Create appointment (C1–C8) + client error mapping  
3. Update status (U1–U6)  
4. Preview (P1–P6)  
5. Get slots (S1–S7)  
6. List by org (L1–L6)  
7. List by patient (Pat1–Pat5)  
8. `GetAppntmentByID` (G1–G5) — update prescription call site to pass logger  
9. Optional: JWT on appointment group + shared sentinel errors  

---

## 10. Open decisions for review

Please mark yes/no or alternatives:

1. **Omit patient/doctor PII and `notes` from all appointment logs?** — **Recommend: yes**
2. **Pass `*zap.Logger` into appointment service methods?** — **Recommend: yes**
3. **Map HTTP statuses** (400 / 404 / 500) instead of always `409`? — **Recommend: yes** (with shared sentinels)
4. **Omit `search` text; log `has_search` only?** — **Recommend: yes**
5. **Slots success: include `slot_count` / `available_count`?** — **Recommend: yes** (cheap, useful)
6. **Create: if notification details fail after DB insert, return 200 + ERROR log (appointment kept)?** — **Recommend: yes** (avoid orphan UX / silent 409 after successful insert)
7. **Unknown status on update: reject with `invalid_status` instead of defaulting to `scheduled`?** — **Recommend: yes**
8. **Include `GetAppntmentByID` in this pass?** — **Recommend: yes** (used by prescription)
9. **Add JWT `Authenticate` to appointment routes in this pass?** — **Recommend: follow-up** (security win, but separate from logging)
10. **Log `appointment_date` on create success?** — **Recommend: yes** (operational, not PII)
11. **Frontend: field-specific validation messages vs single `invalid request`?** — **Recommend: keep short server messages for known validation strings for now**, or unify — pick one
12. **Logger style:** required `log *zap.Logger` first arg everywhere (incl. internal get-by-id) — **yes** (no variadic)

---

## 11. Approval

Once this list looks right, next step is implementing in the order in §9 without expanding into series, delete, or other domains.
