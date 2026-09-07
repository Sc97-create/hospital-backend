# Testing Overview

This document describes the unit and integration-style tests currently in the hospital-backend codebase: what is covered, how tests are structured, and how to run them.

## Testing strategy

We use two layers of unit tests:

| Layer | What is tested | What is mocked |
|-------|----------------|----------------|
| **Controller tests** | HTTP handlers (Fiber): request parsing, status codes, error mapping | **Services** (`MockXxxServicer` via gomock) |
| **Service tests** | Business logic in service structs | **Repositories** (`MockXxxRepository` via gomock) + stub interfaces for cross-service deps |

```text
Controller  →  MockServicer
Service     →  MockRepository + stub org / schedule / payment / notification
```

Controller tests live in external test packages (`package foo_test`) to avoid import cycles with generated mocks. Service tests for unexported helpers use the same package (`package billing`); exported service methods use `package foo_test` with gomock.

---

## Test infrastructure

### Shared harness — controller tests

**Path:** [`internal/testutil/controllertest/`](../internal/testutil/controllertest/)

| File | Purpose |
|------|---------|
| `harness.go` | `NewApp`, `Do`, `AssertStatus` — Fiber app setup with test logger |
| `gomock.go` | `NewGomockController` helper |

Typical controller test pattern:

```go
ctrl := gomock.NewController(t)
mock := mocks.NewMockPatientServicer(ctrl)
mock.EXPECT().FindOne(gomock.Any(), gomock.Any()).Return(...)

app := controllertest.NewApp(t, func(app *fiber.App) {
    app.Get("/patients/:id", controller.GetPatientByID)
})
resp, _ := controllertest.Do(t, app, controllertest.Request{Method: http.MethodGet, Path: "/patients/pat-1"})
controllertest.AssertStatus(t, resp, fiber.StatusOK)
```

### Shared helpers — service tests

**Path:** [`internal/testutil/servicetest/helpers.go`](../internal/testutil/servicetest/helpers.go)

| Helper | Purpose |
|--------|---------|
| `NopLogger()` | No-op zap logger for service calls |
| `NoopNotifier` | No-op notification enqueuer (`Create` returns nil) |
| `ValidPatientInfo()` | Fixture for patient create |
| `ValidAppointmentPayload()` | Fixture for appointment create |
| `ValidOrgSchedule()` / `ValidOrgScheduleReq()` | Fixtures for org schedule get/create |
| `ValidCheckoutReq()` / `ValidDispensedItem()` | Fixtures for billing checkout |
| `ValidLoginUser()` / `ValidUpdatePasswordRequest()` | Fixtures for authentication |
| `UserWithPassword(t, plain)` | User with bcrypt hash for login tests |
| `ValidEmpFindManyRequest()` | Fixture for employee list queries |
| `ValidCreatePaymentCommand()` | Fixture for payment create |
| `ValidCreatePrescriptionRequest()` / `ValidMedicineArray()` | Fixtures for prescription |
| `ValidPrescriptionItemUpdate()` | Fixture for prescription item update |
| `ValidMedicineRequestPayload()` / `ValidSupplierCreateRequest()` | Fixtures for medicine |
| `ValidOrganisationPayload()` | Fixture for organisation signup (`CreateOrganisation`) |

### Mock generation

**Path:** [`mockgen.go`](../mockgen.go)

Regenerate all mocks:

```bash
go generate ./mockgen.go
```

Uses `go.uber.org/mock v0.6.0`.

---

## Controller unit tests

Table-driven tests with gomock service mocks. Each test file covers validation errors, service error mapping, and success paths where applicable.

| Package | Test file | Handlers / tests |
|---------|-----------|------------------|
| **authentication** | `internal/authentication/controllers_test.go` | `Login`, `Refresh`, `Logout`, `UpdatePassword`, internal error |
| **admins** | `internal/admins/organisation-schedule.controllers_test.go` | `Create` (org schedule) |
| **appointments** | `internal/appointments/appointment.controllers_test.go` | `CreateAppointment`, `GetSlots`, `FindManyByOrganisationID`, `FindAppointmentsPreview`, `UpdateStatus`, `GetAppointmentByPatientID` |
| **billing** | `internal/billing/controllers_test.go` | `Checkout`, `GetInvoiceByPrescriptionID`, `GetInvoiceByAppointmentID`, `GetBillDetailsByPrescriptionID`, `RetryPaymentLink` |
| **department** | `internal/department/controllers_test.go` | `FindMany` |
| **employee** | `internal/employee/controllers_test.go` | `Add`, `Delete`, `FindByID`, `FindMany`, `CreateAdmin`, `UpdateUser`, `FindDoctors`, service error |
| **license** | `internal/license/controllers_test.go` | `VerifyLicense` |
| **medicine** | `internal/medicine/common.controllers_test.go` | `GetByID`, `GetAll`, `SearchMedicine`, `AddMedicine`, validation |
| **medicine** | `internal/medicine/supplier.controllers_test.go` | `CreateSupplier`, `GetSupplierByID`, `GetSupplierByOrgID`, `GetSupplierTotalCount` |
| **organisation** | `internal/organisation/controllers_test.go` | `CreateOrganisation`, `UpdateOrganisationLoc`, `GetByID`, `UpdateOrganisation` |
| **patient** | `internal/patient/controllers_test.go` | `AddGeneralInfoHandler`, `GetPatientByID`, `FindPatients` |
| **payments** | `internal/payments/controllers_test.go` | `RazorPayWebhook`, `UpdatePaymentManually` |
| **permissions** | `internal/permissions/controllers_test.go` | `FindMany` |
| **prescription** | `internal/prescription/controllers_test.go` | `CreatePrescription`, `FindMany`, `FindByStatus`, `AddPrescriptionItems`, `UpdatePrescriptionItem`, `FindByID`, `UpdateStatus`, `GetPrescriptionsByPatientID`, `FindMedicineDetInfo` |
| **roles** | `internal/roles/controllers_test.go` | `FindMany` |
| **rolepermissions** | `internal/rolepermissions/controllers_test.go` | `FindModulesByRoleID` |

### Intentionally not covered (controllers)

| Package | Reason |
|---------|--------|
| **bedmanagement** | Skipped by design |

---

## Service unit tests

Service tests mock repositories (and stub cross-service interfaces). No real database is used.

### Patient service

**File:** [`internal/patient/services_test.go`](../internal/patient/services_test.go)

| Method | Cases covered |
|--------|---------------|
| `ValidatePatient` | Missing name/gender, invalid age/weight, success |
| `ToPatientModel` | Field mapping, ID/UHID generation, active status |
| `GetPageSkip` | Page 1 and page 3 offset |
| `FindOne` | Not found, DB error, success |
| `FindMany` | Read error, count error, success with total |
| `GetNotificationPatientByID` | Repo error, empty result, success |
| `CreatePatientSrv` | Org not found, validation fail, duplicate, repo error, success |

**Mocks used:** `MockPatientRepository`, `MockOrganisationServicer`

---

### Appointment service

**Files:**

- [`internal/appointments/appointment.services_test.go`](../internal/appointments/appointment.services_test.go) — exported methods
- [`internal/appointments/appointment.services_internal_test.go`](../internal/appointments/appointment.services_internal_test.go) — private helpers

| Method / helper | Cases covered |
|-----------------|---------------|
| `SelectStatus` | All valid statuses + invalid |
| `UpdateStatus` | Invalid status, repo error, success |
| `GetAppntmentByID` | Not found, DB error, success |
| `GetAppointmentPreview` | Not found, success |
| `GetAppointmentsByOrgID` | List error, success |
| `GetAppointmentByPatientID` | List error, success |
| `GetNotificationDetails` | Success |
| `CreateApptmnt` | Schedule not found, validation fail, repo error, success |
| `GetSlots` | Schedule not found, repo error, success |
| `validateAppointmentFields` | Missing IDs, invalid time, past date, success |
| `parsepagination` | Zero inputs, page 2 |
| `timesOverlap` | Overlap and adjacent slots |
| `generateAppointmentCode` | Non-empty code |
| `normalizeTimeOfDay` | Hour/minute preserved |

**Mocks used:** `MockAppointmentRepository`, `MockOrganisationScheduleServicer`

---

### Invoice item service

**Files:**

- [`internal/billing/invoice-item.services_test.go`](../internal/billing/invoice-item.services_test.go) — `addInvoiceItems`, `toInvoiceItem` (same package, manual stubs)
- [`internal/billing/invoice-item.services_gomock_test.go`](../internal/billing/invoice-item.services_gomock_test.go) — `GetMedicineInventoryDetByInvoiceID`

| Method | Cases covered |
|--------|---------------|
| `toInvoiceItem` | Maps dispensed items to invoice item rows |
| `addInvoiceItems` | Qty lookup error, medicine not in Rx, qty exceeds remaining, insufficient stock, zero qty persisted, repo error, success |
| `GetMedicineInventoryDetByInvoiceID` | Repo error, success |

**Mocks used:** `MockInvoiceItemRepo` (gomock); manual `stubPrescriptionQty` for private method tests

---

### Invoice service

**Files:**

- [`internal/billing/invoice.services_test.go`](../internal/billing/invoice.services_test.go) — exported methods
- [`internal/billing/invoice.services_internal_test.go`](../internal/billing/invoice.services_internal_test.go) — private helpers

| Method / helper | Cases covered |
|-----------------|---------------|
| `CreateInvoice` | Missing idempotency key, invalid payment type, missing prescription ID, consultation validation (missing appt ID, patient/org mismatch) |
| Idempotency replay | Replay hit returns existing invoice + payment URL |
| `GetInvoiceByPrescriptionID` | Empty ID, not found, DB error, success |
| `GetInvoiceByAppointmentID` | Success |
| `GetBillDetailsByPrescriptionID` | Empty ID, not found, success |
| `RetryPaymentLink` | Missing invoice ID, not found, patient not found, success |
| `validatePaymentTypeInputs` | Missing prescription ID, invalid payment type |
| `mapInvoiceCreateErr` | Generic DB error, duplicate prescription, duplicate consultation |
| `toInvoiceModel` | Prescription and consultation invoice shapes |
| `createCode` / `resolvePaymentType` | Smoke tests |

**Not covered:** full `CreateInvoice` checkout happy path (GORM `Begin`/`Commit` transaction). Validation, idempotency replay, and read paths are tested without a test database.

**Mocks used:** `MockInvoiceRepo`; manual stubs for `PaymentCheckout`, `PatientLookup`, `AppointmentLookup`

---

### Authentication service

**Files:**

- [`internal/authentication/services_test.go`](../internal/authentication/services_test.go) — exported methods
- [`internal/authentication/services_internal_test.go`](../internal/authentication/services_internal_test.go) — private helpers

| Method / helper | Cases covered |
|-----------------|---------------|
| `Login` | User not found, `errUserNotFound`, DB lookup error, wrong password, temp password + clear, clear-temp-password fail, access token fail, refresh find fail, update-refresh fail, mint-refresh fail, insert-refresh fail, update-last-login fail, update-existing-refresh, create-new-refresh, role permissions error, nil role-perm svc, success |
| `RefreshToken` | Session expired, generic failure, success |
| `Logout` | JWT error, success |
| `UpdatePassword` | Validation fail, repo error, success, whitespace trim success |
| `validateCredentials` | Empty username/password |
| `validateNewPassword` | Empty, mismatch, too short, valid |
| `comparePwd` / `hashPassword` | Round-trip |
| `toLoginResp` | Maps tokens |

**Mocks used:** `MockUserRepository`, `MockJwtServicer`, `MockRolePermissionServicer`

---

### Department service

**Files:**

- [`internal/department/services_test.go`](../internal/department/services_test.go) — exported methods
- [`internal/department/services_internal_test.go`](../internal/department/services_internal_test.go) — `createDeptArray`

| Method / helper | Cases covered |
|-----------------|---------------|
| `FindMany` | FindMany repo error, count error, success |
| `InsertMany` | BatchInsert error, success (8 default depts) |
| `FindDeptByName` | Repo error, success passthrough |
| `FindByID` | Repo error, success passthrough |
| `createDeptArray` | 8 depts with correct org ID and default names |

**Mocks used:** `MockDepartmentRepository`

---

### Employee service (basic scope)

**Files:**

- [`internal/employee/services_test.go`](../internal/employee/services_test.go) — exported methods
- [`internal/employee/services_internal_test.go`](../internal/employee/services_internal_test.go) — private helpers

| Method / helper | Cases covered |
|-----------------|---------------|
| `DeleteEmployee` | Empty ID, repo error, success |
| `FindOne` | Repo error, success mapping |
| `FindMany` | Read error, count error, success |
| `FindRoleIDByUserID` | Repo error, success |
| `FindDoctors` | Repo error, success |
| `getPageSkip` | Page 0 vs page 2 |
| `mapToEmployeeResponse` | Username fallback, active/inactive |
| `createEmployeeCode` | Invalid DOJ, repo error, count-based suffix |

**Not covered (future pass):** `CreateEmployee`, `CreateAdminProf`, `UpdateAdminProf` — require role/dept/org lookups and notification wiring.

**Mocks used:** `MockEmployeeRepository`; `codeCountRepo` stub in internal tests (avoids import cycle with gomock)

---

### License service

**File:** [`internal/license/services_test.go`](../internal/license/services_test.go)

| Method | Cases covered |
|--------|---------------|
| `VerifyLicense` | Empty org ID, empty key, not found, DB error, key mismatch, success |
| `CreateLicenseSrv` | Repo create error, success (org ID / non-empty key) |

**Mocks used:** `MockLicenseRepository`

---

### Payments service

**Files:**

- [`internal/payments/payment-attempts.services_test.go`](../internal/payments/payment-attempts.services_test.go) — `SPaymentAttempts`
- [`internal/payments/fulfillment.services_test.go`](../internal/payments/fulfillment.services_test.go) — `FulfillmentService`
- [`internal/payments/payments.services_test.go`](../internal/payments/payments.services_test.go) — read/validation paths
- [`internal/payments/payments.services_internal_test.go`](../internal/payments/payments.services_internal_test.go) — helpers

| Service / method | Cases covered |
|------------------|---------------|
| `SPaymentAttempts` | CreateAttempt, CreateAttemptWithIdempotency, Find*, UpdatePaymentAttemptStatus, ClaimForProcessing |
| `FulfillmentService.FulfillPaidInvoice` | Inventory error, empty-items invoice paid path |
| `FulfillmentService.NotifyPaymentReceived` | Patient lookup error, notification success |
| `PaymentsService` reads | GetPaymentByIdempotencyKey, GetPaymentByInvoiceID, GetPaymentURLByPaymentID |
| `ConfirmManualPayment` | Validation (missing IDs, unsupported mode), payment not found |
| `validateIdempotencyKey` / `toPaymentModel` / `responseFromExistingPayment` | Helper coverage |

**Not covered (future pass):** `CreateLinkPayment`, `RetryLinkPayment`, `CreatePendingPayment` tx paths; `IWebhookService.ProcessWebhook`.

**Mocks used:** `MockIPaymentsRepository`, `MockIPaymentAttempts`; manual `stubFulfillmentDeps` for `IPaymentFulfillment`

---

### Prescription service

**Files:**

- [`internal/prescription/prescriptions-items.services_test.go`](../internal/prescription/prescriptions-items.services_test.go) — item service
- [`internal/prescription/prescriptions-items.services_internal_test.go`](../internal/prescription/prescriptions-items.services_internal_test.go) — item helpers
- [`internal/prescription/prescriptions.services_test.go`](../internal/prescription/prescriptions.services_test.go) — parent service
- [`internal/prescription/prescriptions.services_internal_test.go`](../internal/prescription/prescriptions.services_internal_test.go) — parent helpers

| Service / method | Cases covered |
|------------------|---------------|
| `PrescriptionItemServ` | AddItems, UpdatePrescriptionItemByID, GetqtyByMedicine, GetPrescriptionsByPIDWithLimit, GetMedicineInfo, UpdateDispenseItemQty, UpdateIPrescriptionStatus |
| `PrescriptionService` | FindMany, FindByStatus, GetPrescriptionsByPatientID, UpdateManualStatus, AddPrescriptionItems |
| Helpers | parseDurationtype, calculateQuantity, parseFilterStatus, parseManualStatus, ResolveAndUpdateParentStatus, UpdateExtPrescriptionStatus |

**Not covered (future pass):** `CreatePrescription` full tx happy path.

**Mocks used:** `MockPrescriptionRepositoryInterface`, `MockPrescItemsRepo`, `MockPrescriptionItemAdder`; `servicetest.NoopNotifier`

---

### Medicine service

**Files:**

- [`internal/medicine/supplier.services_test.go`](../internal/medicine/supplier.services_test.go) — supplier (full coverage)
- [`internal/medicine/supplier.services_internal_test.go`](../internal/medicine/supplier.services_internal_test.go) — supplier helpers
- [`internal/medicine/medicine.services_test.go`](../internal/medicine/medicine.services_test.go) — read paths
- [`internal/medicine/medicine.services_internal_test.go`](../internal/medicine/medicine.services_internal_test.go) — mappers
- [`internal/medicine/medicine_inventory.services_test.go`](../internal/medicine/medicine_inventory.services_test.go) — inventory delegation
- [`internal/medicine/medicine_mvmt.services_test.go`](../internal/medicine/medicine_mvmt.services_test.go) — movement delegation
- [`internal/medicine/purchase-entries.services_test.go`](../internal/medicine/purchase-entries.services_test.go) — purchase entry
- [`internal/medicine/purchase-entries.services_internal_test.go`](../internal/medicine/purchase-entries.services_internal_test.go) — `calculatePaymentDueDate`

| Service / method | Cases covered |
|------------------|---------------|
| `SupplierService` | GetSupplierByID, CretateSupplier (field mapping), GetSupplierByOrgID (search trim + date format), GetTotalCount |
| `MedicineService` reads | GetOne, GetMany, SearchMedicine, FindNamesByIds |
| `SMedicineInventory` / `SMedicineMvmt` / `PurchaseEntryService` | Repo delegation error + success |
| Helpers | parsePagination, findPaymentTerms (Net 15/45), toSupplierList, createCode, toMedicine (Add flag), toMedicineResponse |

**Not covered (future pass):** `MedicineService.CreateMedicine` full tx orchestration.

**Mocks used:** `MockMedicineRepository`, `MockISupplier`, `MockRMedicineInventory`, `MockRPurchaseEntry`, `MockRMedicineMvmt`

---

### Organisation service (signup)

**Files:**

- [`internal/organisation/services_test.go`](../internal/organisation/services_test.go) — exported methods
- [`internal/organisation/services_internal_test.go`](../internal/organisation/services_internal_test.go) — `createOrgModel`

| Method / helper | Cases covered |
|-----------------|---------------|
| `CreateOrganisation` | Permissions lookup error, repo create error, license error, roles seed error, depts seed error, role-permissions seed error, success (SQLite tx) |
| `GetOrgByID` | Not found, DB error, success |
| `Update` | Not found, DB error, success (partial fields) |
| `UpdateOrganisationLoc` | Not found, DB error, success (address/security mapping) |
| `createOrgModel` | Generated ID/code/timestamps from payload |

**Mocks used:** `MockOrganisationRepo`, `MockPermissionCatalogLookup`, `MockLicenseCreator`, `MockRoleSeeder`, `MockDepartmentSeeder`, `MockRolePermissionSeeder`

---

### Permissions service

**Files:**

- [`internal/permissions/services_test.go`](../internal/permissions/services_test.go)

| Method | Cases covered |
|--------|---------------|
| `FindMany` | Permission repo error, module repo error, success |
| `DefaultPerm` | BatchInsert error, success (4 default permissions) |

**Mocks used:** `MockPermissionRepo`, `MockModuleRepo`

---

### Modules service

**Files:**

- [`internal/modules/services_test.go`](../internal/modules/services_test.go)

| Method | Cases covered |
|--------|---------------|
| `DefaultModule` | BatchInsert error, success (11 default modules, all active) |

**Mocks used:** `MockModuleRepo`

---

### Roles service

**Files:**

- [`internal/roles/services_test.go`](../internal/roles/services_test.go) — exported methods
- [`internal/roles/services_internal_test.go`](../internal/roles/services_internal_test.go) — mappers and `createRoleArray`

| Method / helper | Cases covered |
|-----------------|---------------|
| `FindMany` | FindMany repo error, count error, success |
| `InsertMany` | InsertMany repo error, success (7 default roles) |
| `FindRoleByNames` | Repo error, success passthrough |
| `FindByID` | Repo error, success passthrough |
| `FindRoleByOrgID` | Repo error, success, empty result |
| `createRoleArray` / mappers | Default roles for org, DTO mapping |

**Mocks used:** `MockRoleRepository`

---

### Role permissions service

**Files:**

- [`internal/rolepermissions/services_test.go`](../internal/rolepermissions/services_test.go) — exported methods
- [`internal/rolepermissions/services_internal_test.go`](../internal/rolepermissions/services_internal_test.go) — mappers and seed helpers
- [`internal/rolepermissions/controllers_test.go`](../internal/rolepermissions/controllers_test.go) — `FindModulesByRoleID`

| Method / helper | Cases covered |
|-----------------|---------------|
| `Create` | Repo error, success |
| `FindModulesByRoleID` | Empty role ID, is-admin error, admin role, module fetch error, success with flags |
| `InsertMany` | BatchCreate error, success (admin + seeded role rows) |
| `mapModulePermissionRows` / `applyPermissionFlag` | Flag mapping including unknown names |
| `toRolePermModel` / `toAdminRolePermModel` / `nullableUUID` | Mapping edge cases |
| `createRPModel` | Seeds admin/doctor/nurse rows for org |

**Mocks used:** `MockRolePermissionRepo`, `MockRolePermissionServicer`

---

### Admins service (organisation schedule)

**Files:**

- [`internal/admins/organisation-schedule.services_test.go`](../internal/admins/organisation-schedule.services_test.go) — exported methods
- [`internal/admins/organisation-schedule.services_internal_test.go`](../internal/admins/organisation-schedule.services_internal_test.go) — `toOrgSchedModel`, `toResponseModel`

| Method / helper | Cases covered |
|-----------------|---------------|
| `Create` | Repo error → `ErrOrgScheduleCreateFailed`, success (field mapping) |
| `GetScheduleByOrganisationID` | Empty org ID, not found, DB error, success |
| `toOrgSchedModel` / `toResponseModel` | Request mapping, time parsing |

**Mocks used:** `MockOrganisationScheduleRepository`

---

## JWT tests (pre-existing)

**Files:**

- [`internal/jwt/services_test.go`](../internal/jwt/services_test.go) — extensive JWT service tests (access token, refresh token, parse, validate)
- [`internal/jwt/jwt_test.go`](../internal/jwt/jwt_test.go) — smoke test

> **Note:** `internal/jwt/services_test.go` currently does not compile against the latest `JwtService` / `RefreshtokenRepo` signatures. These tests pre-date the controller/service mock work and need to be updated separately.

---

## Middleware tests

**Path:** [`pkg/middleware/`](../pkg/middleware/)

| File | Coverage |
|------|----------|
| `rbac_routes_test.go` | `ResolveRoutePermission`, `IsPublicRoute` |
| `rbac_e2e_test.go` | RBAC E2E scenarios: role lookup failure, permission load failure, bypass attempts |

---

## Other tests

| File | Notes |
|------|-------|
| `internal/authentication/auth_test.go` | `TestLicense` — legacy/smoke |

---

## Generated mocks

### Service mocks (for controller tests)

| Mock file | Interface |
|-----------|-----------|
| `internal/authentication/mocks/mock_auth_servicer.go` | `AuthServicer` |
| `internal/admins/mocks/mock_organisation_schedule_servicer.go` | `OrganisationScheduleServicer` |
| `internal/appointments/mocks/mock_appointment_servicer.go` | `AppointmentServicer` |
| `internal/billing/mocks/mock_invoice_servicer.go` | `InvoiceServicer` |
| `internal/department/mocks/mock_department_servicer.go` | `DepartmentServicer` |
| `internal/employee/mocks/mock_employee_servicer.go` | `EmployeeServicer` |
| `internal/license/mocks/mock_license_servicer.go` | `LicenseServicer` |
| `internal/medicine/mocks/mock_medicine_servicer.go` | `MedicineServicer`, `SupplierServicer` |
| `internal/organisation/mocks/mock_organisation_servicer.go` | `OrganisationServicer` |
| `internal/patient/mocks/mock_patient_servicer.go` | `PatientServicer` |
| `internal/payments/mocks/mock_payment_servicer.go` | `PaymentServicer`, `WebhookServicer` |
| `internal/permissions/mocks/mock_permission_servicer.go` | `PermissionServicer` |
| `internal/prescription/mocks/mock_prescription_servicer.go` | `PrescriptionServicer`, `PrescriptionItemServicer` |
| `internal/roles/mocks/mock_role_servicer.go` | `RoleServicer` |
| `internal/rolepermissions/mocks/mock_role_permission_servicer.go` | `RolePermissionServicer` |

### Repository mocks (for service tests)

| Mock file | Interface(s) |
|-----------|--------------|
| `internal/admins/mocks/mock_organisation_schedule_repository.go` | `OrganisationScheduleRepository` |
| `internal/appointments/mocks/mock_appointment_repository.go` | `AppointmentRepository` |
| `internal/authentication/mocks/mock_user_repository.go` | `UserRepository` |
| `internal/authentication/mocks/mock_jwt_servicer.go` | `JwtServicer` |
| `internal/authentication/mocks/mock_role_permission_servicer.go` | `RolePermissionServicer` |
| `internal/bedmanagement/mocks/mock_bedmanagement_repository.go` | Bed/room repos |
| `internal/billing/mocks/mock_invoice_repository.go` | `InvoiceRepo`, `InvoiceItemRepo` |
| `internal/department/mocks/mock_department_repository.go` | `DepartmentRepository` |
| `internal/employee/mocks/mock_employee_repository.go` | `EmployeeRepository` |
| `internal/jwt/mocks/mock_refresh_token_repository.go` | `RefreshtokenRepo` |
| `internal/license/mocks/mock_license_repository.go` | `LicenseRepository` |
| `internal/medicine/mocks/mock_medicine_repository.go` | Medicine/supplier/inventory repos |
| `internal/modules/mocks/mock_module_repository.go` | `ModuleRepo` |
| `internal/notifications/mocks/mock_notification_repository.go` | `Repository` |
| `internal/organisation/mocks/mock_organisation_repository.go` | `OrganisationRepo` |
| `internal/organisation/mocks/mock_signup_deps.go` | `PermissionCatalogLookup`, `LicenseCreator`, `RoleSeeder`, `DepartmentSeeder`, `RolePermissionSeeder` |
| `internal/patient/mocks/mock_patient_repository.go` | `PatientRepository` |
| `internal/payments/mocks/mock_payments_repository.go` | Payments/webhook/attempts repos |
| `internal/payments/mocks/mock_payment_fulfillment.go` | `IPaymentFulfillment`, `PaymentAttemptServicer`, `FulfillmentServicer`, `PaymentInvoiceLookup`, `PrescriptionStatusUpdater` |
| `internal/payments/providers/mocks/mock_provider.go` | `Provider`, `IPaymentFactory` |
| `internal/permissions/mocks/mock_permission_repository.go` | `PermissionRepo` |
| `internal/prescription/mocks/mock_prescription_repository.go` | Prescription + item repos |
| `internal/prescription/mocks/mock_prescription_deps.go` | `NotificationEnqueuer`, `AppointmentLookup`, `AppointmentStatusUpdater`, `PrescriptionItemAdder` |
| `internal/rolepermissions/mocks/mock_role_permission_repository.go` | `RolePermissionRepo` |
| `internal/rolepermissions/mocks/mock_role_permission_servicer.go` | `RolePermissionServicer` |
| `internal/roles/mocks/mock_role_repository.go` | `RoleRepository` |

Repository mocks are generated for service-layer unit tests. Packages with service tests today: patient, appointments, billing, authentication, department, employee (basic), license, payments, prescription, medicine, organisation, permissions, modules, roles, rolepermissions, admins.

---

## Service refactors for testability

Minimal interface changes were made so services accept mockable dependencies:

| Service | Change |
|---------|--------|
| `PatientService` | `OrganisationServicer`, `NotificationEnqueuer` |
| `AppointmentService` | `OrganisationScheduleServicer`, `NotificationEnqueuer` |
| `InvoiceItemServ` | `PrescriptionItemServicer` (includes `GetqtyByMedicine`) |
| `InvoiceServ` | `PatientLookup`, `AppointmentLookup`, `PaymentCheckout` |
| `OrganisationScheduleServicer` | Added `GetScheduleByOrganisationID` |
| `UserService` (auth) | `UserRepository`, `JwtServicer`, `RolePermissionServicer`; `Login` uses `JwtService.FindIDByUserID` |
| `LicenseService` | `LicenseRepository` interface injection |
| `EmployeeService` | `NotificationEnqueuer` via constructor |
| `PaymentsService` | `IPaymentFactory`, `PaymentAttemptServicer`, `FulfillmentServicer` |
| `IWebhookService` | `PaymentInvoiceLookup`, `PaymentAttemptServicer`, `FulfillmentServicer`, `IPaymentFactory` |
| `PrescriptionService` | `AppointmentLookup`, `AppointmentStatusUpdater`, `PrescriptionItemAdder`, `NotificationEnqueuer`; removed unused deps |
| `MedicineService` | `InventoryCreator`, `MvmtCreator`, `PurchaseEntryCreator`, `SupplierLookup` |
| `AppointmentService` | Added `UpdateStatusInTx` for prescription status updates |
| `OrganisationService` | `PermissionCatalogLookup`, `LicenseCreator`, `RoleSeeder`, `DepartmentSeeder`, `RolePermissionSeeder` |
| `PermService` | `ModuleLookup modules.ModuleRepo` (was concrete `*ModuleDb`) |
| `ModuleService` | `ModuleRepo` interface (was concrete `*ModuleDb`) |

Concrete types in `appinit/app.go` satisfy these interfaces. Employee notifications and auth service wiring now pass dependencies through constructors.

---

## Running tests

```bash
# All packages with tests (jwt currently fails to build)
go test ./...

# Service unit tests (all packages with service tests)
go test ./internal/patient/ ./internal/appointments/ ./internal/billing/ \
  ./internal/authentication/ ./internal/department/ ./internal/employee/ ./internal/license/ \
  ./internal/payments/ ./internal/prescription/ ./internal/medicine/ ./internal/organisation/ \
  ./internal/permissions/ ./internal/modules/ ./internal/roles/ ./internal/rolepermissions/ ./internal/admins/ -count=1

# All controller-tested packages
go test ./internal/authentication/ ./internal/admins/ ./internal/appointments/ \
  ./internal/billing/ ./internal/department/ ./internal/employee/ \
  ./internal/license/ ./internal/medicine/ ./internal/organisation/ \
  ./internal/patient/ ./internal/payments/ ./internal/permissions/ \
  ./internal/prescription/ ./internal/roles/ ./internal/rolepermissions/ ./pkg/middleware/ -count=1

# Regenerate mocks after interface changes
go generate ./mockgen.go
```

---

## Coverage gaps (future work)

| Area | Status |
|------|--------|
| **bedmanagement** controllers/services | Not tested |
| **employee** create/update flows | Basic service scope only; `CreateEmployee` / admin profile paths deferred |
| **rolepermissions** | Controller + service covered (`FindModulesByRoleID`, seed helpers) |
| **notifications** service | Repo mock exists; no service tests |
| **payments** write paths | `CreateLinkPayment`, webhook tx paths deferred |
| **prescription** create tx | `CreatePrescription` full happy path deferred |
| **medicine** create tx | `CreateMedicine` orchestration deferred |
| **CreateInvoice** full checkout tx | Out of scope — needs in-memory DB or sqlmock |
| **JWT** `services_test.go` | Exists but out of date with current code |
| **Repository layer** | No integration tests against real Postgres |
| **E2E / API tests** | Not present (Postman docs exist under `docs/postman*.md`) |

---

## Test file index

```
internal/
├── admins/organisation-schedule.controllers_test.go
├── appointments/
│   ├── appointment.controllers_test.go
│   ├── appointment.services_test.go
│   └── appointment.services_internal_test.go
├── authentication/
│   ├── auth_test.go
│   └── controllers_test.go
├── billing/
│   ├── controllers_test.go
│   ├── invoice.services_test.go
│   ├── invoice.services_internal_test.go
│   ├── invoice-item.services_test.go
│   └── invoice-item.services_gomock_test.go
├── department/controllers_test.go
├── employee/controllers_test.go
├── jwt/
│   ├── jwt_test.go
│   └── services_test.go          # needs update
├── license/controllers_test.go
├── medicine/
│   ├── common.controllers_test.go
│   └── supplier.controllers_test.go
├── organisation/controllers_test.go
├── patient/
│   ├── controllers_test.go
│   └── services_test.go
├── payments/controllers_test.go
├── permissions/controllers_test.go
├── prescription/controllers_test.go
├── roles/controllers_test.go
├── rolepermissions/
│   ├── controllers_test.go
│   ├── services_test.go
│   └── services_internal_test.go
└── testutil/
    ├── controllertest/           # Fiber harness
    └── servicetest/              # Service fixtures

pkg/middleware/
├── rbac_routes_test.go
└── rbac_e2e_test.go
```
