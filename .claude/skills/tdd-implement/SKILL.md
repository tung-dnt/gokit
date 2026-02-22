---
name: tdd-implement
description: Per-task TDD driver. Reads an Obsidian task file, writes failing tests first, waits for user approval, then implements until make check passes. Invoke with `/tdd-implement "<task path or title>"`.
user-invocable: true
---

You are a TDD implementation driver for a Go developer following strict Red-Green-Refactor discipline. When the user invokes `/tdd-implement`, guide one task from failing tests to a passing `make check`.

## What was invoked

The user ran: `/tdd-implement "<task path or title>"`

If a file path was provided, read the task file using the Read tool. If only a title was given, ask for the full path or search for the file in:
```
/Users/tung-dnt/Library/Mobile Documents/iCloud~md~obsidian/Documents/Personal/TaskNotes/Tasks/
```

---

## Pipeline

### Step 1 — Read the task

Read the Obsidian task file. Extract:
- **Goal**: what this task accomplishes
- **Domain**: which `biz/<domain>/` package is affected
- **Implementation steps**: the ordered list from the task body
- **Acceptance criteria**: the checklist

Display a summary to the user:
```
## Task: <goal>
Domain: biz/<domain>/
Steps: <N> implementation steps
```

---

### Step 2 — RED phase (failing tests)

Invoke the `tdd-guide` agent to write the failing tests.

Provide the agent with:
- The domain name
- The specific operations this task covers (from the implementation steps)
- The test cases specified in the feature-planner plan (if referenced in the task)

The tdd-guide agent will write:
- `biz/<domain>/service_test.go` — table-driven tests for service methods
- `biz/<domain>/controller_test.go` — HTTP handler tests via Echo + httptest
- `biz/<domain>/dto/dto_test.go` — validator tag tests (if DTOs are new)

**STOP here.** Do not implement anything yet.

---

### Step 3 — Test approval gate

Show the written tests to the user and ask:

> "Here are the failing tests for this task. Please review:
>
> **Service tests:** `biz/<domain>/service_test.go`
> **Handler tests:** `biz/<domain>/controller_test.go`
>
> Do the test cases correctly capture the expected behaviour? Reply 'yes' to proceed with implementation, or request changes."

Do NOT write any implementation code until the user explicitly approves the tests.

---

### Step 4 — GREEN phase (implementation)

On user approval, implement the minimum code to make the tests pass.

Follow this order (matches the feature-planner schema-first approach):

1. **DB + codegen** (if this task covers it):
   - Write `repo/sqlite/migrations/<domain>.sql`
   - Write `repo/sqlite/queries/<domain>.sql`
   - Run: `go tool sqlc generate`
   - Verify: `go build ./...`

2. **Model + DTOs** (if this task covers it):
   - Write `biz/<domain>/model.go` (entity + input types + errNotFound)
   - Write `biz/<domain>/dto/dto.go` (validate + example tags)

3. **Service** (if this task covers it):
   - Write `biz/<domain>/service.go`
   - Run: `go test ./biz/<domain>/... -run TestXxxService -v`

4. **Route + Handlers** (if this task covers it):
   - Write `biz/<domain>/route.go`
   - Write `biz/<domain>/controller.go` (with swag annotations)
   - Run: `go test ./biz/<domain>/... -run TestController -v`

5. **Wire-up + Swagger** (if this task covers it):
   - Update `cmd/http/main.go` — add to `registerRouters()`
   - Run: `make swagger`

After each file is written, the PostToolUse hook automatically runs `gofmt + make check`. Watch for failures and fix immediately.

---

### Step 5 — make check loop

Run `make check` explicitly:

```bash
make check
```

This runs: `make fmt && make vet && make lint && make test`

**If it fails:**
1. Read the error output carefully
2. Fix the specific issue (lint warning, test failure, vet error)
3. Re-run `make check`
4. Repeat until all checks pass

Do NOT mark the task done until `make check` exits with code 0.

---

### Step 6 — Code review (background)

After `make check` passes, invoke the `go-code-reviewer` agent **in background** (`run_in_background: true`):

Provide it with the domain path: `biz/<domain>/`

Continue to the next step while the review runs.

---

### Step 7 — Update Obsidian task

Update the task file:
- Change `status: todo` → `status: done`
- Update `dateModified` to the current timestamp (ISO 8601 with +07:00 offset)
- Add `completedDate: <YYYY-MM-DD>` field

Use the Edit tool to make these changes.

---

### Step 8 — Handoff

Present the completion summary:

```
## Task complete: <goal>

### Quality gate
- make check: ✅ PASS
- go-code-reviewer: running in background

### Files changed
- <list of files written/modified>

### Obsidian task
- Status updated: todo → done

### Next steps
```

If this task added REST API endpoints:
```
This task added new API endpoints. Run:

  /update-perf-tests
```

If there are more tasks remaining:
```
Continue with the next task:

  /tdd-implement "<path/to/next-task.md>"
```

---

## Rules

- NEVER write implementation code before tests are written and approved
- NEVER run `make check` only once — loop until it fully passes
- NEVER skip updating the Obsidian task status when done
- If the go-code-reviewer returns blocking issues, fix them before marking done
- Use `errors.Is()` — never `==` for sentinel errors
- All service methods must accept `ctx context.Context` as first parameter
- Wrap all errors: `fmt.Errorf("opName: %w", err)`
- Only `Controller` is exported from each domain package
