# Authentication Logging Plan

Review doc for what to log in the authentication flow and where. No implementation yet — approve / adjust this list, then we instrument.

Builds on the existing logging stack (`pkg/logger`, `RequestLogger` middleware, `middleware.GetLogger`). Request-level `request started` / `request completed` / `client error` / `request failed` already cover HTTP envelope logs. This plan covers **domain auth events** on top of that.

---

## 1. Scope

| Area | Endpoints / entry points | Files |
|------|--------------------------|-------|
| Login | `POST /v1/authentication/login` | `controllers.go`, `services.go`, `repository.go` |
| Refresh | `POST /v1/authentication/refresh` | `controllers.go`, `services.go` + `internal/jwt` |
| Logout | `POST /v1/authentication/logout` | `controllers.go`, `services.go` + `internal/jwt` |
| Access-token gate | `Authenticate` middleware on protected routes | `pkg/middleware/functions.go`, `internal/jwt` |

**Out of scope for this pass:** employee signup / password reset (not present yet), Fiber’s built-in HTTP logger (already registered separately).

---

## 2. Security rules (non-negotiable)

Never log:

- Passwords or password hashes
- Access tokens or refresh tokens (full JWT strings)
- Cookie values containing tokens
- Authorization header raw value

Safe to log:

- `user_id`, `organisation_id` (after lookup / after successful auth)
- Username / email used for login (**only** on failure paths if useful for support; prefer hashing or truncating if PII policy is strict — default proposal: log username on failed login at WARN, omit on success or log `user_id` only)
- Token metadata: `jti` (refresh ID), expiry reason (`expired` / `invalid` / `missing`)
- Boolean outcomes: `has_refresh_cookie`, `token_rotated`, etc.

---

## 3. Propagation approach

Per `docs/logging.md`:

```
RequestLogger (request_id, method, path, ip)
  → Controller: middleware.GetLogger(c)
  → Service: accept *zap.Logger (or fiber.Ctx and GetLogger) as first arg
  → JwtService: accept *zap.Logger on methods called from auth
  → Repository: log only on DB errors (optional Debug on miss)
```

**Decision to confirm:** pass `*zap.Logger` into service/JWT methods vs pass `*fiber.Ctx`. Recommendation: pass `*zap.Logger` so services stay HTTP-agnostic and remain testable.

Controllers already have a partial start (`Login request received`). Align all three handlers to the same pattern.

---

## 4. What to log — by flow

### 4.1 Login (`POST /authentication/login`)

| # | Layer | When | Level | Message (suggested) | Fields |
|---|--------|------|-------|---------------------|--------|
| L1 | Controller | Request entered (after payload parse OK) | INFO | `login attempt` | `username` |
| L2 | Controller | Payload / field parse failure | WARN | `login request invalid` | `error`, missing field name if known |
| L3 | Service | User not found | WARN | `login failed` | `username`, `reason=user_not_found` |
| L4 | Service | Password mismatch | WARN | `login failed` | `username` or `user_id`, `reason=invalid_credentials` |
| L5 | Service | Access token generation failed | ERROR | `login failed` | `user_id`, `org_id`, `reason=access_token`, `error` |
| L6 | Service | Refresh token create/update/save failed | ERROR | `login failed` | `user_id`, `org_id`, `reason=refresh_token`, `error` |
| L7 | Service | `UpdateLastLoginAttempt` failed | ERROR | `login failed` | `user_id`, `reason=last_login_update`, `error` |
| L8 | Service | Success | INFO | `login success` | `user_id`, `organisation_id`, `refresh_action=create\|update` |
| L9 | Repository | DB error on `GetUserID` / `UpdateLastLoginAttempt` | ERROR | `auth repo error` | `op`, `error` (no password) |

**Notes**

- Do **not** log bcrypt internals beyond failure reason.
- Logs keep distinct `reason` values (L3–L7). Frontend must **not** mirror those reasons — see §4.1.1.
- Skip DEBUG for happy-path password compare unless needed later.

### 4.1.1 Frontend-facing login errors (API response)

Logs and UI responses are separate concerns:

| Case | Internal log `reason` | HTTP status (proposed) | Response body shown to frontend | UI copy (suggested) |
|------|----------------------|------------------------|----------------------------------|---------------------|
| Missing / invalid body fields | `login request invalid` | `400` (or keep `409` if you prefer current) | `{ "error": "invalid request", "code": 400 }` | “Please enter username and password.” |
| User not found (L3) | `user_not_found` | `401` | `{ "error": "invalid credentials", "code": 401 }` | “Invalid username or password.” |
| Wrong password (L4) | `invalid_credentials` | `401` | **Same as above** | **Same as above** |
| Empty username/password after parse | `invalid_credentials` | `401` | **Same as above** | **Same as above** |
| Access / refresh token / last-login update failure (L5–L7) | `access_token` / `refresh_token` / `last_login_update` | `500` | `{ "error": "login failed", "code": 500 }` | “Something went wrong. Please try again.” |
| Unexpected DB / infra error | `auth repo error` | `500` | **Same as above** | **Same as above** |

**Why only two failure messages for auth vs server?**

1. **Security** — L3 and L4 must return the same client message so attackers cannot probe which emails exist.
2. **UX** — Users cannot fix token/DB failures; a generic retry message is enough. Ops use logs + `request_id` (`X-Request-ID`) to debug L5–L7.

**Current gap:** `wrapError.Wrap` returns `err.Error()` as-is, so today the frontend can see `"user not found"`, `"invalid credentials"`, `"failed to generate access token"`, etc. When we implement logging, map service errors to the **two client messages** above before wrapping.

Optional later: add a stable `error_code` for the frontend (`INVALID_CREDENTIALS` | `INVALID_REQUEST` | `LOGIN_FAILED`) without exposing internal `reason`.

### 4.2 Refresh (`POST /authentication/refresh`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| R1 | Controller | Missing refresh cookie | WARN | `refresh failed` | `reason=missing_cookie` |
| R2 | Controller | Entered with cookie present | INFO | `refresh attempt` | `has_refresh_cookie=true` (no token value) |
| R3 | JwtService | Parse / signature invalid | WARN | `refresh failed` | `reason=invalid_token`, `error` (sanitized) |
| R4 | JwtService | DB row missing / expired / user mismatch | WARN | `refresh failed` | `reason=not_found\|expired\|user_mismatch`, `jti`, `user_id` if known |
| R5 | JwtService | New access/refresh issue or DB update failed | ERROR | `refresh failed` | `user_id`, `jti`, `reason=...`, `error` |
| R6 | JwtService / Service | Success | INFO | `refresh success` | `user_id`, `organisation_id`, `jti` |
| R7 | Controller | Response write failure | ERROR | `refresh response failed` | `error` |

#### Frontend-facing refresh errors

Refresh is usually silent (axios interceptor / session renew). Frontend needs a **signal to re-login**, not detailed copy.

| Case | Internal log `reason` | HTTP | API `error` | Frontend action |
|------|----------------------|------|-------------|-----------------|
| Missing cookie / invalid / expired / mismatch (R1, R3, R4) | `missing_cookie` / `invalid_token` / `expired` / … | `401` | `session expired` | Clear local auth state → redirect to login. Optional toast: “Session expired. Please sign in again.” |
| Token rotate / DB failure (R5) | rotate/update reasons | `500` | `refresh failed` | Retry once; if still failing → same as session expired. Optional: “Something went wrong. Please sign in again.” |

Do **not** expose distinct reasons (`user_mismatch`, `not_found`, etc.) to the client.

### 4.3 Logout (`POST /authentication/logout`)

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| O1 | Controller | Logout called | INFO | `logout attempt` | `has_refresh_cookie` |
| O2 | JwtService | Empty cookie (idempotent success) | INFO | `logout success` | `reason=no_cookie` |
| O3 | JwtService | Malformed/expired token — still clear cookie | INFO | `logout success` | `reason=token_unusable` |
| O4 | JwtService | Delete by `jti` / `user_id` failed | WARN | `logout cleanup failed` | `jti` / `user_id`, `error` (logout still returns 200 today) |
| O5 | JwtService / Service | Tokens deleted | INFO | `logout success` | `user_id`, `jti` |

**Note:** Controller currently ignores logout errors (`_ = a.AuthService.Logout(...)`). Logs should still record cleanup failures even if HTTP stays 200.

#### Frontend-facing logout errors

**No dedicated frontend error surface needed.**

| Case | HTTP | API | Frontend action |
|------|------|-----|-----------------|
| Always (success or cleanup failure) | `200` | `{ "message": "logged out successfully" }` | Clear local state, clear cookies if any client-side, go to login |

Reasons:

1. Logout is idempotent — user intent is “end session locally” even if server cleanup fails.
2. Showing “logout failed” confuses users who are already leaving.
3. Server cleanup failures belong in **backend WARN logs** (O4) only; ops investigate via `request_id`.

If the request itself never reaches the server (network down), frontend can still clear local state and redirect; no special API error contract required.

### 4.4 Authenticate middleware (protected routes)

These are high-volume. Prefer **WARN on failure only**; do not INFO on every successful auth (request middleware already logs completion).

| # | Layer | When | Level | Message | Fields |
|---|--------|------|-------|---------|--------|
| A1 | Middleware | Missing `Authorization` | WARN | `auth rejected` | `reason=missing_header`, `path`, `method` |
| A2 | Middleware | Bad Bearer format | WARN | `auth rejected` | `reason=invalid_format` |
| A3 | Middleware / Jwt | Invalid or expired access token | WARN | `auth rejected` | `reason=invalid_or_expired`, `error` if useful |
| A4 | Middleware | Success | — | **skip** (or DEBUG only) | optionally enrich request logger with `user_id` once claims are available |

**Enhancement (optional, recommend later):** after successful validate, set `c.Locals("user_id", ...)` and `reqLogger.With(zap.String("user_id", ...))` so all downstream logs inherit `user_id`. Requires parsing subject from access token (today `ValidateAccessToken` only returns bool).

---

## 5. Where — file checklist

| File | Responsibility |
|------|----------------|
| `internal/authentication/controllers.go` | Entry/exit + input validation for login/refresh/logout; extract logger; pass into service |
| `internal/authentication/services.go` | Business outcomes (success / fail reasons); pass logger into JWT calls |
| `internal/authentication/repository.go` | DB errors only |
| `internal/jwt/services.go` | Token parse/validate/rotate/logout outcomes used by auth |
| `pkg/middleware/functions.go` | Access-token rejection reasons on protected routes |
| `pkg/middleware/request_logger.go` | Already done — no auth-specific changes required for MVP |

---

## 6. Suggested log field conventions

Common fields (already or to add):

| Field | Source |
|-------|--------|
| `request_id` | RequestLogger child logger |
| `method`, `path`, `ip` | RequestLogger |
| `user_id` | After user lookup / claims |
| `organisation_id` | After user lookup / claims |
| `username` | Login request only (failures; optional on success) |
| `reason` | Stable enum-like string for filtering |
| `jti` | Refresh token ID (not the JWT body) |
| `op` | Repository operation name |
| `error` | `zap.Error(err)` on ERROR/WARN with error |

Example success line:

```json
{
  "level": "info",
  "msg": "login success",
  "request_id": "...",
  "user_id": "...",
  "organisation_id": "...",
  "refresh_action": "update"
}
```

Example failure line:

```json
{
  "level": "warn",
  "msg": "login failed",
  "request_id": "...",
  "username": "user@example.com",
  "reason": "invalid_credentials"
}
```

---

## 7. What we will **not** add in this pass

- Audit table / persistent auth audit trail (DB) — logs only for now
- Rate-limit / lockout events (no lockout logic yet; `last_login_attempt` update can stay without extra DEBUG)
- Logging full JWT claims maps
- INFO on every successful `Authenticate` middleware call
- Duplicate “request completed” style logs inside controllers (middleware already covers HTTP status + latency)

---

## 8. Current gaps (as of today)

| Item | Status |
|------|--------|
| RequestLogger + `X-Request-ID` | Done |
| `Login` controller start + parse error | Done |
| Login success / fail reasons in service | Done |
| Refresh / Logout controller + service logs | Done |
| JWT validate / rotate / logout logs | Done |
| Auth middleware rejection logs | Done |
| Repository DB error logs | Done |
| Logger passed into service/JWT | Done |
| Client-safe API errors (login / refresh / logout) | Done |

> Optional follow-up still open: enrich request logger with `user_id` after successful access-token validation (§4.4 / decision 5).

---

## 9. Implementation order (after approval)

1. Thread `*zap.Logger` from controllers → `UserService` → `JwtService` methods used by auth  
2. Login (controller polish + service + repo errors)  
3. Refresh  
4. Logout  
5. `Authenticate` middleware rejection logs  
6. (Optional) enrich request logger with `user_id` after access-token validation  

---

## 10. Open decisions for review

Please mark yes/no or alternatives:

1. **Distinct failure reasons in logs** for `user_not_found` vs `invalid_credentials` (client response stays generic)? — **Recommend: yes**
2. **Log username on failed login?** — **Recommend: yes at WARN**; omit or replace with `user_id` on success
3. **Pass `*zap.Logger` into service methods** (signature change) vs only log in controllers? — **Recommend: pass into service + JWT** so reasons are accurate
4. **Authenticate middleware:** WARN on reject only, no success INFO? — **Recommend: yes**
5. **Optional follow-up:** attach `user_id` to request logger after successful JWT validate? — **Recommend: yes, as follow-up**
6. **Logout cleanup DB errors:** WARN even though HTTP 200? — **Recommend: yes**
7. **Frontend login errors:** only `invalid credentials` (401) for L3+L4 and `login failed` (500) for L5–L7 — never expose internal `reason`? — **Recommend: yes** (see §4.1.1)
8. **Frontend refresh:** `session expired` (401) vs `refresh failed` (500); no detailed reasons? — **Recommend: yes**
9. **Frontend logout:** always `200`, no error UI; cleanup failures log-only? — **Recommend: yes**

---

## 11. Approval

Once this list looks right, next step is implementing in the order in §9 without expanding into other domains.
