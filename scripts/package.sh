#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
OS_NAME=$(uname -s)
case "$OS_NAME" in
  Linux*) PLATFORM="linux" ;;
  Darwin*) PLATFORM="darwin" ;;
  *) echo "Unsupported OS for package.sh: $OS_NAME" >&2; exit 1 ;;
esac
STAGING="$DIST/echo-service-$PLATFORM"
TAR_PATH="$DIST/echo-service-$PLATFORM.tar.gz"

mkdir -p "$DIST"
rm -rf "$STAGING"
mkdir -p "$STAGING"

cp -R "$ROOT/runtime/$PLATFORM" "$STAGING/runtime"
cp -R "$ROOT/config" "$STAGING/config"

chmod +x "$STAGING/runtime/echo-service.sh" 2>/dev/null || true

rm -f "$TAR_PATH"
tar -czf "$TAR_PATH" -C "$STAGING" .
echo "Created $TAR_PATH"
