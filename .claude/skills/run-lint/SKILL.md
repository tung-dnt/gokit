---
name: run-lint
description: Run golangci-lint against the codebase, report issues, and apply auto-fixes where possible
---

Run the project linter and report or fix issues. The project uses `golangci-lint` with 36 linters configured in `.golangci.yml`.

## Run lint (read-only, report only)

```bash
golangci-lint run ./...
```

## Run lint with auto-fix

Many linters support `--fix` to automatically apply safe corrections:

```bash
golangci-lint run --fix ./...
go build ./...
```

## Run lint on a specific package

```bash
golangci-lint run ./domain/user/...
golangci-lint run ./infra/...
```

## Run lint with verbose output (shows which linter triggered each issue)

```bash
golangci-lint run --verbose ./...
```

## Common issues and fixes

| Linter | Issue | Fix |
|--------|-------|-----|
| `gofmt` | Formatting | `gofmt -w .` |
| `errcheck` | Unchecked error return | Assign to `_` or handle the error |
| `govet` | Suspicious constructs | Fix the code per vet output |
| `staticcheck` | Dead code, wrong usage | Fix per message |
| `revive` | Style violations | Apply suggested name/style changes |
| `cyclop` | Function too complex | Extract helper functions |
| `gocognit` | High cognitive complexity | Simplify logic or extract helpers |
| `gosec` | Security issues | Address or suppress with `//nolint:gosec` |
| `unused` | Unexported unused symbols | Remove or export |
| `wrapcheck` | Error not wrapped | Use `fmt.Errorf("op: %w", err)` |

## Suppress a specific false positive

```go
result, err := doThing() //nolint:errcheck
```

Only suppress if you have verified the error can be safely ignored.

## Project linter config

`.golangci.yml` — 23 linters, 5m timeout. Key settings:
- `errcheck`: checks all error returns
- `govet`: enabled with shadow checking
- `cyclop`: max complexity 15, package average 10.0
- `gocognit`: max cognitive complexity 15
- `gosec`: security checks (excludes G115)
- `gochecknoglobals`: no package-level vars (except in docs/)
- `exhaustive`: all enum switch cases covered
- `contextcheck`: context passed correctly throughout call chains

## After fixing

Always verify after lint fixes:
```bash
make check
```
