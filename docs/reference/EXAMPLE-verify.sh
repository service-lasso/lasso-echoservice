#!/usr/bin/env bash
set -euo pipefail

CONTRACT="${1:-./verify/service-harness.json}"

if ! command -v service-lasso-harness >/dev/null 2>&1; then
  echo "service-lasso-harness not found in PATH" >&2
  exit 1
fi

service-lasso-harness run --contract "$CONTRACT"
