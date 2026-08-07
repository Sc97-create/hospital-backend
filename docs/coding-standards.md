# Coding Standards & Function Checkpoints

Use this document to validate every new or changed function before merge.  
Limits below are hard caps unless a PR notes an explicit, justified exception.

---

## 1. Function complexity checkpoints

| Checkpoint | Limit | How to measure | Fail action |
|---|---|---|---|
| Cognitive complexity | **≤ 15** | Sonar / `gocognit` style: +1 per branch (`if`/`else`/`switch` case/`for`/`select`), +1 per nested level of those, +1 per boolean operator in conditions (`&&`/`\|\|`) | Split into helpers or early-return |
| Cyclomatic complexity | **≤ 12** | Independent paths through the function | Extract decision branches |
| Max nesting depth | **≤ 2** | Count of nested `if` / `for` / `switch` / `select` blocks (function body = depth 0) | Guard clauses, extract loops |
| Lines of code (excl. blanks/comments) | **≤ 60** | Count statements in function body | Extract private helpers |
| Statements per function | **≤ 40** | Rough statement count | Extract |
| Parameters | **≤ 5** | Including receivers does **not** count; `context` / `*zap.Logger` / `*gorm.DB` each count as 1 | Introduce a request/options struct |
| Return values | **≤ 3** | e.g. `(T, error)` or `(T, int, error)` | Bundle into a result struct |
| Named returns | Prefer none | Avoid named results except short clear cases | Prefer explicit `return` |

### Nesting examples

```go
// ✅ OK — depth 1 inside loop (loop is depth 1, if is depth 2)
for _, item := range items {
    if item.Qty == 0 {
        continue
    }
    // work
}

// ❌ FAIL — depth 3
for _, item := range items {
    if item.Qty > 0 {
        if err := save(item); err != nil {
            if isRetryable(err) { // depth 3 — not allowed
                // ...
            }
        }
    }
}

// ✅ Prefer early continue / extract
for _, item := range items {
    if item.Qty == 0 {
        continue
    }
    if err := processItem(item); err != nil {
        return err
    }
}
```

### Cognitive complexity examples

```go
// High complexity: nested branches + compound conditions
if a && b || c {
    if d {
        for ... {
            switch ...
        }
    }
}

// Lower: early returns, helpers, single-level branches
if err := validate(req); err != nil {
    return err
}
if !allowed {
    return wrapError.ErrInvalidRequest
}
return doWork(req)
```

---

## 2. Control-flow & structure

| Rule | Standard |
|---|---|
| Prefer guard clauses | Fail/return early; happy path stays left-aligned |
| No deeply nested `else` | After `return` in `if`, do not add `else` |
| `switch` over long `if-else if` | Prefer `switch` when ≥ 3 related cases |
| Loops | One concern per loop; filter in a separate pass if it clarifies |
| Magic numbers / strings | Named constants (`pkg/constants` or package-level) |
| Boolean params | Avoid `doThing(true, false)`; use options struct or named constants |

---

## 3. Layering (hospital-backend)

Keep responsibilities in the correct layer. Do not skip layers for convenience.

| Layer | May do | Must not do |
|---|---|---|
| **Controller** | Bind/validate HTTP, call service, map errors → status/JSON | Business rules, raw SQL/GORM queries, multi-step domain orchestration |
| **Service** | Business rules, transactions orchestration, call repos / other services | Parse Fiber context, set HTTP status codes |
| **Repository** | Persistence (GORM/SQL), map DB errors | Business validation beyond “row exists”, HTTP concerns |
| **DTO / models** | Shape of request/response/entities | Side effects |

### Function naming by layer

- Controllers: `CreateX`, `GetX`, `UpdateX`, `ListX` (HTTP verbs / resource)
- Services: verb + domain (`FulfillPaidInvoice`, `addInvoiceItems`)
- Repositories: `Create`, `GetByID`, `Update`, `List` — keep CRUD-clear

---

## 4. Errors

| Checkpoint | Standard |
|---|---|
| Sentinel errors | Use `shared/error` (`wrapError.Err…`) for client/domain failures |
| Propagation | Return errors upward; do not swallow unless intentionally ignored with a comment |
| Logging + return | Log at the layer that has context, then return a stable domain error |
| Wrapping | Prefer domain sentinels over leaking driver/`gorm` errors to HTTP |
| Panic | No `panic` in request path; reserve for impossible init failures |

```go
// ✅
if err != nil {
    log.Error("payment fulfillment failed",
        zap.String("invoice_id", invoiceID),
        zap.String("reason", "stock"),
        zap.Error(err),
    )
    return wrapError.ErrPaymentFulfillFailed
}

// ❌
if err != nil {
    return err // raw DB error to client without domain mapping
}
```

---

## 5. Logging

| Checkpoint | Standard |
|---|---|
| Logger | `*zap.Logger` (structured fields) |
| Ensure logger | Call `ensureLog(log)` (or package equivalent) at service entry |
| Fields | Include stable IDs (`invoice_id`, `prescription_id`, `organisation_id`, …) and a short `reason` on failures |
| Levels | `Error` unexpected/system; `Warn` business rejection; `Info` significant lifecycle; avoid noisy `Debug` in hot paths for MVP |
| Secrets | Never log passwords, tokens, card data, full webhook signatures |

Align with domain logging docs under `docs/*-logging.md` when present.

---

## 6. Transactions & concurrency

| Checkpoint | Standard |
|---|---|
| Who owns `tx` | Document in comment: caller commits/rolls back vs function commits |
| Pass `*gorm.DB` | Services that participate in a larger unit of work accept `db`/`tx` explicitly |
| Goroutines | Do not spawn unconstrained goroutines in handlers; if needed, document lifecycle |
| Shared maps/slices | No concurrent write without sync; prefer local builders then single write |

---

## 7. Types, nullability & Go style

| Checkpoint | Standard |
|---|---|
| Exported API | Clear names; comment on non-obvious exported funcs (`// FuncName …`) |
| Interfaces | Define where used (consumer), keep small |
| Pointers | Use for optional / mutable large structs; avoid `*string` when empty string is enough |
| `context.Context` | First param when the function does I/O that should respect cancel/deadline |
| Imports | Group stdlib / internal / third-party; no unused imports |
| Formatting | `gofmt` / `goimports` clean |

---

## 8. Testing checkpoint (when tests exist)

| Checkpoint | Limit / rule |
|---|---|
| Table-driven tests | Prefer for multi-case logic |
| Complexity of test helpers | Same complexity caps as production |
| External I/O | Mock repos/gateways; no live payment/DB in unit tests |

---

## 9. Per-function review checklist

Copy this into a PR or self-review for **each** new/changed function:

```text
Function: _______________________________
File: ___________________________________

[ ] Cognitive complexity ≤ 15
[ ] Cyclomatic complexity ≤ 12
[ ] Nesting depth ≤ 2
[ ] Body ≤ 60 LOC (excl. blanks/comments)
[ ] ≤ 5 parameters
[ ] ≤ 3 return values
[ ] Early returns / no deep else nesting
[ ] Correct layer (controller / service / repo)
[ ] Domain errors from shared/error where applicable
[ ] Structured zap logging on failure paths (IDs + reason)
[ ] No secrets in logs
[ ] Transaction ownership clear (if DB writes)
[ ] gofmt / goimports clean
[ ] Names match layer conventions

Result: PASS / FAIL — notes: ________________
```

---

## 10. How to validate (tooling)

Recommended local / CI checks (optional but preferred):

```bash
# Cognitive complexity (fail if > 15)
go install github.com/uudashr/gocognit/cmd/gocognit@latest
gocognit -over 15 ./...

# Cyclomatic complexity (fail if > 12)
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
gocyclo -over 12 .

# Nesting / style (via golangci-lint nestif, funlen, gocognit, gocyclo)
# Example .golangci.yml snippets:
#   linters:
#     enable:
#       - gocognit
#       - gocyclo
#       - funlen
#       - nestif
#   linters-settings:
#     gocognit:
#       min-complexity: 16
#     gocyclo:
#       min-complexity: 13
#     funlen:
#       lines: 60
#       statements: 40
#     nestif:
#       min-complexity: 3   # flags nesting that starts looking like depth > 2
```

Manual review still required for layering, logging fields, and transaction ownership — linters do not catch those.

---

## 11. Exceptions

Exceptions need **all** of:

1. Short comment above the function: `// complexity-exception: <reason>`
2. PR description calling out the function and why split is worse (e.g. atomic payment fulfillment steps that must stay visible together)
3. No increase of nesting beyond 3 even with an exception

Prefer extracting helpers over living with exceptions.

---

## 12. Quick reference card

| Metric | Max |
|---|---|
| Cognitive complexity | 15 |
| Cyclomatic complexity | 12 |
| Nesting depth | 2 |
| Function LOC | 60 |
| Parameters | 5 |
| Returns | 3 |
| Nesting with approved exception | 3 |
