# Service Template - Example Repo Tree

_Status: draft example_

This is the first concrete example repo tree for the future `service-template` repo.

It is intentionally small.

```text
service-template/
  README.md
  CHANGELOG.md
  LICENSE
  service.json
  verify/
    service-harness.json
  scripts/
    verify.ps1
    verify.sh
    package.ps1
    package.sh
  runtime/
    win32/
      echo-service.ps1
    linux/
      echo-service.sh
    darwin/
      echo-service.sh
  config/
    example.env
  docs/
    service-contract.md
    packaging.md
    validation.md
```

## Notes

- `service.json`
  - canonical service manifest for the sample service
- `verify/service-harness.json`
  - machine-readable validation contract for the shared harness
- `scripts/verify.*`
  - thin wrappers that call the shared harness binary
- `scripts/package.*`
  - reference packaging entrypoints
- `runtime/`
  - minimal payload/runtime files for the sample service
- `config/`
  - example config inputs if needed
- `docs/`
  - service-author-facing explanation in the real template repo

## Why this shape

This tree is meant to be:
- small enough to understand quickly
- concrete enough to be copied by service authors
- compatible with the harness-first validation direction
- compatible with per-OS payload expectations where needed
