# Echo RAM file consumer (#12)

The opt-in [sample manifest](../examples/webdav/service.json) declares an
ephemeral `demo-config.json` resolving `echo.DEMO_CREDENTIAL` from the Broker
namespace `shared/echo`. For a demonstration, create the Broker secret
`shared/echo/echo.DEMO_CREDENTIAL` with value `synthetic-demo-credential` before
starting. Its required scoped import fails launch if the secret is unavailable.
Core renders it into Broker RAM and supplies `${SERVICE_LASSO_SECRETS_DIR}` as
`ECHO_SECRET_FILES_DIR`. Existing environment delivery continues to work.

For source development, place that manifest beside `main.go` as `service.json`
and discover/start that folder with Service Lasso, with Broker running. For a
packaged service, retain its packaged executable/args and add the sample's
`config.files` and `ECHO_SECRET_FILES_DIR` / `ECHO_SECRET_FILE_NAME` environment
entries. Use the Echo build containing this consumer; older releases lack it.
Replace the synthetic credential with an appropriate scoped SecretRef template
for a real app; never place actual credentials in a checked-in manifest.

Echo reads and validates the JSON on startup. `GET /secret-file` reports only
`status`, `sizeBytes`, `reads` and `lastReadAt`. `POST /secret-file` rereads the
file; its successful reads appear in Broker's WebDAV download counters. The
sample reports `disabled` when unconfigured and `unavailable` on read/validation
failure; this optional demonstration does not change other harness health modes.
Stopping/restarting through Core revokes/replaces the old grant.

No value or capability URL enters the public status, logs, state or env dump.
HTTP transport is strictly `127.0.0.1` with proxies and redirects disabled and
a 256 KiB/three-second bound. Core's Windows UNC directory converts to the
same loopback HTTP request, so this Go sample needs no mapped drive/WebClient.
Other applications may use the native UNC path directly.

Verification: full Go tests and vet passed on Windows. Positive/negative consumer
tests passed on native Ubuntu; real Core + production Broker + Echo consumed this
sample twice, inventory matched counts/bytes, environment and durable files
withheld private material, and stop removed the grant. Source builds do not
imply package publication or deployment.
