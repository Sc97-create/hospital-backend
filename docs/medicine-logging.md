# Medicine Logging Plan

Implemented. Builds on the existing logging stack (`pkg/logger`, `RequestLogger`, `middleware.GetLogger`) and the same patterns used for other domains.

## Layer logging

| Layer | What we log |
|-------|-------------|
| **Controller** | Entry / invalid request; HTTP status mapping; pass `*zap.Logger` |
| **Service** | Business outcomes (`attempt` / `success` / `failed` + stable `reason`) |
| **Repo** | **DB errors only** (`op` + `error`) — no success INFO spam |

Package: `internal/medicine`  
Routes:
- `/api/v1/medicine/*` — `RegisterMedicineRoutes`
- `/api/v1/supplier/*` — `RegisterSupplierRoutes` (same package; included this pass)

---

## Layer logging (same as other domains)

| Layer | What we log |
|-------|-------------|
| **Controller** | Entry / invalid request; HTTP status mapping; pass `*zap.Logger` |
| **Service** | Business outcomes (`attempt` / `success` / `failed` + stable `reason`) |
| **Repo** | **DB errors only** (`op` + `error`) — no success INFO spam |

Internal callers (payment fulfillment stock decrement / movements): prefer request logger when available; else `logger.Log` / `ensureLog(nil)` via adapter (same as billing/prescription). Prefer **not** expanding `IPaymentFulfillment` with logger this pass — log at medicine repo on DB/insufficient-stock errors when adapter calls through.

---

## 1. Scope

### 1.1 HTTP — medicine

| Area | Endpoint / entry | Files | Status today |
|------|------------------|-------|--------------|
| Add / restock medicine (purchase) | `POST /v1/medicine/addMedicine` | `common.controllers.go`, `medicine.services.go`, purchase/inventory/mvmt | Live — highest value |
| Search by name | `GET /v1/medicine/searchMedicine` | controller + `medicine.repository.go` | Live |
| Get by ID | `GET /v1/medicine/getMedicineByID` | controller | **Stub** (hardcoded `"medicine"`) |
| List medicines | `GET /v1/medicine/GetMedicines` | controller | **Stub** (+ service `GetMany` has infinite recursion if wired) |

### 1.2 HTTP — supplier (same package)

| Area | Endpoint | Files |
|------|----------|-------|
| Create supplier | `POST /v1/supplier/createSupplier` | `supplier.controllers.go`, `supplier.services.go`, repo |
| Get by ID | `GET /v1/supplier/getSupplierByID` | same |
| List by org | `GET /v1/supplier/getSupplierByOrgID` | same |
| Count by org | `GET /v1/supplier/getTotalCount` | same |

### 1.3 Internal (no dedicated medicine HTTP)

| Area | Caller | Files |
|------|--------|-------|
| Atomic stock decrement | Payment fulfillment / webhook | `medicine_inventory.repository.go` via `appinit/payment_fulfillment.go` |
| Dispense stock movements | Same | `medicine_mvmt.services.go` / repo |
| Supplier lookup during add | `CreateMedicine` → `GetSupplierByID` | supplier service/repo |
| `FindNamesByIds` / `GetOne` | Mostly unused / stubs | service + repo |

**Auth note:** medicine + supplier routes are **not** behind JWT today. RequestLogger still applies globally. Product risk (follow-up).

**Out of scope for this pass**

- Wiring / fixing stub `getMedicineByID` + `GetMedicines` beyond logging (product work)
- Fixing `GetMany` infinite recursion (note only unless we take stubs live)
- Full payment fulfillment domain logs (see `docs/payments-logging.md`) — only medicine-side stock/mvmt DB errors
- Prescription medicine-info endpoint (prescription doc owns it)
- JWT on medicine/supplier routes

---

## 2. Security / PII / commercial rules (non-negotiable)

Medicine is inventory + commercial B2B data (not patient PHI), but still sensitive.

**Never log**

- Full request bodies / `medicine_info` arrays
- Medicine **names** on INFO success (search query string is borderline — see decisions)
- **Batch numbers** on INFO
- Pricing: `mrp`, `purchase_price`, `selling_price`, `discount`, `unit_price`, `credit_limit`
- Supplier **email**, **contact_number**, **drug_license_number**, **gst_number**
- Full supplier response dumps
- Shelf location on INFO (optional ops field; omit by default)

**Safe to log**

- `organisation_id`, `user_id` / `created_by`, `supplier_id`
- `medicine_id`, `medicine_inventory_id`, `purchase_entry_id`
- `invoice_no` (ops correlation; not secret)
- Counts: `item_count`, `new_medicine_count`, `inventory_count`, `movement_count`, `result_count`
- Flags: `has_payment_due_date`, `supplier_found`
- Stock op: `medicine_inventory_id`, `dispensed_qty` (qty OK; no prices)
- Stable `reason` enums

**Borderline**

- **Search `name` query** — recommend log `name_len` or omit; never log full free-text if it could be long / noisy
- **`invoice_no`** — recommend allow (ops)
- **Stock units / qty** — allow on ERROR / DEBUG; omit aggregate stock from search success INFO

---

## 3. Propagation approach

```
RequestLogger
  → Controller: middleware.GetLogger(c)
  → MedicineService / SupplierService / inventory+mvmt services: *zap.Logger first arg
  → Repos: DB errors only
```

**Recommendation:** required `log *zap.Logger` first arg. Add `internal/medicine/log.go` with `ensureLog`.

Fulfillment path: keep adapter using `logger.Log` for now; add repo-level ERROR on insufficient stock / batch create fail so ops see it under payment flows.

---

## 4. What to log — by flow

### 4.1 Add medicine / purchase (`POST /medicine/addMedicine`)

Creates (in one tx): purchase entry → medicines (only `Add=true` rows) → inventory batches → purchase stock movements. Looks up supplier first.

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| A1 | Controller | Parse / missing mandatory field | WARN | `medicine purchase request invalid` | `field` (`user_id` / `supplier_id` / `organisation_id` / `invoice_no` / `medicine_info`) |
| A2 | Controller | Entered OK | INFO | `medicine purchase attempt` | `organisation_id`, `supplier_id`, `user_id`, `invoice_no`, `item_count` |
| A3 | Service | Supplier not found | WARN | `medicine purchase failed` | `supplier_id`, `organisation_id`, `reason=supplier_not_found` |
| A4 | Service | Purchase entry fail | ERROR | `medicine purchase failed` | `organisation_id`, `supplier_id`, `reason=purchase_entry`, `error` |
| A5 | Service | Medicines batch create fail | ERROR | `medicine purchase failed` | `organisation_id`, `reason=medicines_create`, `error` |
| A6 | Service | Inventory batch fail | ERROR | `medicine purchase failed` | `organisation_id`, `purchase_entry_id`, `reason=inventory_create`, `error` |
| A7 | Service | Movement batch fail | ERROR | `medicine purchase failed` | `organisation_id`, `reason=stock_mvmt`, `error` |
| A8 | Service | Success | INFO | `medicine purchase success` | `organisation_id`, `supplier_id`, `purchase_entry_id`, `invoice_no`, `item_count`, `new_medicine_count`, `inventory_count`, `movement_count` |

**Mandatory today:** `user_id`, `supplier_id`, `organisation_id`, `invoice_no`, `medicine_info` (array).  
**Optional:** `payment_due_date` (mostly ignored by buggy due-date logic — note only).

**Item-level validation:** controller currently ignores per-field errors (`Getstring` / `Getfloat` with `_`). Optional harden: reject empty name / zero boxes / zero units with WARN `field=...`.

#### 4.1.1 Frontend-facing add errors

| Case | HTTP (proposed) | API `error` |
|------|-----------------|-------------|
| Invalid / missing fields | `400` | `invalid request` |
| Supplier not found | `404` | `supplier not found` |
| DB / tx failure | `500` | `failed to create medicine purchase` |

**Decision:** medicine routes today use **409** + raw `err.Error()`. Recommend mapping as above.

---

### 4.2 Search medicine (`GET /medicine/searchMedicine`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| S1 | Controller | Missing `name` / `organisation_id` | WARN | `medicine search request invalid` | `field` |
| S2 | Controller | Entered OK | INFO | `medicine search attempt` | `organisation_id`, `name_len` (or omit name) |
| S3 | Service/Repo | DB error | ERROR | `medicine search failed` | `organisation_id`, `reason=db_read`, `error` |
| S4 | Service | Success | INFO | `medicine search success` | `organisation_id`, `result_count` |

Do not log medicine names / stock totals from result rows.

#### 4.2.1 Frontend-facing search errors

| Case | HTTP | API `error` |
|------|------|-------------|
| Missing params | `400` | `invalid request` |
| DB | `500` | `failed to search medicine` |

---

### 4.3 Get by ID / list (stubs)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| G1 | Controller | Handler hit | WARN | `medicine get stub called` / `medicine list stub called` | `path` or fixed message |

**Recommendation this pass:** log WARN when stub is invoked so traffic is visible; do **not** pretend success domain logs. Wire real get/list + fix `GetMany` recursion as a **separate product task**.

---

### 4.4 Supplier create (`POST /supplier/createSupplier`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| C1 | Controller | Missing field | WARN | `supplier create request invalid` | `field` |
| C2 | Controller | Entered OK | INFO | `supplier create attempt` | `organisation_id`, `user_id`, `payment_terms` |
| C3 | Service | DB fail | ERROR | `supplier create failed` | `organisation_id`, `reason=db_create`, `error` |
| C4 | Service | Success | INFO | `supplier create success` | `organisation_id`, `supplier_id`, `supplier_code`, `payment_terms` |

Never log name / email / phone / drug license / GST / credit_limit.

Create already uses 400/500 in places — align remaining 409s on get/list.

---

### 4.5 Supplier get / list / count

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| Q1 | Controller | Missing `supplier_id` / `organisation_id` | WARN | `supplier get/list request invalid` | `field` |
| Q2 | Service | Not found | WARN | `supplier get failed` | `supplier_id`, `reason=not_found` |
| Q3 | Service | DB error | ERROR | `supplier get/list failed` | `reason=db_read`, `error` |
| Q4 | Service | Success | DEBUG or INFO | `supplier get success` / `supplier list success` | `supplier_id` **or** `organisation_id`, `result_count`, `limit`, `page_no` |

Recommend **DEBUG** for single get success (high volume possible); **INFO** for create; list success INFO with counts only.

HTTP: `400` invalid, `404` not found, `500` DB.

---

### 4.6 Internal — stock decrement (fulfillment)

Owned outcome remains `payment fulfillment failed` in payments. Medicine repo should still emit DB-level errors:

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| I1 | Repo | Update error | ERROR | `medicine repo error` | `op=UpdateMedInventoryStock`, `error` |
| I2 | Repo | Insufficient stock (0 rows) | WARN or ERROR | `medicine stock decrement failed` | `medicine_inventory_id`, `reason=insufficient_stock` (omit qty on INFO; qty OK on ERROR) |
| I3 | Repo | Movement batch fail | ERROR | `medicine repo error` | `op=CreateMedicineMvmtInBatch`, `error` |

Optional service wrapper logs if we thread logger through `SMedicineInventory` / `SMedicineMvmt`.

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/medicine/common.controllers.go` | Add / search / stub get-list entry + HTTP mapping |
| `internal/medicine/medicine.services.go` | Purchase tx step outcomes; search |
| `internal/medicine/medicine.repository.go` | DB errors; search |
| `internal/medicine/purchase-entries.services.go` (+ repo) | Purchase entry create errors |
| `internal/medicine/medicine_inventory.services.go` (+ repo) | Inventory create; stock decrement |
| `internal/medicine/medicine_mvmt.services.go` (+ repo) | Movement create |
| `internal/medicine/supplier.controllers.go` | Supplier HTTP |
| `internal/medicine/supplier.services.go` (+ repo) | Supplier outcomes |
| `internal/medicine/log.go` | `ensureLog` (new) |
| `shared/error/structures.go` | Optional sentinels (below) |
| `appinit/payment_fulfillment.go` | Keep `logger.Log`; no interface expand this pass |

**Suggested sentinels**

```go
ErrMedicineNotFound
ErrMedicinePurchaseFailed
ErrMedicineSearchFailed
ErrSupplierNotFound
ErrSupplierCreateFailed
ErrSupplierFetchFailed
ErrInsufficientStock   // may already exist from billing — reuse if same semantics
```

Billing already has `ErrInsufficientStock` for checkout validation — reuse for inventory decrement or add `ErrMedicineStockInsufficient` if we want domain clarity (open).

---

## 6. Suggested log field conventions

| Field | When |
|-------|------|
| `request_id` | RequestLogger |
| `organisation_id`, `user_id`, `supplier_id` | Purchase / supplier |
| `purchase_entry_id`, `invoice_no` | Purchase |
| `medicine_id`, `medicine_inventory_id` | Stock ops / errors |
| `item_count`, `new_medicine_count`, `inventory_count`, `movement_count`, `result_count` | Success |
| `payment_terms` | Supplier (enum string only) |
| `reason`, `field`, `op`, `error` | Failures |

Example purchase success:

```json
{
  "level": "info",
  "msg": "medicine purchase success",
  "request_id": "...",
  "organisation_id": "...",
  "supplier_id": "...",
  "purchase_entry_id": "...",
  "invoice_no": "INV-1001",
  "item_count": 3,
  "new_medicine_count": 1,
  "inventory_count": 3,
  "movement_count": 3
}
```

---

## 7. Known quirks (log / fix decisions)

| Quirk | Today | Recommend for this pass |
|-------|-------|-------------------------|
| `getMedicineByID` / `GetMedicines` stub | Always 200 fake data | WARN `*_stub_called`; no fake domain success |
| `GetMany` calls itself | Infinite recursion if wired | Out of scope unless stubs go live |
| Add medicine HTTP always **409** | Raw `err.Error()` | Map 400/404/500 + sentinels |
| Per-item fields ignored (`_`) | Empty name / zero qty can enter | Optional validation WARN harden |
| `PaymentDueDate` logic buggy | Due date mostly forced to `time.Now()` | Note only; don’t log due date |
| `CreateInBatches` with empty new-medicine list | Existing meds only (`Add=false`) | Still log `new_medicine_count=0` on success |
| Supplier get returns raw 409 | Inconsistent with create’s 400/500 | Align 404/500 |
| No JWT | Anyone can mutate inventory | Follow-up |
| Stock decrement string error | `"insufficient stock for medicine inventory %s"` | Map to sentinel + structured log |

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger on medicine/supplier routes | Done (global) |
| Medicine controller domain logs | Done |
| Purchase (`addMedicine`) service/repo logs | Done |
| Search domain logs | Done |
| Stub get/list visibility | Done (WARN) |
| Supplier create/get/list logs | Done |
| Inventory stock decrement / mvmt repo logs | Done |
| Client-safe HTTP mapping (medicine) | Done (400/404/500) |
| Sentinels | Done |
| JWT | Not present (follow-up) |

---

## 9. Implementation order (after approval)

1. `log.go` + sentinels  
2. Purchase path (`addMedicine`) — highest risk / value  
3. Search + HTTP mapping  
4. Supplier CRUD/list (if included this pass)  
5. Inventory decrement + movement repo errors (fulfillment)  
6. Stub WARN logs (optional thin)  
7. Optional: JWT / wire real get-list (separate)

---

## 10. Open decisions for review

1. **Include supplier routes in this medicine logging pass?** — **Recommend: yes** (same package, small surface)
2. **Omit medicine names / prices / batch / supplier PII from INFO?** — **Recommend: yes**
3. **Log search query text?** — **Recommend: no**; use `name_len` or omit
4. **Map medicine HTTP to 400/404/500 instead of 409?** — **Recommend: yes**
5. **Harden per-item validation on add?** — **Recommend: yes for empty name / boxes / units**; deeper pricing rules later
6. **Stub handlers: WARN only this pass?** — **Recommend: yes**
7. **Stock insufficient: WARN vs ERROR?** — **Recommend: ERROR** (blocks payment fulfill)
8. **Expand `IPaymentFulfillment` with logger?** — **Recommend: no this pass** (repo logs + adapter `logger.Log`)
9. **Reuse billing `ErrInsufficientStock`?** — **Recommend: reuse** unless we want medicine-specific sentinel
10. **Logger style:** required `log *zap.Logger` first arg — **yes**
11. **JWT on medicine/supplier this pass?** — **Recommend: follow-up**

---

## 11. What we will **not** add in this pass

- Logging names, prices, batches, supplier contact/license fields
- Fixing stub get/list into real APIs (unless you want that bundled)
- Fixing purchase due-date calculation
- Prescription / billing medicine-info logging (other docs)
- JWT

---

## 12. Approval

Once this list looks right, implement in §9 order. Coordinate with `docs/payments-logging.md` so fulfillment stock failures don’t double-noise (payments = fulfill outcome; medicine = `op` / `insufficient_stock` at inventory repo).
