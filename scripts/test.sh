#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for path in \
  "$ROOT/service.json" \
  "$ROOT/verify/service-harness.json"; do
  if [[ ! -f "$path" ]]; then
    echo "Missing required file: $path" >&2
    exit 1
  fi
done

SERVICE_ID=$(python3 - <<'PY'
import json, pathlib
path = pathlib.Path('service.json')
print(json.loads(path.read_text())['id'])
PY
)
if [[ "$SERVICE_ID" != "echo-service" ]]; then
  echo "service.json id mismatch" >&2
  exit 1
fi

CONTRACT_ID=$(python3 - <<'PY'
import json, pathlib
path = pathlib.Path('verify/service-harness.json')
print(json.loads(path.read_text())['serviceId'])
PY
)
if [[ "$CONTRACT_ID" != "echo-service" ]]; then
  echo "service-harness.json serviceId mismatch" >&2
  exit 1
fi

OS_NAME=$(uname -s)
case "$OS_NAME" in
  Linux*) RUNTIME="$ROOT/runtime/linux/echo-service.sh" ;;
  Darwin*) RUNTIME="$ROOT/runtime/darwin/echo-service.sh" ;;
  *) echo "Unsupported OS for test.sh: $OS_NAME" >&2; exit 1 ;;
esac

chmod +x "$RUNTIME"
OUTPUT=$(ECHO_MESSAGE='pipeline test message' "$RUNTIME")
if [[ "$OUTPUT" != *"pipeline test message"* ]]; then
  echo "Echo runtime output mismatch" >&2
  exit 1
fi

echo "Template tests passed ($OS_NAME)"
