# Service Contract

This starter repo demonstrates the first-pass Service Lasso service contract.

Key starter files:
- `service.json` - service manifest
- `verify/service-harness.json` - harness validation contract
- `scripts/verify.*` - thin wrappers that call the shared harness binary
- `scripts/package.*` - reference packaging entrypoints
- `runtime/` - sample payload/runtime files
- `config/` - example config inputs
- `docs/service-json-reference.md` - one-stop reference for `service.json` fields, healthcheck setup, and first-pass contract guidance

This starter is intentionally minimal. It is meant to prove the contract shape, not to be a full production service.
