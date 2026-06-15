# Notification Integration Tasks

## Objective

Integrate the notification package across all applicable workflows and ensure notification attempts are tracked for both successful and failed deliveries.

---

## General Implementation Steps

### 1. Identify Notification Injection Points

Review all business workflows and identify locations where notifications should be triggered.

Current scope:

* Patient Created
* Patient Updated
* Appointment Created
* Appointment Updated
* Prescription Created
* Other existing notification events

---

### 2. Add Notification Type Constants

Create notification type constants in the models package.

Reference:

* Appointment notification constants already implemented in appointment models.

Follow the same pattern for all notification types currently supported by the system.

Example:

```go
const (
    NotificationTypePatientCreated   = "patient_created"
    NotificationTypePatientUpdated   = "patient_updated"
    NotificationTypeAppointmentCreated = "appointment_created"
    NotificationTypeAppointmentUpdated = "appointment_updated"
)
```

---

### 3. Fetch Notification Data

Before triggering notifications:

* Fetch all required notification data from the database.
* Use JOIN queries wherever necessary.
* Refer to `database-operations.md` for query patterns and repository implementation guidelines.

Notification payload should contain all fields required by the corresponding notification template.

---

### 4. Trigger Notification

After successful business operation:

```go
notification.Create(
    context.Background(),
    notificationData,
)
```

Guidelines:

* Trigger notification only after successful completion of the operation.
* Use `context.Background()` while creating notifications.
* Ensure all required template fields are populated.

---

### 5. Update Parse Function

Inside the notification package:

* Review the parse function implementation.
* Add mappings for any newly introduced template variables.
* Ensure all placeholders used by templates are supported.

Example:

```text
{{patient_name}}
{{hospital_name}}
{{appointment_code}}
```

must be populated during parsing.

---

### 6. Notification Attempt Tracking

Create notification attempt entries for:

* Successful notifications
* Failed notifications

Required tracking information:

* Notification ID
* Notification Type
* Delivery Channel
* Status
* Failure Reason (if applicable)
* Attempt Timestamp

Statuses:

```go
const (
    NotificationStatusSent   = "sent"
    NotificationStatusFailed = "failed"
)
```

Every notification processing attempt must create an attempt record.

---

## Template Reference

Notification templates are located under:

```text
notifications/templates/
```

Before implementation:

1. Review the corresponding template.
2. Identify all required fields.
3. Ensure repository query returns those fields.
4. Validate parser support for placeholders.

---

## Current Scope

### Implement Now

* Patient Created
* Patient Updated
* Appointment Created
* Appointment Updated
* Prescription Created
* Existing basic notifications already supported by templates

---

### Deferred Implementation

The following notifications require additional business logic and will be implemented later:

* Medicine Adherence Reminder
* Follow-up Reminder

Only template and notification type preparation can be done at this stage.

Business scheduling logic will be added separately.

---

## Tasks To Be Completed

### High Priority

* [ ] Add notification support for Patient Created
* [ ] Add notification support for Patient Updated
* [ ] Add notification support for Appointment Updated
* [ ] Verify existing Appointment Created notification implementation
* [ ] Add notification type constants
* [ ] Add notification attempt tracking
* [ ] Validate template parser mappings
* [ ] Verify all required data joins

### Future Tasks

* [ ] Implement Medicine Adherence workflow
* [ ] Implement Follow-up Reminder workflow

---

## Implementation Flow

```text
Business Operation
        ↓
Fetch Notification Data
        ↓
Build Notification Payload
        ↓
notification.Create()
        ↓
Parse Template
        ↓
Send Notification
        ↓
Create Attempt Record
        ↓
Mark Sent / Failed
```
