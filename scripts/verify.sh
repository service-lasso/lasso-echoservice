#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CONTRACT="${1:-./verify/service-harness.json}"
OUTPUT_DIR="${2:-./output/verify}"
mkdir -p "$OUTPUT_DIR"
go run ./cmd/verify-harness --contract "$CONTRACT" --output-dir "$OUTPUT_DIR"
