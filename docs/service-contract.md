# Service Contract

This repo implements the Echo Service harness as a real Service Lasso service repo.

Key files:
- `service.json` - service manifest
- `main.go` - the Go harness service
- `verify/service-harness.json` - harness validation contract
- `scripts/verify.*` - validation wrappers
- `scripts/package.*` - packaging entrypoints
- `runtime/` - generated runtime files during local runs
- `config/` - example config inputs
- `docs/service-json-reference.md` - one-stop reference for `service.json` fields, healthcheck setup, and first-pass contract guidance

This service is intentionally harness-oriented.
It exists to support runtime integration, supervision, persistence, and demo verification rather than end-user product behavior.

Current harness-specific surfaces include:
- dedicated HTTP and TCP health targets for realistic health-probe testing
- env and global-env reporting endpoints
- a Service Lasso oriented output endpoint at `GET /service-lasso/output`
- stdout and stderr emission actions for log-capture testing
