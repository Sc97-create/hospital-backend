# Organisation Schedule Logging Plan

Implemented. Package: `internal/admins`  
Route: `POST /api/v1/admins/organisationSchedule/create`

Internal: `GetScheduleByOrganisationID` used by appointments (create + slots).

## Layer logging

| Layer | What we log |
|-------|-------------|
| **Controller** | Entry / invalid request; HTTP mapping |
| **Service** | Create/get outcomes + stable `reason` |
| **Repo** | DB errors only; empty Scan → not found |

## Security

Safe: `organisation_id`, `schedule_id`, `slot_duration`, `week_off_count`, `is_closed`.  
Do not dump full week-off day arrays on INFO unless needed (count is enough).

## Behavior fix

`GetScheduleByOrganisationID` previously returned **empty + nil error** on failure. It now returns `ErrOrgScheduleNotFound` / `ErrOrgScheduleFetchFailed` so appointment flows classify correctly.

## HTTP mapping (create)

| Case | HTTP | Sentinel |
|------|------|----------|
| Missing / invalid fields | `400` | `invalid request` |
| DB | `500` | `failed to create organisation schedule` |

## Gaps

| Item | Status |
|------|--------|
| Domain logs + sentinels | Done |
| Empty Scan → not found | Done |
| Logger threaded into appointment callers | Done |
| Client-safe HTTP mapping on create | Done |
| JWT | Not present (follow-up) |
| Update/get HTTP endpoints | Not present (create-only API today) |
