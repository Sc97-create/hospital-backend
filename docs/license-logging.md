# License Logging Plan

Implemented. Package: `internal/license`  
Route: `PATCH /api/v1/license/verifylicense/:organisationID`

Also: `CreateLicenseSrv` during organisation signup (caller: `organisation` service).

## Layer logging

| Layer | What we log |
|-------|-------------|
| **Controller** | Entry / invalid request; HTTP mapping |
| **Service** | Verify/create outcomes + stable `reason` |
| **Repo** | DB errors only |

## Security

**Never log** `license_key` (full or truncated). Safe: `organisation_id`, `license_id`, `planspan`, `planday`, `reason`.

## HTTP mapping (verify)

| Case | HTTP | Sentinel |
|------|------|----------|
| Missing org / key | `400` | `invalid request` |
| Not found / invalid key / format / expiry mismatch | `401` | `license not found` / `invalid license` |
| DB | `500` | `failed to verify license` |

## Gaps

| Item | Status |
|------|--------|
| Domain logs + sentinels | Done |
| Never log license key | Done |
| Client-safe HTTP mapping | Done |
| JWT on verify | Not present (follow-up) |
