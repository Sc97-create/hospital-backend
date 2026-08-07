# Postman Test Scenarios — Add Patient API

Quick reference for manual and automated testing of `POST /api/v1/patients/addGeneralInfo`.

## Summary Checklist

| ID | Scenario | Expected Status |
|---|---|---|
| A1 | Missing Authorization header | 401 |
| A2 | Invalid token format | 401 |
| A3 | Invalid or expired Bearer token | 401 |
| B1 | Valid complete payload | 200 |
| B2 | Verify patient via getpatientByID | 200 |
| C1–C10 | Missing required field (each field) | 409 |
| C11 | Empty body `{}` | 409 |
| C12 | Malformed JSON | 409 |
| C13 | `age` / `weight` as JSON numbers | 409 |
| D1 | Empty `name` | 409 |
| D2 | Empty `gender` | 409 |
| D3 | Negative `age` | 409 |
| D4 | Zero `weight` | 409 |
| D5 | Negative `weight` | 409 |
| D6 | Non-numeric `age` | 200 (edge case) |
| E1 | Duplicate `email_id` | 409 |
| E2 | Duplicate `mobile_number` | 409 |

---

## 1. Setup

### Environment variables

Create a Postman environment with:

| Variable | Example | Description |
|---|---|---|
| `base_url` | `http://localhost:9069` | Server URL (default from `deploy/local/.env`) |
| `access_token` | *(set by login)* | JWT from authentication login |
| `organisation_id` | `11111111-2222-3333-4444-555555555555` | Valid organisation UUID from DB |
| `user_id` | `ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb` | Valid user UUID (creator) |
| `patient_id` | *(set by B1)* | Created patient UUID for follow-up tests |

### Prerequisite — Login

Obtain a JWT before calling patient APIs (all patient routes require auth).

**Request**

```
POST {{base_url}}/api/v1/authentication/login
Content-Type: application/json
```

**Body**

```json
{
  "user_name": "admin@example.com",
  "password": "your-password"
}
```

**Expected response (200)**

```json
{
  "token": "<jwt>",
  "refresh_token": "<refresh>"
}
```

**Postman Tests**

```javascript
pm.test("Login successful", () => pm.response.to.have.status(200));
const json = pm.response.json();
pm.environment.set("access_token", json.token);
```

---

## 2. Target API

| Item | Value |
|---|---|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/patients/addGeneralInfo` |
| Auth | `Authorization: Bearer {{access_token}}` |
| Content-Type | `application/json` |

### Request fields

All fields are required. Values must be **JSON strings** (controller uses `params.Getstring`).

| Field | JSON key | Notes |
|---|---|---|
| Name | `name` | Non-empty after service validation |
| Gender | `gender` | Non-empty after service validation |
| Age | `age` | String, e.g. `"30"` — not a JSON number |
| Weight | `weight` | String, e.g. `"70"` — must parse to > 0 |
| Email | `email_id` | Unique in DB |
| Mobile | `mobile_number` | Unique in DB |
| Blood group | `blood_group` | Key required; value may be `""` |
| Address | `address` | Key required; value may be `""` |
| Organisation | `organisation_id` | UUID string |
| User | `user_id` | UUID string (creator) |

### Base valid payload

Use this as the starting body for happy-path and mutation tests. Change one field per scenario.

```json
{
  "name": "John Doe",
  "gender": "male",
  "age": "30",
  "weight": "70",
  "email_id": "john.doe.test@example.com",
  "mobile_number": "9876543210",
  "blood_group": "O+",
  "address": "123 Main Street, City",
  "organisation_id": "{{organisation_id}}",
  "user_id": "{{user_id}}"
}
```

> **Tip:** Use unique `email_id` and `mobile_number` per run to avoid E1/E2 failures during happy-path testing.

### Success response (200)

```json
{
  "message": "general info added",
  "patient_id": "<uuid>",
  "code": 200
}
```

### Error response (409)

```json
{
  "error": "<message>",
  "code": 409
}
```

### Auth error response (401)

```json
{
  "error": "Missing Authorization header"
}
```

---

## 3. Test Scenarios

### A. Authentication

#### A1 — Missing Authorization header

**Preconditions:** None

**Request**

```
POST {{base_url}}/api/v1/patients/addGeneralInfo
Content-Type: application/json
(no Authorization header)
```

**Body:** Base valid payload

**Expected:** `401`

```json
{
  "error": "Missing Authorization header"
}
```

**Postman Tests**

```javascript
pm.test("Status is 401", () => pm.response.to.have.status(401));
pm.test("Missing auth error", () => {
  pm.expect(pm.response.json().error).to.eql("Missing Authorization header");
});
```

---

#### A2 — Invalid token format

**Request**

```
POST {{base_url}}/api/v1/patients/addGeneralInfo
Authorization: Token invalid-format
Content-Type: application/json
```

**Body:** Base valid payload

**Expected:** `401`

```json
{
  "error": "Invalid token format"
}
```

**Postman Tests**

```javascript
pm.test("Status is 401", () => pm.response.to.have.status(401));
pm.test("Invalid format error", () => {
  pm.expect(pm.response.json().error).to.eql("Invalid token format");
});
```

---

#### A3 — Invalid or expired Bearer token

**Request**

```
POST {{base_url}}/api/v1/patients/addGeneralInfo
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid
Content-Type: application/json
```

**Body:** Base valid payload

**Expected:** `401`

```json
{
  "error": "Invalid or expired token"
}
```

**Postman Tests**

```javascript
pm.test("Status is 401", () => pm.response.to.have.status(401));
pm.test("Invalid token error", () => {
  pm.expect(pm.response.json().error).to.eql("Invalid or expired token");
});
```

---

### B. Happy path

#### B1 — Valid complete payload

**Preconditions:** Valid `access_token`; unique email and mobile

**Request**

```
POST {{base_url}}/api/v1/patients/addGeneralInfo
Authorization: Bearer {{access_token}}
Content-Type: application/json
```

**Body:** Base valid payload (use fresh `email_id` / `mobile_number`)

**Expected:** `200`

```json
{
  "message": "general info added",
  "patient_id": "<uuid>",
  "code": 200
}
```

**Postman Tests**

```javascript
pm.test("Status is 200", () => pm.response.to.have.status(200));
pm.test("Response shape", () => {
  const json = pm.response.json();
  pm.expect(json.message).to.eql("general info added");
  pm.expect(json.code).to.eql(200);
  pm.expect(json.patient_id).to.be.a("string").and.not.empty;
});
pm.environment.set("patient_id", pm.response.json().patient_id);
```

---

#### B2 — Verify patient via getpatientByID

**Preconditions:** Run B1 first; `patient_id` set in environment

**Request**

```
GET {{base_url}}/api/v1/patients/getpatientByID/{{patient_id}}
Authorization: Bearer {{access_token}}
```

**Expected:** `200`

```json
{
  "data": {
    "patient_id": "<uuid>",
    "patient_name": "John Doe",
    "patient_gender": "male",
    "patient_age": 30,
    "patient_weight": 70,
    "patient_phone": "9876543210",
    "patient_email": "john.doe.test@example.com",
    "patient_bg": "O+",
    "patient_address": "123 Main Street, City",
    "patient_status": "active",
    "patient_code": "CLI-<number>"
  },
  "code": 200
}
```

**Postman Tests**

```javascript
pm.test("Status is 200", () => pm.response.to.have.status(200));
pm.test("Patient data matches", () => {
  const data = pm.response.json().data;
  pm.expect(data.patient_id).to.eql(pm.environment.get("patient_id"));
  pm.expect(data.patient_name).to.eql("John Doe");
  pm.expect(data.patient_gender).to.eql("male");
  pm.expect(data.patient_age).to.eql(30);
});
```

---

### C. Missing / invalid payload — controller layer

For C1–C10: start from base valid payload and **omit one field**. All return `409` with `"required payload is missing"` (first missing field encountered depends on controller parse order: `name` → `blood_group` → `address` → `age` → `user_id` → `weight` → `gender` → `organisation_id` → `email_id` → `mobile_number`).

| ID | Omit field | Expected error |
|---|---|---|
| C1 | `name` | `required payload is missing` |
| C2 | `gender` | `required payload is missing` |
| C3 | `age` | `required payload is missing` |
| C4 | `weight` | `required payload is missing` |
| C5 | `email_id` | `required payload is missing` |
| C6 | `mobile_number` | `required payload is missing` |
| C7 | `blood_group` | `required payload is missing` |
| C8 | `address` | `required payload is missing` |
| C9 | `organisation_id` | `required payload is missing` |
| C10 | `user_id` | `required payload is missing` |

**Example — C1 (missing name)**

```json
{
  "gender": "male",
  "age": "30",
  "weight": "70",
  "email_id": "test.c1@example.com",
  "mobile_number": "9876543211",
  "blood_group": "O+",
  "address": "123 Main Street",
  "organisation_id": "{{organisation_id}}",
  "user_id": "{{user_id}}"
}
```

**Postman Tests (reusable for C1–C10)**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Missing field error", () => {
  pm.expect(pm.response.json().error).to.eql("required payload is missing");
});
```

---

#### C11 — Empty body

**Body**

```json
{}
```

**Expected:** `409` — `"required payload is missing"`

---

#### C12 — Malformed JSON

**Body** (raw text, not valid JSON)

```
{ name: "John Doe" }
```

**Expected:** `409` — JSON unmarshal error message

---

#### C13 — age / weight as JSON numbers

**Body** (invalid types for controller)

```json
{
  "name": "John Doe",
  "gender": "male",
  "age": 30,
  "weight": 70,
  "email_id": "test.c13@example.com",
  "mobile_number": "9876543213",
  "blood_group": "O+",
  "address": "123 Main Street",
  "organisation_id": "{{organisation_id}}",
  "user_id": "{{user_id}}"
}
```

**Expected:** `409` — `"required payload is missing"` (numbers fail `Getstring` type check)

---

### D. Service validation

Start from base valid payload; change only the noted field. Use unique email/mobile per request.

#### D1 — Empty name

```json
"name": ""
```

**Expected:** `409` — `"please provide name"`

---

#### D2 — Empty gender

```json
"gender": ""
```

**Expected:** `409` — `"please provide valid gender"`

---

#### D3 — Negative age

```json
"age": "-1"
```

**Expected:** `409` — `"age should be greater then 0"`

---

#### D4 — Zero weight

```json
"weight": "0"
```

**Expected:** `409` — `"weight should not be 0"`

---

#### D5 — Negative weight

```json
"weight": "-5"
```

**Expected:** `409` — `"weight should not be 0"`

---

#### D6 — Non-numeric age (edge case)

```json
"age": "abc"
```

**Expected:** `200` — `strconv.Atoi` returns 0; age `0` passes validation (`age < 0` check only). Patient is created with `age: 0`.

> Document as known behavior. Consider fixing in service layer separately.

**Postman Tests**

```javascript
pm.test("Status is 200 (edge case)", () => pm.response.to.have.status(200));
```

---

**Postman Tests (reusable for D1–D5)**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Validation error present", () => {
  pm.expect(pm.response.json().error).to.be.a("string").and.not.empty;
});
```

---

### E. Database constraints

#### E1 — Duplicate email_id

**Preconditions:** Run B1 first with a known email; submit again with same `email_id` and a **new** `mobile_number`

**Expected:** `409` — PostgreSQL unique violation on `email_id`

---

#### E2 — Duplicate mobile_number

**Preconditions:** Run B1 first with a known mobile; submit again with same `mobile_number` and a **new** `email_id`

**Expected:** `409` — PostgreSQL unique violation on `mobile_no`

**Postman Tests (reusable for E1–E2)**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Duplicate constraint error", () => {
  pm.expect(pm.response.json().error).to.be.a("string").and.not.empty;
});
```

---

### F. Optional edge cases

| Scenario | Expected behavior |
|---|---|
| Invalid UUID for `organisation_id` or `user_id` | Likely `409` from DB insert failure |
| `address` longer than 500 characters | May truncate or fail depending on DB |
| `blood_group` / `address` as empty strings `""` | Allowed — only key presence is validated |
| `age: "0"` | Allowed — passes service validation |

---

## 4. Recommended Postman collection order

1. Login (prerequisite)
2. A1 → A3 (auth)
3. B1 → B2 (happy path + verify)
4. C1 → C13 (missing / invalid payload)
5. D1 → D6 (service validation)
6. E1 → E2 (uniqueness)

---

## 5. Common headers (all addGeneralInfo requests)

```
Authorization: Bearer {{access_token}}
Content-Type: application/json
Accept: application/json
```

---

## 6. Related endpoints (out of scope, except B2)

| Method | URL | Purpose |
|---|---|---|
| GET | `/api/v1/patients/getPatients?limit=10&page_no=1&organisation_id={{organisation_id}}` | List patients |
| GET | `/api/v1/patients/getpatientByID/:patientID` | Get single patient (B2 verification) |

Patient update API is not yet implemented — see [patient-update.md](./patient-update.md).

---

# Postman Test Scenarios — Add Medicine API

Quick reference for manual and automated testing of `POST /api/v1/medicine/addMedicine`.

> **Note:** Medicine routes do **not** require JWT auth (unlike patient routes).

## Summary Checklist

| ID | Scenario | Expected Status |
|---|---|---|
| M-B1 | Add new medicine (no `medicine_id`) | 200 |
| M-B2 | Add batch to existing medicine | 200 |
| M-B3 | Multiple items in one request | 200 |
| M-B4 | Verify via searchMedicine | 200 |
| M-C1 | Missing `user_id` | 409 |
| M-C2 | Missing `supplier_id` | 409 |
| M-C3 | Missing `organisation_id` | 409 |
| M-C4 | Missing `invoice_no` | 409 |
| M-C5 | Missing `medicine_info` | 409 |
| M-C6 | Empty body `{}` | 409 |
| M-C7 | Malformed JSON | 409 |
| M-C8 | `medicine_info` not an array | 409 |
| M-D1 | Invalid / non-existent `supplier_id` | 409 |
| M-D2 | Empty `medicine_info` array | 200 (no-op insert) |
| M-D3 | Missing item fields (defaults applied) | 200 |

---

## 1. Setup

### Additional environment variables

| Variable | Example | Description |
|---|---|---|
| `supplier_id` | Valid supplier UUID from DB | Required FK for purchase entry |
| `medicine_id` | *(set by M-B1)* | Existing medicine UUID for restock scenario |
| `invoice_no` | `GST-2026-0001` | Unique per test run to avoid duplicate invoice issues |

### Prerequisite — Create supplier (if none exists)

```
POST {{base_url}}/api/v1/supplier/createSupplier
Content-Type: application/json
```

**Body**

```json
{
  "user_id": "{{user_id}}",
  "organisation_id": "{{organisation_id}}",
  "name": "Test Pharma Supplier",
  "payment_terms": "Net30",
  "email_id": "supplier.test@example.com",
  "drug_license_number": "DL-TEST-12345",
  "contact_number": "9876500000",
  "credit_limit": 100000,
  "gst_number": "29ABCDE1234F1Z5"
}
```

**Expected:** `200` — `"supplier created successfully"`

Use the returned supplier UUID as `{{supplier_id}}`, or fetch via:

```
GET {{base_url}}/api/v1/supplier/getSupplierByID?supplier_id={{supplier_id}}
```

---

## 2. Target API

| Item | Value |
|---|---|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/medicine/addMedicine` |
| Auth | None (no JWT middleware on medicine routes) |
| Content-Type | `application/json` |

### Top-level request fields

| Field | JSON key | Required | Type | Notes |
|---|---|---|---|---|
| User | `user_id` | Yes | string | Creator UUID |
| Supplier | `supplier_id` | Yes | string | Must exist in DB |
| Organisation | `organisation_id` | Yes | string | UUID |
| Invoice number | `invoice_no` | Yes | string | Purchase invoice reference |
| Payment due | `payment_due_date` | No | string | RFC3339; optional |
| Items | `medicine_info` | Yes | array | At least one item for meaningful test |

### `medicine_info` item fields

| Field | JSON key | Required | Type | Notes |
|---|---|---|---|---|
| Medicine ID | `medicine_id` | No | string | Omit or `""` to create **new** medicine |
| Name | `name` | No* | string | Required in practice for new medicines |
| Form | `form` | No | string | e.g. `TABLET`, `CREAM` |
| Strength | `strength` | No | string | e.g. `650mg` |
| HSN code | `hsn_code` | No | string | |
| Batch | `batch_no` | No | string | e.g. `BAT-4029` |
| Expiry | `expiry_date` | No | string | RFC3339 e.g. `2028-12-31T00:00:00Z` |
| Shelf | `shelf_location` | No | string | e.g. `Rack 3-B` |
| Boxes purchased | `purchase_qty_boxes` | No | number | Used for stock: `boxes × units_per_box` |
| Units per box | `units_per_box` | No | number | Pack size |
| MRP | `mrp` | No | number | |
| Purchase price | `purchase_price` | No | number | Per box |
| Discount | `discount` | No | number | |
| Selling price | `selling_price` | No | number | Defaults to `mrp` if `0` |
| Reorder level | `reorder_level` | No | number | |
| Max stock target | `max_stock_target` | No | number | |
| Quantity | `quantity` | No | number | Parsed but stock uses boxes × units |

\*Controller does not enforce non-empty `name`, but DB may reject empty `name` on insert.

### Base valid payload — new medicine

Use a **unique** `invoice_no` and `batch_no` per run.

```json
{
  "user_id": "{{user_id}}",
  "supplier_id": "{{supplier_id}}",
  "organisation_id": "{{organisation_id}}",
  "invoice_no": "GST-2026-0001",
  "payment_due_date": "2026-07-20T12:00:00Z",
  "medicine_info": [
    {
      "name": "Dolo 650mg",
      "form": "TABLET",
      "strength": "650mg",
      "hsn_code": "30049011",
      "batch_no": "BAT-4029",
      "expiry_date": "2028-12-31T00:00:00Z",
      "shelf_location": "Rack 3-B",
      "purchase_qty_boxes": 10,
      "units_per_box": 15,
      "mrp": 30.00,
      "purchase_price": 21.50,
      "discount": 5.00,
      "selling_price": 28.50,
      "reorder_level": 50,
      "max_stock_target": 200
    }
  ]
}
```

### Success response (200)

```json
{
  "message": "created successfully"
}
```

### Error response (409)

```json
{
  "error": "<message>",
  "code": 409
}
```

### What gets created (happy path)

In one DB transaction:

1. **Purchase entry** — invoice linked to supplier
2. **Medicine** — new row only when `medicine_id` is empty (auto-generated UUID)
3. **Medicine inventory** — batch row with `current_stock_units = purchase_qty_boxes × units_per_box`
4. **Stock movement** — `purchase` type with qty added

---

## 3. Test Scenarios

### M-B. Happy path

#### M-B1 — Add new medicine (no `medicine_id`)

**Preconditions:** Valid `supplier_id` exists; unique `invoice_no`

**Request**

```
POST {{base_url}}/api/v1/medicine/addMedicine
Content-Type: application/json
```

**Body:** Base valid payload (omit `medicine_id` in item)

**Expected:** `200` — `"created successfully"`

**Postman Tests**

```javascript
pm.test("Status is 200", () => pm.response.to.have.status(200));
pm.test("Created message", () => {
  pm.expect(pm.response.json().message).to.eql("created successfully");
});
```

**Follow-up:** Search to capture generated medicine — run M-B4 with `name=Dolo`

---

#### M-B2 — Add batch to existing medicine

**Preconditions:** M-B1 completed; set `medicine_id` from search or DB

**Body change:** Include existing `medicine_id`; new `batch_no` and unique `invoice_no`

```json
{
  "user_id": "{{user_id}}",
  "supplier_id": "{{supplier_id}}",
  "organisation_id": "{{organisation_id}}",
  "invoice_no": "GST-2026-0002",
  "medicine_info": [
    {
      "medicine_id": "{{medicine_id}}",
      "name": "Dolo 650mg",
      "form": "TABLET",
      "strength": "650mg",
      "batch_no": "BAT-9112",
      "expiry_date": "2027-06-30T00:00:00Z",
      "shelf_location": "Rack 3-B",
      "purchase_qty_boxes": 5,
      "units_per_box": 15,
      "mrp": 30.00,
      "purchase_price": 21.50,
      "discount": 0,
      "selling_price": 28.50
    }
  ]
}
```

**Expected:** `200` — inventory batch added; no duplicate medicine master row (`Add=false` when `medicine_id` provided)

---

#### M-B3 — Multiple medicines in one request

**Body:** Two items in `medicine_info` — one new (no `medicine_id`), one with existing `medicine_id`

```json
{
  "user_id": "{{user_id}}",
  "supplier_id": "{{supplier_id}}",
  "organisation_id": "{{organisation_id}}",
  "invoice_no": "GST-2026-0003",
  "medicine_info": [
    {
      "name": "Elocon Ointment",
      "form": "CREAM",
      "strength": "5g",
      "hsn_code": "30049012",
      "batch_no": "CRM-8812",
      "expiry_date": "2027-04-30T00:00:00Z",
      "shelf_location": "Fridge-2",
      "purchase_qty_boxes": 5,
      "units_per_box": 1,
      "mrp": 120.00,
      "purchase_price": 95.00,
      "discount": 0,
      "selling_price": 120.00
    },
    {
      "medicine_id": "{{medicine_id}}",
      "name": "Dolo 650mg",
      "form": "TABLET",
      "strength": "650mg",
      "batch_no": "BAT-5500",
      "expiry_date": "2028-01-01T00:00:00Z",
      "purchase_qty_boxes": 2,
      "units_per_box": 15,
      "mrp": 30.00,
      "purchase_price": 21.50,
      "selling_price": 28.50
    }
  ]
}
```

**Expected:** `200`

---

#### M-B4 — Verify via searchMedicine

**Request**

```
GET {{base_url}}/api/v1/medicine/searchMedicine?name=Dolo
```

**Expected:** `200` with matching medicine in `data` array

**Postman Tests**

```javascript
pm.test("Status is 200", () => pm.response.to.have.status(200));
pm.test("Medicine found", () => {
  const data = pm.response.json().data;
  pm.expect(data).to.be.an("array").that.is.not.empty;
  if (data.length > 0) {
    pm.environment.set("medicine_id", data[0].id);
  }
});
```

---

### M-C. Missing / invalid payload — controller layer

Start from base valid payload; omit one field per test. All return `409`.

| ID | Omit / change | Expected error |
|---|---|---|
| M-C1 | `user_id` | `required payload is missing` |
| M-C2 | `supplier_id` | `required payload is missing` |
| M-C3 | `organisation_id` | `required payload is missing` |
| M-C4 | `invoice_no` | `required payload is missing` |
| M-C5 | `medicine_info` | `not a valid type` |
| M-C6 | Empty `{}` body | `required payload is missing` |
| M-C7 | Malformed JSON | JSON unmarshal error |
| M-C8 | `"medicine_info": "not-an-array"` | `not a valid type` |

**Postman Tests (reusable for M-C1–M-C8)**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Error present", () => {
  pm.expect(pm.response.json().error).to.be.a("string").and.not.empty;
});
```

---

### M-D. Business / dependency scenarios

#### M-D1 — Invalid supplier_id

**Body change:** `"supplier_id": "00000000-0000-0000-0000-000000000000"`

**Expected:** `409` — supplier lookup / FK failure from `GetSupplierByID`

---

#### M-D2 — Empty `medicine_info` array

```json
{
  "user_id": "{{user_id}}",
  "supplier_id": "{{supplier_id}}",
  "organisation_id": "{{organisation_id}}",
  "invoice_no": "GST-2026-EMPTY",
  "medicine_info": []
}
```

**Expected:** `200` — purchase entry created; no medicines/inventory/m movements inserted

---

#### M-D3 — Minimal item (defaults applied)

**Body:** Single item with only `name`, `batch_no`, `purchase_qty_boxes`, `units_per_box`

```json
{
  "user_id": "{{user_id}}",
  "supplier_id": "{{supplier_id}}",
  "organisation_id": "{{organisation_id}}",
  "invoice_no": "GST-2026-MINIMAL",
  "medicine_info": [
    {
      "name": "Minimal Test Med",
      "batch_no": "BAT-MIN-001",
      "purchase_qty_boxes": 1,
      "units_per_box": 10
    }
  ]
}
```

**Expected:** `200` — new medicine created with defaults (`selling_price` → `mrp`, zero values where omitted)

---

### M-E. Optional edge cases

| Scenario | Expected behavior |
|---|---|
| Duplicate `invoice_no` for same supplier | May fail on DB unique constraint → 409 |
| Invalid `expiry_date` format | Parses as zero time; may still return 200 |
| `selling_price: 0` | Controller sets `selling_price = mrp` |
| `purchase_qty_boxes: 0` | `current_stock_units = 0` |
| Same `batch_no` for same medicine | Depends on DB constraints |

---

## 4. Recommended Postman collection order

1. Create supplier (prerequisite)
2. M-B1 → M-B4 (happy path + verify)
3. M-B2 → M-B3 (restock + multi-item)
4. M-C1 → M-C8 (validation)
5. M-D1 → M-D3 (dependencies + edge cases)

---

## 5. Common headers

```
Content-Type: application/json
Accept: application/json
```

---

## 6. Related endpoints

| Method | URL | Purpose |
|---|---|---|
| GET | `/api/v1/medicine/searchMedicine?name={{name}}` | Search medicine by name (M-B4) |
| GET | `/api/v1/medicine/getMedicineByID?id={{medicine_id}}` | Get by ID (stub response currently) |
| POST | `/api/v1/supplier/createSupplier` | Create supplier prerequisite |
| GET | `/api/v1/supplier/getSupplierByID?supplier_id={{supplier_id}}` | Verify supplier exists |

See [medicine-inventory.md](../medicine-inventory.md) for full purchase payload reference and stock calculation rules.

---

# Postman Test Scenarios — Checkout API

Quick reference for manual and automated testing of `POST /api/v1/billing/create`.

This endpoint creates an invoice, inserts invoice items (with quantity validation), and returns a Razorpay payment link. Inventory deduction and prescription status updates happen **after** the patient pays via the webhook.

## Summary Checklist

| ID | Scenario | Expected Status |
|---|---|---|
| CK-B1 | Full valid checkout — single item | 200 |
| CK-B2 | Full valid checkout — multiple items | 200 |
| CK-B3 | Verify invoice and items via DB / audit | 200 |
| CK-C1 | Missing `prescription_id` | 409 |
| CK-C2 | Missing `patient_id` | 409 |
| CK-C3 | Missing `cashier_id` | 409 |
| CK-C4 | Missing `organisation_id` | 409 |
| CK-C5 | Missing `supplier_id` | 409 |
| CK-C6 | Missing `financials.total_amount` | 409 |
| CK-C7 | Missing `dispense_items` | 409 |
| CK-C8 | Empty body `{}` | 409 |
| CK-D1 | `dispensed_qty` > `current_stock_units` (inventory exceeded) | 409 |
| CK-D2 | `dispensed_qty` > `prescribed_qty - balance_after_dispense` (prescription exceeded) | 409 |
| CK-D3 | `dispensed_qty` exactly equals remaining prescribed qty | 200 |
| CK-D4 | Item not in prescription (`medicine_id` not found in prescription) | 409 |
| CK-D5 | Invalid `patient_id` (no patient row) | 409 |
| CK-D6 | Invalid `prescription_id` (prescription not in DB) | 409 |
| CK-E1 | Partial payment — `accept_partial` response (Razorpay mock) | 200 (link returned) |

---

## 1. Setup

### Additional environment variables

| Variable | Example | Description |
|---|---|---|
| `prescription_id` | UUID | Active prescription for the patient |
| `prescription_item_id` | UUID | ID of an item row inside that prescription |
| `medicine_id` | UUID | Medicine UUID matching the prescription item |
| `medicine_inventory_id` | UUID | Batch UUID (from `medicine_inventories`) |
| `batch_no` | `BAT-4029` | Batch number string |
| `current_stock_units` | `150` | Current stock from `medicine_inventories.current_stock_units` |
| `prescribed_qty` | `30` | `prescription_items.quantity` |
| `balance_after_dispense` | `0` | Already dispensed: `prescription_items.balance_after_dispense` |

### How to get prerequisite UUIDs

Run these queries in your DB before testing (or via a GET endpoint):

```sql
-- Active prescription for a patient
SELECT id, patient_id FROM prescriptions WHERE patient_id = '<patient_uuid>' LIMIT 1;

-- Items inside that prescription
SELECT id, medicine_id, quantity, balance_after_dispense
FROM prescription_items WHERE prescription_id = '<prescription_uuid>';

-- Inventory batch for that medicine
SELECT id, batch_no, current_stock_units
FROM medicine_inventories WHERE medicine_id = '<medicine_uuid>' LIMIT 1;
```

---

## 2. Target API

| Item | Value |
|---|---|
| Method | `POST` |
| URL | `{{base_url}}/api/v1/billing/create` |
| Auth | `Authorization: Bearer {{access_token}}` |
| Content-Type | `application/json` |

### Top-level request fields

| Field | JSON key | Required | Type | Notes |
|---|---|---|---|---|
| Prescription | `prescription_id` | Yes | string UUID | Links invoice to prescription |
| Patient | `patient_id` | Yes | string UUID | Used to fetch patient info for Razorpay |
| Cashier | `cashier_id` | Yes | string UUID | Staff initiating checkout |
| Organisation | `organisation_id` | Yes | string UUID | |
| Supplier | `supplier_id` | Yes | string UUID | |
| Payment mode | `payment_mode` | No | string | e.g. `"upi"`, `"card"` |
| Financials | `financials` | Yes | object | See below |
| Items | `dispense_items` | Yes | array | One entry per medicine batch |

### `financials` object

| Field | JSON key | Required | Notes |
|---|---|---|---|
| Subtotal | `sub_total_amount` | Yes | Before tax/discount |
| Tax | `tax_amount` | Yes | Can be `0` |
| Discount | `discount_amount` | No | Defaults to `0` |
| Total | `total_amount` | Yes | What Razorpay link is created for |

### `dispense_items` array item fields

| Field | JSON key | Required | Notes |
|---|---|---|---|
| Medicine | `medicine_id` | Yes | UUID |
| Inventory batch | `medicine_inventory_id` | Yes | UUID — specific batch being dispensed |
| Prescription item | `prescription_item_id` | Yes | UUID — links to `prescription_items.id` |
| Batch number | `batch_no` | Yes | String, used in error messages |
| Current stock | `current_stock_units` | Yes | Frontend must pass live stock count for validation |
| Qty to dispense | `quantity_sold_units` | Yes | Integer |
| Unit price | `unit_price_charged` | No | Selling price per unit |
| Item subtotal | `computed_item_total` | No | `unit_price × qty` |
| Item total | `total_amount` | Yes | Final price for this item |

### Base valid payload

Replace all `{{...}}` with real UUIDs from your DB.

```json
{
  "prescription_id": "{{prescription_id}}",
  "patient_id": "{{patient_id}}",
  "cashier_id": "{{cashier_id}}",
  "organisation_id": "{{organisation_id}}",
  "supplier_id": "{{supplier_id}}",
  "payment_mode": "upi",
  "financials": {
    "sub_total_amount": 171.00,
    "tax_amount": 0.00,
    "discount_amount": 0.00,
    "total_amount": 171.00
  },
  "dispense_items": [
    {
      "medicine_id": "{{medicine_id}}",
      "medicine_inventory_id": "{{medicine_inventory_id}}",
      "prescription_item_id": "{{prescription_item_id}}",
      "batch_no": "{{batch_no}}",
      "current_stock_units": 150,
      "quantity_sold_units": 6,
      "unit_price_charged": 28.50,
      "computed_item_total": 171.00,
      "total_amount": 171.00
    }
  ]
}
```

> **Financial consistency:** `total_amount` in `financials` should equal the sum of all `dispense_items[].total_amount`. Mismatch is not validated server-side today but will cause Razorpay link amount to differ.

### Success response (200)

```json
{
  "message": "stored",
  "payment_link": "https://rzp.io/l/xxxxxxxxxxxx"
}
```

### Error response (409)

```json
{
  "error": "<message>",
  "code": 409
}
```

---

## 3. Test Scenarios

### CK-B. Happy path

#### CK-B1 — Single item checkout

**Preconditions:**
- Valid `prescription_id` with at least one active item
- `balance_after_dispense < prescribed_qty` for that item
- `current_stock_units >= quantity_sold_units`
- Valid `patient_id` with phone/email (needed for Razorpay SMS/email)

**Request**

```
POST {{base_url}}/api/v1/billing/create
Authorization: Bearer {{access_token}}
Content-Type: application/json
```

**Body:** Base valid payload above (qty = 6, stock = 150, prescribed = 30, already dispensed = 0)

**Expected:** `200`

```json
{
  "message": "stored",
  "payment_link": "https://rzp.io/l/xxxxxxxxxxxx"
}
```

**Postman Tests**

```javascript
pm.test("Status is 200", () => pm.response.to.have.status(200));
pm.test("Payment link returned", () => {
  const json = pm.response.json();
  pm.expect(json.message).to.eql("stored");
  pm.expect(json.payment_link).to.be.a("string").and.include("rzp.io");
});
pm.environment.set("payment_link", pm.response.json().payment_link);
```

---

#### CK-B2 — Multiple items checkout

**Preconditions:** Prescription has at least 2 active items with separate inventory batches

**Body:** Extend `dispense_items` with a second medicine

```json
{
  "prescription_id": "{{prescription_id}}",
  "patient_id": "{{patient_id}}",
  "cashier_id": "{{cashier_id}}",
  "organisation_id": "{{organisation_id}}",
  "supplier_id": "{{supplier_id}}",
  "payment_mode": "upi",
  "financials": {
    "sub_total_amount": 411.00,
    "tax_amount": 0.00,
    "discount_amount": 0.00,
    "total_amount": 411.00
  },
  "dispense_items": [
    {
      "medicine_id": "{{medicine_id}}",
      "medicine_inventory_id": "{{medicine_inventory_id}}",
      "prescription_item_id": "{{prescription_item_id}}",
      "batch_no": "{{batch_no}}",
      "current_stock_units": 150,
      "quantity_sold_units": 6,
      "unit_price_charged": 28.50,
      "computed_item_total": 171.00,
      "total_amount": 171.00
    },
    {
      "medicine_id": "{{medicine_id_2}}",
      "medicine_inventory_id": "{{medicine_inventory_id_2}}",
      "prescription_item_id": "{{prescription_item_id_2}}",
      "batch_no": "{{batch_no_2}}",
      "current_stock_units": 50,
      "quantity_sold_units": 4,
      "unit_price_charged": 60.00,
      "computed_item_total": 240.00,
      "total_amount": 240.00
    }
  ]
}
```

**Expected:** `200` — invoice created with 2 `invoice_items` rows; single Razorpay link for combined amount

**Postman Tests**

```javascript
pm.test("Status is 200", () => pm.response.to.have.status(200));
pm.test("Payment link returned", () => {
  pm.expect(pm.response.json().payment_link).to.be.a("string").and.not.empty;
});
```

---

#### CK-B3 — Dispense exact remaining qty (boundary)

**Setup:** Set `quantity_sold_units` = `prescribed_qty - balance_after_dispense` (exact remaining)

Example: `prescribed_qty = 30`, `balance_after_dispense = 24` → `quantity_sold_units = 6`

**Expected:** `200` — passes validation exactly at boundary

---

### CK-C. Missing required fields

For each test: start from base valid payload and **remove one field**. All return `409`.

| ID | Remove | Expected error |
|---|---|---|
| CK-C1 | `prescription_id` | `required payload is missing` |
| CK-C2 | `patient_id` | `required payload is missing` |
| CK-C3 | `cashier_id` | `required payload is missing` |
| CK-C4 | `organisation_id` | `required payload is missing` |
| CK-C5 | `supplier_id` | `required payload is missing` |
| CK-C6 | `financials.total_amount` | `required payload is missing` |
| CK-C7 | `dispense_items` | `required payload is missing` |
| CK-C8 | Entire body `{}` | `required payload is missing` |

**Postman Tests (reusable for CK-C1 → CK-C8)**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Error present", () => {
  pm.expect(pm.response.json().error).to.be.a("string").and.not.empty;
});
```

---

### CK-D. Business validation

#### CK-D1 — Inventory stock exceeded

**Setup:** Set `quantity_sold_units` **greater than** `current_stock_units`

```json
"current_stock_units": 5,
"quantity_sold_units": 10
```

**Expected:** `409`

```json
{
  "error": "insufficient stock in batch BAT-4029: requested 10, available 5"
}
```

**Postman Tests**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Insufficient stock error", () => {
  pm.expect(pm.response.json().error).to.include("insufficient stock in batch");
});
```

---

#### CK-D2 — Prescription qty exceeded

**Setup:** Set `quantity_sold_units` > `prescribed_qty - balance_after_dispense`

Example: `prescribed_qty = 10`, `balance_after_dispense = 8` → try `quantity_sold_units = 5` (remaining = 2, exceed by 3)

```json
"quantity_sold_units": 5,
"current_stock_units": 100
```

**Expected:** `409`

```json
{
  "error": "dispensed qty 5 exceeds remaining prescribed qty 2 for medicine <uuid>"
}
```

**Postman Tests**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Prescription qty exceeded error", () => {
  pm.expect(pm.response.json().error).to.include("exceeds remaining prescribed qty");
});
```

---

#### CK-D3 — Medicine not found in prescription

**Setup:** Use a `medicine_id` that exists in the DB but is **not** in the given `prescription_id`

**Expected:** `409`

```json
{
  "error": "medicine <uuid> not found in prescription"
}
```

**Postman Tests**

```javascript
pm.test("Status is 409", () => pm.response.to.have.status(409));
pm.test("Medicine not in prescription", () => {
  pm.expect(pm.response.json().error).to.include("not found in prescription");
});
```

---

#### CK-D4 — Invalid patient_id

**Setup:** `"patient_id": "00000000-0000-0000-0000-000000000000"`

**Expected:** `409` — `PatientServ.FindOne` returns not found after invoice + items are created (invoice TX committed, but payment link creation fails)

> **Note:** This exposes a potential partial write (invoice rows exist but no payment link). Worth tracking as a future improvement.

---

#### CK-D5 — Invalid prescription_id

**Setup:** `"prescription_id": "00000000-0000-0000-0000-000000000000"`

**Expected:** `409` — `GetqtyByMedicine` returns empty map; first item fails with "not found in prescription"

---

### CK-E. Partial payment scenario (Razorpay)

#### CK-E1 — Checkout succeeds; Razorpay accepts partial

Checkout itself does not check `accept_partial` — the payment link is always created.

**Verify:**
1. Complete CK-B1 — get a payment link
2. In Razorpay test dashboard, pay only **half** the amount
3. Razorpay fires `payment_link.paid` webhook with `accept_partial: true` and `amount_paid < amount`
4. Verify `payment_attempts.payment_status = "partially_paid"` in DB
5. Verify `invoices.status` stays `"unpaid"` (no inventory was deducted)

**Expected after webhook:** DB shows `partially_paid`; no stock change; prescription unchanged

---

## 4. What gets written to DB on success

| Table | Operation | Notes |
|---|---|---|
| `invoices` | INSERT | `status = "unpaid"` |
| `invoice_items` | INSERT | One row per item in `dispense_items` |
| `payments` | INSERT | Links to invoice; holds Razorpay link amount |
| `payment_attempts` | INSERT | `payment_status = "pending"` |

> Inventory, prescription items, and stock movements are **not** touched at checkout. They are only updated after the webhook confirms full payment.

---

## 5. What the webhook does after payment

| Event | DB changes |
|---|---|
| `payment_link.paid` (full) | Deducts stock per item, increments `balance_after_dispense`, sets item/prescription status, marks invoice `paid` |
| `payment_link.paid` (partial) | Sets attempt to `partially_paid`; no inventory changes |
| `payment_link.cancelled` | Sets attempt to `cancelled`; invoice stays `unpaid` |
| `payment_link.expired` | Sets attempt to `expired`; invoice stays `unpaid` |

See [checkout-webhook-flow.md](./checkout-webhook-flow.md) for the full step-by-step breakdown.

---

## 6. Recommended Postman collection order

1. Login (prerequisite — get `access_token`)
2. CK-B1 → CK-B3 (happy path)
3. CK-C1 → CK-C8 (missing fields)
4. CK-D1 → CK-D5 (business validation)
5. CK-E1 (partial payment — requires Razorpay test mode)

---

## 7. Common headers

```
Authorization: Bearer {{access_token}}
Content-Type: application/json
Accept: application/json
```

---

## 8. Related endpoints

| Method | URL | Purpose |
|---|---|---|
| GET | `/api/v1/prescriptions/findMany?organisation_id={{organisation_id}}` | List prescriptions to find a valid `prescription_id` |
| GET | `/api/v1/medicine/searchMedicine?name={{name}}` | Find `medicine_id` and batch info |
| POST | `/api/v1/payments/webhook` | Razorpay posts here after patient pays |

See [checkout-webhook-flow.md](./checkout-webhook-flow.md) for end-to-end architecture.

