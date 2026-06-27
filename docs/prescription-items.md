# Finalized Prescription Schema Design

## Objective

Replace the existing `medicines JSONB` field in the `prescriptions` table with a normalized relational model that supports:

* Multi-tenancy
* Prescription history
* Reporting and analytics
* Future pharmacy module
* Future insurance integrations
* Future AI/RAG capabilities
* Compliance and auditability

---

# Entity Relationship

```text
Patient
   |
   |
Consultation
   |
   |
Prescription (1)
   |
   +----< Prescription Item (Many)
                     |
                     |
                 Medicine
```

---

# Table: prescriptions

Stores prescription-level information.

```sql
prescriptions
-------------
id UUID PRIMARY KEY

organisation_id UUID NOT NULL

patient_id UUID NOT NULL

doctor_id UUID NOT NULL

consultation_id UUID NOT NULL

diagnosis TEXT

notes TEXT

version INTEGER DEFAULT 1

created_by UUID
updated_by UUID

created_at TIMESTAMP
updated_at TIMESTAMP
deleted_at TIMESTAMP NULL
```

## Notes

* One prescription belongs to one consultation.
* One prescription belongs to one patient.
* One prescription can contain multiple medicines.
* Version field reserved for future prescription revisions.
* Soft delete supported through `deleted_at`.

---

# Table: medicines

Master medicine catalog.

```sql
medicines
---------
id UUID PRIMARY KEY

name VARCHAR(255) NOT NULL

generic_name VARCHAR(255)

strength VARCHAR(100)

manufacturer VARCHAR(255)

created_at TIMESTAMP

updated_at TIMESTAMP
```

## Notes

* Shared medicine catalog across all organizations.
* Prevents duplicate medicine definitions.
* Supports autocomplete and pharmacy modules.
* Can later integrate with external medicine databases.

---

# Table: prescription_items

Stores individual medicines prescribed within a prescription.

```sql
prescription_items
------------------
id UUID PRIMARY KEY

organisation_id UUID NOT NULL

prescription_id UUID NOT NULL

medicine_id UUID NULL

medicine_name VARCHAR(255) NOT NULL

strength VARCHAR(100)

dosage VARCHAR(100)

frequency_code VARCHAR(20)

duration_value INTEGER

duration_unit VARCHAR(20)

quantity INTEGER

instructions TEXT

status VARCHAR(50)

created_at TIMESTAMP

updated_at TIMESTAMP

deleted_at TIMESTAMP NULL
```

---

# Snapshot Fields

The following fields intentionally duplicate data from the medicines table:

```sql
medicine_name
strength
```

## Reason

Prescriptions are legal medical records.

Example:

2026:

```text
Paracetamol 500mg
```

2030:

Medicine catalog updated.

```text
Paracetamol 650mg
```

Old prescriptions must still display:

```text
Paracetamol 500mg
```

Therefore prescription records maintain a historical snapshot.

---

# Frequency Codes

Standard values:

```text
OD  = Once Daily

BD  = Twice Daily

TDS = Three Times Daily

QID = Four Times Daily

SOS = As Needed
```

Examples:

```text
Paracetamol
Frequency: BD

Amoxicillin
Frequency: TDS
```

---

# Duration Structure

Instead of storing:

```text
5 Days
```

Store:

```sql
duration_value = 5
duration_unit  = DAYS
```

Supported units:

```text
DAYS
WEEKS
MONTHS
```

Benefits:

* Easier reporting
* Medication reminders
* Future AI recommendations

---

# Status Values

```text
ACTIVE

COMPLETED

DISCONTINUED

CANCELLED
```

Initial status:

```text
ACTIVE
```

---

# Index Recommendations

## prescriptions

```sql
idx_prescriptions_org
(organisation_id)

idx_prescriptions_patient
(patient_id)

idx_prescriptions_doctor
(doctor_id)

idx_prescriptions_consultation
(consultation_id)
```

## prescription_items

```sql
idx_prescription_items_org
(organisation_id)

idx_prescription_items_prescription
(prescription_id)

idx_prescription_items_medicine
(medicine_id)
```

---

# Migration Strategy

## Current Structure

```sql
prescriptions
-------------
id
...
medicines JSONB
```

Example:

```json
[
  {
    "medicine_name": "Paracetamol",
    "dosage": "500mg",
    "frequency": "BD",
    "duration": "5 Days"
  }
]
```

---

## Migration Steps

### Step 1

Create:

* medicines
* prescription_items

tables.

### Step 2

Update prescription creation flow.

Old:

```text
Create Prescription
      ↓
Store medicines JSONB
```

New:

```text
Create Prescription
      ↓
Insert Prescription
      ↓
Insert Prescription Items
```

### Step 3

Backfill historical data.

For each prescription:

```text
Read medicines JSONB
      ↓
Create medicine record if required
      ↓
Insert prescription_items
      ↓
Validate migration
```

### Step 4

Update all read APIs to use:

```sql
prescriptions
JOIN prescription_items
LEFT JOIN medicines
```

### Step 5

Remove JSONB dependency.

### Step 6

Drop medicines JSONB column after validation.

---

# Final Decision

Approved tables:

1. prescriptions
2. medicines
3. prescription_items

Key architectural decisions:

* Multi-tenant support through organisation_id
* Historical medicine snapshots preserved
* Structured duration and frequency
* Soft delete support
* Audit-friendly design
* Future-ready for pharmacy, insurance, and AI/RAG modules
* Elimination of medicines JSONB storage
