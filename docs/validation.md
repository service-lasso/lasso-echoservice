# Validation

Reference validation files:
- `verify/service-harness.json`
- `scripts/verify.ps1`
- `scripts/verify.sh`

Current validation direction:
- the repo runs direct local verification of the real Echo Service harness
- the verification path builds the service binary, runs it, exercises the API surface, and checks persistence artifacts
- local and CI usage should share the same harness contract path
- the manifest still uses `process` as the primary health model, while the harness now also exposes dedicated HTTP and TCP health targets for future runtime tests

Ref/code-backed donor healthchecks[] types observed:
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
  - `/health`, `/health/http`, `/health/tcp`, `/state`, `/logs`, `/sqlite`, `/env`, `/global-env`, and `/service-lasso/output`
  - action handling for `write-log`, `write-state`, `write-sqlite`, `write-stdout`, `write-stderr`, `http-health`, `tcp-health`, `error`, `fork-child`, and `start-child`
  - persistence of log, state, and SQLite artifacts
  - stdout and stderr capture
  - dedicated HTTP health transitions through healthy, error, stopped, then restored healthy
  - dedicated TCP health transitions through healthy, error, stopped, then restored healthy
  - clean close behavior after verification
