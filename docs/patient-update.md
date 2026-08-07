# Patient Update API - Implementation Design

## Objective

Implement patient update functionality using dynamic update query generation.

Only fields that have changed should be updated.

The application follows the existing architecture:

```text
Controller
    ↓
Service
    ↓
Repository
    ↓
GORM
    ↓
PostgreSQL
```

---

# Controller Layer

Create API:

```http
PUT /patients/{patientId}
```

Mandatory fields:

* patientId
* organisationId

Reuse the existing CreatePatient request model.

---

# Service Flow

```text
UpdatePatient
      ↓
Validate Request
      ↓
GetPatientByID
      ↓
Compare Existing vs Incoming Data
      ↓
Generate Dynamic Update Query
      ↓
Repository.UpdatePatient
      ↓
Trigger Notification
      ↓
Return Response
```

---

# Existing Repository Method

Use existing repository method:

```go
GetPatientByID(
    ctx context.Context,
    patientID string,
    organisationID string,
) (*models.Patient, error)
```

Purpose:

* Fetch current patient data.
* Used for comparison against incoming request.

---

# Service Layer Responsibilities

The Service layer is responsible for:

* Fetching existing patient.
* Comparing old and new values.
* Building update query.
* Building query arguments.
* Calling repository for execution.

Repository should only execute the generated query.

---

# Dynamic Query Generation

Initialize:

```go
baseQuery := `UPDATE patients SET `
```

Variables:

```go
var (
    setClauses []string
    args       []interface{}
    argPos     = 1
)
```

---

# Compare Existing vs New Values

Example:

```go
if patient.FirstName != req.FirstName {
    setClauses = append(
        setClauses,
        fmt.Sprintf("first_name = $%d", argPos),
    )

    args = append(args, req.FirstName)
    argPos++
}

if patient.MobileNumber != req.MobileNumber {
    setClauses = append(
        setClauses,
        fmt.Sprintf("mobile_number = $%d", argPos),
    )

    args = append(args, req.MobileNumber)
    argPos++
}
```

Repeat for all supported patient fields.

---

# Updated Timestamp

Always update:

```go
setClauses = append(
    setClauses,
    "updated_at = NOW()",
)
```

---

# Build Final Query

```go
baseQuery += strings.Join(
    setClauses,
    ", ",
)
```

Append WHERE clause:

```go
baseQuery += fmt.Sprintf(`
WHERE patient_id = $%d
AND organisation_id = $%d
`,
argPos,
argPos+1,
)
```

Append arguments:

```go
args = append(
    args,
    patientID,
    organisationID,
)
```

---

# Example Generated Query

```sql
UPDATE patients
SET
    first_name = $1,
    mobile_number = $2,
    updated_at = NOW()
WHERE patient_id = $3
AND organisation_id = $4
```

Arguments:

```go
[]interface{}{
    "Sachin",
    "9999999999",
    patientID,
    organisationID,
}
```

---

# Repository Layer

Repository should not perform any comparison logic.

Repository responsibility:

```go
UpdatePatient(
    ctx context.Context,
    query string,
    args []interface{},
) error
```

Implementation:

```go
func (r *PatientRepository) UpdatePatient(
    ctx context.Context,
    query string,
    args []interface{},
) error {

    return r.db.WithContext(ctx).
        Exec(query, args...).
        Error
}
```

Since GORM is already being used, execute the generated PostgreSQL query through:

```go
db.Exec(query, args...)
```

No query generation should exist inside repository.

---

# No Change Scenario

Before repository call:

```go
if len(setClauses) == 1 {
    return errors.New("no fields updated")
}
```

Reason:

Only `updated_at` exists and no business fields changed.

Do not execute update query.

---

# Notification Integration

After successful update:

1. Fetch notification data.
2. Trigger Patient Updated notification.

Example:

```go
notification.Create(
    context.Background(),
    notificationData,
)
```

Notification Type:

```go
models.NotificationTypePatientUpdated
```

---

# Tasks

* [ ] Create UpdatePatient API
* [ ] Reuse CreatePatient request model
* [ ] Validate patientId
* [ ] Validate organisationId
* [ ] Fetch patient using GetPatientByID
* [ ] Compare existing and incoming values
* [ ] Generate dynamic PostgreSQL update query in service
* [ ] Generate dynamic argument list
* [ ] Execute query using repository
* [ ] Trigger Patient Updated notification
* [ ] Add notification attempt tracking
* [ ] Add unit tests
* [ ] Add integration tests
