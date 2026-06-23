# Review — PR #12 (Amplop v2 Phase 4: local run, configurable sslmode, docs)

## Blocking

_None._

## Resolved

_None yet._

## Non-blocking (nit)

- [ ] `DB_SSL_MODE` is not validated against a known set. A typo (e.g. `verfy-full`)
  is passed straight to lib/pq and surfaces only as a connect error. Acceptable
  given the strict `verify-full` default; flagging for awareness.
  (internal/platform/config/config.go:44-47)

## Notes (verified, no action)

- verify-full path unchanged for real production config: `sslrootcert` always
  appended when `SSLMode != "disable"`; `password` appended when non-empty.
  Omitting password (empty) is the correct fix for the lib/pq dbname-drop bug.
  (internal/platform/database/database.go:30-37)
- `disable` is the only mode that drops the CA cert and is strictly opt-in;
  default remains `verify-full`. README warns against `disable` in prod.
- Test refactor in expense/budget/subscription repo_test.go is behavior-
  preserving (Open+Ping → database.Connect = Open+single-conn+Ping). All three
  retain `//go:build integration`, so `go test ./...` skips them as documented.
- Docs accurate: `make run-expense` exists; `functions.HTTP("Expense",...)`
  matches `--entry-point=Expense`; `--runtime=go121` matches go.mod 1.21.3.
- build / vet / gofmt / `go test ./...` all clean.
