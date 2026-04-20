# Service Lasso - Architecture Decisions

_Status: working draft / decision log_

Important reconciliation note:
- this file is a decision log, but `QUESTION-LIST-AND-CODE-VALIDATION.md` is the canonical transcript+code reconciliation source for the main donor/runtime-boundary questions
- if an older decision section here drifts away from that reconciled source, the reconciled question doc should win unless the discussion is explicitly reopened

Purpose:

- capture the important architectural decisions discussed so far
- keep critical decisions out of chat-only memory
- provide a concise companion to the larger architecture notes

Related docs:

- `ARCHITECTURE.md`
- `SERVICE-STRUCTURE-REVIEW.md`
- `QUESTION-LIST-AND-CODE-VALIDATION.md`
- `REFERENCE-APP-NODE.md`
- `REFERENCE-APP-ELECTRON.md`
- `REFERENCE-APP-TAURI.md`
- `REFERENCE-APP-DOCKER.md`
- `REFERENCE-APP-PKG.md`
- `REFERENCE-APP-NEXE.md`

---

## 1. Core Product Boundary

### Decision
The core product is `service-lasso`.

### Meaning
It should be:
- Node-first
- npm-publishable
- buildable to plain JS
- executable with `node`
- the canonical first implementation of the architecture

### Not part of the core boundary
- Electron shell concerns
- Tauri shell concerns
- Docker packaging concerns
- `pkg` / `nexe` packaging concerns

These belong in separate reference apps/packages.

---

## 2. Runtime Shape

### Decision
Service Lasso should be a Node-first local service orchestration runtime.

### Meaning
The core should provide:
- runtime engine
- service/instance orchestration
- local control API
- CLI/runtime entrypoints
- shared contracts used by optional UI/reference hosts

Important clarification:
- the admin/operator UI is **not** privileged core
- it should be an optional separate service/reference consumer of the API
- the core owns orchestration + API + CLI + contracts, not a mandatory built-in UI shell

---

## 3. Implementation Hierarchy

### Decision
Node/JS is the canonical first implementation.

### Meaning
The first full working implementation lives in `service-lasso`.

Future native ports should be:
- `service-lasso-go`
- `service-lasso-rust`

These must be **true ports**, not divergent rewrites.

## 4. App / Instance Model

### Decision
An app start creates an app instance at runtime.

### Meaning
- when one app starts, it negotiates ports and loads all required services
- when another app starts, it does the same again on a different set of ports
- each app startup is therefore an instance in practice
- services come from shared service repos/packages, but app runs create separate instance-level runtime state and negotiated ports
- if a service needs its own internal sub-instance/forking behavior, that can be handled within that service itself

---

## 5. Repo Visibility Rule

### Decision
All Service Lasso ecosystem repos should be public.

### Meaning
- core repo
- service repos
- reference app repos
- template repos

Current rationale:
- public visibility keeps the pipeline/release flow working as intended

## 6. Reference App Set

### Decision
Reference apps/packages are separate from the core.

### Current set
#### Host/environment references
- `service-lasso-app-node`
- `service-lasso-app-electron`
- `service-lasso-app-tauri`
- `service-lasso-app-docker`

#### Packaging references
- `service-lasso-app-pkg`
- `service-lasso-app-nexe`

### Rule
Reference apps/packages must consume the core package and must not contain hidden orchestration logic absent from core.

---

## 7. Electron Position

### Decision
Electron is excluded from the core architecture and belongs only in a reference app.

### Meaning
Electron may be used to:
- host the UI
- wrap/package the runtime
- provide desktop shell behavior

But it must not define the core runtime architecture.

---

## 8. IPC vs API

### Decision
Electron-style renderer IPC should not define the core control plane.

### Meaning
The core should expose a local API/control plane.

Reference hosts like Electron/Tauri may adapt to that API, but the core should not depend on host-specific IPC semantics.

---

## 9. Service Folder Model

### Decision
The `services/` folder should remain dynamic, not fixed.

### Meaning
Service Lasso should support:
- discovering installed services under `services/`
- downloading/installing services from release sources
- updating/removing services
- treating `services/` as an installed local inventory

It should not depend on a permanently source-controlled monolithic service tree.

Current clarified local service layout direction:
- each service has its own folder under `services/`
- each service keeps one canonical metadata file (`service.json`)
- each service has a mutable local state file (`service.state`) for install/resolution/state info
- each service has a pid file (`service.pid`) for current running process state
- each service has config folders coming from the repo/package
- each service has an `installed/` folder created by the install process
- install commonly means extracting the service archive (for example zip / 7zip, possibly chunked for large payloads)

---

## 10. Service Template Priority

### Decision
The first template repo to create should be the public service template repo.

### Meaning
- it should be created before the wider service repo migration begins
- it should be the canonical base for all later service repos
- it should include a minimal echo-style sample service proving the lifecycle/install/log contract
- the sample script behavior should just echo sample text
- it should include explicit OS subfolders such as `win32/`, `linux/`, and `darwin/`
- the pipeline should use those OS subfolders as packaging inputs
- it should include a simple reference release pipeline (for example `*.ps1` and `*.sh`) that produces compressed release artifacts for end-to-end testing
- archive/package creation in that pipeline should be done using 7zip-based packaging config

## 11. Service Source / Install Model

### Decision
All services must live in their own repos.

### Meaning
- one service = one repo
- service repos are independently versioned and released
- Service Lasso installs services from those external repos/releases
- the core repo should not become a permanent monorepo of service implementations
- services support explicit versions and should resolve to exact release versions when selected

Service Lasso should support a flow like:
- resolve service source
- choose exact release/version (for example `@node:1.18.0`)
- download release package
- install/unpack into local `services/`
- run setup/install exec commands if defined
- register and run the service

Important clarification:
- install is not just archive extraction
- install may include both archive extraction and command-based setup steps

The service CLI/app flow should support service acquisition modes such as:
- `embed` -> download/install now
- `package` -> defer inclusion to package/build time
- `runtime` -> resolve/download/install when the app/runtime starts

---

## 12. Action Model

### Decision
Services should use one finite built-in action vocabulary.

### Current action set
- `install`
- `uninstall`
- `update`
- `start`
- `stop`
- `restart`
- `config`

### Meaning of `config`
`config` is a real operational action.

It should allow services to:
- generate templated config
- merge service-provided config with app-space overrides
- inject runtime-resolved values (ports, URLs, cert paths, env-derived values)
- write/update effective config into the runtime/installed area

Important example:
- for `@traefik`, `config` can regenerate routing config when services are added/changed

### Model rule
All services share:
- one registry
- one broad banded state machine
- one finite action vocabulary

Services differ by:
- which actions they support
- which states are meaningful for them in practice
- their role/category metadata

Important clarification:
- diagnostics and status inspection should primarily be exposed through UI/API/state, not separate built-in actions like `validate` / `inspect`
- actions should stay operational and finite

---

## 13. Packaging Modes

### Decision
App/reference hosts should support multiple service-acquisition modes.

### Modes
#### `embed`
- download/install the selected service now

#### `package`
- defer the selected service to package/build time inclusion

#### `runtime`
- resolve/download/install the selected service when the app/runtime starts

### Meaning
This rule should apply consistently across:
- node
- electron
- tauri
- docker
- pkg
- nexe
- later go/rust ports

Important rule:
- packaging some services with an app at package time does **not** change the ownership model
- those services still belong in their own repos and are only bundled as selected release artifacts
- by default services may track latest, unless an app/service selection pins an exact version

---

## 13. UI Hosting Model

### Decision
The UI should be treated as a separate optional service.

### Meaning
- apps typically roll their own UI
- Service Lasso should provide its own UI as an optional service used in reference apps
- the UI can register into the routing layer like other services

### Example
- `admin.servicelasso.localhost`

---

## 14. Naming Convention

### Decision
Service repos should use one simple shared repo prefix:

- `lasso-`

After removing that prefix, the remainder should be the exact local service folder name.

### Examples
- `lasso-@archive` -> local folder `@archive`
- `lasso-@localcert` -> local folder `@localcert`
- `lasso-@node` -> local folder `@node`
- `lasso-fastapi` -> local folder `fastapi`
- `lasso-mongo` -> local folder `mongo`

### Meaning
This provides:
- simple repo-to-folder mapping
- clear visual grouping in GitHub/org listings
- easy automation
- consistent naming across all service repos

### Base/system services
Future base/system services should likely use `@` naming rather than `_` naming.

Example direction:
- `@archive`
- `@localcert`
- `@node`
- `@python`
- `@java`
- `@traefik`

### Important rule
Naming is a grouping/ordering signal.
Actual semantics must still be defined in metadata.

---

## 15. Base Service Classification

### Decision
Not all donor underscore services remain equal in the future model, and none of them are mandatory by default.

### Utility/provider capabilities
- `archive`
- `localcert`

### Runtime providers
- `node`
- `python`
- `java`

### Infrastructure
- `traefik`

### Not base by default
- `keycloak`

Important clarification:
- base/system services remain optional
- reference apps may include a selected default set
- some users may only want a subset (for example only `@archive`, or no utility services at all if their service does not require them)

---

## 16. `archive` Interpretation

### Decision
`archive` remains a service-registry-visible utility service.

### Meaning
It provides:
- archive extraction capability
- install/unpack support for service packages
- executable path/global env bindings
- independently updateable archive binaries

Important clarification:
- it is still treated as a service in the registry
- it is not expected to behave like a normal long-running app service

---

## 17. `localcert` Interpretation

### Decision
`localcert` remains a service-registry-visible utility/bootstrap service.

### Meaning
It exists to:
- generate local cert material
- support local HTTPS flows
- provide cert/root-CA paths to the routing layer and other services

Important clarification:
- it runs first-time/setup behavior
- it is considered a utility service
- it is not expected to be a normal startable long-running service
- it still appears in the service registry like `archive`

---

## 18. Runtime Provider Interpretation

### Decision
`node`, `python`, and `java` are base runtime providers.

### Meaning
Other services may declare that they require one of these runtimes.

These providers should remain explicit first-class concepts in the architecture.

---

## 19. Routing Layer Interpretation

### Decision
`traefik` should remain part of the platform foundation, but as just another service.

### Meaning
It is more than a convenience proxy. It acts as:
- routing layer
- ingress/service-host layer
- stable local URL provider

This supports URLs like:
- `admin.servicelasso.localhost`

Important clarification:
- browser/frontend interaction typically uses routed URLs
- service-to-service communication is typically done directly to ports, not routed hostnames

---

## 20. Keycloak Classification

### Decision
`keycloak` should be removed from base services.

### Meaning
It is better treated as optional infrastructure/auth service, not universal baseline capability.

---

## 21. Environment Isolation Model

### Decision
All services are sandboxed from OS-level environment variables.

### Meaning
- services should not implicitly inherit the host OS environment
- a service creator must explicitly specify what the service needs
- services are spawned only with what is available through the Service Lasso-controlled environment model
- shared service-visible environment should come from `globalenv` (and explicit service env derived from it)

### Why
This keeps service execution:
- more reproducible
- more portable
- less dependent on hidden machine state
- easier to reason about across Node/Go/Rust implementations

## 22. Debug Mode Goal

### Decision
Debug mode should keep the runtime and UI useful even if services are not fully started.

### Meaning
Debug mode should allow:
- runtime up
- UI up
- services stopped or partially unavailable
- visibility into blocked/missing service payloads

The UI should not become useless simply because full readiness has not been reached.

---

## 23. Donor Snapshot Status

### Decision
The donor code in `ref/typerefinery-service-manager-donor/` is reference material only.

### Meaning
It has been cleaned into a standalone node-only runtime shape for analysis, but it is not the final tracked Service Lasso runtime.

The real product should be built intentionally from the target architecture, not by preserving donor structure unchanged.

---

## 24. Manifest Structure Direction

### Decision
The donor-style giant `execconfig` block should be cleaned up into clearer named sections.

### Meaning
Service Lasso should not preserve one large catch-all `execconfig` shape forever.

Current direction is to split manifest concerns into clearer structured sections such as:
- runtime
- install
- network
- env
- health
- dependencies

Important clarification:
- this is a schema cleanup/normalization direction
- exact final section names and boundaries can still be refined
- the key decision is to move away from one overloaded donor catch-all block toward cleaner structured manifest sections

---

## 25. Actions Semantics

### Decision
`actions` should be treated as **service-specific custom commands that override Service Lasso default behavior for a named service action**.

### Meaning
`actions` is not:
- an arbitrary free-form custom action registry
- generic metadata
- a separate capability system unrelated to service behavior

Instead, `actions` exists so the service creator can explicitly map their service-specific command when that service needs to override what Service Lasso would normally do by default for a named action.

Examples of what this means:
- a service may define `stop` to run a graceful shutdown command instead of default process termination
- if no action override is defined, Service Lasso should use its default behavior for that action
- action expansion should be evidence-driven rather than speculative; do not widen the overrideable action list until a concrete service shows a real need

Important clarification:
- action names are still owned by Service Lasso, not invented ad hoc per service
- `actions` supplies per-service override command mappings for those named actions
- this keeps the action surface finite while still allowing service-specific behavior

---

## 26. Install Semantics Clarification

### Decision
`install` should converge a service into its current expected installed/configured state.

### Meaning
Install may include:
- acquiring/downloading payloads
- extracting/unpacking artifacts
- running setup/preparation commands
- rewriting Lasso-managed effective config/output

Important clarification:
- install should not be treated as a destructive full revert of all content back to pristine defaults
- install is closer to reconcile/materialize desired installed state than to wipe-and-reset behavior
- if install already rewrites managed config during normal convergence, a separate first-class `config` action should only be kept when a concrete lighter-weight regeneration/update scenario requires it

---

## 27. `service.state` Structure Direction

### Decision
Managed service state should move toward a `.state/` folder, with structured JSON state rather than a single overloaded flat blob.

### Meaning
Current preferred areas are:
- service reference info
- install
- content
- config
- runtime

Important clarification:
- backups should live under `.state/backups/` and be tracked as a list/history rather than a single backup field
- logs should remain logs by default; they do not need to become state snapshots unless a concrete need appears
- avoid a separate pointer/current file unless a concrete need appears
- avoid a separate legacy `service.pid` file unless a concrete need appears; PID/runtime info can live in the structured state JSON itself

Current lifecycle-write rule:
- when a service starts, its state JSON is created
- when the service stops, that same state JSON is updated with the last action/result
- other lifecycle events should generally write logs for now rather than creating additional timestamped state files by default

This keeps managed operational state simpler while preserving a clear last-known service record.

---

## 28. Canonical Reconciliation Source

### Decision
`QUESTION-LIST-AND-CODE-VALIDATION.md` is the canonical reconciliation doc for the main donor/runtime-boundary questions.

### Meaning
If later planning text drifts away from:
- the settled transcript answers
- the donor code evidence
- the reconciled four-question boundary set

then the reconciled question doc should be treated as more authoritative than older speculative planning sections unless the discussion is explicitly reopened.

---

## 29. Working Summary

Service Lasso is currently intended to become:

- a canonical Node-first orchestration runtime
- a publishable npm module
- a buildable plain-JS runtime executable with Node
- a dynamic installed-service platform
- a system with explicit utility services, runtime providers, and routing infrastructure
- a product with separate reference hosts/packages and later true Go/Rust ports
- a platform where services are independently versioned repos resolved to exact releases when chosen, but may default to latest unless pinned

---

_Add new decisions here as they become stable._
