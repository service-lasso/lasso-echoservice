# Service Template - OpenSpec Tracker

_Status: working tracker_

## Current repo target
- `service-template`

## Draft Spec Register

| Draft Spec | Area | Status | Main Source Docs | Intended Repo Target | Notes |
| --- | --- | --- | --- | --- | --- |
| `SPEC-SERVICE-TEMPLATE-REPO.md` | Template | `draft` | `docs/reference/SERVICE-TEMPLATE-REPO.md`, `docs/reference/SERVICE-STRUCTURE-REVIEW.md`, `docs/reference/PROPOSED-CODEBASE-STRUCTURE.md` | `service-template` | Canonical template/service-author contract draft. |

## Current focus
1. lock the canonical template repo contract
2. define the minimum sample service + packaging/release path
3. define the first harness-facing validation example shape
4. only then broaden into more detailed template variants if still needed

## Current local source set
- `README.md`
- `docs/openspec-drafts/SPEC-SERVICE-TEMPLATE-REPO.md`
- `docs/reference/SERVICE-TEMPLATE-REPO.md`
- `docs/reference/SERVICE-STRUCTURE-REVIEW.md`
- `docs/reference/PROPOSED-CODEBASE-STRUCTURE.md`
- `docs/reference/DECISION-CONTEXT.md`
- `docs/reference/EXAMPLE-REPO-TREE.md`
- `docs/reference/EXAMPLE-service.json`
- `docs/reference/EXAMPLE-service-harness.json`
- `docs/reference/EXAMPLE-verify.ps1`
- `docs/reference/EXAMPLE-verify.sh`

## Shared/adjacent context now copied locally
These are now available inside this folder for standalone review:
- `docs/reference/shared-runtime/QUESTION-LIST-AND-CODE-VALIDATION.md`
- `docs/reference/shared-runtime/ARCHITECTURE-DECISIONS.md`
- `docs/reference/shared-runtime/SERVICE-MANAGER-BEHAVIOR.md`
- `docs/reference/adjacent/SPEC-SERVICE-LASSO-HARNESS.md`

## Current gaps
Still needs explicit implementation-ready work for:
- deciding which starter-file fields are canonical first-pass contract versus illustrative placeholders
- normalizing the exact health schema around `process` default plus explicit `http|tcp|file|variable` overrides
- replacing the starter CI package/test flow with a real released-harness invocation once `service-lasso-harness` exists as a binary
