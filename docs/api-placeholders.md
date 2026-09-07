# API placeholder catalog

Reference map of every registered HTTP API from `pkg/middleware/routers/functions.go`, using snake_case keys:

```text
appointment_status => appointment/updateStatus
```

## Naming convention

| Piece | Rule |
|-------|------|
| **Key** | `{module}_{intent}` in snake_case (e.g. `appointment_status`, `patient_create`) |
| **Value** | `{group}/{handlerPath}` as registered (preserve route casing) |
| **Full path** | `/api/v1/{value}` (plus `:params` where present) |

These keys are **documentation placeholders only**. Runtime RBAC uses **module + CRUD action** (`create` / `update` / `view` / `delete`) via `pkg/middleware/rbac_routes.go`, not these snake_case names.

The `permissions` table holds only the four CRUD names: `create`, `update`, `view`, `delete`.

### RBAC status column

| Status | Meaning |
|--------|---------|
| `public` | Listed in `PublicRoutes` (no module permission check; some still require JWT) |
| `create` / `update` / `view` / `delete` | Mapped in Create/Update/View/Delete route maps (module in parentheses) |
| `unmapped` | Not in any RBAC map; intentional until a module exists |

Sources of truth: `pkg/middleware/routers/functions.go`, `pkg/middleware/rbac_routes.go`.

---

## Quick lookup (key => path)

```text
auth_login => authentication/login
auth_refresh => authentication/refresh
auth_logout => authentication/logout
auth_update_password => authentication/updatePassword
auth_update_password_first_login => authentication/updatePasswordFirstLogin
auth_request_password_reset => authentication/requestPasswordReset
role_get_many => role/getRoles
department_get_many => department/getDepartments
permission_get_all => permission/getAll
patient_create => patients/addGeneralInfo
patient_get_many => patients/getPatients
patient_get_by_id => patients/getpatientByID/:patientID
employee_get_doctors => employee/getDoctors
employee_find_by_id => employee/findbyID
employee_add => employee/addEmployee
employee_create => employee/create
employee_update => employee/update
employee_delete => employee/delete
employee_get_many => employee/getEmployees
organisation_signup => organisation/signupOrg
organisation_update_location => organisation/updateLocation
organisation_get_by_id => organisation/getbyid/:organisation_id
organisation_update => organisation/update
medicine_add => medicine/addMedicine
medicine_get_by_id => medicine/getMedicineByID
medicine_get_many => medicine/GetMedicines
medicine_search => medicine/searchMedicine
prescription_create => prescription/create
prescription_get_many => prescription/get
prescription_get_by_status => prescription/getByStatus
prescription_update => prescription/updatePrescriptions
prescription_update_item => prescription/updatePrescriptionItem
prescription_get_by_id => prescription/getprescriptionbyPid
prescription_get_by_appointment_id => prescription/getPrescriptionByAppointmentID
prescription_get_by_patient_id => prescription/getPrescriptionByPatientID
prescription_update_status => prescription/updateStatus
prescription_get_medicine_info => prescription/getMedicineInfo/:prescription_id
supplier_get_by_id => supplier/getSupplierByID
supplier_get_by_org_id => supplier/getSupplierByOrgID
supplier_get_total_count => supplier/getTotalCount
supplier_create => supplier/createSupplier
appointment_create => appointment/create
appointment_get_time_slots => appointment/getTimeSlots
appointment_get_by_org_id => appointment/getappointmentbyOrgID
appointment_get_preview => appointment/getAppointmentsPreview
appointment_status => appointment/updateStatus
appointment_get_by_patient_id => appointment/getappointmentByPatientID
billing_create => billing/create
billing_get_invoice_by_prescription_id => billing/getInvoiceByPrescriptionID/:prescriptionID
billing_get_invoice_by_appointment_id => billing/getInvoiceByAppointmentID/:appointmentID
billing_get_bill_details_by_prescription_id => billing/getBillDetailsByPrescriptionID/:prescriptionID
billing_retry_payment_link => billing/invoices/:invoiceID/retry-payment-link
payment_webhook => payment/webhook
payment_confirm => payment/confirm
bed_create_room_type => bed/createRoomType
bed_get_room_type_data => bed/getRoomTypeData
bed_create_room => bed/createRoom
bed_create => bed/createBed
bed_generate => bed/generateBeds
bed_get_available => bed/getAvailableBeds
bed_get_available_rooms => bed/getAvailableRooms
bed_get_available_room_types => bed/getAvailableRoomTypes
bed_create_allotment => bed/createBedAllotment
org_schedule_create => admins/organisationSchedule/create
```

---

## Authentication

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `auth_login` | POST | `authentication/login` | public |
| `auth_refresh` | POST | `authentication/refresh` | public |
| `auth_logout` | POST | `authentication/logout` | public |
| `auth_update_password` | PATCH | `authentication/updatePassword` | public |
| `auth_update_password_first_login` | PATCH | `authentication/updatePasswordFirstLogin` | public (JWT required) |
| `auth_request_password_reset` | POST | `authentication/requestPasswordReset` | public |

## Role

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `role_get_many` | GET | `role/getRoles` | view (`role`) |

## Department

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `department_get_many` | GET | `department/getDepartments` | view (`department`) |

## Permission

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `permission_get_all` | GET | `permission/getAll` | view (`role`) |

## Patient

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `patient_create` | POST | `patients/addGeneralInfo` | create (`patient`) |
| `patient_get_many` | POST | `patients/getPatients` | view (`patient`) |
| `patient_get_by_id` | GET | `patients/getpatientByID/:patientID` | view (`patient`) |

## Employee

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `employee_get_doctors` | GET | `employee/getDoctors` | public (JWT required) |
| `employee_find_by_id` | GET | `employee/findbyID` | public (JWT required) |
| `employee_add` | POST | `employee/addEmployee` | create (`employee`) |
| `employee_create` | POST | `employee/create` | create (`employee`) |
| `employee_update` | PATCH | `employee/update` | update (`employee`) |
| `employee_delete` | DELETE | `employee/delete` | delete (`employee`) |
| `employee_get_many` | GET | `employee/getEmployees` | view (`employee`) |

## Organisation

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `organisation_signup` | POST | `organisation/signupOrg` | public |
| `organisation_update_location` | PATCH | `organisation/updateLocation` | unmapped |
| `organisation_get_by_id` | GET | `organisation/getbyid/:organisation_id` | unmapped |
| `organisation_update` | PATCH | `organisation/update` | unmapped |

## Medicine

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `medicine_add` | POST | `medicine/addMedicine` | create (`medicine`) |
| `medicine_get_by_id` | GET | `medicine/getMedicineByID` | view (`medicine`) |
| `medicine_get_many` | GET | `medicine/GetMedicines` | view (`medicine`) |
| `medicine_search` | GET | `medicine/searchMedicine` | view (`medicine`) |

## Prescription

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `prescription_create` | POST | `prescription/create` | create (`prescription`) |
| `prescription_get_many` | POST | `prescription/get` | view (`prescription`) |
| `prescription_get_by_status` | GET | `prescription/getByStatus` | view (`prescription`) |
| `prescription_update` | PATCH | `prescription/updatePrescriptions` | update (`prescription`) |
| `prescription_update_item` | PATCH | `prescription/updatePrescriptionItem` | update (`prescription`) |
| `prescription_get_by_id` | GET | `prescription/getprescriptionbyPid` | view (`prescription`) |
| `prescription_get_by_appointment_id` | POST | `prescription/getPrescriptionByAppointmentID` | view (`prescription`) |
| `prescription_get_by_patient_id` | GET | `prescription/getPrescriptionByPatientID` | view (`prescription`) |
| `prescription_update_status` | PATCH | `prescription/updateStatus` | update (`prescription`) |
| `prescription_get_medicine_info` | GET | `prescription/getMedicineInfo/:prescription_id` | view (`prescription`) |

## Supplier

Supplier routes use the **medicine** module for RBAC.

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `supplier_get_by_id` | GET | `supplier/getSupplierByID` | view (`medicine`) |
| `supplier_get_by_org_id` | POST | `supplier/getSupplierByOrgID` | view (`medicine`) |
| `supplier_get_total_count` | GET | `supplier/getTotalCount` | view (`medicine`) |
| `supplier_create` | POST | `supplier/createSupplier` | create (`medicine`) |

## Appointment

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `appointment_create` | POST | `appointment/create` | create (`appointment`) |
| `appointment_get_time_slots` | GET | `appointment/getTimeSlots` | view (`appointment`) |
| `appointment_get_by_org_id` | POST | `appointment/getappointmentbyOrgID` | view (`appointment`) |
| `appointment_get_preview` | GET | `appointment/getAppointmentsPreview` | view (`appointment`) |
| `appointment_status` | PATCH | `appointment/updateStatus` | update (`appointment`) |
| `appointment_get_by_patient_id` | POST | `appointment/getappointmentByPatientID` | view (`appointment`) |

## Billing

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `billing_create` | POST | `billing/create` | create (`billing`) |
| `billing_get_invoice_by_prescription_id` | GET | `billing/getInvoiceByPrescriptionID/:prescriptionID` | view (`billing`) |
| `billing_get_invoice_by_appointment_id` | GET | `billing/getInvoiceByAppointmentID/:appointmentID` | view (`billing`) |
| `billing_get_bill_details_by_prescription_id` | GET | `billing/getBillDetailsByPrescriptionID/:prescriptionID` | view (`billing`) |
| `billing_retry_payment_link` | POST | `billing/invoices/:invoiceID/retry-payment-link` | update (`billing`) |

## Payment

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `payment_webhook` | POST | `payment/webhook` | public |
| `payment_confirm` | POST | `payment/confirm` | update (`billing`) |

## Bed

JWT auth only today; no bed module in RBAC maps.

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `bed_create_room_type` | POST | `bed/createRoomType` | unmapped |
| `bed_get_room_type_data` | GET | `bed/getRoomTypeData` | unmapped |
| `bed_create_room` | POST | `bed/createRoom` | unmapped |
| `bed_create` | POST | `bed/createBed` | unmapped |
| `bed_generate` | POST | `bed/generateBeds` | unmapped |
| `bed_get_available` | GET | `bed/getAvailableBeds` | unmapped |
| `bed_get_available_rooms` | GET | `bed/getAvailableRooms` | unmapped |
| `bed_get_available_room_types` | GET | `bed/getAvailableRoomTypes` | unmapped |
| `bed_create_allotment` | POST | `bed/createBedAllotment` | unmapped |

## Admins / organisation schedule

| Key | Method | Path value | RBAC |
|-----|--------|------------|------|
| `org_schedule_create` | POST | `admins/organisationSchedule/create` | unmapped |

---

## Unmapped / needs module

These 13 routes are registered but intentionally absent from Create/Update/View/Delete/Public maps until dedicated modules exist. Documented in `rbac_routes.go`.

| Key | Path value | Notes |
|-----|------------|-------|
| `bed_create_room_type` | `bed/createRoomType` | JWT only; no bed module |
| `bed_get_room_type_data` | `bed/getRoomTypeData` | JWT only; no bed module |
| `bed_create_room` | `bed/createRoom` | JWT only; no bed module |
| `bed_create` | `bed/createBed` | JWT only; no bed module |
| `bed_generate` | `bed/generateBeds` | JWT only; no bed module |
| `bed_get_available` | `bed/getAvailableBeds` | JWT only; no bed module |
| `bed_get_available_rooms` | `bed/getAvailableRooms` | JWT only; no bed module |
| `bed_get_available_room_types` | `bed/getAvailableRoomTypes` | JWT only; no bed module |
| `bed_create_allotment` | `bed/createBedAllotment` | JWT only; no bed module |
| `organisation_update_location` | `organisation/updateLocation` | No organisation module; no JWT/RBAC on group |
| `organisation_get_by_id` | `organisation/getbyid/:organisation_id` | No organisation module; no JWT/RBAC on group |
| `organisation_update` | `organisation/update` | No organisation module; no JWT/RBAC on group |
| `org_schedule_create` | `admins/organisationSchedule/create` | No schedule module; no JWT/RBAC on group |

To wire any of these into RBAC later:

1. Add module constant(s) if needed (`bed`, `organisation`, …).
2. Add path → module entries in `rbac_routes.go`.
3. Switch the router group to `UseProtected` (or equivalent) so maps enforce.
4. Grant module actions in `defaultRolePermissionMatrix` for new orgs.
