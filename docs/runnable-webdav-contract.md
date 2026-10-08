# Runnable Broker secret-file reference (#14)

The complete Echo example requests scoped Broker-owned vault-to-file provision
through Core (Core #1743, Broker #201), receives its WebDAV directory via the
managed child environment, and reads the file with the existing consumer.
Provide setup code that builds this Echo source into a fresh service folder,
preserves any existing folder, and writes a complete executable manifest.
Provide a check command that exercises safe GET/POST consumer status without
printing credentials/paths. Execute actual Core/Broker/Echo provisioning on
Ubuntu; retain positive/negative setup, access, read, replacement and stop proof.
No source helper bootstraps insecure credentials or bypasses first-run Broker
setup. Keep encrypted vault/runtime prerequisites explicit and link the lesson.
