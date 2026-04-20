# lasso-echoservice

`lasso-echoservice` is the canonical repository for the Echo Service harness used in the Service Lasso ecosystem.

It is a real service repo, not a core-runtime fixture repo.

Its job is to provide:
- a runnable Go service
- a small browser UI for manual checks
- an HTTP API for automation, CLI, and integration tests
- predictable behavior-simulation actions for runtime supervision work
- dedicated HTTP and TCP health targets that can be toggled between healthy, stopped, and error modes
- structured env and global-env output that Service Lasso can consume directly
- stdout and stderr emission actions for log-capture testing
- durable log, state, and SQLite writes for persistence testing

## Main endpoints

- `GET /`
- `GET /health`
- `GET /health/http`
- `GET /health/tcp`
- `GET /state`
- `GET /logs`
- `GET /sqlite`
- `GET /env`
- `GET /global-env`
- `GET /service-lasso/output`

## Main actions

- `POST /action/write-log`
- `POST /action/write-state`
- `POST /action/write-sqlite`
- `POST /action/write-stdout`
- `POST /action/write-stderr`
- `POST /action/http-health`
- `POST /action/tcp-health`
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
- `ECHO_HTTP_HEALTH_PORT`
- `ECHO_TCP_PORT`
- `ECHO_LOG_PATH`
- `ECHO_STATE_PATH`
- `ECHO_DB_PATH`
- `SERVICE_LASSO_GLOBAL_ENV_JSON`

## Health simulation

The harness now exposes three health surfaces:
- process health via the main service at `GET /health`
- dedicated HTTP health via `http://127.0.0.1:${ECHO_HTTP_HEALTH_PORT}/health`
- dedicated TCP health via `127.0.0.1:${ECHO_TCP_PORT}`

The browser UI and API can switch the dedicated HTTP/TCP health targets between:
- `healthy`
- `error`
- `stopped`

That makes the harness useful for future Service Lasso work on real HTTP/TCP health probing without changing the service’s primary manifest healthcheck yet.

## Verification

Local verification commands:

```powershell
go test ./...
pwsh -NoLogo -NoProfile -File .\scripts\verify.ps1
pwsh -NoLogo -NoProfile -File .\scripts\package.ps1
```

The verifier now proves:
- API and UI-backed persistence actions
- stdout and stderr emission
- dedicated HTTP health transitions
- dedicated TCP health transitions
- env/global-env and Service Lasso output surfaces
- child-process tracking
- clean close behavior

## Relationship to `service-lasso`

The `service-lasso` repo remains the core runtime and contract repo.

This repo is the canonical home of the actual Echo Service implementation.

`service-lasso` may still keep a thin local fixture manifest for `echo-service` so the core runtime test suite remains self-contained, but the service implementation itself now lives here.
