# Run Echo with a Broker-provisioned RAM secret file

This is a complete runnable Service Lasso service, with setup code, the real Go
consumer and a read checker. Use a learning host with Core's Broker-owned file
provisioning, an initialized/unlocked Secrets Broker and Service Admin's RAM
files view. Node 22+ and the Go version in `go.mod` are needed to prepare it;
the resulting service launches the compiled binary and needs no Go installation.

1. Clone this repository's `develop` branch outside your host's services root.
   From this checkout, run:

   ```powershell
   $servicesRoot = Read-Host 'Existing services root for your learning host'
   node examples/webdav/prepare.mjs $servicesRoot
   ```

   Linux/macOS:

   ```bash
   read -r -p 'Existing services root for your learning host: ' servicesRoot
   node examples/webdav/prepare.mjs "$servicesRoot"
   ```

   The command compiles Echo and writes a complete `echo-webdav/service.json`.
   It refuses an existing `echo-webdav` directory. Windows gets an `.exe`;
   Linux/macOS get the native binary. The service keeps all its endpoints and
   launches relative to its service folder. A failed build leaves that new folder
   available for inspection; it does not overwrite an existing service.

2. In Service Admin's Secrets Broker view, create namespace `shared/echo`,
   ref `echo.DEMO_CREDENTIAL`, value `synthetic-demo-credential`.
   This is deliberately synthetic demo data. It is stored in Broker's encrypted
   vault, not in the checked-in manifest.

3. Refresh discovery; Install, Configure and Start `echo-webdav` through Core.
   Core sends the declared reference and `demo-config.json` template to Broker
   under a scoped launch lease. Broker resolves the reference, creates the file
   in RAM and returns its WebDAV directory. Core passes that directory through
   `ECHO_SECRET_FILES_DIR`. This file-only import is never returned to Core as
   a plaintext `/v1/resolve` response. Echo appends `demo-config.json` and reads it.

4. Copy Echo's allocated **Service HTTP** origin from its Network view, then run:

   ```powershell
   $echoOrigin = Read-Host 'Echo Service HTTP origin (http://127.0.0.1:<port>)'
   node examples/webdav/check.mjs $echoOrigin
   ```

   Linux/macOS: `node examples/webdav/check.mjs "$echoOrigin"`, using that same
   allocated origin. The checker proves startup loaded a valid file, rereads it
   once and prints only status, size and read count. It exits nonzero if unavailable,
   remote, redirected or not advancing. A successful initial run shows `loaded`,
   `sizeBytes: 46` and `reads: 2` for this exact demo value.

5. Open **Secrets Broker → RAM files**, filter `echo-webdav`, then refresh.
   `demo-config.json` should show the same size and two completed downloads on
   an otherwise idle grant. Repeat the checker to see both counts advance.
   No credential content or capability token appears in these views.

6. Stop Echo through Core: its grant disappears. Start again: Broker provisions
   a fresh file/path from the vault and counters restart. For a negative check,
   stop this learning service, delete only the synthetic ref, then try Start:
   provisioning must fail before spawn. Restore the ref to recover.

[`service.json`](service.json) is the full manifest; `prepare.mjs` changes only
its executable filename for the target OS. The Go implementation is
[`ram_secret_file.go`](../../ram_secret_file.go); positive and negative tests are
[`ram_secret_file_test.go`](../../ram_secret_file_test.go). Run Go tests with
`go test ./...`, and setup/check tests with
`node --test examples/webdav/example.test.mjs`.

`config.files[].ephemeral: true` requests file provisioning;
`broker.imports` supplies allowed bindings. `${SERVICE_LASSO_SECRETS_DIR}` is the
returned directory, not an on-disk working directory. Plain `_FILE` names are
app conventions; choose variables your consumer understands. Explicit secrets
in `env` remain supported separately. Broker text substitution does not JSON-escape
arbitrary secret values: this synthetic value is JSON-safe. For arbitrary raw
credentials, declare a single-value file and use an app that reads that format.

On Windows Echo converts the supplied UNC directory to loopback HTTP, so this
sample needs neither drive mapping nor Windows WebClient setup. No direct Echo
launch can provision the file: startup must go through Core and Broker.
