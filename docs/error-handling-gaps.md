# Error Handling Gaps

Inventory of places that **do not** yet follow the project standard:

- Domain sentinels from `shared/error` (`wrapError.Err…`)
- Correct HTTP status via `wrapError.Wrap` **and** `return`
- Structured zap logging on failure paths
- No raw GORM/driver/`fmt.Errorf` messages exposed to HTTP

Standard reference: `docs/coding-standards.md` §4–5.

Last audited: 2026-08-04.

---

## Summary

| Domain | Status | Domain sentinels? |
|---|---|---|
| Payments (manual confirm / fulfillment) | Mostly done | Yes |
| Billing, prescription, appointments, org, license, patient, auth, org-schedule | Mostly done | Yes |
| **Employee** | Not used | **None** |
| **Bedmanagement** | Not used | **None** |
| **Department** | Not used | **None** |
| **Roles / permissions / rolepermissions** | Not used | **None** |
| Payments webhook / attempts (internals) | Partial | Defined but underused |
| Medicine stubs | Bypassed | N/A (always 200) |
| JWT helpers | Partial (remapped by auth) | Inline `errors.New` |

---

## P0 — Fix first

### 1. Employee — nil err into `Wrap` (panic risk)

`UpdateUser` compares passwords but wraps the **old** `err` (still `nil` after successful `params.New`):

```159:160:internal/employee/controllers.go
	if AdminReq.Password != confirmPassword {
		return wrapError.Wrap(err, c, 409)
```

`Wrap` calls `err.Error()` → **panic** when passwords mismatch.

### 2. Bedmanagement — `Wrap` without `return`

Response is written, then execution continues (may double-write or succeed after failure):

| File | Lines (examples) |
|---|---|
| `internal/bedmanagement/controllers/beds.controllers.go` | 37, 42, 47, 52, 57, 73, 82 |
| `internal/bedmanagement/controllers/rooms.controllers.go` | 31, 36, 41, 46, 51, 56, 61, 66 |
| `internal/bedmanagement/controllers/roomtype.controllers.go` | 35, 40, 50, 55, 60, 78 |

Pattern:

```go
errwrap.Wrap(err, c, 409) // missing return
```

### 3. Department / roles / permissions — raw `return err`

Unmapped parser/DB errors go straight to Fiber:

| File | Lines |
|---|---|
| `internal/department/controllers.go` | 23, 31 |
| `internal/department/services.go` | 25 |
| `internal/roles/controllers.go` | 13, 21 |
| `internal/permissions/controllers.go` | 8 |
| `internal/permissions/services.go` | 31 |

---

## P1 — Domain not wired to `shared/error`

### Employee (whole module)

| Issue | Where |
|---|---|
| No employee sentinels in `shared/error` | — |
| Every failure → HTTP **409** + raw message | `internal/employee/controllers.go` (many `Wrap(err, c, 409)`) |
| Inline `errors.New` | `internal/employee/services.go:70`, `:215` |
| Raw repo `return err` | `internal/employee/repository.go:25` |
| No zap on failure paths | controllers / services |

### Bedmanagement (whole module)

| Issue | Where |
|---|---|
| No bed/room/allotment sentinels | — |
| Inline `errors.New` for validation | `beds.controllers.go:56`, `133–139`; `rooms.controllers.go:89–95`; `roomtype.controllers.go:45`, `72`, `86` |
| Repo inline / raw errors | `roomtype.repository.go:42` (`"ward already exits"`); other repos `return err` |
| Services propagate raw err | `beds.services.go`, `rooms.services.go`, `bedallotment.services.go`, `roomsummary.services.go` |
| Blanket 409 | Most controllers |
| No zap | Controllers / services |

### Role permissions

| Issue | Where |
|---|---|
| Inline `errors.New` | `internal/rolepermissions/repository.go:24` |
| Raw propagate | `internal/rolepermissions/services.go:29` |

---

## P2 — Partial (sentinels exist or callers remap, still inconsistent)

### Payments webhook / attempts / create helpers

| Gap | Where |
|---|---|
| `ErrWebhookProcessFailed` defined but webhook HTTP uses ad-hoc JSON only | `internal/payments/controllers.go:33–41` |
| Raw `return err` in attempts | `payment-attempts.services.go:38`, `48`, `69`, `79`, `164`, `178` |
| Raw repo errors | `payment-attempts.repository.go`, `webhook_repository.go:16`, `payments.repository.go` |
| `fmt.Errorf` validation (can leak to callers) | `payments.services.go:176`, `212`, `608` |
| Provider inline errors | `providers/razorpay/client.go:46`, `gateway.go:100`, `106`; `providers/factory.go:26` |
| Notify path returns raw patient err | `fulfillment_service.go:167` |

Manual confirm + most of fulfillment **do** use wrapError correctly.

### Medicine stubs

| Gap | Where |
|---|---|
| Always HTTP 200 + fake `"medicine"` payload (no real errors) | `common.controllers.go:31–37` (`GetByIDHandler`), `:40–61` (`GetAllHandler`) |

Search / purchase / supplier paths use wrapError.

### Auth logout / JSON write

| Gap | Where |
|---|---|
| Logout discards service error (no log) | `authentication/controllers.go:110` (`_ = …Logout`) |
| Login/refresh JSON write failures `return err` (Fiber) | `controllers.go:68`, `100` |
| Local `errors.New` (usually remapped before HTTP) | `services.go:155`, `163`; `repository.go:10` |

### JWT package

Inline `errors.New` / `fmt.Errorf` for token/key failures — OK if auth always remaps; not HTTP-safe if called directly:

- `internal/jwt/services.go:178`, `189`, `285–341`, `302–317`

---

## Missing sentinels in `shared/error`

Add when wiring these modules (none exist today):

```text
Employee:     ErrEmployeeNotFound, ErrEmployeeCreateFailed, ErrEmployeeUpdateFailed,
              ErrEmployeeFetchFailed, ErrPasswordMismatch, …
Beds:         ErrBedNotFound, ErrBedCreateFailed, ErrRoomNotFound, ErrRoomTypeNotFound,
              ErrBedAllotmentFailed, ErrWardAlreadyExists, …
Department:   ErrDepartmentNotFound, ErrDepartmentCreateFailed, …
Roles:        ErrRoleNotFound, ErrRoleCreateFailed, …
Permissions:  ErrPermissionNotFound, ErrPermissionCreateFailed, …
```

---

## Checklist (per gap area)

Use when closing an item:

```text
[ ] Domain Err* added to shared/error
[ ] Service maps gorm.ErrRecordNotFound → NotFound; else → *Failed
[ ] Controller: errors.Is switch + correct status (400/401/404/409/500)
[ ] Every Wrap(...) is returned
[ ] Zap Warn/Error with ids + reason on failure
[ ] No fmt.Errorf / errors.New reaching HTTP
[ ] No discarded errors (_ =) on critical paths without comment + log
```

---

## Suggested fix order

1. Employee `UpdateUser` nil-`err` panic  
2. Bedmanagement missing `return` after `Wrap`  
3. Department / roles / permissions controller mapping  
4. Employee + bedmanagement full wrapError + zap pass  
5. Payments webhook/attempts remapping + use `ErrWebhookProcessFailed`  
6. Medicine real GetByID/GetAll (replace stubs)  
7. Auth logout logging; JWT/auth inline cleanup  

---

## What *is* covered (for contrast)

These areas generally map to `shared/error` and log on failure:

- Payments: `ConfirmManualPayment`, `FulfillPaidInvoice` (stock/dispense/invoice paths)
- Billing checkout / invoice services  
- Prescription create/update/items (controller switches)  
- Appointments, organisation, license, patient, org-schedule  
- Auth login / refresh (client vs internal classification)  
- Medicine search / purchase / supplier (non-stub handlers)  
