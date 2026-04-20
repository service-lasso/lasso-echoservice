# service-template

_Status: starter template repo_

`service-template` is the canonical starting point for new Service Lasso service repos.

Use this repo when you want to create a new service repo that already has:
- the expected service repo layout
- a starter `service.json`
- starter packaging scripts
- starter verify scripts and harness contract shape
- starter docs for service contract, packaging, and validation
- starter CI scaffolding

## Use this template

Recommended flow:

1. Create a new repo from this GitHub template.
2. Rename the sample service files/content for the real service.
3. Replace the sample runtime payload with the real service payload.
4. Update `service.json`, `verify/service-harness.json`, and the docs for the new service.
5. Run the local package + test flow.
6. Wire the repo into the real released `service-lasso-harness` binary once that integration path is enabled.

## Quick start

### Local package

```powershell
pwsh -NoLogo -NoProfile -File .\scripts\package.ps1
```

### Local tests

```powershell
pwsh -NoLogo -NoProfile -File .\scripts\test.ps1
```

## Start here for deeper design context

Read in this order if you need the underlying design/spec context:

1. `docs/openspec-drafts/SPEC-SERVICE-TEMPLATE-REPO.md`
2. `docs/openspec-drafts/OPENSPEC-TRACKER.md`
3. `docs/service-contract.md`
4. `docs/service-json-reference.md`
5. `docs/packaging.md`
6. `docs/validation.md`
7. `docs/reference/adjacent/SPEC-SERVICE-LASSO-HARNESS.md`

## What is here

### OpenSpec drafts
- `docs/openspec-drafts/SPEC-SERVICE-TEMPLATE-REPO.md`
- `docs/openspec-drafts/OPENSPEC-TRACKER.md`

### Supporting reference/planning docs
- `docs/service-contract.md`
- `docs/service-json-reference.md`
- `docs/packaging.md`
- `docs/validation.md`
- `docs/reference/SERVICE-TEMPLATE-REPO.md`
- `docs/reference/SERVICE-STRUCTURE-REVIEW.md`
- `docs/reference/PROPOSED-CODEBASE-STRUCTURE.md`
- `docs/reference/DECISION-CONTEXT.md`
- `docs/reference/shared-runtime/QUESTION-LIST-AND-CODE-VALIDATION.md`
- `docs/reference/shared-runtime/ARCHITECTURE-DECISIONS.md`
- `docs/reference/shared-runtime/SERVICE-MANAGER-BEHAVIOR.md`
- `docs/reference/adjacent/SPEC-SERVICE-LASSO-HARNESS.md`
- `docs/reference/EXAMPLE-REPO-TREE.md`
- `docs/reference/EXAMPLE-service.json`
- `docs/reference/EXAMPLE-service-harness.json`
- `docs/reference/EXAMPLE-verify.ps1`
- `docs/reference/EXAMPLE-verify.sh`

## Purpose

This repo is the canonical starting point for Service Lasso service repos.

Its role is to define:
- one-service-per-repo expectations
- service repo layout
- service author documentation expectations
- sample service expectations
- packaging/release expectations
- validation-harness integration expectations

## Current status

This repo is usable now as a starter template.

It currently includes:
- actual starter repo files (`service.json`, `verify/`, `scripts/`, `runtime/`, `config/`, `.github/workflows/`)
- a packaged first-pass sample artifact at `dist/echo-service-win32.zip`
- a starter multi-OS GitHub Actions workflow that packages release archives and runs basic tests
- starter harness-contract files and thin verify wrappers
- supporting reference/spec docs for deeper design work

Important current validation note:
- the pipeline now downloads and invokes the released `service-lasso-harness` binary in CI
- it still keeps the starter local package/test flow alongside harness verification
- the current harness version is pinned in workflow config and can be advanced intentionally over time
