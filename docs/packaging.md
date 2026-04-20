# Packaging

Reference packaging scripts:
- `scripts/package.ps1`
- `scripts/package.sh`

Current first-pass direction:
- package the minimal sample service into a release artifact under `dist/`
- include `service.json`, runtime payload, and config
- use the produced artifact as the thing later consumed by the shared harness
