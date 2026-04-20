# Spec Draft - Service Template Repo

_Status: draft_

## Intent
Define the canonical `service-template` repo as the standard starting point for future Service Lasso service repos. This matters because service authors should have one authoritative source for repo shape, manifest expectations, packaging/release behavior, validation-harness integration, and service-author documentation instead of reconstructing the contract from scattered donor-analysis notes.

## Scope
Included in this spec:
- the role of `service-template` as the canonical base repo for future services
- one-service-per-repo discipline and its relationship to app bundling/distribution
- minimum repo layout/documentation expectations for service authors
- minimum sample service direction
- reference release/packaging expectations
- validation-harness integration expectations
- naming direction and practical bootstrap/repo-creation guidance
- the requirement that the template repo explain the service contract directly and comprehensively

Explicitly out of scope for this spec:
- the full Service Lasso core runtime spec
- the full harness implementation spec beyond the template integration expectations
- UI/admin behavior
- the final detailed implementation of every future service role/type
- every possible future template variant beyond the first canonical template

## Acceptance Criteria
- `AC-1`: The spec states that each Service Lasso service should live in its own repo.
- `AC-2`: The spec states that `service-template` is the canonical base repo for future service repos.
- `AC-3`: The spec defines the minimum repo/layout/documentation expectations that the template must provide to service authors.
- `AC-4`: The spec states what the template docs must explain directly about `service.json`, defaults vs overrides, `actions`, install/config/start semantics, uninstall vs reset, and managed `.state/` usage.
- `AC-5`: The spec defines the minimum canonical folder/layout direction for the first real `service-template` repo.
- `AC-6`: The spec defines the standard validation-harness direction and records that services should be able to prove they work under real Service Lasso semantics locally and in CI.
- `AC-7`: The spec defines expectations for a minimal sample service plus a reference packaging/release pipeline.
- `AC-8`: The spec records the practical GitHub repo/template creation flow for `service-template`.
- `AC-9`: The spec makes the current canonical repo name explicit as `service-template`.
- `AC-10`: Open questions are recorded explicitly instead of being left implicit in planning notes.

## Tests and Evidence
Planning/reference evidence currently available:
- `docs/reference/SERVICE-TEMPLATE-REPO.md`
- `docs/reference/SERVICE-STRUCTURE-REVIEW.md`
- `docs/reference/PROPOSED-CODEBASE-STRUCTURE.md`
- `docs/reference/shared-runtime/QUESTION-LIST-AND-CODE-VALIDATION.md`
- `docs/reference/shared-runtime/ARCHITECTURE-DECISIONS.md`
- `docs/reference/shared-runtime/SERVICE-MANAGER-BEHAVIOR.md`
- `docs/reference/adjacent/SPEC-SERVICE-LASSO-HARNESS.md`
- representative donor service manifests and READMEs under the donor `services/` tree

Implementation evidence required later:
- a real `service-template` repo structure matching the documented contract
- a minimal sample service proving the template lifecycle
- reference packaging/release scripts and output artifacts
- a harness-facing validation example that the shared harness can execute

Current first-pass starter evidence now present locally:
- `service.json`
- `verify/service-harness.json`
- `scripts/verify.ps1`
- `scripts/verify.sh`
- `scripts/package.ps1`
- `scripts/package.sh`
- `runtime/win32/echo-service.ps1`
- `runtime/linux/echo-service.sh`
- `runtime/darwin/echo-service.sh`
- `.github/workflows/validate-template.yml`
- packaged sample artifact `dist/echo-service-win32.zip`

## Documentation Impact
This spec is expected to govern or inform:
- `README.md`
- future service-author onboarding docs in this repo
- future sample-service docs and packaging scripts
- future validation-contract example files (`scripts/verify.*`, `verify/service-harness.json`)
- future release/CI guidance for service repos derived from this template

## Verification
Verify this spec by checking that the template repo docs and future implementation consistently enforce the same model:
- one service per repo remains explicit
- `service-template` is clearly the canonical starting point
- the template layout is concrete enough to start from without donor archaeology
- service-author documentation requirements are explicit enough to avoid reconstructing the contract from scratchpads
- sample service, packaging, and harness integration expectations are clear enough to drive a real first implementation slice

## Change Notes
- Initial template spec draft was created from donor-analysis/template planning work.
- This revision rewrites the template spec into the project’s actual standard/governance shape instead of leaving it as a ref-style planning note.
- Current canonical repo name recorded explicitly here: `service-template`.

## Current Contract Direction

### Canonical repo name
Current canonical repo name:
- `service-template`

This is the intended first public template repo for the Service Lasso ecosystem.

### Template repo role
`service-template` should be the canonical base repo for future Service Lasso service repos.

Its job is to:
- establish the service-repo contract in one place
- provide a standard repo/layout/doc shape
- provide a minimal sample service
- provide reference packaging/release behavior
- provide the first harness-facing validation example

### One service per repo
Each service should live in its own repo.

This remains true even when:
- some services are bundled into app packages at package time
- some services are preinstalled in a base distribution
- some services are downloaded dynamically later

Bundling is a distribution choice, not a repo-ownership model change.

### Naming direction
Current naming direction for service repos:
- use `lasso-` as the shared repo prefix
- after removing `lasso-`, the remainder maps to the local service folder name

Examples:
- `lasso-@archive`
- `lasso-@node`
- `lasso-fastapi`

### What the template must standardize
The template must standardize at least:
- canonical `service.json` usage and shape guidance
- defaults vs overrides guidance
- how `actions` works in practice
- install/config/start expectations at the service-author level
- uninstall vs reset expectations
- managed `.state/` direction for mutable operational state
- runtime/config/content folder conventions
- per-OS folders where relevant (`win32/`, `linux/`, `darwin/`)
- release artifact shape
- build/package/verify script expectations
- documentation structure
- versioning/release conventions

### Minimum canonical repo structure
The first real `service-template` repo should make the minimum expected structure explicit.

Current minimum direction:
- `service.json`
- `README.md`
- `CHANGELOG.md`
- `LICENSE`
- `config/`
- `scripts/`
- `runtime/` or another clear payload/runtime folder
- per-OS folders where the service actually needs OS-specific payloads
- release/packaging config sufficient to produce installable artifacts

The repo should be small enough to stay understandable, but concrete enough that a new service can be created from it without donor archaeology.

### What the template docs must explain directly
The template docs should explain directly, in one place:
- what `service.json` is for
- what each top-level section means
- how defaults work vs overrides
- what `actions` means in practice
- how install works conceptually
- how config differs from install and when it reruns
- how uninstall differs from reset
- how managed `.state/` usage works at the service-author level
- that `.state/` is the preferred mutable operational state area and legacy standalone `service.pid` should not be treated as mandatory unless a concrete need survives review
- how release artifacts, packaging, and per-OS folders fit together
- how a service creator should think about service-specific command mappings versus Lasso defaults

### Sample service direction
The template should include a deliberately minimal sample service.

Current preferred direction:
- use a simple echo service
- avoid real application complexity
- include explicit per-OS folders as needed
- make the service easy to understand and easy to package

The sample should prove:
- manifest shape
- install flow
- action/default behavior
- state/log behavior
- release artifact flow

Current example artifacts have now been added under `docs/reference/`:
- `EXAMPLE-REPO-TREE.md`
- `EXAMPLE-service.json`
- `EXAMPLE-service-harness.json`
- `EXAMPLE-verify.ps1`
- `EXAMPLE-verify.sh`

### Health model direction
Current health-model direction:
- default health model is **process**
- other health models are allowed when explicitly declared by the service config

Ref/code-backed donor healthcheck types observed:
- `http`
- `tcp`
- `file`
- `variable`

Relevant donor/runtime evidence:
- `docs/reference/shared-runtime/SERVICE-MANAGER-BEHAVIOR.md`
- `docs/reference/SERVICE-STRUCTURE-REVIEW.md`
- donor runtime implementation in `runtime/Service.ts` (copied evidence path remains in donor-ref source)

For `service-template`, the first sample should stay aligned with this rule:
- use `process` as the default/simple case
- only use another health model when the service contract needs it and declares it explicitly

### Validation-harness integration direction
Each service should be able to prove that it works inside Service Lasso, not only as a standalone app/script.

Current preferred direction:
- use one shared Lasso validation harness model across service repos
- let each service repo provide a small service-specific validation contract rather than bespoke orchestration logic
- make the same validation flow runnable both locally and in CI

The template should provide a standard example shape, likely including:
- `scripts/verify.ps1`
- `scripts/verify.sh`
- an example machine-readable validation contract/manifest
- example CI wiring that runs the same harness contract in pipeline

A first concrete example contract and thin verify wrappers now exist under `docs/reference/` for review:
- `EXAMPLE-service-harness.json`
- `EXAMPLE-verify.ps1`
- `EXAMPLE-verify.sh`

The `service-template` repo should include a golden harness example that proves:
- the sample service artifact can be packaged
- the sample service can be installed into an isolated harness root
- the sample service can be started via Service Lasso semantics
- expected health/log/state outputs appear
- cleanup/stop behavior works cleanly

### Reference release pipeline direction
The template should include a simple reference release pipeline.

Current preferred direction:
- include simple `*.ps1` and `*.sh` scripts
- use per-OS subfolders as packaging inputs
- create compressed release artifacts
- use 7zip-based packaging configuration where needed
- make the resulting artifacts usable for end-to-end install/release testing
- make the resulting artifacts usable by the shared harness in CI

### Practical bootstrap guidance
Useful GitHub CLI direction for the canonical template repo:

```bash
gh repo create service-lasso/service-template --public --clone
gh repo edit service-lasso/service-template --template
```

Important clarification:
- create it as a normal repo first
- then mark it as a GitHub template repo

### Recommended first value
The first real `service-template` repo should be valuable immediately as both:
- a service repo starting point
- a release/pipeline test fixture
- a harness example proving what “works in Lasso” means in practice

## Open Questions
- What exact folder layout should become canonical beyond the current minimum direction?
- What exact sample service should ship first beyond the current echo-service direction?
- What exact release artifact structure and packaging scripts should be mandatory versus optional?
- How much generated/example `.state/` material should be included versus merely documented?
- Which parts of the service-author contract should live directly in the template repo versus separate central docs copied or linked into it?
- Which fields in the first example `service.json` and `service-harness.json` should be treated as canonical first-pass fields versus illustrative placeholders?
- Do we want to explicitly normalize the health field names/shapes now around `process` default plus `http|tcp|file|variable` overrides, or leave exact schema locking to the next contract pass?
- Should there be one single base template only at first, or a later layered template strategy for runtime-provider / infrastructure / app-service variants after the canonical first template is proven?
