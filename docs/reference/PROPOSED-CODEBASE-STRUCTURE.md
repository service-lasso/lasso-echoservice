# Service Lasso - Proposed Codebase Structure

_Status: working draft_

Important reconciliation note:
- this file is a structural sketch, not a canonical decision log
- use `QUESTION-LIST-AND-CODE-VALIDATION.md` and `ARCHITECTURE-DECISIONS.md` to decide what is actually settled before treating any proposed layer/folder split here as fixed

Purpose:

- document the proposed target codebase structure for `service-lasso`
- separate donor file boundaries from the future architectural layers
- provide a concrete structure to refine while design decisions continue

Related docs:

- `ARCHITECTURE.md`
- `ARCHITECTURE-DECISIONS.md`
- `SERVICE-STRUCTURE-REVIEW.md`
- `SERVICE-MANAGER-BEHAVIOR.md`
- `RUNTIME-API-INDEX.md`
- `SERVICE-TEMPLATE-REPO.md`

---

## 1. Main Goal

The donor runtime should be restructured into clear layers for:

- runtime/orchestration
- services/instances
- install/update/bootstrap
- env/ports/health/routing
- API
- UI
- CLI
- shared contracts/types

The future structure should follow conceptual ownership, not donor file history.

---

## 2. Proposed Main Repo Structure

```text
service-lasso/
  src/
    core/
      runtime/
      services/
      instances/
      install/
      env/
      ports/
      health/
      routing/
    api/
      server/
      routes/
      schemas/
    ui/
      app/
      assets/
    cli/
    shared/
      contracts/
      utils/
      system/
    paths/
  templates/
    service-repo/
    app-node/
  examples/
  docs/
  package.json
  tsconfig.json
```

---

## 3. Core Layer Breakdown

### `src/core/runtime/`
Responsibilities:
- top-level runtime startup/shutdown
- runtime lifecycle
- runtime events/state
- wiring managers together

Candidate files:
- `Runtime.ts`
- `RuntimeConfig.ts`
- `RuntimeLifecycle.ts`
- `RuntimeEvents.ts`

### `src/core/services/`
Responsibilities:
- service definitions
- service discovery/registry
- service manager logic
- service dependency graph
- service state model

Candidate files:
- `Service.ts`
- `ServiceManager.ts`
- `ServiceRegistry.ts`
- `ServiceDefinition.ts`
- `ServiceState.ts`
- `ServiceDependencyGraph.ts`

### `src/core/instances/`
Responsibilities:
- app/instance manifests
- instance lifecycle
- grouping services into a named instance

Candidate files:
- `Instance.ts`
- `InstanceManager.ts`
- `InstanceManifest.ts`
- `InstanceState.ts`

### `src/core/install/`
Responsibilities:
- install/update/bootstrap flows
- archive extraction
- service source resolution
- installed service tracking
- bundled vs lightweight bootstrap behavior

Candidate files:
- `InstallManager.ts`
- `Installer.ts`
- `ArchiveProvider.ts`
- `BootstrapInstaller.ts`
- `ServiceSource.ts`
- `GitHubReleaseSource.ts`
- `InstalledServiceStore.ts`

### `src/core/env/`
Responsibilities:
- service env generation
- global env propagation or equivalent replacement model
- template substitution
- runtime provider env helpers

Candidate files:
- `EnvBuilder.ts`
- `GlobalEnvRegistry.ts`
- `TemplateVars.ts`
- `RuntimeProviderEnv.ts`

### `src/core/ports/`
Responsibilities:
- port reservation/allocation
- port inspection/tracking
- future port-block support

Candidate files:
- `PortRegistry.ts`
- `PortAllocator.ts`
- `PortReservation.ts`
- `PortInspector.ts`

### `src/core/health/`
Responsibilities:
- health check definitions
- health check runners
- readiness/liveness aggregation

Candidate files:
- `HealthCheck.ts`
- `HealthCheckRunner.ts`
- `HealthStatus.ts`

### `src/core/routing/`
Responsibilities:
- local routed hostnames
- service host registration
- reverse-proxy/routing integration

Candidate files:
- `RoutingManager.ts`
- `RouteDefinition.ts`
- `ServiceHostname.ts`

---

## 4. API Layer

### `src/api/`
This becomes the primary control plane.

Candidate structure:

```text
src/api/
  server/
    createServer.ts
    RuntimeServer.ts
  routes/
    runtime.ts
    services.ts
    instances.ts
    installs.ts
    logs.ts
    health.ts
  schemas/
    runtime.ts
    services.ts
    instances.ts
```

The API should replace host-specific IPC as the default core control surface.

---

## 5. UI Layer

### `src/ui/`
The UI should be separable from the core runtime bootstrap.

Candidate structure:

```text
src/ui/
  app/
    pages/
    components/
    hooks/
    client/
  assets/
```

The UI should support two modes:
- embedded UI served directly by the runtime
- separate managed UI service registered into the routing layer

---

## 6. CLI Layer

### `src/cli/`
The CLI should expose runtime/service/install/instance operations.

Candidate structure:

```text
src/cli/
  index.ts
  commands/
    runtime.ts
    service.ts
    instance.ts
    install.ts
```

---

## 7. Shared Contracts / Utilities

### `src/shared/contracts/`
This should hold the canonical contract types that later ports must mirror.

Candidate files:
- `service-manifest.ts`
- `instance-manifest.ts`
- `install-manifest.ts`
- `base-service-types.ts`

### `src/shared/utils/`
Cross-cutting helper functions.

### `src/shared/system/`
Platform/process/system abstractions.

---

## 8. Path Layer

### `src/paths/`
Replace donor `Resources.ts` with explicit runtime path modules.

Candidate files:
- `RuntimePaths.ts`
- `DataPaths.ts`
- `ServicePaths.ts`

---

## 9. Templates and Examples

### `templates/service-repo/`
Canonical template for one-service-per-repo service packages.

### `templates/app-node/`
Reference Node host/app template.

### `examples/`
Examples of:
- basic instances
- multi-service instances
- maybe packaged/bundled vs lightweight setups later

---

## 10. Local Installed Service Layout (Outside `src/`)

The installed `services/` directory should be treated as a local dynamic inventory.

Example direction:

```text
services/
  @archive/
  @localcert/
  @node/
  @python/
  @java/
  @traefik/
  fastapi/
  mongo/
  postgredb/
```

Potential per-service layout:

```text
services/
  <service-name>/
    service.json
    service.state
    service.pid
    config/
    installed/
```

Current direction:
- `service.json` is the canonical manifest/contract
- `service.state` stores mutable local install/resolution/state information
- `service.pid` stores the current running pid/process state
- `config/` comes from the repo/package
- `installed/` is created by the install process

---

## 11. Mapping From Donor Files

### Donor `Service.ts`
Likely split across:
- `src/core/services/Service.ts`
- `src/core/install/Installer.ts`
- `src/core/health/HealthCheckRunner.ts`
- `src/core/env/EnvBuilder.ts`
- `src/core/ports/PortReservation.ts`

### Donor `ServiceManager.ts`
Likely split across:
- `src/core/services/ServiceManager.ts`
- `src/core/services/ServiceRegistry.ts`
- `src/core/runtime/Runtime.ts`
- `src/core/ports/PortRegistry.ts`

### Donor `Services.ts`
Likely split across:
- `src/api/server/createServer.ts`
- `src/ui/...`
- `src/cli/...`
- maybe `src/core/runtime/bootstrap.ts`

### Donor `Utils.ts`
Likely split across:
- `src/shared/utils/`
- `src/shared/system/`
- `src/core/ports/`
- `src/core/env/`

### Donor `Resources.ts`
Likely replaced by:
- `src/paths/RuntimePaths.ts`

---

## 12. Reference Hosts / Separate Repos

Outside the core repo, reference hosts remain separate:

- `service-lasso-app-node`
- `service-lasso-app-electron`
- `service-lasso-app-tauri`
- `service-lasso-app-docker`
- `service-lasso-app-pkg`
- `service-lasso-app-nexe`

The core repo should not absorb their host-specific concerns.

---

## 13. Service Repos / Separate Repos

All services should live in their own repos.

Naming rule currently documented elsewhere:
- repo prefix `lasso-`
- remove `lasso-`
- remainder is exact local service folder name

Examples:
- `lasso-@archive`
- `lasso-@node`
- `lasso-fastapi`
- `lasso-mongo`

---

## 14. Working Summary

The proposed future structure separates the current donor runtime into:
- core runtime/orchestration
- install/update/bootstrap
- services/instances
- ports/env/health/routing
- API
- UI
- CLI
- shared contracts

This should make the final Service Lasso codebase much easier to reason about and much closer to the architecture we are actually designing.
