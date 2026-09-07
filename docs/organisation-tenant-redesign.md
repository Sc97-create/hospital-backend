# Organisation & Tenant Redesign

Design note for introducing **tenants**, reshaping **organisation**, replacing product **licenses** with **subscription**, and adding an org-scoped **service** catalog.

`organisation_tax_config` is **out of scope** for this pass.

---

## 1. Business model (target)

| Concept | Rule |
|---|---|
| **Tenant** | Top-level account master. Owns one or more organisations. |
| **Organisation** | Child of a tenant. **Exactly one hospital / facility** per organisation. |
| **Address** | Stored **per organisation** (not shared across orgs under the same tenant). |
| **Data sharing** | Stored **per organisation** (consent / sharing prefs for that facility). |
| **Service** | Billable / catalog items owned by an organisation (e.g. consultation fee, procedures). |
| **Subscription** | Entitlement / plan for the product. Replaces the current `licenses` product-license flow. |
| **Tax config** | Deferred — do not implement `organisation_tax_config` yet. |

Hierarchy:

```text
tenant (1) ──< organisation (N) ──< service (N)
                 │
                 └── address, data_sharing (on org row / JSONB)
```

Subscription is expected to attach at **tenant** level (one commercial agreement covering all orgs under the tenant). Exact subscription schema is a follow-up; this doc only defines removal of product license and the hooks signup/auth must use instead.

---

## 2. Target ERD (in scope)

```
                         ┌──────────────────┐
                         │     tenants      │
                         ├──────────────────┤
                         │ id PK            │
                         │ name             │
                         │ status           │
                         │ created_at       │
                         │ updated_at       │
                         └────────┬─────────┘
                                  │ 1:N
                                  ▼
                    ┌────────────────────────┐
                    │     organisations      │
                    ├────────────────────────┤
                    │ id PK                  │
                    │ tenant_id FK           │
                    │ legal_entity_name      │
                    │ display_name           │
                    │ organisation_type      │
                    │ facility_name          │
                    │ registration_no        │
                    │ license_number         │  ← facility/regulatory (not product license)
                    │ license_expiry         │
                    │ gstin                  │
                    │ address                │  JSONB, per org
                    │ data_sharing           │  JSONB, per org
                    │ status                 │
                    │ created_at             │
                    │ updated_at             │
                    └───────────┬────────────┘
                                │ 1:N
                                ▼
                      ┌─────────────────┐
                      │    services     │
                      ├─────────────────┤
                      │ id PK           │
                      │ org_id FK       │
                      │ name            │
                      │ description     │
                      │ price           │
                      │ status          │
                      │ created_at      │
                      │ updated_at      │
                      └─────────────────┘
```

> **Naming:** Prefer GORM plural table `organisations` / `services` / `tenants` to match existing migration style. Column `org_id` in the sketch maps to `organisation_id` in Go models for consistency with the rest of the codebase.

---

## 3. Current implementation (baseline)

### 3.1 Organisation today

| Area | Current |
|---|---|
| Model | `internal/organisation/structures.go` — flat org, **no tenant** |
| Fields | `id`, `code`, `hospital_type`, `legal_entity_name`, `organisation_name`, `address` (JSONB), `security` (JSONB), timestamps |
| Create | `POST /api/v1/organisation/signupOrg` seeds org + **product license** + roles + depts + role_permissions in one TX |
| Update | `PATCH .../update` (name / legal / hospital_type), `PATCH .../updateLocation` (address + security) |
| Get | `GET .../getbyid/:organisation_id` |
| Repo | `Create`, `GetOrganisationByID`, `UpdateLocationByID`, `Update` |

### 3.2 Product license today (to eradicate)

| Area | Current |
|---|---|
| Table | `licenses` (`organisation_id`, `license_key`, `issued_at`, `expires_at`) |
| Create | Inside `OrganisationService.CreateOrganisation` via `LicenseCreator.CreateLicenseSrv` (default 6 months) |
| Verify API | `PATCH /api/v1/license/verifylicense/:organisationID` |
| Module | `internal/license/*`, RBAC module `constants.License`, migration `AutoMigrate(&license.License{})` |
| Wiring | `appinit/app.go` injects `LicenseService` into org service |

Facility fields like `license_number` / `license_expiry` on the **new** organisation row are **hospital regulatory metadata**, not the SaaS product license. Those stay; the **`licenses` table and verify/create product-license flow go away**.

### 3.3 What does not exist yet

- `tenants` table / package
- `tenant_id` on organisation
- `services` catalog table / CRUD
- `display_name`, `facility_name`, `registration_no`, `gstin`, `status`, `data_sharing`
- Subscription entity / APIs (placeholder only in this redesign)

---

## 4. Field mapping (organisation)

| Current | Target | Notes |
|---|---|---|
| `id` | `id` | Unchanged |
| — | `tenant_id` | **Required** FK → `tenants.id` |
| `legal_entity_name` | `legal_entity_name` | Unchanged |
| `organisation_name` | `display_name` | Rename column + JSON |
| `hospital_type` | `organisation_type` | Rename; same semantic (clinic / hospital / etc.) |
| — | `facility_name` | Physical facility name (one hospital per org) |
| — | `registration_no` | Facility registration |
| — | `license_number` | Facility / clinical license number |
| — | `license_expiry` | Facility license expiry date |
| — | `gstin` | Tax identity (org-level; tax *config* table still deferred) |
| `address` | `address` | Keep JSONB **per org** |
| `security` | *(retire)* | Replace with `data_sharing` (see below) |
| — | `data_sharing` | New JSONB per org |
| — | `status` | e.g. `active` / `inactive` / `suspended` |
| `code` | *(drop or keep)* | Not in target ERD — **drop** unless product still needs a public hospital code; if kept, document as non-ERD extension |
| `created_at` / `updated_at` | same | Unchanged |

### 4.1 Suggested JSONB shapes

**`address`** (keep existing shape; fix typo in security path separately):

| Field | Purpose |
|---|---|
| `country_id` | Country ref |
| `state` | State ref / name |
| `city` | City ref / name |
| `created_at` / `created_by` | Audit of address write |

Optional later: line1, pincode, geo — not required for this pass.

**`data_sharing`** (replaces `security` for org-level sharing prefs):

| Field | Purpose |
|---|---|
| `allow_cross_org_patient_lookup` | Share patient identity across orgs under same tenant |
| `allow_analytics_export` | Aggregate analytics export |
| `enable_audit_logs` | Migrate from `security.enable_audit_logs` |
| `emergency_access` | Migrate from `security.emergency_access` |

Exact keys can be tightened with product; migration should copy `security` → `data_sharing` then drop `security`.

---

## 5. Tenants

### 5.1 Model

```go
type Tenant struct {
    ID        string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    Name      string    `json:"name" gorm:"type:varchar(255);not null"`
    Status    string    `json:"status" gorm:"type:text;not null"` // active | inactive | suspended
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

### 5.2 Business rules

1. Creating the **first** organisation for a new customer usually creates the **tenant** in the same transaction (unless an existing `tenant_id` is supplied by a platform admin).
2. Additional organisations under the same tenant must pass a valid `tenant_id` and inherit commercial subscription from the tenant (once subscription exists).
3. Soft business invariant: one organisation = one facility (`facility_name` required on create/update).
4. Listing organisations is always scoped: by `tenant_id` for platform/tenant-admin, or by JWT `organisation_id` for facility users.
5. Deactivating a tenant should cascade policy to orgs (set org `status` inactive / block login) — implement as service rule when status APIs land; do not hard-delete.

### 5.3 Package sketch

```text
internal/tenant/
  structures.go
  repository.go
  services.go
  controllers.go
  dto/
```

---

## 6. Subscription replaces product license

### 6.1 Remove

| Item | Action |
|---|---|
| `licenses` table | Stop migrating; plan data migration / drop after cutover |
| `internal/license/*` | Delete or quarantine after callers removed |
| `LicenseCreator` in org signup | Remove from `interfaces.go`, constructor, mocks |
| `CreateLicenseSrv` call in `CreateOrganisation` TX | Remove |
| `PATCH /license/verifylicense/:organisationID` | Remove route + RBAC map entry |
| `constants.License` module | Remove from constants / default role permissions / RBAC routes |
| `appinit` wiring of `LicenseService` | Remove |
| Tests asserting license create on org signup | Rewrite for tenant + subscription hooks |

### 6.2 Introduce (stub for this pass)

Minimum hook so signup does not leave entitlement undefined:

1. **Interface** on org/tenant create, e.g. `SubscriptionProvisioner.ProvisionDefault(tx, tenantID)`.
2. Default plan / trial period decided by product (e.g. 14-day trial) — **do not** encode license keys.
3. Auth / middleware later checks `subscription.status` + `expires_at` on **tenant**, not org license key verify.
4. Full subscription CRUD / billing provider integration is a **separate doc**.

Until the subscription table exists, org create should:

- Create tenant + organisation (+ seed roles/depts/permissions as today)
- **Not** call license APIs
- Optionally write a TODO / no-op provisioner so DI stays clean

---

## 7. Service catalog (new)

### 7.1 Purpose

Per-hospital catalog of priced offerings used by ops and (later) billing:

- Consultation fees
- Procedures / packages
- Other non-medicine charge lines

This is **not** the Go `*Service` application layer naming collision — table/package should be clear, e.g. package `orgservice` or `facilitieservice`, table `services`.

### 7.2 Model

```go
type Service struct {
    ID             string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null;index"`
    Name           string    `json:"name" gorm:"type:varchar(255);not null"`
    Description    string    `json:"description" gorm:"type:text"`
    Price          float64   `json:"price" gorm:"type:numeric(12,2);not null"`
    Status         string    `json:"status" gorm:"type:text;not null"` // active | inactive
    CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

### 7.3 Business rules

1. Every service row **belongs to exactly one organisation**; never shared across orgs.
2. Create/update/list/delete always filter by `organisation_id` (tenant isolation via org → tenant).
3. Soft-deactivate via `status=inactive` rather than hard delete when the service may appear on historical invoices (future).
4. Unique name per org recommended: unique index `(organisation_id, lower(name))` where `status=active`.
5. Price ≥ 0; currency assumed INR for now (no currency column until multi-currency).
6. **Signup seed (recommended):** after org create, seed a default `Consultation` service with price `0` or a configurable default so billing can later resolve consultation amount from catalog instead of hardcoding.
7. **Billing integration (follow-up, not blocking this redesign):** consultation invoice creation should eventually resolve amount from an active service (by name/code or `service_id` on appointment). Do **not** change invoice schema in this pass unless product requires it immediately.
8. Tax is **not** applied via `organisation_tax_config` yet; service `price` is pre-tax list price until tax config lands.

### 7.4 Package sketch

```text
internal/orgservice/   # or internal/servicecatalog/
  structures.go
  repository.go
  services.go
  controllers.go
  dto/
  mocks/
```

---

## 8. API changes

### 8.1 Tenant APIs (new)

| Method | Path | Intent |
|---|---|---|
| `POST` | `/api/v1/tenant/create` | Platform/admin create tenant |
| `GET` | `/api/v1/tenant/getbyid/:tenant_id` | Fetch tenant |
| `PATCH` | `/api/v1/tenant/update` | Update name/status |
| `GET` | `/api/v1/tenant/list` | List tenants (platform) |

RBAC: new module `tenant` (or platform-only until module exists — mark `unmapped` like current org routes if needed).

### 8.2 Organisation APIs (change)

| Current | Change |
|---|---|
| `POST /organisation/signupOrg` | Accept new fields; create **tenant** (if new) + org; **no license**; optionally seed default **service** |
| `PATCH /organisation/update` | Map to new columns (`display_name`, `organisation_type`, facility/regulatory/gstin/status) |
| `PATCH /organisation/updateLocation` | Keep address update; move audit/emergency flags into **`data_sharing`** payload (or split `PATCH .../updateDataSharing`) |
| `GET /organisation/getbyid/:organisation_id` | Response includes `tenant_id` + new fields; drop product-license noise |
| — | **New** `GET /organisation/listByTenant/:tenant_id` |

#### Create / update request shape (target)

```json
{
  "tenant_id": "",
  "tenant_name": "Acme Health Group",
  "legal_entity_name": "Acme Health Pvt Ltd",
  "display_name": "Acme City Hospital",
  "organisation_type": "hospital",
  "facility_name": "Acme City Hospital - Main Campus",
  "registration_no": "REG-KA-123",
  "license_number": "CL-KA-456",
  "license_expiry": "2027-12-31",
  "gstin": "29AAAAA0000A1Z5",
  "address": {
    "country_id": "...",
    "state": "...",
    "city": "..."
  },
  "data_sharing": {
    "enable_audit_logs": true,
    "emergency_access": false,
    "allow_cross_org_patient_lookup": false
  },
  "status": "active"
}
```

Rules:

- If `tenant_id` empty → create tenant from `tenant_name` (required) in same TX.
- If `tenant_id` set → validate tenant exists and is `active`; do not create a second tenant.
- `display_name` + `facility_name` + `organisation_type` + `legal_entity_name` required on create.

#### Response shape (get)

Return full org model including `tenant_id`, `data_sharing`, `status`. Do not return product `license_key`.

### 8.3 License APIs (remove)

| Method | Path | Action |
|---|---|---|
| `PATCH` | `/api/v1/license/verifylicense/:organisationID` | **Remove** |

Update `docs/api-placeholders.md` and Postman collections accordingly.

### 8.4 Service catalog APIs (new)

| Method | Path | Intent |
|---|---|---|
| `POST` | `/api/v1/service/create` | Create service for org |
| `GET` | `/api/v1/service/getbyid/:service_id` | Get one (org-scoped) |
| `GET` | `/api/v1/service/listByOrg` | List by `organisation_id` (+ optional status filter) |
| `PATCH` | `/api/v1/service/update` | Update name/description/price/status |
| `DELETE` or `PATCH` | `/api/v1/service/deactivate` | Prefer deactivate |

Request create:

```json
{
  "organisation_id": "...",
  "name": "Consultation",
  "description": "General OPD consultation",
  "price": 500.00,
  "status": "active"
}
```

RBAC: new module `service` (or under `organisation` until product decides). Update `rbac_routes.go` + `api-placeholders.md`.

---

## 9. Layer changes (API → service → repo)

### 9.1 Tenant

| Layer | Work |
|---|---|
| **DTO** | create/update/get payloads |
| **Controller** | bind/validate `name`, `status` |
| **Service** | create/update/get/list; status transitions; block delete if orgs exist |
| **Repo** | `Create`, `GetByID`, `Update`, `List`; status filter helpers |
| **Migration** | `AutoMigrate(&tenant.Tenant{})` **before** organisation alter |
| **appinit / routers** | wire + register routes |

### 9.2 Organisation

| Layer | File(s) | Work |
|---|---|---|
| **Model** | `structures.go` | Add `TenantID`, rename fields, add facility/regulatory/gstin/status/`DataSharing`; remove or stop writing `Security` / `Code` / `HospitalType` / `OrganisationName` |
| **DTO** | `DTO/organisation_request.go` | Align JSON tags; add tenant + new fields |
| **Controller** | `controllers.go` | Validate new required fields; stop requiring `hospital_type` / `organisation_name` names |
| **Service** | `services.go` | TX: tenant resolve/create → org create → seed roles/depts/permissions → seed default service; **remove license**; map updates to new columns; location update writes `address` + optionally `data_sharing` |
| **Interfaces** | `interfaces.go` | Remove `LicenseCreator`; add `TenantCreator` / `ServiceSeeder` (or inject concrete deps) |
| **Repo** | `repository.go` | Extend `GetOrganisationByID` SELECT list; `Update` map keys; optional `ListByTenantID` |
| **Mocks / tests** | `*_test.go`, mocks | Drop license expectations; add tenant + service seed cases |
| **Migration** | `shared/migration/functions.go` | Alter org columns; migrate data; drop `licenses` when safe |

### 9.3 Service catalog

| Layer | Work |
|---|---|
| **DTO / Controller** | CRUD + org scope validation |
| **Service** | price/status rules; unique-name check; deactivate |
| **Repo** | `Create`, `GetByID`, `ListByOrg`, `Update`, `Deactivate` — always `WHERE organisation_id = ?` |
| **Migration** | `AutoMigrate` service model |
| **Wiring** | container in `appinit`, routes in `Register*Routes` |

### 9.4 License (eradicate)

| Layer | Work |
|---|---|
| Controllers / services / repo / utils / mocks | Remove from build path |
| Routers + RBAC | Unregister verify route + module maps |
| Default role permissions | Remove license module grants (`internal/rolepermissions/...`) |
| Org signup tests | Assert no license side effect |

### 9.5 Cross-cutting

| Area | Work |
|---|---|
| JWT / auth context | Eventually include `tenant_id` alongside `organisation_id` for list/filter guards |
| Middleware | Replace any license-verify assumption with subscription check (later) |
| ERD doc | Update `docs/erd.md` section “Organisation & license” → “Tenant, organisation & services” |
| Logging docs | Update `docs/organisation-logging.md`, retire/replace `docs/license-logging.md` |

---

## 10. Signup / create transaction (target flow)

```text
BEGIN
  1. Resolve tenant
       - if tenant_id provided: lock/load active tenant
       - else: INSERT tenant(name, status=active)
  2. INSERT organisation(... tenant_id, display_name, facility_name, ...)
  3. Seed roles, departments, role_permissions (existing)
  4. Seed default service "Consultation" (new, recommended)
  5. Provision subscription for tenant (stub / follow-up)  ← replaces CreateLicenseSrv
COMMIT
```

Failure at any step rolls back the whole signup (same as today).

---

## 11. Data migration plan

1. Create `tenants` table.
2. For each existing `organisations` row:
   - Insert a tenant (`name` = old `organisation_name` / `display_name`).
   - Set `organisations.tenant_id`.
3. Rename columns:
   - `organisation_name` → `display_name`
   - `hospital_type` → `organisation_type`
4. Add nullable then backfill: `facility_name` (default from display name), `registration_no`, `license_number`, `license_expiry`, `gstin`, `status` (default `active`), `data_sharing`.
5. Copy `security` JSON → `data_sharing`; drop `security` when clients migrated.
6. Drop `code` if unused by clients.
7. Create `services`; optionally backfill one Consultation row per org.
8. Stop writing `licenses`; after verify API removed and no readers remain, drop `licenses`.

Prefer explicit SQL migration scripts for renames/backfills; do not rely on AutoMigrate alone for renames.

---

## 12. Out of scope (this pass)

- `organisation_tax_config` table and tax percentage logic
- Full subscription billing / payment-provider integration
- Multi-facility under one organisation (explicitly **not** allowed)
- Changing invoice schema to require `service_id` (follow-up once catalog is live)
- Cross-tenant data access

---

## 13. Implementation checklist

### Schema & models

- [ ] Add `tenants` model + migration
- [ ] Alter `organisations` to target columns + `tenant_id`
- [ ] Add `services` model + migration
- [ ] Data backfill / rename migration for existing orgs
- [x] Stop migrating `licenses` (AutoMigrate removed; drop table in DB when ready)

### Organisation module

- [ ] Update structures, DTOs, controllers, services, repo
- [x] Remove `LicenseCreator` from signup TX
- [ ] Tenant resolve/create in signup TX
- [ ] Address + data_sharing update paths
- [ ] List-by-tenant API
- [x] Regenerate mocks; fix unit tests (license expectations removed)

### Service catalog module

- [ ] New package (CRUD layers)
- [ ] Routes + optional RBAC module
- [ ] Default Consultation seed on org create
- [ ] Tests (create/list/update/deactivate + org isolation)

### Tenant module

- [ ] New package (CRUD layers)
- [ ] Routes + wiring

### License eradication

- [x] Remove routes, appinit wiring, constants, module seed list
- [x] Delete `internal/license`
- [x] Update API placeholder docs
- [ ] Drop leftover `licenses` / `modules.name='license'` rows in existing DBs (ops)

### Docs

- [ ] Update `docs/erd.md` organisation section
- [ ] Update `docs/api-placeholders.md`
- [ ] Keep this file as the change contract until implementation lands

---

## 14. Open decisions (product)

1. Is subscription strictly **tenant-scoped** (recommended) or also overridable per org?
2. Default trial length / plan when provisioning on signup?
3. Exact `data_sharing` key set and defaults for multi-org tenants?
4. Should `code` (public hospital code) be kept for integrations?
5. Package name for catalog: `orgservice` vs `servicecatalog` (avoid colliding with `*Service` types)?
6. When should billing start resolving consultation amount from `services`?
