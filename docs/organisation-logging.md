# Organisation Logging Plan

Implemented. Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for other domains.

## Layer logging

| Layer | What we log |
|-------|-------------|
| **Controller** | Entry / invalid request; HTTP status mapping |
| **Service** | Business outcomes (`attempt` / `success` / `failed` + stable `reason`) |
| **Repo** | **DB errors only** (`op` + `error`); empty Scan / 0-row update → not found |

Package: `internal/organisation`  
Routes: `/api/v1/organisation/*` (see `RegisterOrganisationRoutes`)

---

## 1. Scope

| Area | Endpoint / entry | Files |
|------|------------------|-------|
| Signup / create org | `POST /v1/organisation/signupOrg` | `controllers.go`, `services.go`, `repository.go` (+ license/roles/depts seed in tx) |
| Update location | `PATCH /v1/organisation/updateLocation` | same |
| Get by ID | `GET /v1/organisation/getbyid/:organisation_id` | same |
| Update profile fields | `PATCH /v1/organisation/update` | same |
| Get by ID (internal) | `GetOrgByID` — patient create | `services.go`, `repository.go` |

**Auth note:** organisation routes are **not** behind JWT. Signup is intentionally public; get/update being public is a product risk (open decision).

**Signup adjacent (out of scope for this pass — document only)**

| Step | Route | Package |
|------|-------|---------|
| Super Admin create | `POST /employee/create` | employee |
| License verify | `PATCH /license/verifylicense/:organisationID` | license — see `docs/license-logging.md` |
| Org schedule | `POST /admins/organisationSchedule/create` | admins — see `docs/organisation-schedule-logging.md` |

Typical sequence: `signupOrg` → `employee/create` → `verifylicense` → optional schedule.

**Out of scope**

- Full employee / license / schedule logging (separate docs later)
- Prescription’s unused `orgService` injection
- JWT (follow-up unless decided here)

---

## 2. Security / PII rules (non-negotiable)

Org is B2B tenant metadata (not patient PHI) but still identifiable.

**Never log**

- Full request bodies
- Admin signup PII if ever folded into org create: name, email, mobile, password
- License key (omit; don’t log truncated key unless support explicitly needs it)
- Full `GetByID` response / address JSON blob on INFO
- Email/notification payloads containing `hospital_name` + employee contact

**Safe to log**

- `organisation_id`, `code`
- `hospital_type`
- Security flags: `enable_audit_logs`, `emergency_access`
- Address as **IDs only**: `country_id`, `state_id`, `city_id`
- Seed flags: `license_created`, `roles_seeded`, `depts_seeded`
- Stable `reason` enums

**Borderline**

- `organisation_name`, `legal_entity_name` — **omit on INFO success**; allow `field=` on validation WARN only

---

## 3. Propagation approach

```
RequestLogger
  → Controller: middleware.GetLogger(c)
  → OrganisationService: *zap.Logger as first arg
  → Repository: DB errors only
```

Internal callers (`GetOrgByID` from patient): pass request logger (patient already has logging).

**Recommendation:** required `log *zap.Logger` first arg. Add `internal/organisation/log.go` with `ensureLog`.

---

## 4. What to log — by flow

### 4.1 Create organisation (`POST /signupOrg`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| C1 | Controller | Parse / missing mandatory field | WARN | `organisation create request invalid` | `error`, `field` (`organisation_name` / `legal_entity_name` / `hospital_type`) |
| C2 | Controller | After parse OK | INFO | `organisation create attempt` | `hospital_type` only |
| C3 | Service | Permissions lookup fail | ERROR | `organisation create failed` | `reason=permissions_lookup`, `error` |
| C4 | Service | Org insert fail | ERROR | `organisation create failed` | `reason=db_create`, `error` |
| C5 | Service | License create fail | ERROR | `organisation create failed` | `organisation_id`, `reason=license_create`, `error` |
| C6 | Service | Roles seed fail | ERROR | `organisation create failed` | `organisation_id`, `reason=roles_seed`, `error` |
| C7 | Service | Depts seed fail | ERROR | `organisation create failed` | `organisation_id`, `reason=depts_seed`, `error` |
| C8 | Service | Role-permissions seed fail | ERROR | `organisation create failed` | `organisation_id`, `reason=role_permissions_seed`, `error` |
| C9 | Service | Success | INFO | `organisation create success` | `organisation_id`, `code`, `hospital_type`, `license_created=true`, `roles_seeded=true`, `depts_seeded=true` |
| C10 | Repository | `Create` DB error | ERROR | `organisation repo error` | `op=Create`, `error` |

**Mandatory:** `organisation_name`, `legal_entity_name`, `hospital_type`.  
**Note:** empty string may pass `Getstring` today — optional harden with validation WARN.

### 4.1.1 Frontend-facing create errors

| Case | HTTP (proposed) | API `error` |
|------|-----------------|-------------|
| Invalid / missing fields | `400` | `invalid request` |
| DB / seed / license | `500` | `failed to create organisation` |

**Decision:** today everything is **409** + raw `err.Error()`. Recommend mapping as above.

---

### 4.2 Update location (`PATCH /updateLocation`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| L1 | Controller | Mandatory field missing | WARN | `organisation location update request invalid` | `field` |
| L2 | Controller | Entered OK | INFO | `organisation location update attempt` | `organisation_id`, `country_id`, `state_id`, `city_id`, `enable_audit_logs`, `emergency_access` |
| L3 | Service | Update fail | ERROR | `organisation location update failed` | `organisation_id`, `reason=db_update`, `error` |
| L4 | Service | Success | INFO | `organisation location update success` | `organisation_id`, `enable_audit_logs`, `emergency_access` |
| L5 | Optional | 0 rows updated | WARN | `organisation location update failed` | `organisation_id`, `reason=not_found` |

**Mandatory:** `organisation_id`, `state_id`, `city_id`, `country_id`.  
**Optional:** `enable_audit_logs`, `emergency_access` (default false).

---

### 4.3 Get by ID (`GET /getbyid/:organisation_id`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| G1 | Controller | Missing path param | WARN | `organisation get request invalid` | `reason=missing_organisation_id` |
| G2 | Service | Lookup started | DEBUG | `organisation get by id` | `organisation_id` |
| G3 | Service | Empty row / not found | WARN | `organisation get by id failed` | `organisation_id`, `reason=not_found` |
| G4 | Service | DB error | ERROR | `organisation get by id failed` | `organisation_id`, `reason=db_read`, `error` |
| G5 | Service | Success | DEBUG | `organisation get by id success` | `organisation_id`, `hospital_type` |

**Important quirk:** repo uses `Raw().Scan()` — zero rows often return **empty struct + nil error**. When implementing, treat empty `ID` as `not_found` (fixes patient org lookup too).

### 4.3.1 Frontend-facing get errors

| Case | HTTP | API `error` |
|------|------|-------------|
| Missing ID | `400` | `invalid request` |
| Not found | `404` | `organisation not found` |
| DB | `500` | `failed to fetch organisation` |

---

### 4.4 Update profile (`PATCH /update`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| U1 | Controller | Mandatory missing | WARN | `organisation update request invalid` | `field` |
| U2 | Controller | Entered OK | INFO | `organisation update attempt` | `organisation_id`, `hospital_type` |
| U3 | Service | Update fail | ERROR | `organisation update failed` | `organisation_id`, `reason=db_update`, `error` |
| U4 | Service | Success | INFO | `organisation update success` | `organisation_id`, `hospital_type` |
| U5 | Optional | 0 rows | WARN | `organisation update failed` | `organisation_id`, `reason=not_found` |

**Mandatory at controller:** `organisation_id`, `organisation_name`, `legal_entity_name`, `hospital_type`. Do not log names on INFO.

---

### 4.5 Internal get (`GetOrgByID`)

Used by patient create (and similar).

```go
func (s *OrganisationService) GetOrgByID(log *zap.Logger, id string) (Organisation, error)
```

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| I1 | Service | Started | DEBUG | `organisation get by id` | `organisation_id` |
| I2 | Service | Not found / empty | WARN | `organisation get by id failed` | `organisation_id`, `reason=not_found` |
| I3 | Service | DB error | ERROR | `organisation get by id failed` | `organisation_id`, `reason=db_read`, `error` |
| I4 | Service | Success | DEBUG | `organisation get by id success` | `organisation_id`, `hospital_type` |

Patient already maps org miss to `ErrOrganisationNotFound` — align empty-Scan fix so that path is real.

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/organisation/controllers.go` | Entry / invalid request / pass logger / HTTP mapping |
| `internal/organisation/services.go` | Create tx step outcomes; update/get |
| `internal/organisation/repository.go` | DB errors; empty-Scan → not found |
| `internal/organisation/log.go` | `ensureLog` (new) |
| `shared/error/structures.go` | Optional: `ErrOrganisationCreateFailed`, `ErrOrganisationUpdateFailed`, `ErrOrganisationFetchFailed` (reuse `ErrOrganisationNotFound`) |
| `internal/patient/services.go` | Pass logger into `GetOrgByID` when signature changes |

License create during signup: log only as `license_created` / `reason=license_create` from org service — not full license package instrumentation.

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | RequestLogger |
| `organisation_id`, `code` | After create / on get/update |
| `hospital_type` | Create / update |
| `country_id`, `state_id`, `city_id` | Location update |
| `enable_audit_logs`, `emergency_access` | Location |
| `license_created`, `roles_seeded`, `depts_seeded` | Create success |
| `reason`, `field`, `op`, `error` | Failures |

Example create success:

```json
{
  "level": "info",
  "msg": "organisation create success",
  "request_id": "...",
  "organisation_id": "...",
  "code": "...",
  "hospital_type": "clinic",
  "license_created": true,
  "roles_seeded": true,
  "depts_seeded": true
}
```

---

## 7. What we will **not** add in this pass

- Logging org/legal names on success
- Full employee create domain logs (separate pass)
- JWT (unless approved in decisions)

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on org HTTP routes | Done (global) |
| Organisation controller domain logs | Done |
| Organisation service domain logs | Done |
| Organisation repo DB error logs | Done |
| Empty Scan treated as not found | Done |
| 0-row update treated as not found | Done |
| Client-safe HTTP mapping | Done (400/404/500) |
| Logger on `GetOrgByID` for patient | Done |
| JWT on org routes | Not present (follow-up) |

---

## 9. Implementation order (after approval)

1. `log.go` + optional sentinels  
2. Fix repo empty-Scan / optional 0-row update detection  
3. Thread logger; create (C1–C10) + HTTP mapping  
4. Update location + update profile  
5. Get by ID HTTP + internal `GetOrgByID` (update patient call site)  
6. Optional: JWT on update/get (leave signup public)  

---

## 10. Open decisions for review

1. **Omit `organisation_name` / `legal_entity_name` from INFO logs?** — **Recommend: yes**
2. **Pass `*zap.Logger` into org service + `GetOrgByID`?** — **Recommend: yes**
3. **Map HTTP 400/404/500 instead of always 409?** — **Recommend: yes**
4. **Treat empty Scan / 0-row update as `not_found`?** — **Recommend: yes**
5. **Step-level create failure reasons** (license/roles/depts)? — **Recommend: yes**
6. **Include license verify / employee create in this pass?** — **Recommend: no** (cross-link only)
7. **JWT on get/update/location (signup stays public)?** — **Recommend: follow-up / product call**
8. **Log address IDs on location update?** — **Recommend: yes**
9. **Logger style:** required `log *zap.Logger` first arg — **yes**

---

## 11. Approval

Once this list looks right, implement in §9 order without expanding into employee/license/schedule packages beyond create-tx handoff flags.
