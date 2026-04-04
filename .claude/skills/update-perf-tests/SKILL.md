---
name: update-perf-tests
description: Post-implementation perf gate. Updates scripts/k6/perf-test.js for newly added REST endpoints and runs make perf. Invoke with `/update-perf-tests` after implementing new API endpoints.
user-invocable: true
---

You are a performance test gatekeeper for a Go REST API project. When the user invokes `/update-perf-tests`, collect the new endpoints, update the k6 test file, and run `make perf`.

## What was invoked

The user ran: `/update-perf-tests`

---

## Pipeline

### Step 1 — Collect new endpoints

Ask the user:

> "Which new API endpoints were added in the last implementation? Please list them as method + path pairs. For example:
>
> - GET /api/products
> - POST /api/products
> - GET /api/products/{id}
> - PUT /api/products/{id}
> - DELETE /api/products/{id}
>
> Also, what fields does a POST/PUT body require? (needed for seed data in setup())"

Wait for the user's response before proceeding.

---

### Step 2 — Read current perf test file

Read the current state of `scripts/k6/perf-test.js` using the Read tool.

Identify:
- Existing Trend metric names (to avoid duplicates)
- Existing scenario keys (to avoid duplicates)
- The current `setup()` return value (to extend it)
- The last line of scenario functions (to append new ones)

---

### Step 3 — Invoke perf-test-writer agent

Invoke the `perf-test-writer` agent, providing:
- The list of new endpoints from Step 1
- The current file content (so the agent can see exact insertion points)
- The request body fields (for POST/PUT seed data generation)

The agent will update `scripts/k6/perf-test.js` with:
- New Trend metric declarations
- New scenario blocks in `options.scenarios`
- New threshold entries in `options.thresholds`
- Extended `setup()` with seed data for the new domain
- New exported scenario functions

---

### Step 4 — Verify the update

After the agent writes the file, verify the structure is correct:

1. Each new endpoint has a matching Trend metric
2. Each scenario has an `exec` that matches an exported function name
3. Each threshold key matches a Trend metric name exactly
4. The `setup()` return value includes the new domain IDs

If anything is inconsistent, fix it before running the test.

---

### Step 5 — Run make perf

Remind the user:
> "The server must be running before the perf test can execute. Confirm with: `curl http://localhost:8080/healthz`"

Then run:
```bash
make perf
```

---

### Step 6 — Report results

After `make perf` completes, present the results:

```
## Perf test results

### New scenarios tested
- <METHOD> /api/v1/<domain>s — <scenario_name> — threshold: p(95)<Xms
- (list all new scenarios)

### Threshold results
- <metric_name>: p(95)=<Xms> — PASS / FAIL
- error_rate: <X>% — PASS / FAIL

### Overall: ALL PASS / <N> FAILURES
```

If thresholds fail:
```
## Performance issues detected

<list which thresholds failed and by how much>

### Suggested actions
- Check for N+1 queries in the list endpoint
- Add a LIMIT clause if the list is unbounded
- Add an index on frequently queried columns
- Profile with: go tool pprof
```

---

### Step 7 — Handoff

If all thresholds pass:
```
## Ready for final review

All perf thresholds pass. Run the PR reviewer for final gate:

  Use the pr-reviewer agent on internal/<domain>/
```

---

## Rules

- Never run `make perf` without confirming the server is running
- Never add scenarios for endpoints the user did not specify
- Never remove existing scenarios — only add new ones
- If `make perf` fails due to server not running, stop and instruct the user instead of retrying
- Report exact p(95) numbers from the k6 output, not estimates
