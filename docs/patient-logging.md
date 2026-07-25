# Patient Logging Plan

Review doc for what to log in the patient domain and where. No implementation yet — approve / adjust this list, then we instrument.

Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for authentication. Request-level `request started` / `request completed` / `client error` / `request failed` already cover the HTTP envelope. This plan covers **domain patient events** on top of that.

---

## 1. Scope

| Area | Endpoint / entry | Files |
|------|------------------|-------|
| Create patient | `POST /v1/patients/addGeneralInfo` | `controllers.go`, `services.go`, `repository.go` |
| List patients | `GET /v1/patients/getPatients` | same |
| Get patient by ID | `GET /v1/patients/getpatientByID/:patientID` | same |
| Notification lookup | `GetNotificationPatientByID` (called from payment fulfillment, not HTTP) | `services.go`, `repository.go` |

All HTTP patient routes are behind JWT `Authenticate` middleware (auth rejection already logged).

**Out of scope for this pass**

- Patient update API (`docs/patient-update.md` — not implemented yet); add logging when that feature lands
- Prescription / appointment / billing logs that only *reference* a patient ID
- Notification delivery internals (email/SMS provider) — only the fire-and-forget handoff from patient create

---

## 2. Security / PII rules (non-negotiable)

Patient data is sensitive. Prefer IDs over PII in logs.

**Never log**

- Full patient payloads / request bodies
- Full address
- Full mobile number or email on success paths
- Blood group, weight, age as standalone success fields (low value, still PII-adjacent)

**Safe to log**

- `patient_id`, `uhid` / patient code, `organisation_id`, `created_by` (`user_id`)
- Pagination: `limit`, `page_no`, `total` (counts only)
- Validation **field names** on failure (e.g. `field=name`), not field values when the value is PII
- Boolean / enum outcomes: `notification_enqueued=true`

**Borderline (recommend omit on INFO success; allow only if needed for support)**

- Patient name — **omit** by default
- Email / mobile — **omit**; on unique-constraint DB failures log `reason=duplicate` without the conflicting value

---

## 3. Propagation approach

Same as auth:

```
RequestLogger (request_id, method, path, ip)
  → Controller: middleware.GetLogger(c)
  → Service: accept *zap.Logger as first arg
  → Repository: log only on DB errors
```

For non-HTTP callers (`GetNotificationPatientByID` from payment fulfillment): pass a logger from the caller, or fall back to `logger.Log` / `zap.NewNop()` via `ensureLog`.

**Recommendation:** pass `*zap.Logger` into service methods (HTTP-agnostic, testable).

---

## 4. What to log — by flow

### 4.1 Create patient (`POST /patients/addGeneralInfo`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| C1 | Controller | Payload / field parse failure | WARN | `patient create request invalid` | `error`, `field` if known |
| C2 | Controller | After parse OK | INFO | `patient create attempt` | `organisation_id`, `created_by` (`user_id`) |
| C3 | Service | Org lookup failed | WARN | `patient create failed` | `organisation_id`, `reason=org_not_found` (or `org_lookup`), `error` |
| C4 | Service | Validation failed | WARN | `patient create failed` | `organisation_id`, `reason=validation`, `field` (`name` / `gender` / `age` / `weight`) |
| C5 | Service | DB create failed | ERROR | `patient create failed` | `organisation_id`, `reason=db_create` (or `duplicate` if unique violation), `error` |
| C6 | Service | Notification payload build failed | ERROR | `patient create failed` | `patient_id`, `organisation_id`, `reason=notification_payload`, `error` |
| C7 | Service | Patient saved; notification enqueued (fire-and-forget) | INFO | `patient create success` | `patient_id`, `uhid`, `organisation_id`, `created_by`, `notification_enqueued=true` |
| C8 | Repository | `Create` DB error | ERROR | `patient repo error` | `op=Create`, `error` |

**Notes**

- Today `notifications.Create` is fire-and-forget with **no error checked**. Do **not** fail the HTTP create if notify fails later; optionally log at DEBUG that enqueue was triggered (covered by C7 `notification_enqueued`).
- Do not log name / email / mobile / address on C2 or C7.
- `ValidatePatient` currently ignores `Atoi`/`ParseFloat` parse errors (invalid age becomes `0`). Logging `field=age` / `field=weight` is enough; fixing parse behavior is **out of scope** unless you want it bundled.

### 4.1.1 Frontend-facing create errors

| Case | Internal log `reason` | HTTP (proposed) | API `error` | UI copy (suggested) |
|------|----------------------|-----------------|-------------|---------------------|
| Missing / invalid body fields | `patient create request invalid` | `400` | `invalid request` | “Please check the patient details and try again.” |
| Validation (name, gender, age, weight) | `validation` | `400` | keep short message **or** map to `invalid request` | Same / field-specific if product wants it |
| Org not found / org lookup | `org_not_found` / `org_lookup` | `400` or `404` | `organisation not found` | “Organisation not found.” |
| Duplicate email/mobile/UHID | `duplicate` | `409` | `patient already exists` | “A patient with these details already exists.” |
| DB / unexpected | `db_create` / other | `500` | `failed to create patient` | “Something went wrong. Please try again.” |

**Decision to confirm:** today everything returns `409` with raw `err.Error()`. Recommend mapping as above (aligned with auth). Until then, logs still use structured `reason`.

---

### 4.2 List patients (`GET /patients/getPatients`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| L1 | Controller | Request entered | INFO | `patient list attempt` | `organisation_id`, `limit`, `page_no` |
| L2 | Controller | Missing `organisation_id` (if we enforce) | WARN | `patient list request invalid` | `reason=missing_organisation_id` |
| L3 | Service / Repo | ReadMany or Count failed | ERROR | `patient list failed` | `organisation_id`, `reason=db_read` / `db_count`, `error` |
| L4 | Service | Success | INFO | `patient list success` | `organisation_id`, `count` (returned rows), `total` |
| L5 | Repository | DB errors on `ReadMany` / `Count` | ERROR | `patient repo error` | `op`, `error` |

**Notes**

- Do not log individual patient rows.
- Empty list (`total=0`) is still **success** (INFO), not WARN.
- Controller today swallows wrap return on error (`errwrap.Wrap` without `return`) — fix return path when implementing logs.

### 4.2.1 Frontend-facing list errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Missing org (if enforced) | `400` | `invalid request` | Show validation |
| DB / unexpected | `500` | `failed to fetch patients` | Retry / toast |

---

### 4.3 Get patient by ID (`GET /patients/getpatientByID/:patientID`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| G1 | Controller | Request entered | INFO | `patient get attempt` | `patient_id` |
| G2 | Controller | Empty `patientID` param | WARN | `patient get request invalid` | `reason=missing_patient_id` |
| G3 | Service | Not found | WARN | `patient get failed` | `patient_id`, `reason=not_found` |
| G4 | Service / Repo | DB error | ERROR | `patient get failed` | `patient_id`, `reason=db_read`, `error` |
| G5 | Service | Success | INFO | `patient get success` | `patient_id`, `organisation_id` (if available), `uhid` |
| G6 | Controller | Response encode failure | ERROR | `patient get response failed` | `patient_id`, `error` |
| G7 | Repository | `ReadOne` DB error | ERROR | `patient repo error` | `op=ReadOne`, `error` |

### 4.3.1 Frontend-facing get errors

| Case | HTTP | API `error` | Frontend |
|------|------|-------------|----------|
| Not found | `404` | `patient not found` | Empty state / redirect |
| Invalid ID | `400` | `invalid request` | — |
| DB / unexpected | `500` | `failed to fetch patient` | Retry |

---

### 4.4 Notification patient lookup (`GetNotificationPatientByID`)

Used by payment fulfillment for notification context — not a public patient route.

**Logger API:** required `*zap.Logger` as the first argument (same pattern as create/list/get and auth). No variadic.

```go
func (p *PatientService) GetNotificationPatientByID(log *zap.Logger, patientID string) (map[string]interface{}, error) {
    log = ensureLog(log) // nil-safe fallback to nop/global if caller passes nil by mistake
    // ...
}
```

| Caller | How to pass logger |
|--------|-------------------|
| Payment fulfillment (has request logger) | `GetNotificationPatientByID(reqLogger, id)` |
| Payment fulfillment (no request logger yet) | `GetNotificationPatientByID(logger.Log, id)` or `ensureLog(nil)` via passing `nil` only as temporary fallback |
| Future HTTP path | `GetNotificationPatientByID(middleware.GetLogger(c), id)` |

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| N1 | Service | Lookup started | DEBUG | `patient notification lookup` | `patient_id` |
| N2 | Service | Not found / DB error | ERROR | `patient notification lookup failed` | `patient_id`, `reason`, `error` |
| N3 | Service | Success | DEBUG | `patient notification lookup success` | `patient_id`, `organisation_id` |

**Recommendation:** DEBUG on happy path (high volume from payments); ERROR on failure. Do not log email/mobile from the joined query result.
---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/patient/controllers.go` | Entry logs, invalid request, pass logger into service; fix error `return` paths on Find / GetPatientByID |
| `internal/patient/services.go` | Business outcomes (create / list / get / notification lookup) |
| `internal/patient/repository.go` | DB errors only (`op` + `error`) |
| `shared/error/structures.go` | Optional: add patient client sentinels (`ErrPatientNotFound`, `ErrPatientCreateFailed`, …) next to auth errors |

No changes required to `RequestLogger` for this pass.

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | From RequestLogger child logger |
| `patient_id` | After create / on get / notification lookup |
| `uhid` | Create success, get success |
| `organisation_id` | Create / list / get when known |
| `created_by` | Create (from `user_id` in payload) |
| `limit`, `page_no`, `count`, `total` | List |
| `reason` | Stable enum for failures |
| `field` | Validation / missing body field name |
| `op` | Repository operation |
| `error` | `zap.Error(err)` on WARN/ERROR with an error |

Example create success:

```json
{
  "level": "info",
  "msg": "patient create success",
  "request_id": "...",
  "patient_id": "...",
  "uhid": "CLI-123",
  "organisation_id": "...",
  "created_by": "...",
  "notification_enqueued": true
}
```

Example create failure:

```json
{
  "level": "warn",
  "msg": "patient create failed",
  "request_id": "...",
  "organisation_id": "...",
  "reason": "validation",
  "field": "weight"
}
```

---

## 7. What we will **not** add in this pass

- Logging full patient PII (name, email, phone, address) on success
- INFO log per row in list responses
- Changing notification provider implementation
- Patient update endpoint logging (feature not shipped)
- Attaching `user_id` from JWT to request logger (auth follow-up; nice-to-have for `created_by` correlation later)

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on patient HTTP routes | Done (global middleware) |
| Patient controller domain logs | Done |
| Patient service domain logs | Done |
| Patient repo DB error logs | Done |
| Logger passed into patient service | Done |
| Client-safe / consistent HTTP status mapping | Done |
| `GetNotificationPatientByID` logs | Done |

---

## 9. Implementation order (after approval)

1. Thread `*zap.Logger` controller → service → repo  
2. Create patient (C1–C8) + client error mapping  
3. Get by ID (G1–G7)  
4. List (L1–L5)  
5. `GetNotificationPatientByID` (N1–N3) — update call sites (e.g. payment fulfillment) to pass `*zap.Logger`  
6. Add shared patient sentinel errors if we adopt the status mapping in §4.x.1  

---

## 10. Open decisions for review

Please mark yes/no or alternatives:

1. **Omit name / email / mobile / address from all patient logs?** — **Recommend: yes**
2. **Pass `*zap.Logger` into patient service methods?** — **Recommend: yes**
3. **Map HTTP statuses** (400 / 404 / 409 / 500) instead of always `409`? — **Recommend: yes** (with shared sentinels)
4. **List: require `organisation_id` and WARN if missing?** — **Recommend: yes**
5. **Create: treat unique constraint as `409 patient already exists`?** — **Recommend: yes**
6. **Notification lookup: DEBUG success, ERROR failure only?** — **Recommend: yes**
7. **Include `GetNotificationPatientByID` in this pass?** — **Recommend: yes** (small, used by payments)
8. **Frontend: field-specific validation messages vs single `invalid request`?** — **Recommend: keep short server messages for validation for now** (current strings like “please provide name”), or unify to `invalid request` — pick one
9. **Logger style for all patient service methods (including notification lookup):** required `log *zap.Logger` first arg — **yes** (no variadic)

---

## 11. Approval

Once this list looks right, next step is implementing in the order in §9 without expanding into update API or other domains.
