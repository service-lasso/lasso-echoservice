# Service Lasso - Service Template Repo

_Status: working draft_

Important reconciliation note:
- this file is a migration/template planning doc
- use `QUESTION-LIST-AND-CODE-VALIDATION.md` and `ARCHITECTURE-DECISIONS.md` first when deciding what is already settled versus what remains template-design work

Purpose:

- define the role of the canonical Service Lasso service template repo
- clarify how individual services should be migrated out into their own repos
- document the rule that packaging some services with an app does not change service ownership
- serve as the place that must eventually explain the full service contract comprehensively, not just by shorthand references to other scratchpad docs

Related docs:

- `ARCHITECTURE.md`
- `ARCHITECTURE-DECISIONS.md`
- `SERVICE-STRUCTURE-REVIEW.md`

---

## 1. Core Rule

All services must live in their **own repos**.

Important current decisions:
- the service template repo should be **public**
- all Service Lasso ecosystem repos should be public
- it should be the **first repo created** in the new Service Lasso ecosystem because it unlocks migration/creation of all later service repos

This remains true even when:

- some services are bundled into app packages at package time
- some services are preinstalled in bundled/base distributions
- some services are downloaded dynamically on first start

Bundling does **not** change ownership.

---

## 2. Template Repo Role

There should be a canonical **service template repo** that defines the standard shape for a Service Lasso service repo.

Every migrated service repo should use that template as its starting point.

Current direction:
- we only need **one** service template repo first
- we do not need multiple template repos before migration starts
- this template should be the canonical base for all later service repos

This ensures consistent:

- repo layout
- service contract structure
- release packaging
- metadata conventions
- install/update behavior
- CI/release expectations

---

## 3. Migration Rule

For each donor-era service that survives into the new platform model:

1. create its own repo
2. initialize it from the canonical service template repo
3. migrate its service-specific code/config/assets into that repo
4. make it publishable/releasable independently
5. have Service Lasso install it from its released artifacts

---

## 4. Naming Convention

Service repos should use one shared prefix:

- `lasso-`

Rule:
- remove `lasso-` from the repo name
- the remainder is the exact local service folder name

Examples:
- `lasso-@archive` -> `@archive`
- `lasso-@node` -> `@node`
- `lasso-fastapi` -> `fastapi`

This convention should be part of the template/migration standard.

## 5. What The Template Must Standardize

The template repo should standardize at least:

- canonical `service.json` manifest shape
- expected local `service.state` usage
- expected local `service.pid` usage
- release artifact shape
- runtime/config layout
- explicit per-OS subfolders (for example `win32/`, `linux/`, `darwin/`)
- scripts for build/package/verify
- documentation structure
- versioning/release conventions
- a minimal sample service implementation that proves the lifecycle
- a simple reference release pipeline for producing compressed test artifacts
- packaging configuration for building release archives with 7zip

## 6. Template Sample Service

The template repo should include a deliberately minimal sample service.

Current direction:
- use a simple **echo service** as the sample
- it should not do any real application work
- it should exist only to prove the template/service lifecycle contract
- the template should include explicit OS subfolders such as `win32/`, `linux/`, and `darwin/`

Recommended sample behavior:
- install/setup runs cleanly
- during install/setup it writes a simple echo message into the log
- the sample script itself should just echo sample text
- it proves service registration, install flow, state handling, and log output without adding heavy runtime complexity

Why this is good:
- easiest possible template to understand
- proves the manifest/release/install flow
- avoids dragging real app/runtime complexity into the template repo

### Reference pipeline requirement
The template repo should also include a simple reference release pipeline.

Current direction:
- include a simple `*.ps1` script and `*.sh` script in the template repo
- the sample script behavior should just echo sample text
- these should use the per-OS subfolders as packaging inputs
- these should be used to create compressed release artifacts
- archiving/package creation should be done using 7zip
- the pipeline/template should include the necessary packaging config for 7zip-based release creation
- the resulting release artifacts should be usable for testing the Service Lasso install/release flow end-to-end

This gives the template immediate value as both:
- a service repo starting point
- a release/pipeline test fixture

## 7. Packaging Rule

---

## 7. Packaging Rule

Some apps/reference hosts may package a selected set of services at package time.

That is allowed.

But those packaged services must still:

- originate from their own independent service repos
- follow the service template repo contract
- remain independently releasable/versioned

So app packaging is a **distribution choice**, not a repo-ownership model.

---

## 8. Why This Matters

Without a template repo and one-service-per-repo discipline, the platform drifts back toward a giant mixed donor tree.

The template repo keeps migration clean and makes the service ecosystem scalable.

---

## 9. Documentation Expectation

The canonical service template repo doc will need a **comprehensive explanation of the full service contract**.

That explanation should eventually cover, in one place:

- what `service.json` is for
- what each top-level section means
- how defaults work versus overrides
- what `actions` means in practice
- how install/config/start relate
- what `service.state` and `service.pid` are for
- how releases, packaging, and per-OS folders fit together
- how a service creator should reason about defaults versus service-specific command mappings

Important rule:
- this should be explained directly and comprehensively in the template repo docs
- future service authors should not need to reconstruct the contract by piecing together scattered scratchpad notes

---

## 10. GitHub CLI Repo/Template Creation Commands

Useful `gh` CLI commands to remember when creating the Service Lasso ecosystem repos:

### Create a normal public repo
```bash
gh repo create service-lasso/<repo-name> --public
```

### Create and clone a normal public repo
```bash
gh repo create service-lasso/<repo-name> --public --clone
```

### Create a public repo from the current folder and push it
```bash
gh repo create service-lasso/<repo-name> --public --source . --push
```

### Add a description at creation time
```bash
gh repo create service-lasso/<repo-name> --public --description "<description>"
```

### Mark an existing repo as a template repo
```bash
gh repo edit service-lasso/<repo-name> --template
```

### Recommended flow for the canonical service template repo
```bash
gh repo create service-lasso/service-template --public --clone
gh repo edit service-lasso/service-template --template
```

### Verify the repo afterwards
```bash
gh repo view service-lasso/<repo-name>
```

Important clarification:
- GitHub template repos are created by first creating a normal repo, then marking it with `gh repo edit ... --template`
- use `--source . --push` when the local folder already exists and should become the remote repo immediately

---

## 11. Working Summary

Service Lasso should have:

- one canonical service template repo
- one service per repo
- optional app/package-time bundling of selected service releases
- dynamic install/update flows that still treat each service as independently owned and released
