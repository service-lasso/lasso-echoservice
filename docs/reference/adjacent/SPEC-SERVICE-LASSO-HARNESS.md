# Spec Draft - Service Lasso Harness

_Status: draft_

## Intent
Define the dedicated `service-lasso-harness` contract as the shared validation runner for Service Lasso service repos. This matters because service authors need one standard way to prove that a service artifact works under real Service Lasso semantics — install, config, start, health, stop, and evidence capture — rather than relying on ad hoc repo-local smoke scripts that drift from each other.

## Scope
Included in this spec:
- the role and boundary of `service-lasso-harness` as a standalone shared validation utility
- the split between service-repo-owned validation contracts and harness-owned execution behavior
- preferred v1 implementation language and distribution model
- the minimum v1 validation flow the harness must perform
- minimum result/evidence outputs from a harness run
- how consumer service repos and CI are expected to invoke the harness
- the relationship between `service-template` and the harness as the first golden example
- current role-based validation direction for runtime-provider, infrastructure, app, and utility/bootstrap services

Explicitly out of scope for this spec:
- the full Service Lasso core runtime spec
- the full `service-template` spec beyond what the harness needs to consume
- the final canonical `service.json` structure for all services
- UI/admin behavior or route design
- implementation details for every future validation role/profile beyond the minimum first-pass direction

## Acceptance Criteria
- `AC-1`: The spec states that `service-lasso-harness` exists to validate service artifacts under real Service Lasso semantics, not only by direct standalone execution.
- `AC-2`: The spec states that consumer service repos own a small validation contract and thin verify entrypoints, while the harness owns the shared execution engine.
- `AC-3`: The preferred v1 implementation language is explicitly defined as **Go**.
- `AC-4`: The preferred distribution model is explicitly defined as downloadable **GitHub release binaries**.
- `AC-5`: The spec states that service repos and CI should consume released harness binaries rather than depending on a local Node toolchain for harness execution.
- `AC-6`: The spec defines the minimum v1 validation flow: isolated root creation, artifact install, dependency bootstrapping where required, `install`/`config`/`start`/health/`stop` execution, and evidence capture.
- `AC-7`: The spec defines the minimum required outputs from a harness run, including machine-readable result status plus enough evidence to debug failures.
- `AC-8`: The spec defines the current role-based validation direction for runtime-provider, infrastructure, app, and utility/bootstrap services.
- `AC-9`: The spec defines `service-template` as the first intended golden-example consumer of the harness contract.
- `AC-10`: Open questions are recorded explicitly instead of being left implicit in surrounding planning text.

## Tests and Evidence
Planning evidence currently available:
- `README.md`
- `docs/usage-flow.md`
- `docs/openspec-drafts/OPENSPEC-TRACKER.md`
- `C:\projects\service-lasso\service-template\docs\openspec-drafts\SPEC-SERVICE-TEMPLATE-REPO.md`
- `C:\projects\service-lasso\service-template\docs\reference\SERVICE-TEMPLATE-REPO.md`
- `C:\projects\service-lasso\service-template\docs\reference\SERVICE-STRUCTURE-REVIEW.md`
- `C:\projects\service-lasso\service-lasso\ref\typerefinery-service-manager-donor\QUESTION-LIST-AND-CODE-VALIDATION.md`
- `C:\projects\service-lasso\service-lasso\ref\typerefinery-service-manager-donor\ARCHITECTURE-DECISIONS.md`

Implementation evidence required later:
- a Go runner binary that can execute the first stable `run` flow against a service contract
- a machine-readable contract/schema for `verify/service-harness.json`
- at least one end-to-end validation proof against the `service-template` sample service
- CI proof that the released harness binary can be acquired and executed by a consumer service repo

## Documentation Impact
This spec is expected to govern or inform:
- `README.md`
- `docs/usage-flow.md`
- future `docs/validation-contract.md`
- future `schemas/service-harness.schema.json`
- future CLI/runner docs for the released binary
- future `service-template` example verify-contract docs and sample usage

## Verification
Verify this spec by checking that the harness docs and future implementation consistently enforce the same model:
- service repos declare a small validation contract instead of embedding a bespoke validation engine
- the harness is clearly treated as the shared executor
- Go is the explicit v1 implementation choice
- GitHub release binaries are the explicit distribution mechanism
- consumer repos and CI are expected to invoke the released harness binary
- the minimum run flow covers isolated setup, artifact/dependency preparation, lifecycle execution, health/readiness checks, and evidence capture
- run outputs are explicit enough that a future implementation can prove pass/fail and support debugging

## Change Notes
- Initial dedicated harness spec draft was created after the harness work was split into its own standalone local folder.
- This revision rewrites the harness spec into the project’s actual standard spec/governance structure instead of a freeform planning note.
- Current preferred direction recorded here: implement v1 in Go, distribute via GitHub release binaries, and have service repos consume the released harness binary.

## Current Contract Direction

### Harness role
`service-lasso-harness` is the shared validation runner for Service Lasso service repos.

Its purpose is to answer:

> does this service artifact actually work inside Service Lasso?

It should not be treated as a replacement for unit tests, a UI test framework, a service template repo, or a general workflow engine.

### Service repo interaction model
A consumer service repo is expected to provide a small contract surface such as:
- `scripts/verify.ps1`
- `scripts/verify.sh`
- `verify/service-harness.json`

Those files should be thin wrappers/contracts rather than full validation engines.

Conceptual direction for invocation:

```powershell
service-lasso-harness.exe run --contract .\verify\service-harness.json
```

Equivalent non-Windows shell invocation should follow the same contract model.

### Harness-owned execution behavior
The harness should own the shared validation engine behavior:
- isolated Lasso root creation
- artifact installation
- dependency bootstrapping where required for validation
- lifecycle execution (`install`, `config`, `start`, health/readiness, `stop`)
- logs/state/result artifact capture
- machine-readable pass/fail output

### Preferred v1 implementation language
Preferred v1 implementation language:
- **Go**

Reasoning:
- single-binary distribution is cleaner for a shared cross-repo utility
- easier version pinning for service repos and CI
- easier GitHub-release consumption model
- avoids requiring a local Node toolchain solely to execute the harness runner

### Preferred distribution model
Preferred distribution model:
- **GitHub release binaries**

Current intended direction:
- publish platform-specific binaries per release
- let consumer repos and CI download/use a pinned harness release asset
- treat released binaries as the default execution path for service verification

### Minimum v1 run flow
The minimum meaningful v1 harness flow should be:
1. create an isolated temporary Lasso validation root
2. materialize test config if required
3. install the target service artifact
4. install declared validation dependencies where required
5. run `install`
6. run `config` if required
7. run `start`
8. wait for readiness / health
9. validate expected outputs
10. run `stop`
11. capture and emit result artifacts

`uninstall` / `reset` checks may begin as narrower or later slices if needed, but the overall direction should remain explicit.

### Minimum run outputs
Each harness run should produce enough evidence to trust passes and debug failures.

Minimum direction:
- machine-readable pass/fail result
- failing stage / failure reason when not successful
- logs
- relevant runtime/state outputs
- timing information
- artifact summary and/or artifact paths

### Role-based validation direction
Current role-based validation direction:

#### Runtime-provider services
Examples:
- `@node`
- `@python`
- `@java`

Expected direction:
- runtime installs successfully
- binary is callable
- expected env is exported
- a dependent sample service can execute through it

#### Infrastructure services
Examples:
- postgres
- traefik
- mongo

Expected direction:
- payload/archive install works
- config materializes correctly
- service port opens
- health succeeds
- stop behavior works correctly

#### App services
Examples:
- node app
- python app
- API service

Expected direction:
- runtime dependency resolves
- app starts correctly
- health/readiness succeeds
- expected logs/state are written

#### Utility/bootstrap services
Examples:
- `@archive`
- `@localcert`

Expected direction:
- expected action completes
- expected artifacts are produced
- exported values are usable by dependent services

### Relationship to `service-template`
`service-template` is the first intended golden-example consumer of the harness contract.

The harness + template relationship should prove:
- how a service declares the validation contract
- how a local verify script calls the harness
- how CI uses the same path
- what a successful evidence bundle looks like

The harness should not overrun the template contract, but it still needs its own standalone spec because it is a separate shared utility with its own release/distribution responsibilities.

## Open Questions
- What exact schema should be locked first for `verify/service-harness.json`?
- How much dependency bootstrapping should the harness own versus the service contract declare?
- Which uninstall/reset checks belong in v1 versus later slices?
- What exact CLI surface should be stabilized first (`run`, `validate-contract`, `version`, etc.)?
- Which output artifacts should be mandatory for every run?
- How much platform-specific logic should live in the Go binary versus service-owned wrappers/config?
- Should there be an optional helper installer/bootstrap path for acquiring the binary, or should pinned direct release-asset download remain the only supported path?
