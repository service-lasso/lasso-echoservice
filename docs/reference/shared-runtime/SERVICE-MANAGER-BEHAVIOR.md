# Service Lasso - Current Service Manager Behavior Review

_Status: working draft / donor behavior analysis_

Important reconciliation note:
- this is a donor behavior/evidence doc
- use `QUESTION-LIST-AND-CODE-VALIDATION.md` for the reconciled transcript+code answers to the main runtime-boundary questions
- this file should explain what the donor currently does, not override settled product-direction answers from transcript

Purpose:

- document how the current donor-derived service manager actually behaves
- record how `service.json` is used by the runtime
- create a place to keep picking apart the current manager behavior as part of the refactor/design process

Related docs:

- `ARCHITECTURE.md`
- `ARCHITECTURE-DECISIONS.md`
- `SERVICE-STRUCTURE-REVIEW.md`
- `SERVICE-TEMPLATE-REPO.md`

---

## 1. Primary Contract File

The current service manager treats:

- `service.json`

as the canonical per-service manifest/runtime contract file.

In the donor-derived runtime:

- `ServiceManager.ts` sets `#serviceConfigFile = "service.json"`
- the manager scans for service manifests using a glob under the services root

Current discovery pattern:

- `*/service.json`

So each direct service folder under the services root is expected to provide a `service.json`.

---

## 2. Discovery Behavior

When the runtime reloads:

1. it clears the current service list
2. it locates service config files
3. it parses each `service.json`
4. it turns each parsed config into a `ServiceConfig`
5. it constructs `Service` objects from those configs
6. it updates global env
7. it sorts services
8. it notifies the UI/app callbacks with service list + global env

This means `service.json` is loaded very early and is the starting point for almost all runtime behavior.

---

## 3. Path Resolution Behavior

For each discovered `service.json`, the manager derives several important paths.

### `servicehome`
The root service folder containing `service.json`.

### `servicepath`
The runtime path used for platform-specific execution.

Behavior:
- if a platform subfolder exists for the current platform (`win32`, `darwin`, `linux`), it prefers that platform folder as the service execution path
- otherwise it falls back to the service root

### `servicesdataroot`
A derived data root based on the service folder name.

Implication:
- service identity is tied closely to the folder name
- platform-specific binaries/payloads are expected to live under per-platform subfolders when applicable

---

## 4. How `service.json` Is Consumed

Each parsed `service.json` becomes the `options` passed into the `Service` constructor.

This means the runtime logic in `Service.ts` treats `service.json` as the main source of truth for how to:

- prepare the service
- resolve its executable
- build commands
- set env
- assign ports
- wait for dependencies
- check health
- start/stop/restart it

In other words, `service.json` is not just descriptive metadata — it is an operational runtime contract.

---

## 5. Main Runtime Concerns Driven By `execconfig`

The most important section inside `service.json` is:

- `execconfig`

This section currently drives most service-manager behavior.

### A. Execution/runtime selection
Fields used include:
- `executable`
- `executablecli`
- `execservice`
- `execshell`
- `execcwd`

This determines:
- which executable to run
- whether the service runs directly or through another runtime provider
- current working directory / shell behavior

### B. Command construction
Fields used include:
- `commandline`
- `commandlinecli`
- `commandconfig`

This determines the actual command/arguments/config expansion used at runtime.

### C. Setup/install behavior
Fields used include:
- `setup`
- `setuparchive`

This determines whether a service needs archive extraction and/or bootstrap steps before runtime.

Important explicit note:
- install/setup in the current manager can include **exec commands**, not just archive extraction
- `setuparchive` covers extraction/unpack behavior
- `setup` covers platform-specific setup command execution
- both are part of the current install/setup flow

### D. Runtime paths/state
Fields used include:
- `datapath`

This influences where service data/state should live.

### E. Networking / ports / URLs
Fields used include:
- `serviceport`
- `serviceportsecondary`
- `serviceportconsole`
- `serviceportdebug`
- `portmapping`
- `urls`

This determines reserved ports, exposed URLs, and how service endpoints are described.

### F. Health/readiness
Fields used include:
- `healthcheck`

This determines how the runtime checks whether a service is started/healthy.

### G. Dependency graph
Fields used include:
- `depend_on`
- `execservice`

This determines both explicit service dependencies and indirect runtime-provider relationships.

### H. Environment generation
Fields used include:
- `env`
- `globalenv`
- `outputvarregex`

This determines both service-local env and the cross-service/global env propagation behavior.

### I. Auth / behavior flags
Fields used include:
- `authentication`
- `ignoreexiterror`
- `debuglog`

These influence startup/runtime handling behavior.

---

## 6. Global Environment Behavior

The service manager aggregates global environment variables across services.

Behavior:
- each service may expose `globalenv`
- the manager merges these into a shared global environment map
- the merged global env is then pushed back into services
- the manager also emits this global env outward to the UI/app callbacks

Current clarified design direction:
- services should be sandboxed from OS-level environment variables
- service creators must explicitly specify what a service needs
- services should be spawned only with the Service Lasso-controlled environment model (shared `globalenv` plus explicit service env derived from it)

Implication:
- `service.json` is currently part of a cross-service env propagation system
- services do not live in total isolation; they contribute to a shared runtime environment graph
- future Service Lasso should treat env exposure as explicit contract, not host-machine leakage

This is one of the strongest reasons the manifest format matters so much.

---

## 7. Startup Ordering Behavior

The manager sorts services using:

- `execconfig.serviceorder`

with a default fallback if not provided.

This means `service.json` currently affects:
- orchestration order
- startup sequencing
- the relative priority of base/runtime/infrastructure services

This is another reason the file is more than just metadata.

---

## 8. Dependency Behavior

The current runtime uses dependencies in two ways:

### Explicit dependencies
- `depend_on`

### Runtime-provider dependencies
- `execservice`

Meaning:
- a service can directly depend on named services
- a service can also depend on another service to provide the executable/runtime used to launch it

This is a powerful model and one of the more important donor concepts worth preserving.

---

## 9. Setup / Install Behavior

The current runtime includes service-local setup logic directly in the service contract.

Examples of behavior supported by the current manager:
- unpacking archives before use
- running setup commands before runtime
- preparing dependent/runtime-provider services first when required
- writing setup state markers used to decide whether the service is already prepared

Important clarified finding from donor code review:
- current donor `setup` is not just a narrow installer stage
- it is functioning as a combined preparation flow that includes install/bootstrap behavior and some config/preparation behavior
- because of that, donor `setup` should be treated as evidence of current entanglement, not as the ideal future action model
- this is one of the strongest code-level reasons to keep future `install` and `config` separate in Service Lasso even though donor currently mixes them
- capturing output values back into env via regex parsing

Important interpretation:
- current install behavior is not just “download + unzip”
- it is a compound install/setup flow that can include both archive extraction and exec-command-based setup steps
- this precedent is important for the future `install` action design

Implication:
- the current manager already mixes install/bootstrap and runtime behavior in the same manifest-driven flow
- this is useful, but also a reason to normalize/split concerns more cleanly in the future

---

## 10. Why JSON Was Useful Here

The current manager behavior makes it clear why JSON was practical.

`service.json` currently serves as a portable machine-readable contract for:

- service discovery
- path resolution
- startup ordering
- dependency graph
- runtime provider selection
- setup/install instructions
- command construction
- env/global env generation
- health checks
- ports and URLs

So JSON was not just chosen for convenience as metadata storage.
It became the core runtime/orchestration contract file.

---

## 11. Current Status-Band Observation

The current service status model uses numeric-ish string codes that are already grouped in rough bands.

Examples observed:
- invalid/error band: negative values
- loaded/setup/archive/install band: `1` -> `30`
- resolve/available band: `35` -> `50`
- stopping/stopped band: `65` -> `80`
- starting/dependencies/health band: `90` -> `120`
- completion band: `200`+

Important interpretation:
- these values are not random
- they already imply grouped lifecycle bands/categories in the current design
- this supports the idea that future Service Lasso can keep the conceptual banding while still improving the clarity of the state model
- the single state machine also has value because it gives one central place for action/lifecycle decisions

So the critique is not that the current model has no structure.
The critique is that the current grouped states are still overloaded into one broad enum that mixes several concerns.

## 12. Main Design Tension Exposed By Current Behavior

The current manifest is useful, but overloaded.

A single `service.json` currently mixes:

- identity/metadata
- source/install concerns
- runtime execution concerns
- env concerns
- dependency concerns
- health concerns
- routing/network concerns

This is one of the central design tensions to resolve in Service Lasso.

Future Service Lasso may still keep JSON, but should likely normalize the structure more clearly.

---

## 13. Working Summary

The current donor-derived service manager uses `service.json` as the canonical runtime contract file for discovery, orchestration, setup, execution, env propagation, and health/status behavior.

This confirms that JSON was serving a real system-level purpose, not just acting as descriptive metadata.

It also confirms that a future Service Lasso refactor should treat the manifest contract as a first-class architectural element, even if the schema is cleaned up and split into clearer conceptual sections.

---

_Add more current-manager behavior notes here as we continue dissecting the runtime._
