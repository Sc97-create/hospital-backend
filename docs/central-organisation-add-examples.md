# Central organisation — add examples

This is **add organisation** under an existing `tenant_id`.  
Tenant create/signup will be added later; for now pass any UUID as `tenant_id` while testing.

Base path: `/api/v1/central/organisation`

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/add` | Add organisation under a tenant |
| `GET` | `/getbyid/:organisation_id` | Fetch one |
| `GET` | `/listByTenant/:tenant_id` | List orgs for a tenant |
| `PATCH` | `/update` | Update org fields |
| `PATCH` | `/updateAddress` | Update address + data_sharing |

---

## 1. Minimal add (required fields only)

```http
POST /api/v1/central/organisation/add
Content-Type: application/json
```

```json
{
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "legal_entity_name": "Acme Health Pvt Ltd",
  "organisation_type": "hospital",
  "facility_name": "Acme City Hospital - Main Campus"
}
```

Expected `200`:

```json
{
  "message": "organisation added successfully",
  "organisation_id": "<uuid>"
}
```

---

## 2. Full add (with address, regulatory, data_sharing)

```json
{
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "legal_entity_name": "City Care Hospitals",
  "organisation_type": "hospital",
  "facility_name": "City Care - Indiranagar",
  "registration_no": "REG-KA-2026-001",
  "license_number": "CL-KA-7788",
  "license_expiry": "2028-06-30",
  "gstin": "29AAAAA0000A1Z5",
  "country_id": "IN",
  "state": "KA",
  "city": "BLR",
  "patient_lookup": true,
  "lab_reports": false,
  "status": "active"
}
```

---

## 3. Clinic under same tenant

```json
{
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "legal_entity_name": "City Care Hospitals",
  "organisation_type": "clinic",
  "facility_name": "City Care - Whitefield Clinic",
  "patient_lookup": true,
  "lab_reports": true
}
```

---

## 4. curl

```bash
BASE_URL="${BASE_URL:-http://localhost:3000}"
TENANT_ID="11111111-1111-1111-1111-111111111111"

curl -sS -X POST "$BASE_URL/api/v1/central/organisation/add" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenant_id\": \"$TENANT_ID\",
    \"legal_entity_name\": \"Acme Health Pvt Ltd\",
    \"organisation_type\": \"hospital\",
    \"facility_name\": \"Acme City Hospital - Main Campus\",
    \"registration_no\": \"REG-KA-123\",
    \"license_number\": \"CL-KA-456\",
    \"license_expiry\": \"2027-12-31\",
    \"gstin\": \"29AAAAA0000A1Z5\",
    \"country_id\": \"IN\",
    \"state\": \"KA\",
    \"city\": \"BLR\",
    \"patient_lookup\": true,
    \"lab_reports\": false
  }"
```

Get by id (replace `ORG_ID`):

```bash
curl -sS "$BASE_URL/api/v1/central/organisation/getbyid/ORG_ID"
```

List by tenant:

```bash
curl -sS "$BASE_URL/api/v1/central/organisation/listByTenant/$TENANT_ID"
```

---

## 5. Validation failures (expect `400`)

Missing `tenant_id`:

```json
{
  "legal_entity_name": "Acme Health Pvt Ltd",
  "organisation_type": "hospital",
  "facility_name": "Main Campus"
}
```

Missing `facility_name`:

```json
{
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "legal_entity_name": "Acme Health Pvt Ltd",
  "organisation_type": "hospital"
}
```

---

## Notes

- `tenant_id` is required on add; tenant entity/API comes in a later pass.
- `legal_entity_name` is the single org name (no separate display name).
- `data_sharing` only supports `patient_lookup` and `lab_reports`.
- Legacy `POST /api/v1/organisation/signupOrg` has been removed; use `POST /api/v1/central/organisation/add`.
