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
BINARY_PATH="$STAGING/echo-service"

mkdir -p "$DIST"
rm -rf "$STAGING"
mkdir -p "$STAGING"

(
  cd "$ROOT"
  if [ "$PLATFORM" = "darwin" ]; then
    # The published darwin asset is consumed by both Apple silicon and Intel
    # hosts, so build a universal binary instead of the runner architecture.
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o "$STAGING/echo-service-amd64" .
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o "$STAGING/echo-service-arm64" .
    lipo -create -output "$BINARY_PATH" "$STAGING/echo-service-amd64" "$STAGING/echo-service-arm64"
    rm -f "$STAGING/echo-service-amd64" "$STAGING/echo-service-arm64"
  else
    go build -o "$BINARY_PATH" .
  fi
)

cp "$ROOT/service.json" "$STAGING/service.json"
cp "$ROOT/README.md" "$STAGING/README.md"
cp -R "$ROOT/config" "$STAGING/config"

chmod +x "$BINARY_PATH"

rm -f "$TAR_PATH"
tar -czf "$TAR_PATH" -C "$STAGING" .
echo "Created $TAR_PATH"
