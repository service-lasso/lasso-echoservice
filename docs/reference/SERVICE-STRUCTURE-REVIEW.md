# Service Lasso - Donor Service Structure Review

_Status: working draft / template-design input_

Important reconciliation note:
- this file reviews donor service structure and extracts patterns
- use `QUESTION-LIST-AND-CODE-VALIDATION.md` and `ARCHITECTURE-DECISIONS.md` first when deciding what is settled versus what is only a proposed migration/template direction

Purpose:

- review the current donor `services/` folder structure
- identify the main service shapes/patterns
- extract what should become the basis for a **template service repo**
- clarify what is donor baggage vs what is a reusable contract

---

## 1. Key Design Intent

The `services/` folder should remain **dynamic**, not fixed.

Target behavior:

- Service Lasso can discover installed services under `services/`
- services can be added from external sources (for example GitHub releases)
- the runtime can download/install a release package into `services/`
- the service manager can then load and run the installed service
- every service should live in its own repo, even when some services are bundled into an app package at package time

So the donor `services/` folder is best interpreted as:

- a **local installed service inventory**
- not a permanently source-controlled monolith

That means this review is about defining the **service package/install contract**.

---

## 2. Donor Service Inventory

Current donor service directories reviewed:

### Runtime / utility services
- `_archive`
- `_java`
- `_keycloak`
- `_localcert`
- `_node`
- `_python`
- `_traefik`

### Application / data / platform services
- `bpmn-server`
- `cms`
- `fastapi`
- `filebeat`
- `files`
- `jupyterlab`
- `messageservice-client`
- `mongo`
- `nginx`
- `openobserve`
- `postgredb`
- `postgredb-admin`
- `totaljs-flow`
- `totaljs-messageservice`
- `typedb`
- `typedb-init`
- `typedb-sample`
- `wsecho`

### Non-service app/example folder observed
- `bpmn-client`

Important note:
- `bpmn-client` does **not** contain `service.json`, so it is not currently in the same runtime-managed contract as the actual services.

---

## 3. What Almost Every Managed Service Has

The donor contract strongly centers on a per-service folder containing:

- `service.json`
- optional README / notes
- optional payload archives or platform binaries
- optional config/data/templates/scripts
- optional app source/runtime files

The single biggest contract artifact is:

- `services/<service-id>/service.json`

This is the thing the runtime discovers and interprets.

---

## 4. Common Top-Level Service Folder Shapes

Across the donor services, the main recurring folder shapes are:

### A. Utility runtime provider services
Examples:
- `_node`
- `_python`
- `_java`
- `_archive`

Typical contents:
- `service.json`
- platform folders like `win32/`, `darwin/`, `linux/`
- setup archive(s) or runtime payloads
- README / notes

Purpose:
- provide a runtime binary/toolchain used by other services
- often referenced via `execservice`

### B. Archive-installed infrastructure services
Examples:
- `mongo`
- `postgredb`
- `_traefik`
- `openobserve`
- `typedb`
- `_keycloak`

Typical contents:
- `service.json`
- payload archives or split archive parts
- platform-specific runtime folders
- config/data directories
- README / notes

Purpose:
- unpack and run a packaged infrastructure dependency

### C. Source/application services run via a utility runtime
Examples:
- `fastapi`
- `messageservice-client`
- `postgredb-admin`
- `wsecho`
- `typedb-sample`
- `typedb-init`
- `bpmn-server`
- `files`
- `totaljs-flow`
- `totaljs-messageservice`

Typical contents:
- `service.json`
- app source/runtime files
- language-specific dependency manifests
  - `package.json`
  - `requirements.txt`
- config/data/templates/scripts

Purpose:
- run app code using another service/runtime provider such as `node` or `python`

### D. Config-heavy proxy/cert/bootstrap services
Examples:
- `nginx`
- `_localcert`
- `_traefik`

Typical contents:
- `service.json`
- config folders
- command templates / command config
- generated cert/data folders

Purpose:
- support the wider service graph rather than act as end-user apps

---

## 5. `service.json` Contract - Main Observed Structure

Most services follow this high-level shape:

```json
{
  "id": "...",
  "name": "...",
  "description": "...",
  "enabled": true,
  "status": "...",
  "icon": "...",
  "servicetype": 50,
  "servicelocation": 10,
  "execconfig": {
    "...": "..."
  }
}
```

### Common top-level fields observed
- `id`
- `name`
- `description`
- `enabled`
- `status`
- `logoutput`
- `icon`
- `servicetype`
- `servicelocation`
- `version` (some services)
- `actions` (some services)
- `execconfig`

### Common `execconfig` fields observed
- `serviceorder`
- `serviceport`
- `serviceportsecondary`
- `serviceportconsole`
- `serviceportdebug`
- `portmapping`
- `executable`
- `executablecli`
- `execservice`
- `execshell`
- `execcwd`
- `commandline`
- `commandlinecli`
- `commandconfig`
- `setup`
- `setuparchive`
- `datapath`
- `env`
- `globalenv`
- `depend_on`
- `healthcheck`
- `authentication`
- `outputvarregex`
- `urls`
- `ignoreexiterror`
- `debuglog`

---

## 6. Main Runtime Patterns In The Donor Contract

### Pattern 1 - `execservice`
A service can depend on another service to provide the executable/runtime.

Examples in spirit:
- app service runs via `node`
- app service runs via `python`
- dependent tool uses `java`

This is useful and should survive.

### Pattern 2 - `setuparchive`
A service can install/unpack a payload archive before runtime.

This also maps well to the future GitHub-release install model.

### Pattern 3 - `depend_on`
Services explicitly declare dependency order.

This is core and should survive.

### Pattern 4 - `healthcheck`
Services declare their runtime health expectations.

This is core and should survive, but may need normalization.

### Pattern 5 - environment templating
Services rely heavily on `env` and `globalenv` generation.

This is useful, but the exact donor shape is likely too app-specific and should be cleaned.

---

## 7. Donor Issues / Weaknesses Observed

These are important because the template repo should avoid them.

### A. Mixed concerns in a single folder
Many donor service folders mix:
- install payloads
- app source
- generated data
- runtime state
- config
- docs

This makes services less portable and harder to reason about.

### B. Bundled payload assumptions
Some donor services assume archives/binaries are already present locally.

For Service Lasso, these should be installable dynamically from a release source, not assumed to already exist.

### C. Runtime state mixed with package content
Some folders contain data/runtime/generated state next to source/config.

The cleaner model is:
- service package content
- separate installed/runtime data/state locations

### D. Service contract is powerful but under-normalized
`service.json` currently mixes several concerns:
- metadata
- install instructions
- runtime instructions
- env generation
- health
- dependency graph
- auth/config hints

This should probably be split conceptually, even if a single file remains possible for compatibility.

---

## 8. Recommended Conceptual Split For A Service Package

For Service Lasso, each service should conceptually have three layers:

### 1. Source definition
Where does the service come from?

Examples:
- GitHub release
- local path
- bundled core service

### 2. Installed artifact
What is installed locally right now?

Examples:
- installed version
- install path
- checksum
- installed timestamp

### 3. Runtime definition
How is it executed and supervised?

Examples:
- executable or execservice
- ports
- env
- dependencies
- health checks

The donor `service.json` currently tends to compress all of this into one place.

---

## 9. Proposed Template Service Repo Goal

A template service repo should make it easy to publish a service that Service Lasso can install and run.

That template should be designed so a service can be:

- built/published independently
- packaged for release
- downloaded/installed into `services/`
- discovered by the runtime
- started with predictable setup/runtime behavior

---

## 10. Proposed Template Service Repo Structure

Suggested baseline template:

```text
service-template/
  service.json
  README.md
  CHANGELOG.md
  LICENSE
  release/
    manifest.json
  config/
  scripts/
  assets/
  runtime/
  package/
```

If the service is source-based, maybe:

```text
service-template/
  service.json
  README.md
  src/
  config/
  scripts/
  package.json | requirements.txt | etc
```

But for release/installability, I think the final packaged artifact should always converge on a predictable installed shape.

---

## 11. Proposed Installed Service Folder Shape

Suggested installed layout under local `services/`:

```text
services/
  <service-id>/
    service.json
    service.state
    service.pid
    config/
    installed/
```

Possible meanings:

- `service.json`
  - canonical service manifest / contract
- `service.state`
  - mutable local install/resolution/state information
  - for example resolved version, installed version, install timestamps, selected mode, checksum, setup/install flags
- `service.pid`
  - current running pid / process state file
- `config/`
  - config content coming from repo/package
- `installed/`
  - extracted/installed service payload

This keeps the local `services/` directory dynamic and installable while clearly separating immutable contract from mutable runtime/install state.

---

## 12. Proposed Template `service.json` Sections

For the future template, I would normalize the contract into clearer sections.

Example direction:

```json
{
  "id": "example-service",
  "name": "Example Service",
  "version": "1.0.0",
  "kind": "application",
  "source": {
    "type": "github-release"
  },
  "install": {
    "strategy": "archive",
    "entry": "runtime/"
  },
  "runtime": {
    "execservice": "node",
    "command": "main.js",
    "cwd": "runtime",
    "ports": {
      "service": 8000
    },
    "env": {},
    "dependsOn": []
  },
  "health": {
    "type": "http",
    "url": "http://localhost:${SERVICE_PORT}/health"
  }
}
```

This is conceptually cleaner than the current flat donor `execconfig` shape.

---

## 13. Recommended Template Repo Files

The template repo should likely include:

### Required
- `service.json`
- `README.md`
- release packaging config
- runtime entrypoint or executable target

### Strongly recommended
- `CHANGELOG.md`
- `LICENSE`
- `scripts/build.*`
- `scripts/package.*`
- `scripts/verify.*`
- release manifest/checksum file

### Optional
- sample config
- healthcheck probe script
- migration/setup scripts
- platform-specific packaging helpers

---

## 14. Suggested Service Categories For Template Thinking

It may help to define template variants:

### Template A - Runtime provider service
Examples:
- node
- python
- java

### Template B - Archive-installed infrastructure service
Examples:
- mongo
- postgres
- traefik
- openobserve

### Template C - Source/app service using a runtime provider
Examples:
- fastapi app
- node app
- python worker

### Template D - Setup/bootstrap job service
Examples:
- typedb-init
- data/bootstrap/seed services

These likely want slightly different template defaults, even if they share a common contract.

---

## 15. Best Current Extraction For The Template Repo Contract

If we strip the donor clutter away, the main reusable contract appears to be:

A service package should provide:

1. **identity**
   - id, name, version, description
2. **runtime requirements**
   - executable or runtime provider
3. **install strategy**
   - archive/setup/source layout
4. **dependency graph**
   - what must exist/run first
5. **ports**
   - declared or mapped ports
6. **env/config**
   - service env and generated globals
7. **health**
   - how the manager knows it is alive/ready
8. **operator metadata**
   - icon, actions, docs links if desired

---

## 16. Packaging / Installation Modes

Current direction from Max:

All Service Lasso app/reference hosts should support two service-distribution modes:

### A. Bundled / base-services mode
The app package includes a base set of services pre-packaged with the app.

Use cases:
- smoother offline or low-friction startup
- known-good default runtime set
- reference/demo distributions

Implications:
- app packages need a manifest of included base services
- installed-service registration must recognize pre-bundled services
- updates must distinguish bundled version vs locally updated version

### B. Lightweight / bootstrap-download mode
The app package ships light and downloads required/base services on first start.

Use cases:
- smaller installer/package size
- faster app release cycles
- dynamic service acquisition from GitHub releases or other sources

Implications:
- runtime needs bootstrap install flow
- source/install metadata must be first-class
- first-run UX must clearly show install/download progress and failures

This reinforces that the `services/` directory is a dynamic installed inventory, not a fixed static source tree.

## 17. Immediate Recommendation

Use the donor service review to define:

- a **normalized service package contract**
- a **template service repo structure**
- a **local installed service layout** under `services/`

Do **not** copy the donor service folders 1:1 as the final model.

The donor set is valuable as a reference corpus, but the final template/service packaging model should be cleaner, more installable, and explicitly designed for dynamic GitHub-release-based installation.

---

## 18. Base Service Classification Notes

Current direction from discussion:

### Base utility/provider capabilities
- `_archive`
  - donor config confirms this is effectively a packaged archive tool provider (7zip / 7zz)
  - exports global env like `ARCHIVE` and `ARCHIVE_HOME`
  - likely to evolve toward an internal provider/tool abstraction rather than a normal service
- `_localcert`
  - donor config confirms this is a local cert/bootstrap provider
  - generates local cert material and exports values like `CERT_FILE`, `CERT_KEY`, `CERT_PFX`, `CAROOT_CERT`
  - used to support local HTTPS hosting flows for the routing layer

### Base runtime providers
- `_node`
  - donor config confirms this is a packaged Node runtime provider with install archive + exported globals `NODE`, `NODE_HOME`, `NPM`
- `_python`
  - donor config confirms this is a packaged Python runtime provider with setup/bootstrap logic and exported globals `PYTHON`, `PYTHON_HOME`
- `_java`
  - donor config confirms this is a packaged Java runtime provider with exported globals `JAVA`, `JAVA_HOME`

These are strong donor concepts worth preserving, but they should become explicit runtime-provider roles rather than relying on underscore-prefixed naming.

### Base infrastructure
- `_traefik`
  - donor config confirms this is more than a simple proxy convenience
  - acts as the routing/ingress/service-host layer
  - owns stable local URLs and hostnames
  - supports service-to-service communication via local routed hostnames
  - depends on `localcert` and `nginx` in the donor setup

### Removed from base classification
- `_keycloak`
  - should not remain a base service
  - better treated as optional infrastructure/auth service

## 19. Open Questions For Next Iteration

- should `service.json` stay single-file or split into `source/install/runtime` files?
- should installed versions be symlinked or copied into `current/`?
- should a service package contain runtime state/data at all, or must that always live elsewhere?
- how should checksums/signatures be represented?
- how should secrets/config references be expressed?
- should actions/admin UI metadata live in the service contract or in a separate UI descriptor?
- what should the exact release artifact contract look like for GitHub-installed services?

---

## 20. Working Summary

The donor services folder demonstrates a useful runtime contract centered on `service.json`, dependencies, setup archives, runtime providers, and health checks.

For Service Lasso, this should evolve into a cleaner dynamic service package/install model where:

- services are installable from release sources
- local `services/` is an installed inventory
- each service has a clearer split between source, install, and runtime concerns
- a template service repo can be published independently and then installed by the manager

---

_Add per-service notes and refinements as we continue reviewing._
