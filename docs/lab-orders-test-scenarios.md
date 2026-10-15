# Lab Orders — Test Scenarios

Manual QA for lab-order create → pay → complete flow, plus RBAC.


| API                     | Method  | Path                                                 | Permission                      |
| ----------------------- | ------- | ---------------------------------------------------- | ------------------------------- |
| Create lab order        | `POST`  | `/api/v1/lab-orders/create`                          | Lab create                      |
| List lab orders         | `GET`   | `/api/v1/lab-orders/list`                            | Lab view                        |
| Get lab order           | `GET`   | `/api/v1/lab-orders/getbyID/:id`                     | Lab view                        |
| Update order status     | `PATCH` | `/api/v1/lab-orders/update/:id/status`               | Lab update                      |
| Bulk update test status | `PATCH` | `/api/v1/lab-orders/bulkupdatestatus`                | Lab update                      |
| Update one test status  | `PATCH` | `/api/v1/lab-orders/update/:id/tests/:testId/status` | Lab update                      |
| Check billing exists    | `GET`   | `/api/v1/billing/checkByBillingID/:billingID`        | Public (any authenticated role) |
| Checkout                | `POST`  | `/api/v1/billing/create`                             | Billing create                  |
| Confirm payment         | `POST`  | `/api/v1/payment/confirm`                            | Billing update                  |


**Roles**


| Role           | Lab                       | Billing                |
| -------------- | ------------------------- | ---------------------- |
| Doctor         | create + view (no update) | —                      |
| Lab Technician | view + update (no create) | view + create + update |


**Order status flow**

`pending` → (payment) `paid` + `is_payment=true` → (all tests done) `completed`

---



## Prerequisites


| Variable               | Notes                        |
| ---------------------- | ---------------------------- |
| `BASE_URL`             | e.g. `http://localhost:8080` |
| `DOCTOR_JWT`           | Doctor login token           |
| `LAB_TECH_JWT`         | Lab Technician login token   |
| `ORG_ID`               | Organisation UUID            |
| `PATIENT_ID`           | Patient UUID                 |
| `DOCTOR_ID`            | Doctor user UUID             |
| `DIAGNOSTIC_CENTRE_ID` | Diagnostic centre UUID       |
| `LAB_TECH_USER_ID`     | Cashier / lab tech user UUID |


---



## Scenario A — Happy path (create → pay → complete tests)



### A.1 Doctor creates lab order

```bash
curl -X POST "$BASE_URL/api/v1/lab-orders/create" \
  -H "Authorization: Bearer $DOCTOR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "organisation_id": "'"$ORG_ID"'",
    "patient_id": "'"$PATIENT_ID"'",
    "doctor_id": "'"$DOCTOR_ID"'",
    "diagnostic_centre_id": "'"$DIAGNOSTIC_CENTRE_ID"'",
    "tests": [
      {"name": "CBC", "sub_test": ""},
      {"name": "FBS", "sub_test": ""}
    ]
  }'
```

**Expect:** `200`, `lab_order_id`. Save as `LO_ID`.

### A.2 Doctor views lab order

```bash
curl -X GET "$BASE_URL/api/v1/lab-orders/getbyID/$LO_ID" \
  -H "Authorization: Bearer $DOCTOR_JWT"
```

**Expect:** `status=pending`, `is_payment=false`, two tests. Save test ids as `T1`, `T2`.

### A.3 Check not billed yet

```bash
curl -X GET "$BASE_URL/api/v1/billing/checkByBillingID/$LO_ID" \
  -H "Authorization: Bearer $LAB_TECH_JWT"
```

**Expect:** `{ "code": 200, "count": 0 }` → enable Pay button.

### A.4 Lab tech checkout (cash)

```bash
curl -X POST "$BASE_URL/api/v1/billing/create" \
  -H "Authorization: Bearer $LAB_TECH_JWT" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: lab-checkout-1" \
  -d '{
    "payment_type": "lab_order",
    "billing_id": "'"$LO_ID"'",
    "patient_id": "'"$PATIENT_ID"'",
    "cashier_id": "'"$LAB_TECH_USER_ID"'",
    "organisation_id": "'"$ORG_ID"'",
    "payment_mode": "cash",
    "financials": {
      "sub_total_amount": 500,
      "tax_amount": 0,
      "discount_amount": 0,
      "total_amount": 500
    }
  }'
```

**Expect:** `200`, `invoice_id`. Save as `INVOICE_ID`.

> Cash/QR checkout sets `is_payment=true` and `status=paid` when payment mode is cash/qr. Link payments are marked on confirm/webhook.



### A.5 Confirm payment (cash/qr)

```bash
curl -X POST "$BASE_URL/api/v1/payment/confirm" \
  -H "Authorization: Bearer $LAB_TECH_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_id": "'"$INVOICE_ID"'",
    "organisation_id": "'"$ORG_ID"'",
    "payment_mode": "cash"
  }'
```

**Expect:** `200`. Invoice paid; lab order payment flag remains true / status `paid` if still pending.

### A.6 Re-check lab order + billing

```bash
curl -X GET "$BASE_URL/api/v1/lab-orders/getbyID/$LO_ID" \
  -H "Authorization: Bearer $LAB_TECH_JWT"

curl -X GET "$BASE_URL/api/v1/billing/checkByBillingID/$LO_ID" \
  -H "Authorization: Bearer $LAB_TECH_JWT"
```

**Expect:**

- Lab order: `is_payment=true`, `status=paid`
- Billing check: `"count": 1` → disable Pay button



### A.7 Lab tech completes tests (bulk)

```bash
curl -X PATCH "$BASE_URL/api/v1/lab-orders/bulkupdatestatus" \
  -H "Authorization: Bearer $LAB_TECH_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "lab_order_id": "'"$LO_ID"'",
    "test_ids": ["'"$T1"'", "'"$T2"'"],
    "status": "completed"
  }'
```

**Expect:** `200`. GetbyID: both tests `completed`, order `status=completed`.

### A.8 List filter by completed

```bash
curl -X GET "$BASE_URL/api/v1/lab-orders/list?organisation_id=$ORG_ID&status=completed&limit=10&page_no=1" \
  -H "Authorization: Bearer $DOCTOR_JWT"
```

**Expect:** order appears in list.

---



## Scenario B — RBAC


| #   | Who      | Action                         | Expect               |
| --- | -------- | ------------------------------ | -------------------- |
| B.1 | Doctor   | create / list / get            | `200`                |
| B.2 | Doctor   | `bulkupdatestatus`             | `403`                |
| B.3 | Doctor   | `update/:id/status`            | `403`                |
| B.4 | Lab tech | create lab order               | `403`                |
| B.5 | Lab tech | bulk status + billing checkout | `200`                |
| B.6 | Any role | `checkByBillingID`             | `200` (JWT required) |




### B.2 Doctor blocked from status update

```bash
curl -X PATCH "$BASE_URL/api/v1/lab-orders/bulkupdatestatus" \
  -H "Authorization: Bearer $DOCTOR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "lab_order_id": "'"$LO_ID"'",
    "test_ids": ["'"$T1"'"],
    "status": "completed"
  }'
```

**Expect:** `403 Forbidden`.

### B.4 Lab tech blocked from create

```bash
curl -X POST "$BASE_URL/api/v1/lab-orders/create" \
  -H "Authorization: Bearer $LAB_TECH_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "organisation_id": "'"$ORG_ID"'",
    "patient_id": "'"$PATIENT_ID"'",
    "doctor_id": "'"$DOCTOR_ID"'",
    "diagnostic_centre_id": "'"$DIAGNOSTIC_CENTRE_ID"'",
    "tests": [{"name": "CBC", "sub_test": ""}]
  }'
```

**Expect:** `403 Forbidden`.

---



## Scenario C — Negative cases


| #   | Step                               | Expect               |
| --- | ---------------------------------- | -------------------- |
| C.1 | Checkout same `LO_ID` again        | already billed error |
| C.2 | Bulk status with unknown `test_id` | not found            |
| C.3 | Bulk status missing `test_ids`     | `400`                |
| C.4 | Create without tests               | `400`                |
| C.5 | List without `organisation_id`     | `400`                |




### C.1 Double checkout

```bash
curl -X POST "$BASE_URL/api/v1/billing/create" \
  -H "Authorization: Bearer $LAB_TECH_JWT" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: lab-checkout-2" \
  -d '{
    "payment_type": "lab_order",
    "billing_id": "'"$LO_ID"'",
    "patient_id": "'"$PATIENT_ID"'",
    "cashier_id": "'"$LAB_TECH_USER_ID"'",
    "organisation_id": "'"$ORG_ID"'",
    "payment_mode": "cash",
    "financials": {
      "sub_total_amount": 500,
      "tax_amount": 0,
      "discount_amount": 0,
      "total_amount": 500
    }
  }'
```

**Expect:** lab order already billed error.

### C.2 Unknown test id

```bash
curl -X PATCH "$BASE_URL/api/v1/lab-orders/bulkupdatestatus" \
  -H "Authorization: Bearer $LAB_TECH_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "lab_order_id": "'"$LO_ID"'",
    "test_ids": ["00000000-0000-0000-0000-000000000000"],
    "status": "completed"
  }'
```

**Expect:** not found / update failed.

---



## Sign-off checklist

- [x] Doctor can create and view lab orders
- [x] Doctor cannot update test/order status
- [x] Lab tech cannot create lab orders
- [x] Lab tech can checkout + confirm payment
- [x] After payment: `is_payment=true`, `status=paid`
- [x] `checkByBillingID` returns `0` then `1`
- [x] Bulk complete sets tests + order to `completed`
- [x] List filter by `status=completed` works
- [x] Double checkout blocked