# Validation

Reference validation files:
- `verify/service-harness.json`
- `scripts/verify.ps1`
- `scripts/verify.sh`

Current validation direction:
- the repo runs direct local verification of the real Echo Service harness
- the verification path builds the service binary, runs it, exercises the API surface, and checks persistence artifacts
- local and CI usage should share the same harness contract path
- default health model remains `process` until broader health simulation issues land

Ref/code-backed donor healthcheck types observed:
- `http`
- `tcp`
- `file`
- `variable`

Current implementation status:
- `scripts/test.*` runs the Go test suite for the harness itself
- `scripts/verify.*` runs the repo-local verifier under `cmd/verify-harness`
- the verifier currently proves:
  - binary build
  - process health startup
  - `/health`, `/state`, `/logs`, and `/sqlite`
  - action handling for `write-log`, `write-state`, `write-sqlite`, `error`, `fork-child`, and `start-child`
  - persistence of log, state, and SQLite artifacts
  - clean close behavior after verification
