# service.json Reference

_Status: current package reference_

This doc is the package-level reference for the Echo Service `service.json` manifest.

The core Service Lasso repository owns the full schema. This page records the subset used by Echo Service and follows the canonical `endpoints[]` contract.

## What this doc covers

- top-level manifest purpose
- common top-level fields
- `actions`
- `execconfig`
- env, dependencies, and `endpoints[]`
- `healthchecks[]` direction
- examples
- what is currently canonical vs still illustrative

## Important current rule

The current template direction is:
- **default health model = `process`**
- other health models are used only when explicitly declared by service config

Ref/code-backed donor healthchecks[] types observed:
- `http`
- `tcp`
- `file`
- `variable`

## Purpose of `service.json`

`service.json` is the canonical service manifest used by Service Lasso to understand how a service should be discovered, prepared, executed, and monitored.

At a high level it carries:
- identity
- operator metadata
- lifecycle/action hints
- runtime execution settings
- environment settings
- dependency hints
- health expectations

## Canonical endpoint authoring pattern

The checked-in [`service.json`](../service.json) declares each listener as a network endpoint, feeds resolved ports into the existing `ECHO_*` environment variables, and declares operator-facing links as URL endpoints. A reduced example is:

```json
{
  "id": "echo-service",
  "name": "Echo Service",
  "description": "Go-based harness service used for Service Lasso integration, runtime hardening, and supervision testing.",
  "enabled": true,
  "version": "0.3.0",
  "executable": "go",
  "args": ["run", "."],
  "env": {
    "ECHO_PORT": "${endpoint.service.port}",
    "ECHO_HTTP_HEALTH_PORT": "${endpoint.http_health.port}",
    "ECHO_TCP_PORT": "${endpoint.tcp_health.port}"
  },
  "endpoints": [
    {
      "id": "service",
      "kind": "network",
      "transport": "tcp",
      "protocol": "http",
      "bind": "127.0.0.1",
      "port": { "default": 4010, "strategy": "preferred" },
      "exposure": "local",
      "required": true,
      "primary": true
    },
    {
      "id": "ui",
      "kind": "url",
      "target": "service",
      "url": "http://${endpoint.service.bind}:${endpoint.service.port}/",
      "exposure": "local",
      "primary": true
    }
  ],
  "healthchecks": [
    {
      "id": "process-ready",
      "type": "process"
    }
  ]
}
```

The full manifest also declares the dedicated `http_health` and `tcp_health` network endpoints plus the `service_health` and `http_health_url` URL endpoints.

## Top-level fields

### `id`
Unique service identifier.

Example:
```json
"id": "echo-service"
```

Current direction:
- required
- should be stable
- should align with the service repo’s identity

### `name`
Human-facing display name.

Example:
```json
"name": "Echo Service"
```

### `description`
Short operator-facing description.

### `enabled`
Whether the service is enabled by default.

### `version`
Current package/version identity for the service.

### `logoutput`
Whether stdout/stderr style runtime logging should be captured/displayed.

### `icon`
UI/operator-facing icon hint.

### `servicetype`
Current donor-style service type classification value.

### `servicelocation`
Current donor-style service location classification value.

## `actions`

`actions` is where the service defines or overrides named lifecycle actions.

Current intended rule:
- actions correspond to known Service Lasso lifecycle/action names
- service config can override how a named action behaves for that service
- if a service does not override a supported action, Lasso default behavior applies

Current sample actions:
- `install`
- `config`
- `start`
- `stop`

### Current action examples

```json
"actions": {
  "install": {
    "description": "Prepare the sample runtime payload if needed."
  },
  "config": {
    "description": "Materialize effective runtime config for the sample service."
  },
  "start": {
    "description": "Start the sample echo service."
  },
  "stop": {
    "description": "Stop the sample echo service gracefully."
  }
}
```

### Current action semantics direction
- `install`
  - prepare/install payload and required local setup
- `config`
  - materialize effective config from explicit inputs
- `start`
  - launch the service runtime
- `stop`
  - stop the service gracefully

Additional action names may exist later, but this first-pass template should stay small and lifecycle-focused.

## `execconfig`

`execconfig` contains the runtime execution contract.

This is where the service tells Lasso how to run and supervise it.

### `serviceorder`
Startup ordering hint.

Example:
```json
"serviceorder": 100
```

### `execcwd`
Execution working directory.

Example:
```json
"execcwd": "runtime"
```

### `executable`
Executable or executable key/name used for the service runtime.

Example:
```json
"executable": "echo-service"
```

### `env`
Service-local environment variables.

Example:
```json
"env": {
  "ECHO_MESSAGE": "hello from service-template"
}
```

Current direction:
- service env should be explicit
- avoid depending on uncontrolled host-machine env leakage

### `depend_on`
Explicit dependencies.

Example:
```json
"depend_on": []
```

Current direction:
- use this for services that require another service/runtime/provider first
- keep empty for the minimal sample

## Healthchecks

### Default rule
Current rule:
- if a service does not explicitly require another model, the default is **`process`**

Example:
```json
"healthchecks": [
  {
    "id": "process-ready",
    "type": "process"
  }
]
```

This is the right default for a simple sample service.

### Observed donor healthchecks[] types
The donor runtime/code shows these healthchecks[] types:
- `http`
- `tcp`
- `file`
- `variable`

`process` is the current template default direction, even though the donor code paths most explicitly surfaced in ref material are the four types above.

### `process` healthchecks[] item
Use when:
- service health is adequately represented by the process being up/running
- you do not need a deeper readiness endpoint yet

Sample:
```json
"healthchecks": [
  {
    "id": "process-ready",
    "type": "process"
  }
]
```

### `http` healthchecks[] item
Use when:
- the service exposes an HTTP readiness or health endpoint

Sample:
```json
"healthchecks": [
  {
    "id": "http-ready",
    "type": "http",
    "url": "http://localhost:${SERVICE_PORT}/health",
    "expected_status": 200
  }
]
```

### `tcp` healthchecks[] item
Use when:
- readiness is best represented by a socket accepting connections

Sample:
```json
"healthchecks": [
  {
    "id": "tcp-ready",
    "type": "tcp"
  }
]
```

Current donor behavior suggests this relies on the configured service host/port.

### `file` healthchecks[] item
Use when:
- the service creates a file that represents successful readiness/setup

Sample:
```json
"healthchecks": [
  {
    "id": "ready-file",
    "type": "file",
    "file": "${SERVICE_HOME}/.state/runtime/ready.txt"
  }
]
```

### `variable` healthchecks[] item
Use when:
- a specific resolved/exported variable is the readiness signal

Sample:
```json
"healthchecks": [
  {
    "id": "service-url-ready",
    "type": "variable",
    "variable": "${SERVICE_URL}"
  }
]
```

## Other important manifest aspects

### Environment generation
Current broader Service Lasso direction includes:
- explicit service-local env via `env`
- possible cross-service/global env behavior via `globalenv`

The sample template keeps this minimal for now.

### Endpoints

Use top-level `endpoints[]` for every service interface or resource:

- network listeners use `kind: "network"` with a named `id`, bind, protocol, and port policy
- operator-facing links use `kind: "url"` and target their network endpoint
- runtime variables resolve endpoint data with `${endpoint.<id>.<field>}` selectors
- compatibility aliases, when required, stay in `env` or `globalenv`

Do not author new manifests with legacy top-level `ports`, `portmapping`, or `urls`, or with donor-style `serviceport*` fields. Historical donor-analysis documents under `docs/reference/` may still name those fields when describing the source system.

### Runtime-provider relationships
Donor material also shows patterns such as:
- `execservice`

This is relevant when a service is run via another runtime-provider service such as Node, Python, or Java.

The minimal sample does not use this yet.

## Canonical vs illustrative right now

### Treat as current first-pass canonical direction
- one service per repo
- `service.json` as the main service contract file
- lifecycle-focused `actions`
- `execconfig` as the execution contract section
- explicit `env`
- explicit `depend_on`
- canonical `endpoints[]` with `${endpoint.<id>.<field>}` selectors
- default health model of `process`
- explicit override to other health models when needed

### Still illustrative / not fully locked yet
- exact numeric meaning of `servicetype`
- exact numeric meaning of `servicelocation`
- final exact schema shape for all optional `execconfig` fields
- final exact health schema normalization
- final exact release artifact conventions across all service types

## Recommended authoring guidance

For the first template-based service:
1. keep the manifest small
2. use `process` health unless another model is clearly needed
3. explicitly declare env and dependencies
4. avoid donor baggage that mixes generated runtime state into package content
5. prefer clarity over trying to model every advanced donor feature on day one

## Related docs

Start here for the broader template contract:
- `docs/service-contract.md`
- `docs/validation.md`
- `docs/packaging.md`
- `docs/openspec-drafts/SPEC-SERVICE-TEMPLATE-REPO.md`

For deeper donor/runtime context:
- `docs/reference/shared-runtime/SERVICE-MANAGER-BEHAVIOR.md`
- `docs/reference/SERVICE-STRUCTURE-REVIEW.md`
- `docs/reference/shared-runtime/QUESTION-LIST-AND-CODE-VALIDATION.md`
- `docs/reference/shared-runtime/ARCHITECTURE-DECISIONS.md`
