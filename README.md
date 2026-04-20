# lasso-echoservice

`lasso-echoservice` is the canonical repository for the Echo Service harness used in the Service Lasso ecosystem.

It is a real service repo, not a core-runtime fixture repo.

Its job is to provide:
- a runnable Go service
- a small browser UI for manual checks
- an HTTP API for automation, CLI, and integration tests
- predictable behavior-simulation actions for runtime supervision work
- durable log, state, and SQLite writes for persistence testing

## Main endpoints

- `GET /`
- `GET /health`
- `GET /state`
- `GET /logs`
- `GET /sqlite`

## Main actions

- `POST /action/write-log`
- `POST /action/write-state`
- `POST /action/write-sqlite`
- `POST /action/error`
- `POST /action/close`
- `POST /action/abort`
- `POST /action/start-child`
- `POST /action/fork-child`

## Local development

Build:

```powershell
go build .
```

Run:

```powershell
.\echo-service.exe
```

Or:

```powershell
go run .
```

## Default runtime files

The service writes under `./runtime/` by default:
- `runtime/echo.log`
- `runtime/state.json`
- `runtime/echo.sqlite`

These can be overridden with:
- `ECHO_PORT`
- `ECHO_LOG_PATH`
- `ECHO_STATE_PATH`
- `ECHO_DB_PATH`

## Relationship to `service-lasso`

The `service-lasso` repo remains the core runtime and contract repo.

This repo is the canonical home of the actual Echo Service implementation.

`service-lasso` may still keep a thin local fixture manifest for `echo-service` so the core runtime test suite remains self-contained, but the service implementation itself now lives here.
