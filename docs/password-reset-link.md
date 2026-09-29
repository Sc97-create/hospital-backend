# Password reset link API

## Summary

This pass implements **request reset link + email**, **complete reset via public `updatePassword`**, and **first-login password change via JWT `updatePasswordFirstLogin`**.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| `POST` | `/api/v1/authentication/requestPasswordReset` | Public (no JWT / RBAC) |
| `PATCH` | `/api/v1/authentication/updatePassword` | Public (no JWT / RBAC) — reset via token only |
| `PATCH` | `/api/v1/authentication/updatePasswordFirstLogin` | JWT required, RBAC-public — temp-password users only |

### Request password reset

#### Request body

```json
{
  "email_id": "staff@example.com"
}
```

#### Responses

**200 — accepted** (also returned when the email is unknown, to avoid account enumeration):

```json
{
  "message": "If an account exists for that email, a reset link has been sent",
  "code": "200"
}
```

**400** — missing / invalid `email_id`

**429** — another reset was requested (or password was updated) less than 15 minutes ago (`last_pwd_updated`)

**500** — lookup / token persist / email enqueue failure

### Complete password reset (`updatePassword`)

#### Request body

```json
{
  "token": "<plain_token_from_email_link>",
  "password": "newpassword",
  "confirm_password": "newpassword"
}
```

#### Responses

**200** — password updated (reset token cleared, `last_pwd_updated` set)

**400** — missing fields, weak/mismatched password, or invalid token

**500** — persist failure

### First-login password change (`updatePasswordFirstLogin`)

Used after login with a temporary password (`password_cleared: true` in the login response). Requires `Authorization: Bearer <access_token>`. Rejects if the user already has a permanent `password_hash`.

#### Request body

```json
{
  "password": "newpassword",
  "confirm_password": "newpassword"
}
```

#### Responses

**200** — password updated (`temp_password` cleared)

**400** — validation failure or password already set

**401** — missing / invalid JWT

**500** — persist failure

---

## Behaviour (request link)

1. Normalize `email_id` (trim + lowercase).
2. Look up `users` by `email_id`.
3. If no user → return success (no email sent).
4. If `last_pwd_updated` is set and age &lt; 15 minutes → `ErrPasswordResetTooSoon` (HTTP 429); token is **not** rotated and no email is sent.
5. Generate a 32-byte random token (hex). Persist  
   `SHA-256(plain_token)` as `password_reset_token_hash`, and set `last_pwd_updated = now`.
6. Build frontend URL:

   `http://localhost:5173/forgot-password/reset?token=<plain_token>`

   Base URL comes from `PASSWORD_RESET_BASE_URL` (default `http://localhost:5173`).
7. Enqueue notification type `password_reset_requested` with subject `Reset your password`.

Email template fields:

| Field | Source |
|-------|--------|
| Employee name | `first_name` + `last_name` (falls back to email) |
| Employee email | `email_id` |
| Reset URL | link above |
| Cooldown minutes | `15` |

## Behaviour (complete reset)

1. Require `token`, `password`, `confirm_password`.
2. Validate password length (≥ 8) and confirmation match.
3. Hash token and look up user by `password_reset_token_hash`.
4. On match: update `password_hash`, clear `password_reset_token_hash` and `temp_password`, set `last_pwd_updated = NOW()`.

## Behaviour (first-login)

1. Require JWT; resolve `user_id` from access token.
2. Validate password length (≥ 8) and confirmation match.
3. Load user; require empty `password_hash` (permanent password not set yet).
4. Update `password_hash`, clear `temp_password` / reset token, set `last_pwd_updated = NOW()`.

---

## Schema (`users` via AutoMigrate)

| Column | Type | Purpose |
|--------|------|---------|
| `password_reset_token_hash` | text | Stored hash of reset token (never the raw token) |
| `last_pwd_updated` | timestamp | Cooldown clock; also refreshed on successful password update |

Successful password updates clear `password_reset_token_hash` and set `last_pwd_updated = NOW()`.

---

## Config

| Env | Default | Meaning |
|-----|---------|---------|
| `PASSWORD_RESET_BASE_URL` | `http://localhost:5173` | Frontend origin for the reset link |

SMTP settings are unchanged (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`).

---

## Not in this change

- Token expiry beyond the 15-minute re-request cooldown (raw token remains usable until overwritten / password updated)
