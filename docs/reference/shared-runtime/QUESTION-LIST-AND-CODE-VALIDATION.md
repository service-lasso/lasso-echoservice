# Service Lasso - Canonical Question List and Code Validation

_Status: reconciled from chat transcript + donor code review_

Purpose:

- keep one canonical list of the important donor/runtime-boundary questions discussed in chat
- record the answers that were actually settled in transcript
- show what the donor code validates, what it only partially supports, and what remains implementation/spec work
- stop later speculative question batches from drifting away from the grounded discussion

Primary sources:

- Telegram chat export reviewed from `C:\Users\maxbarrass\Downloads\Telegram Desktop\ChatExport_2026-04-06\messages.html`
- donor runtime code under `runtime/Service.ts`, `runtime/ServiceManager.ts`, and `runtime/Services.ts`
- donor service manifests under `services/*/service.json`

Related docs:

- `ARCHITECTURE-DECISIONS.md`
- `SERVICE-MANAGER-BEHAVIOR.md`
- `QUESTIONABLE-CURRENT-RUNTIME-AREAS.md`
- `SERVICE-STRUCTURE-REVIEW.md`

---

## 1. Canonical question list recovered from transcript

These were the important runtime-boundary questions that anchored the discussion:

1. Should `@archive` and `@localcert` use the same lifecycle APIs as normal services, or should they have clearer utility/setup semantics?
2. Do you want `globalenv` to remain merged/ambient within the Service Lasso sandbox, or should it become more explicitly bound per service?
3. Should port negotiation be fully owned by Service Lasso core, with services only declaring needs?
4. How much of setup/install should stay inside service lifecycle vs be owned by a dedicated install manager?

These four should be treated as the canonical donor/runtime-boundary question set for this discussion.

---

## 2. Reconciled answers

### Question 1
**Should `@archive` and `@localcert` use the same lifecycle APIs as normal services, or should they have clearer utility/setup semantics?**

### Settled answer from transcript
Use:

- one registry
- one broad banded state machine
- one finite built-in action vocabulary

But utility services still have **clearer utility/setup semantics** than normal long-running services.

Meaning:

- `@archive` remains a registry-visible service because its binaries/versioning/update path matter independently
- `@localcert` remains a registry-visible utility/bootstrap service
- neither should be treated as a normal daemon-like service in the UI/API
- services differ by supported actions and meaningful states, not by requiring separate service engines

### What donor code validates
The donor code validates this direction strongly:

- `Service.ts` already distinguishes utility services with `servicetype === UTILITY`
- utility services end in `COMPLETED` / `COMPLETEDERROR` rather than daemon-style runtime states
- `Services.ts` and the existing state machine already assume one registry / one service system
- donor manifests for `_archive` and `_localcert` show they are capability/bootstrap providers, not normal user-facing app daemons

### Validation result
**Validated by transcript and broadly supported by code.**

The remaining work is implementation polish:

- define which built-in actions each service role supports
- make UI/API show only meaningful actions
- make service metadata classify role/category explicitly

---

### Question 2
**Do you want `globalenv` to remain merged/ambient within the Service Lasso sandbox, or should it become more explicitly bound per service?**

### Settled answer from transcript
Keep `globalenv` as the **shared Service Lasso-controlled runtime environment**.

Important rule settled in chat:

- services are sandboxed from OS environment variables
- service creators must explicitly specify what they need
- services are spawned only with what is available through Service Lasso-controlled env rules

Meaning:

- `globalenv` remains real and important
- it is not a fallback to host-machine env leakage
- explicit service env is still valuable, but it is derived from the controlled Service Lasso model, not ambient OS state

### What donor code validates
The donor code validates the mechanism directly:

- each service may emit `execconfig.globalenv`
- `Service.globalEnvironmentVariables` resolves and emits those values
- `ServiceManager.getGlobalEnv()` merges emitted values from all services
- the merged env is pushed back into services
- donor manifests for `archive`, `localcert`, `node`, `python`, `java`, and `traefik` all use exported globals as part of the service graph

### Validation result
**Strongly validated by transcript and code.**

Remaining implementation/spec work:

- define precedence rules when multiple services export the same key
- define export/import visibility rules clearly in schema docs
- distinguish shared/exported env from private service-only env sections more cleanly

These are implementation/specification details, not open product-direction questions.

---

### Question 3
**Should port negotiation be fully owned by Service Lasso core, with services only declaring needs?**

### Settled answer from transcript
Yes.

The intended model is:

- services declare port needs / port roles
- Service Lasso core resolves, reserves, negotiates, and publishes final ports

### What donor code validates
The donor code validates this very strongly:

- `ServiceManager.ts` owns port reservation bookkeeping
- services declare ports and mappings through manifest fields like:
  - `serviceport`
  - `serviceportsecondary`
  - `serviceportconsole`
  - `serviceportdebug`
  - `portmapping`
- service startup depends on the manager’s resolved/reserved port model

### Validation result
**Strongly validated by transcript and code.**

Remaining implementation/spec work:

- define exact stability guarantees across reload/install/update
- define exact collision policy and fallback behavior
- define how port blocks are expressed for app-level runtime footprints

Again, these are implementation-contract details, not unresolved product questions.

---

### Question 4
**How much of setup/install should stay inside service lifecycle vs be owned by a dedicated install manager?**

### Settled answer from transcript
The reconciled model is:

- service manifests declare install/setup/config-relevant behavior
- Service Lasso core orchestrates install/config work
- `install` is the main built-in action that converges a service into its expected installed state
- install may include archive extraction, setup commands, and rewriting Lasso-managed effective config/output
- install should not be read as a destructive revert of all content back to pristine defaults

Meaning:

- install is not just unzip/download
- install may include archive extraction and platform-specific setup commands
- install may also rewrite managed effective config/output as part of converging the service into the expected installed/configured state
- this is closer to reconcile/materialize desired state than to hard reset everything
- a separate first-class `config` action may still exist later for lighter-weight regeneration/update scenarios, but should not be assumed unless a concrete use case requires it
- the donor-style service contract remains important, but orchestration should be cleaner than the donor’s current entangled implementation

### What donor code validates
The donor code validates the mechanics but also shows the current entanglement:

- `Service.install()` delegates to setup flow in `Service.ts`
- `setuparchive` supports archive extraction
- `setup` supports platform-specific exec commands
- `commandconfig` supports config file materialization behavior
- the donor current code mixes install/setup/config/runtime concerns more than the future model should

### Validation result
**Validated directionally by transcript and partially validated by donor mechanics.**

The donor code proves that:

- install includes archive extraction
- install includes setup exec commands
- config materialization exists as a real runtime need
- the current donor `setup` flow is doing more than a narrow installer; it already performs behavior that is partly installation/bootstrap and partly configuration/preparation

Important interpretation:

- donor `setup` should not be read as evidence that `setup` is the correct future first-class action name
- instead, it is evidence that the donor currently entangles install-like and config-like work inside one setup flow
- this strengthens the decision to separate future `install` and `config` as distinct built-in actions

But the donor code does **not** yet provide the clean boundary we want.

Remaining implementation/spec work:

- separate install orchestration from runtime execution more cleanly
- define exactly where generated effective config is written
- define install/config state recording in `service.state`
- decide whether separate first-class `config` remains necessary, or whether install fully owns normal config materialization with only lighter-weight regeneration scenarios separated later

These are design-implementation tasks, not open direction questions.

---

## 3. Additional decisions validated from transcript + code

The transcript settled several related points that should be treated as current truth.

### Service contract / repo shape
- one `service.json` should remain the canonical manifest file
- one service = one repo
- services support explicit versions and exact release selection
- `services/` is a dynamic installed inventory, not a fixed source-controlled monolith

### Environment / execution
- services are sandboxed from OS env
- `globalenv` is the shared controlled runtime environment
- service-to-service communication is typically direct to ports
- routed URLs are mainly for browser/frontend/operator interaction

### UI / control plane
- core should own orchestration + API + CLI + contracts
- admin UI should be an optional separate service, not privileged core
- per-service UI remains valuable, but as API-driven admin UI rather than inline bootstrap HTML

### Finite action model
Current settled built-in action set:

- `install`
- `uninstall`
- `update`
- `start`
- `stop`
- `restart`
- `config`

Current clarified interpretation of `actions`:

- `actions` is for **service-specific custom commands that override Service Lasso default behavior for a named service action**
- the service creator explicitly maps their service-specific command when the default is not right for that service
- if no override is present, Service Lasso falls back to its default behavior for that action
- it is not an arbitrary free-form custom action registry
- action names are still owned by Service Lasso
- do not expand the overrideable action list spec pre-emptively; only extend it when a concrete service demonstrates a real need

And explicitly **not** in the current core action set:

- `reinstall`
- `regenconfig`
- `validate`
- `inspect`
- arbitrary custom action names

Diagnostic/status views should instead come from:

- UI/API service detail
- `service.json`
- `service.state`
- `service.pid`
- logs
- status/health fields

---

## 4. What is still a real gap after reconciliation

After transcript + code reconciliation, the main remaining gaps are **not** the original four product-boundary questions.

Those four are effectively answered.

The remaining gaps are implementation/specification work such as:

1. define the exact cleaned section names/boundaries for the normalized `service.json` shape now that the direction away from one giant donor-style `execconfig` block is settled
2. define precise `service.state` content and lifecycle transitions inside the newer `.state/`-folder model, including what is written on start/stop versus what remains log-only
3. define exact install-state / config-state recording rules
4. define exact generated-config output paths and precedence rules
5. define exact port-allocation guarantees and persistence behavior
6. define exact supported-action declaration shape in manifest/schema
7. define UI/API payloads that reflect supported actions, role/category, and current state cleanly

These should be treated as specification and implementation tasks, not as unresolved top-level direction questions.

---

## 5. Working summary

The important outcome of this reconciliation is:

- the four runtime-boundary questions are now consolidated in one place
- transcript answers take precedence over later speculative drift
- donor code is used to validate mechanism and identify where the donor implementation is still too entangled
- what remains is mostly schema/runtime/API specification work, not fresh architecture ideation

If later docs conflict with this file, this file should win unless the discussion is explicitly reopened.
