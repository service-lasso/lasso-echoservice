# Service Template - Decision Context

_Status: local context note_

This file exists so `service-template` can stand on its own without pretending that every relevant decision lives only inside this folder.

## Why this file exists

The template work depends on some shared Service Lasso reconciliation/decision docs that still live under the donor-reference area.

That shared context should not be duplicated casually, but it should be easy to find.

## Shared context to consult

### Local copies of the shared reconciliation docs
- `shared-runtime/QUESTION-LIST-AND-CODE-VALIDATION.md`
- `shared-runtime/ARCHITECTURE-DECISIONS.md`
- `shared-runtime/SERVICE-MANAGER-BEHAVIOR.md`

### Why they matter to `service-template`
They contain the current settled direction for things like:
- `install` vs `config`
- `actions` semantics
- utility/runtime-provider/service boundaries
- port negotiation expectations
- `globalenv` direction
- `.state` direction
- uninstall vs reset semantics
- runtime/service lifecycle expectations that template examples should not contradict

## Local docs in this folder
Within `service-template`, start with:
1. `docs/openspec-drafts/SPEC-SERVICE-TEMPLATE-REPO.md`
2. `docs/reference/SERVICE-TEMPLATE-REPO.md`
3. `docs/reference/SERVICE-STRUCTURE-REVIEW.md`
4. `docs/reference/PROPOSED-CODEBASE-STRUCTURE.md`

## Related adjacent repo
The shared validation runner still lives separately as a project in:
- `C:\projects\service-lasso\service-lasso-harness`

For standalone review convenience, the most relevant adjacent harness spec is also copied locally here:
- `adjacent/SPEC-SERVICE-LASSO-HARNESS.md`

## Working rule

Use this folder as the main working area for template-specific work.

Default to the local copies in this folder first. If those copies drift later, reconcile against their source repos deliberately rather than guessing.
