#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CONTRACT="${1:-./verify/service-harness.json}"
OUTPUT_DIR="${2:-./output/verify}"

resolve_harness() {
  if [[ -n "${SERVICE_LASSO_HARNESS_BIN:-}" ]]; then
    echo "$SERVICE_LASSO_HARNESS_BIN"
    return 0
  fi

  if command -v service-lasso-harness >/dev/null 2>&1; then
    command -v service-lasso-harness
    return 0
  fi

  echo "service-lasso-harness binary not found. Set SERVICE_LASSO_HARNESS_BIN or add it to PATH." >&2
  return 1
}

mkdir -p "$OUTPUT_DIR"
RESOLVED_CONTRACT="./verify/service-harness.ci.json"
RUN_OUTPUT_DIR="$OUTPUT_DIR/harness-run"

python3 - <<'PY' "$CONTRACT" "$RESOLVED_CONTRACT"
import json
import pathlib
import sys

contract_path = pathlib.Path(sys.argv[1]).resolve()
resolved_path = pathlib.Path(sys.argv[2]).resolve()
resolved_path.parent.mkdir(parents=True, exist_ok=True)

doc = json.loads(contract_path.read_text())
doc['artifact']['path'] = '../dist/echo-service-linux.tar.gz'
if sys.platform == 'darwin':
    doc['artifact']['path'] = '../dist/echo-service-darwin.tar.gz'
resolved_path.write_text(json.dumps(doc, indent=2) + '\n')
PY

HARNESS="$(resolve_harness)"
"$HARNESS" validate-contract --contract "$RESOLVED_CONTRACT"
"$HARNESS" run --contract "$RESOLVED_CONTRACT" --output-dir "$RUN_OUTPUT_DIR"
