package docs

# GORM Usage Guidelines

## Purpose

This document defines when to use various GORM functionalities and why. The goal is to maintain consistency across the codebase and avoid confusion when multiple approaches can achieve the same result.

---

# General Principles

## Prefer GORM APIs Over Raw SQL

### Good

```go
db.Where("id = ?", id).First(&user)
```

### Avoid

```go
db.Raw("SELECT * FROM users WHERE id = ?", id).Scan(&user)
```

### Why?

* Database agnostic
* Easier to maintain
* Easier to read
* Leverages GORM features
* Less chance of SQL syntax errors

### Exception

Use Raw SQL when:

* Complex joins become difficult to express in GORM
* Window functions are required
* Recursive queries are required
* Database-specific functions are required
* Performance optimization requires handcrafted SQL

---

# Fetching Records

## Find()

### Use When

Fetching multiple records.

```go
var appointments []Appointment

err := db.
    Where("patient_id = ?", patientID).
    Find(&appointments).
    Error
```

### Why

* Intended for model retrieval
* Automatically maps model fields
* Most readable approach

### Alternative

```go
Scan()
```

### Why Not Scan?

`Scan()` is intended for custom result structures.

---

## First()

### Use When

Fetching exactly one record.

```go
var appointment Appointment

err := db.
    Where("id = ?", id).
    First(&appointment).
    Error
```

### Behavior

Adds:

```sql
LIMIT 1
```

Returns:

```go
gorm.ErrRecordNotFound
```

when record does not exist.

---

## Take()

### Use When

Any single record is acceptable.

```go
db.Take(&appointment)
```

### Difference from First()

First():

```sql
ORDER BY id ASC LIMIT 1
```

Take():

```sql
LIMIT 1
```

No ordering.

---

## Last()

### Use When

Need the last record.

```go
db.Last(&appointment)
```

Equivalent:

```sql
ORDER BY id DESC LIMIT 1
```

---

# Scan()

## Use When

Result is NOT a model.

Example DTO:

```go
type AppointmentPreview struct {
    Name string
    Age  int
}
```

```go
db.Raw(query).
    Scan(&response)
```

or

```go
db.Table("patients").
    Select("name, age").
    Scan(&response)
```

### Why

Used for:

* DTOs
* Aggregations
* Custom projections
* Joins

---

# Raw()

## Use When

GORM query builder becomes difficult.

Example:

```go
query := `
SELECT
    a.id,
    p.name
FROM appointments a
JOIN patients p ON p.id = a.patient_id
`

db.Raw(query).Scan(&response)
```

### Why

Provides full SQL control.

### Avoid When

Simple CRUD operations.

Bad:

```go
db.Raw("SELECT * FROM appointments")
```

Good:

```go
db.Find(&appointments)
```

---

# Create Operations

## Create()

### Use When

Creating a single record.

```go
db.Create(&appointment)
```

---

## Create In Batch

```go
db.Create(&appointments)
```

Where:

```go
appointments []Appointment
```

### Why

More efficient than inserting one-by-one.

---

# Updates

## Update()

### Use When

Updating one field.

```go
db.Model(&appointment).
    Update("status", "completed")
```

---

## Updates()

### Use When

Updating multiple fields.

```go
db.Model(&appointment).
    Updates(map[string]interface{}{
        "status": "completed",
        "notes":  "Done",
    })
```

---

## Save()

### Avoid Generally

```go
db.Save(&appointment)
```

### Why

Save:

* Updates all fields
* Can unintentionally overwrite values
* Harder to control

Prefer:

```go
Updates()
```

---

# Delete Operations

## Delete()

```go
db.Delete(&appointment)
```

---

## Conditional Delete

```go
db.Where("status = ?", "cancelled").
    Delete(&Appointment{})
```

---

# Filtering

## Where()

Primary filtering method.

```go
db.Where("patient_id = ?", patientID)
```

---

## Multiple Conditions

```go
db.Where(
    "patient_id = ? AND organisation_id = ?",
    patientID,
    organisationID,
)
```

---

## Struct Conditions

```go
db.Where(&Appointment{
    PatientID: patientID,
})
```

### Limitation

Zero values are ignored.

Example:

```go
Age = 0
```

will not be included.

---

# Pagination

## Correct

```go
db.
    Order("created_at DESC").
    Limit(limit).
    Offset(offset).
    Find(&appointments)
```

### Why

Pagination without ordering can produce inconsistent results.

---

# Counting Records

## Count()

```go
var total int64

db.Model(&Appointment{}).
    Where("patient_id = ?", patientID).
    Count(&total)
```

### Why

Efficient count query.

### Avoid

```go
db.Find(&appointments)
len(appointments)
```

---

# Transactions

## Use When

Multiple operations must succeed together.

```go
tx := db.Begin()
```

Example:

```go
tx := db.Begin()

if err := tx.Create(&prescription).Error; err != nil {
    tx.Rollback()
    return err
}

if err := tx.Create(&medicine).Error; err != nil {
    tx.Rollback()
    return err
}

tx.Commit()
```

---

# Joins

## Use GORM Join

```go
db.
    Table("appointments a").
    Joins("JOIN patients p ON p.id = a.patient_id")
```

### Use When

Simple joins.

---

## Use Raw SQL

When:

* Multiple joins
* Aggregations
* Subqueries
* Complex reporting queries

---

# Preload()

## Use When

Loading relationships.

Example:

```go
db.Preload("Patient").
    Find(&appointments)
```

### Why

Avoids manual joins.

---

# Select()

## Use When

Need only specific columns.

```go
db.
    Select("id, name").
    Find(&patients)
```

### Why

Reduces data transfer.

---

# Pluck()

## Use When

Fetching one column.

```go
var names []string

db.Model(&Patient{}).
    Pluck("name", &names)
```

### Why

Cleaner than Find + Loop.

---

# Exists Check

## Preferred

```go
var count int64

db.Model(&Appointment{}).
    Where("id = ?", id).
    Count(&count)

exists := count > 0
```

---

# Error Handling

Always check:

```go
if err != nil {
    return err
}
```

For single records:

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil
}
```

---

# Soft Delete

If model contains:

```go
gorm.DeletedAt
```

then:

```go
db.Delete(&user)
```

does NOT physically delete.

It sets:

```sql
deleted_at = CURRENT_TIMESTAMP
```

---

----------
# COALESCE
the COALESCE function evaluates a list of arguments from left to right and 
returns the first non-null value it encounters. If all arguments are null, it returns null.

```sql

SELECT 
    name, 
    COALESCE(email, 'No Email Provided') as contact_email
FROM patients;

```
How it works: If a patient has an email, it shows their email. If their email is NULL, it displays "No Email Provided".

```sql
COALESCE(
    (SELECT json_agg(...) FROM ...), 
    '[]'::json -- Fallback
) AS active_batches

```
If we don't use COALESCE, the API sends "active_batches": null to the browser, which will crash your frontend React/Angular code when it tries to run .map() or .length. By falling back to '[]'::json, the API cleanly sends "active_batches": [], which the frontend can render safely.

------------

# Decision Matrix

| Requirement           | Recommended            |
| --------------------- | ---------------------- |
| Single Record         | First()                |
| Any Single Record     | Take()                 |
| Multiple Records      | Find()                 |
| DTO Response          | Scan()                 |
| Complex Query         | Raw()                  |
| Count Records         | Count()                |
| One Column            | Pluck()                |
| Relationships         | Preload()              |
| One Field Update      | Update()               |
| Multiple Field Update | Updates()              |
| Create Record         | Create()               |
| Transaction           | Begin/Commit/Rollback  |
| Pagination            | Limit + Offset + Order |
| Soft Delete           | Delete()               |
| Specific Columns      | Select()               |

---

# Golden Rule

1. Use GORM APIs whenever possible.
2. Use Raw SQL only when query complexity justifies it.
3. Use Find() for models.
4. Use Scan() for DTOs.
5. Always paginate with Order().
6. Prefer Updates() over Save().
7. Use transactions for related writes.
8. Keep repository code consistent across the project.
