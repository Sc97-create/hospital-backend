# Hospital Backend — Entity Relationship Diagram

Derived from GORM models migrated in `shared/migration/functions.go`. Table names follow GORM’s default pluralization (no custom `TableName()` overrides). Most foreign keys are logical UUID columns; only a few associations are declared with GORM tags.

> Soft delete: `roles` and `departments` use `gorm.DeletedAt`. `roles` also has `is_deleted`.

---

## Overview

```mermaid
erDiagram
    organisations ||--o{ licenses : has
    organisations ||--o{ users : has
    organisations ||--o{ departments : has
    organisations ||--o{ roles : has
    organisations ||--o{ patients : has
    organisations ||--o{ organisation_schedules : has
    organisations ||--o{ appointments : has
    organisations ||--o{ prescriptions : has
    organisations ||--o{ invoices : has
    organisations ||--o{ medicines : has
    organisations ||--o{ suppliers : has
    organisations ||--o{ notifications : has
    organisations ||--o{ role_permissions : has

    departments ||--o{ users : employs
    roles ||--o{ users : assigned
    roles ||--o{ role_permissions : grants
    permissions ||--o{ role_permissions : used_in
    modules ||--o{ role_permissions : used_in

    users ||--o{ refresh_tokens : has
    users ||--o{ appointments : "doctor"
    users ||--o{ prescriptions : "prescribed_by"

    patients ||--o{ appointments : books
    patients ||--o{ prescriptions : receives
    patients ||--o{ invoices : billed
    patients ||--o{ payments : pays
    patients ||--o{ notifications : notified

    organisation_schedules ||--o{ appointments : slots
    appointments ||--o{ prescriptions : generates
    appointments ||--o| invoices : "consultation"

    prescriptions ||--o{ prescription_items : contains
    prescriptions ||--o| invoices : "prescription bill"

    medicines ||--o{ prescription_items : prescribed
    medicines ||--o{ medicine_inventories : stocked
    medicines ||--o{ invoice_items : sold
    medicines ||--o{ medicine_stock_movements : tracked

    suppliers ||--o{ m_purchase_entries : supplies
    suppliers ||--o{ medicine_inventories : batches
    m_purchase_entries ||--o{ medicine_inventories : creates
    medicine_inventories ||--o{ medicine_stock_movements : moves
    medicine_inventories ||--o{ invoice_items : dispensed_from

    invoices ||--o{ invoice_items : lines
    invoices ||--o{ payments : collects
    payments ||--o{ payment_attempts : tries
    payment_attempts ||--o{ refunds : refunds
    payment_attempts ||--o{ webhook_events : receives

    notifications ||--o{ notification_attempts : retries
```

---

## 1. Organisation & license

```mermaid
erDiagram
    organisations {
        uuid id PK
        text code
        text hospital_type
        text legal_entity_name
        varchar organisation_name
        jsonb address
        jsonb security
        timestamp created_at
        timestamp updated_at
    }

    licenses {
        uuid id PK
        uuid organisation_id FK
        text license_key UK
        timestamp issued_at
        timestamp expires_at
    }

    organisations ||--o{ licenses : "organisation_id"
```

**JSONB shapes**

| Column | Fields |
|---|---|
| `organisations.address` | `country_id`, `state`, `city`, `created_at`, `created_by` |
| `organisations.security` | `enable_audit_logs`, `emergency_access` |

---

## 2. RBAC (roles, permissions, modules)

```mermaid
erDiagram
    organisations {
        uuid id PK
    }

    roles {
        uuid id PK
        varchar name
        uuid organisation_id FK
        text created_by
        text updated_by
        boolean is_deleted
        timestamp deleted_at
        timestamp created_at
        timestamp updated_at
    }

    permissions {
        uuid id PK
        varchar name UK
        text description
        timestamp created_at
        timestamp updated_at
    }

    modules {
        uuid id PK
        text name UK
        boolean is_active
        timestamp created_at
        timestamp updated_at
    }

    role_permissions {
        uuid id PK
        uuid role_id FK
        uuid permission_id FK "nullable"
        uuid module_id FK "nullable"
        uuid organisation_id FK
        boolean is_admin
        timestamp created_at
        timestamp updated_at
    }

    organisations ||--o{ roles : has
    organisations ||--o{ role_permissions : scopes
    roles ||--o{ role_permissions : "role_id"
    permissions ||--o{ role_permissions : "permission_id"
    modules ||--o{ role_permissions : "module_id"
```

- Default role names: Super Admin, Hospital Admin, Doctor, Nurse, Receptionist, Lab Technician, Pharmacist.
- Permission names: typically `create`, `update`, `delete`, `view`.
- Admin grants may leave `permission_id` / `module_id` null (`RelaxNullableColumns`).

---

## 3. Users, departments & auth tokens

```mermaid
erDiagram
    organisations {
        uuid id PK
    }

    departments {
        uuid id PK
        varchar name
        text description
        boolean is_active
        uuid organisation_id FK
        text created_by
        text updated_by
        timestamp deleted_at
        timestamp created_at
        timestamp updated_at
    }

    roles {
        uuid id PK
        varchar name
    }

    users {
        uuid id PK
        varchar username "unique per org"
        text first_name
        text last_name
        text password_hash
        text temp_password
        varchar email_id "unique per org"
        text phone_number
        varchar employee_code
        text address
        varchar date_of_birth
        varchar date_of_joining
        varchar shift_start_time
        varchar shift_end_time
        varchar license_no
        text qualification
        varchar employee_type
        varchar emergency_email
        varchar emergency_name
        varchar emergency_contact
        uuid organisation_id FK
        uuid department_id FK
        uuid role_id FK
        boolean is_active
        int last_login_attempt
        text password_reset_token_hash
        timestamp last_pwd_updated
        timestamp created_at
        timestamp updated_at
    }

    refresh_tokens {
        uuid id PK
        uuid user_id FK
        text token_hash
        timestamp created_at
        timestamp expires_at
    }

    organisations ||--o{ users : "organisation_id"
    organisations ||--o{ departments : "organisation_id"
    departments ||--o{ users : "department_id"
    roles ||--o{ users : "role_id"
    users ||--o{ refresh_tokens : "user_id"
```

Unique indexes: `idx_org_username` (`organisation_id` + `username`), `idx_org_email` (`organisation_id` + `email_id`).

---

## 4. Patients

```mermaid
erDiagram
    organisations {
        uuid id PK
    }

    users {
        uuid id PK
    }

    patients {
        uuid id PK
        text uh_id UK
        text name
        int age
        text gender
        int weight
        text email_id UK
        text mobile_no UK
        varchar blood_group
        timestamp last_visit_date
        uuid created_by FK
        uuid organisation_id FK
        text status
        varchar address
        int total_visit
        timestamp last_visit
        timestamp created_at
        timestamp updated_at
    }

    organisations ||--o{ patients : "organisation_id"
    users ||--o{ patients : "created_by"
```

**Status values:** `pending`, `active` (DB default tag also mentions `free`).

---

## 5. Appointments & schedules

```mermaid
erDiagram
    organisations {
        uuid id PK
    }

    organisation_schedules {
        uuid id PK
        uuid organisation_id FK
        text_array day_of_week
        text start_time
        text end_time
        int slot_duration
        text break_start_time
        text break_end_time
        boolean is_closed
        timestamp created_at
    }

    patients {
        uuid id PK
    }

    users {
        uuid id PK
    }

    appointments {
        uuid id PK
        text appointment_code
        uuid schedule_id FK
        text series_id
        uuid patient_id FK
        uuid doctor_id FK
        uuid organisation_id FK
        timestamp appointment_date
        timestamptz start_time
        timestamptz end_time
        text visit_type
        text status
        uuid created_by FK
        timestamp created_at
    }

    organisations ||--o{ organisation_schedules : has
    organisations ||--o{ appointments : has
    organisation_schedules ||--o{ appointments : "schedule_id"
    patients ||--o{ appointments : "patient_id"
    users ||--o{ appointments : "doctor_id"
    users ||--o{ appointments : "created_by"
```

| Field | Values |
|---|---|
| `status` | `scheduled`, `completed`, `cancelled`, `ongoing`, `upcoming`, `missed`, `waiting`, `reschedule_required` |
| `visit_type` | `new_patient`, `follow_up`, `opd` |
| `day_of_week` | `MON`…`SUN` (`text[]`) |

`series_id` is a plain string — there is no appointment-series table.

---

## 6. Prescriptions

```mermaid
erDiagram
    patients {
        uuid id PK
    }

    appointments {
        uuid id PK
    }

    users {
        uuid id PK
    }

    organisations {
        uuid id PK
    }

    medicines {
        uuid id PK
    }

    prescriptions {
        uuid id PK
        text code
        uuid patient_id FK
        uuid appointment_id FK
        uuid prescribed_by FK
        uuid organisation_id FK
        text status
        timestamp created_at
        timestamp updated_at
    }

    prescription_items {
        uuid id PK
        uuid prescription_id FK
        uuid medicine_id FK
        jsonb frequency
        bigint quantity
        float duration_day
        text duration_type
        text food_instruction
        text status
        boolean out_of_stock
        int balance_after_dispense
        uuid created_by
        timestamp created_at
        timestamp updated_at
    }

    patients ||--o{ prescriptions : "patient_id"
    appointments ||--o{ prescriptions : "appointment_id"
    users ||--o{ prescriptions : "prescribed_by"
    organisations ||--o{ prescriptions : "organisation_id"
    prescriptions ||--o{ prescription_items : "prescription_id CASCADE"
    medicines ||--o{ prescription_items : "medicine_id"
```

**`frequency` JSONB:** `{ morning, afternoon, night }` (floats).

**Item status examples:** `pending`, `fully_dispensed`, `partially_dispensed`.

---

## 7. Billing

```mermaid
erDiagram
    patients {
        uuid id PK
    }

    users {
        uuid id PK
    }

    organisations {
        uuid id PK
    }

    prescriptions {
        uuid id PK
    }

    appointments {
        uuid id PK
    }

    medicines {
        uuid id PK
    }

    medicine_inventories {
        uuid id PK
    }

    prescription_items {
        uuid id PK
    }

    invoices {
        uuid id PK
        text invoice_code
        varchar payment_type
        uuid prescription_id FK "nullable UK"
        uuid appointment_id FK "nullable UK"
        uuid patient_id FK
        text status
        uuid cashier_id FK
        uuid organisation_id FK
        numeric sub_total_amount
        numeric tax_amount
        numeric total_amount
        numeric discount_amount
        timestamp created_at
        timestamp updated_at
    }

    invoice_items {
        uuid id PK
        uuid invoice_id FK
        uuid medicine_id FK
        uuid medicine_inventory_id FK
        uuid prescription_item_id FK
        text batch_no
        numeric sub_total_price
        numeric total_price
        int dispensed_qty
        timestamp created_at
    }

    patients ||--o{ invoices : "patient_id"
    users ||--o{ invoices : "cashier_id"
    organisations ||--o{ invoices : "organisation_id"
    prescriptions |o--o| invoices : "prescription_id"
    appointments |o--o| invoices : "appointment_id"
    invoices ||--o{ invoice_items : "invoice_id"
    medicines ||--o{ invoice_items : "medicine_id"
    medicine_inventories ||--o{ invoice_items : "medicine_inventory_id"
    prescription_items ||--o{ invoice_items : "prescription_item_id"
```

| Field | Values |
|---|---|
| `payment_type` | `consultation`, `prescription` |
| `status` | `unpaid`, `paid` |

Unique indexes on `prescription_id` and `appointment_id` guard double-billing. Consultation invoices typically have no line items.

---

## 8. Payments

```mermaid
erDiagram
    invoices {
        uuid id PK
    }

    patients {
        uuid id PK
    }

    payments {
        uuid id PK
        uuid invoice_id FK
        uuid patient_id FK
        numeric amount
        varchar currency
        varchar source
        varchar channel
        varchar initiated_by
        varchar idempotency_key UK
        timestamp created_at
        timestamp updated_at
    }

    payment_attempts {
        uuid id PK
        uuid payment_id FK
        varchar provider_payment_id
        varchar provider
        int attempt_no
        varchar provider_link_id
        varchar provider_order_id
        timestamptz paid_at
        text payment_link
        jsonb provider_request
        varchar payer_account_type
        varchar payment_vpa
        boolean accept_partial
        numeric amount_paid
        numeric amount_transferred
        text payment_error
        varchar payment_error_code
        varchar provider_reference_id
        varchar payment_link_status
        varchar payment_status
        timestamptz expires_at
        timestamp created_at
    }

    refunds {
        uuid id PK
        uuid payment_attempt_id FK
        varchar provider_refund_id
        numeric amount
        text reason
        varchar status
        jsonb provider_data
        jsonb gateway_response
        timestamp created_at
    }

    webhook_events {
        uuid id PK
        uuid payment_attempt_id FK
        varchar event_type
        jsonb provider_response
        timestamp created_at
        timestamp updated_at
    }

    invoices ||--o{ payments : "invoice_id"
    patients ||--o{ payments : "patient_id"
    payments ||--o{ payment_attempts : "payment_id"
    payment_attempts ||--o{ refunds : "payment_attempt_id"
    payment_attempts ||--o{ webhook_events : "payment_attempt_id"
```

Defaults: `currency=INR`, `source=link`, `channel=upi`.

---

## 9. Medicine, inventory & suppliers

```mermaid
erDiagram
    organisations {
        uuid id PK
    }

    users {
        uuid id PK
    }

    suppliers {
        uuid id PK
        varchar supplier_code "unique per org"
        varchar name
        varchar contact_number
        varchar email
        varchar address
        uuid organisation_id FK
        varchar gst_number
        varchar drug_license_no
        varchar payment_terms
        varchar supplier_status
        numeric credit_limit
        uuid created_by FK
        timestamp created_at
        timestamp updated_at
    }

    m_purchase_entries {
        uuid id PK
        varchar invoice_number
        date invoice_date
        uuid supplier_id FK
        date payment_due_date
        uuid organisation_id FK
        timestamp created_at
    }

    medicines {
        uuid id PK
        varchar code UK
        varchar name
        varchar form
        varchar strength
        boolean is_active
        uuid created_by FK
        uuid organisation_id FK
        text hsn_code
        int reorder_level
        int max_stock_target
        timestamp created_at
        timestamp updated_at
    }

    medicine_inventories {
        uuid id PK
        uuid medicine_id FK
        uuid supplier_id FK
        varchar batch_no
        date expires_at
        uuid organisation_id FK
        uuid created_by FK
        uuid purchase_entry_id FK
        int purchase_qty_boxes
        int units_per_box
        int current_stock_units
        varchar shelf_location
        jsonb medicine_pricing
        timestamp created_at
        timestamp updated_at
    }

    medicine_stock_movements {
        uuid id PK
        uuid medicine_id FK
        uuid medicine_inventory_id FK
        uuid organisation_id FK
        varchar movement_type
        int qty_changed
        uuid created_by FK
        varchar source_type
        numeric unit_price_at_time_of_mvmt
        int balance_after_mvmt
        timestamp created_at
        timestamp updated_at
    }

    organisations ||--o{ suppliers : has
    organisations ||--o{ medicines : has
    organisations ||--o{ m_purchase_entries : has
    users ||--o{ suppliers : "created_by"
    users ||--o{ medicines : "created_by"
    suppliers ||--o{ m_purchase_entries : "supplier_id"
    suppliers ||--o{ medicine_inventories : "supplier_id"
    m_purchase_entries ||--o{ medicine_inventories : "purchase_entry_id"
    medicines ||--o{ medicine_inventories : "medicine_id"
    medicines ||--o{ medicine_stock_movements : "medicine_id"
    medicine_inventories ||--o{ medicine_stock_movements : "medicine_inventory_id"
```

| Field | Values |
|---|---|
| `payment_terms` | Net 30, Net 15, Net 45, Advance, Cash |
| `supplier_status` | Active, Inactive |
| `movement_type` | sale, purchase, return, dispense |
| `source_type` | purchase_entry, patient_medicine_order |

**`medicine_pricing` JSONB:** `mrp`, `unit_price`, `discount`, `purchase_price`, `selling_price`.

---

## 10. Notifications

```mermaid
erDiagram
    organisations {
        uuid id PK
    }

    patients {
        uuid id PK
    }

    users {
        uuid id PK
    }

    notifications {
        uuid id PK
        uuid organisation_id FK
        uuid patient_id FK "nullable"
        uuid employee_id FK "nullable"
        text notification_type
        text status
        int retry_count
        timestamp next_retry_at
        text last_error
        timestamp sent_at
        jsonb p_payload
        timestamp created_at
        timestamp updated_at
    }

    notification_attempts {
        uuid id PK
        uuid notification_id FK
        int attempt_no
        text status
        text error_msg
        timestamp created_at
    }

    organisations ||--o{ notifications : has
    patients ||--o{ notifications : "patient_id"
    users ||--o{ notifications : "employee_id"
    notifications ||--o{ notification_attempts : "notification_id"
```

**Status flow:** `PENDING` → `PROCESSING` → `SENT` / `FAILED` / `PERMANENTLY_FAILED`.

**`p_payload` JSONB:** `content`, `channel` (e.g. EMAIL), `subject`, `recipient_email`.

---

## Entity checklist (migrated)

| # | Entity | Table (GORM default) |
|---|---|---|
| 1 | Organisation | `organisations` |
| 2 | License | `licenses` |
| 3 | User | `users` |
| 4 | RefreshToken | `refresh_tokens` |
| 5 | Patient | `patients` |
| 6 | Department | `departments` |
| 7 | Role | `roles` |
| 8 | Permission | `permissions` |
| 9 | RolePermission | `role_permissions` |
| 10 | Modules | `modules` |
| 11 | OrganisationSchedule | `organisation_schedules` |
| 12 | Appointment | `appointments` |
| 13 | Prescription | `prescriptions` |
| 14 | PrescriptionItems | `prescription_items` |
| 15 | Invoice | `invoices` |
| 16 | InvoiceItem | `invoice_items` |
| 17 | Payments | `payments` |
| 18 | PaymentAttempts | `payment_attempts` |
| 19 | Refunds | `refunds` |
| 20 | WebhookEvents | `webhook_events` |
| 21 | Supplier | `suppliers` |
| 22 | MPurchaseEntry | `m_purchase_entries` |
| 23 | Medicine | `medicines` |
| 24 | MedicineInventory | `medicine_inventories` |
| 25 | MedicineStockMovements | `medicine_stock_movements` |
| 26 | Notification | `notifications` |
| 27 | NotificationAttempts | `notification_attempts` |

**Not persisted as tables:** `dashboard` (aggregation), `authentication` (uses `users`), email/Razorpay DTO packages. Bed management (`room_types`, `rooms`, `room_summaries`, `beds`, `bed_allotments`) exists in code but is out of scope for this diagram.

---

## Core clinical flow (relationship path)

```text
Organisation
  └─ Patient
       └─ Appointment ──(doctor)──► User
            └─ Prescription ──(prescribed_by)──► User
                 ├─ PrescriptionItems ──► Medicine
                 │                         └─ MedicineInventory ──► Supplier / PurchaseEntry
                 └─ Invoice (prescription | consultation)
                      ├─ InvoiceItems ──► MedicineInventory
                      └─ Payments
                           └─ PaymentAttempts ──► Refunds / WebhookEvents
```
