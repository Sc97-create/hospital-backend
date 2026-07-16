# Postman Test Scenarios — Suppliers API

Quick reference for manual and automated testing of supplier endpoints under `/api/v1/supplier`.

## Summary Checklist

| ID | Scenario | Endpoint | Expected Status |
|---|---|---|---|
| S1 | Create supplier — valid payload | `POST /createSupplier` | 200 |
| S2 | Create supplier — missing required field | `POST /createSupplier` | 400 |
| S3 | Create supplier — empty body `{}` | `POST /createSupplier` | 400 |
| S4 | Create supplier — optional `gst_number` omitted | `POST /createSupplier` | 200 |
| L1 | List by org — valid request | `GET /getSupplierByOrgID` | 200 |
| L2 | List by org — missing `organisation_id` | `GET /getSupplierByOrgID` | 400 |
| L3 | List by org — page 2 / custom limit | `GET /getSupplierByOrgID` | 200 |
| L4 | List by org — empty result | `GET /getSupplierByOrgID` | 200 |
| L5 | List response excludes sensitive fields | `GET /getSupplierByOrgID` | 200 |
| C1 | Total count — valid | `GET /getTotalCount` | 200 |
| C2 | Total count — missing `organisation_id` | `GET /getTotalCount` | 400 |
| C3 | Total count matches list `total` | `GET /getTotalCount` + list | 200 |
| G1 | Get by ID — valid `supplier_id` | `GET /getSupplierByID` | 200 |
| G2 | Get by ID — unknown `supplier_id` | `GET /getSupplierByID` | 409 |
| G3 | Get by ID — missing `supplier_id` | `GET /getSupplierByID` | 409 |
| F1 | Flow: create → count++ → list contains supplier → getByID | mixed | 200 |

---

## 1. Setup

### Environment variables

| Variable | Example | Description |
|---|---|---|
| `base_url` | `http://localhost:9069` | Server URL |
| `organisation_id` | `11111111-2222-3333-4444-555555555555` | Valid organisation UUID |
| `user_id` | `ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb` | Valid user UUID (creator) |
| `supplier_id` | *(set after S1 / L1)* | Created supplier UUID |
| `empty_org_id` | `00000000-0000-0000-0000-000000000099` | Org with no suppliers (for L4) |

> **Auth note:** Supplier routes are currently **not** wrapped with JWT middleware. Add `Authorization: Bearer {{access_token}}` later if auth is enabled on these routes.

### Base URL prefix

```
{{base_url}}/api/v1/supplier
```

---

## 2. APIs Overview

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/createSupplier` | Create a supplier |
| `GET` | `/getSupplierByOrgID` | Paginated supplier list for an org |
| `GET` | `/getTotalCount` | Total supplier count for an org |
| `GET` | `/getSupplierByID` | Minimal supplier lookup by ID |

---

## 3. Create Supplier

### S1 — Valid complete payload

**Request**

```
POST {{base_url}}/api/v1/supplier/createSupplier
Content-Type: application/json
```

**Body**

```json
{
  "user_id": "{{user_id}}",
  "organisation_id": "{{organisation_id}}",
  "name": "MedLife Distributors",
  "payment_terms": "Net 30",
  "email_id": "orders@medlife.example",
  "drug_license_number": "DL-KA-2026-001",
  "contact_number": "9876543210",
  "credit_limit": 250000,
  "gst_number": "29ABCDE1234F1Z5"
}
```

**Required fields:** `user_id`, `organisation_id`, `name`, `payment_terms`, `email_id`, `drug_license_number`, `contact_number`, `credit_limit`  
**Optional:** `gst_number`

**Payment terms accepted values**

| Input | Stored as |
|---|---|
| `Cash` | Cash |
| `Net 15` | Net 15 |
| `Net 30` | Net 30 |
| `Net 45` | Net 45 |
| anything else | Advance |

**Expected response (200)**

```json
{
  "code": 200,
  "message": "supplier created successfully"
}
```

**Postman Tests**

```javascript
pm.test("Create supplier success", () => pm.response.to.have.status(200));
const json = pm.response.json();
pm.test("Has success message", () => {
  pm.expect(json.code).to.eql(200);
  pm.expect(json.message).to.include("supplier created");
});
```

### S2 — Missing required field

Repeat for each required key: remove one field from the S1 body.

**Expected:** `400`

Example (missing `name`):

```json
{
  "user_id": "{{user_id}}",
  "organisation_id": "{{organisation_id}}",
  "payment_terms": "Net 30",
  "email_id": "orders@medlife.example",
  "drug_license_number": "DL-KA-2026-001",
  "contact_number": "9876543210",
  "credit_limit": 250000
}
```

### S3 — Empty body

```
POST {{base_url}}/api/v1/supplier/createSupplier
Content-Type: application/json

{}
```

**Expected:** `400`

### S4 — Omit optional `gst_number`

Same as S1 without `gst_number`.

**Expected:** `200`

---

## 4. List Suppliers by Organisation

### L1 — Valid request (defaults)

**Request**

```
GET {{base_url}}/api/v1/supplier/getSupplierByOrgID?organisation_id={{organisation_id}}
```

Defaults: `limit=10`, `page_no=1`

**Expected response (200)**

```json
{
  "code": 200,
  "total": 2,
  "data": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "supplier_code": "SUPP-4821",
      "name": "MedLife Distributors",
      "contact_number": "9876543210",
      "email": "orders@medlife.example",
      "payment_terms": "Net 30",
      "supplier_status": "Active",
      "created_at": "14 Jul 2026"
    },
    {
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "supplier_code": "SUPP-1094",
      "name": "CarePlus Pharma",
      "contact_number": "9123456780",
      "email": "sales@careplus.example",
      "payment_terms": "Cash",
      "supplier_status": "Active",
      "created_at": "10 Jul 2026"
    }
  ]
}
```

**Postman Tests**

```javascript
pm.test("List success", () => pm.response.to.have.status(200));
const json = pm.response.json();
pm.test("Has data array and total", () => {
  pm.expect(json.code).to.eql(200);
  pm.expect(json.data).to.be.an("array");
  pm.expect(json.total).to.be.a("number");
});
if (json.data.length > 0) {
  pm.environment.set("supplier_id", json.data[0].id);
}
```

### L2 — Missing `organisation_id`

```
GET {{base_url}}/api/v1/supplier/getSupplierByOrgID
```

**Expected:** `400`

### L3 — Custom pagination

```
GET {{base_url}}/api/v1/supplier/getSupplierByOrgID?organisation_id={{organisation_id}}&limit=5&page_no=2
```

**Expected:** `200`  
Assert: `data.length <= 5`

```javascript
pm.test("Respects limit", () => {
  const json = pm.response.json();
  pm.expect(json.data.length).to.be.at.most(5);
});
```

### L4 — Empty org

```
GET {{base_url}}/api/v1/supplier/getSupplierByOrgID?organisation_id={{empty_org_id}}&limit=10&page_no=1
```

**Expected response (200)**

```json
{
  "code": 200,
  "total": 0,
  "data": []
}
```

### L5 — Sensitive fields excluded

After L1, assert list items do **not** contain:

- `credit_limit`
- `gst_number`
- `drug_license_no`
- `address`
- `organisation_id`
- `created_by`
- `updated_at`

```javascript
pm.test("No sensitive fields in list items", () => {
  const sensitive = [
    "credit_limit",
    "gst_number",
    "drug_license_no",
    "address",
    "organisation_id",
    "created_by",
    "updated_at"
  ];
  const items = pm.response.json().data;
  items.forEach((item) => {
    sensitive.forEach((key) => pm.expect(item).to.not.have.property(key));
  });
});
```

**Allowed list fields:** `id`, `supplier_code`, `name`, `contact_number`, `email`, `payment_terms`, `supplier_status`, `created_at`

---

## 5. Get Total Count

### C1 — Valid

**Request**

```
GET {{base_url}}/api/v1/supplier/getTotalCount?organisation_id={{organisation_id}}
```

**Expected response (200)**

```json
{
  "code": 200,
  "total": 12
}
```

**Postman Tests**

```javascript
pm.test("Count success", () => pm.response.to.have.status(200));
const json = pm.response.json();
pm.test("Has total", () => {
  pm.expect(json.code).to.eql(200);
  pm.expect(json.total).to.be.a("number");
  pm.expect(json.total).to.be.at.least(0);
});
pm.environment.set("supplier_total", String(json.total));
```

### C2 — Missing `organisation_id`

```
GET {{base_url}}/api/v1/supplier/getTotalCount
```

**Expected:** `400`

### C3 — Count matches list total

1. Call `getTotalCount`
2. Call `getSupplierByOrgID` with same `organisation_id`
3. Assert both `total` values are equal

```javascript
// Run after storing supplier_total from C1
pm.test("List total matches getTotalCount", () => {
  const listTotal = pm.response.json().total;
  const countTotal = Number(pm.environment.get("supplier_total"));
  pm.expect(listTotal).to.eql(countTotal);
});
```

---

## 6. Get Supplier by ID

### G1 — Valid ID

**Request**

```
GET {{base_url}}/api/v1/supplier/getSupplierByID?supplier_id={{supplier_id}}
```

**Expected response (200)**

```json
{
  "code": 200,
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "name": "MedLife Distributors",
    "payment_terms": "Net 30"
  }
}
```

> This endpoint currently returns only `id`, `name`, `payment_terms`.

```javascript
pm.test("Get by ID success", () => pm.response.to.have.status(200));
const json = pm.response.json();
pm.test("Has supplier data", () => {
  pm.expect(json.data.id).to.be.ok;
  pm.expect(json.data.name).to.be.a("string");
  pm.expect(json.data.payment_terms).to.be.a("string");
});
```

### G2 — Unknown ID

```
GET {{base_url}}/api/v1/supplier/getSupplierByID?supplier_id=00000000-0000-0000-0000-000000000001
```

**Expected:** `409`

### G3 — Missing `supplier_id`

```
GET {{base_url}}/api/v1/supplier/getSupplierByID
```

**Expected:** `409`

---

## 7. End-to-end Flow (F1)

Run in order:

| Step | Action | Assert |
|---|---|---|
| 1 | `GET getTotalCount` | Store `total_before` |
| 2 | `POST createSupplier` | `200` |
| 3 | `GET getTotalCount` | `total == total_before + 1` |
| 4 | `GET getSupplierByOrgID` | New name appears in `data` |
| 5 | Save `data[0].id` → `supplier_id` | — |
| 6 | `GET getSupplierByID` | Name / payment_terms match create payload |

**Suggested create body for flow** (unique email/contact each run):

```json
{
  "user_id": "{{user_id}}",
  "organisation_id": "{{organisation_id}}",
  "name": "Postman Flow Supplier",
  "payment_terms": "Cash",
  "email_id": "flow.supplier.{{$timestamp}}@example.com",
  "drug_license_number": "DL-TEST-{{$timestamp}}",
  "contact_number": "9{{$randomInt}}",
  "credit_limit": 10000
}
```

---

## 8. Suggested Postman Collection Order

1. **S1** Create supplier  
2. **C1** Get total count  
3. **L1** List by org (save `supplier_id`)  
4. **L5** Assert no sensitive fields  
5. **C3** Compare totals  
6. **G1** Get by ID  
7. **L2 / C2 / S2 / G2** Negative cases  

---

## 9. Quick cURL Reference

```bash
# Create
curl -X POST "$BASE_URL/api/v1/supplier/createSupplier" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id":"'"$USER_ID"'",
    "organisation_id":"'"$ORG_ID"'",
    "name":"MedLife Distributors",
    "payment_terms":"Net 30",
    "email_id":"orders@medlife.example",
    "drug_license_number":"DL-KA-2026-001",
    "contact_number":"9876543210",
    "credit_limit":250000,
    "gst_number":"29ABCDE1234F1Z5"
  }'

# List
curl "$BASE_URL/api/v1/supplier/getSupplierByOrgID?organisation_id=$ORG_ID&limit=10&page_no=1"

# Total count
curl "$BASE_URL/api/v1/supplier/getTotalCount?organisation_id=$ORG_ID"

# Get by ID
curl "$BASE_URL/api/v1/supplier/getSupplierByID?supplier_id=$SUPPLIER_ID"
```
