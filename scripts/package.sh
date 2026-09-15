#!/usr/bin/env bash
# Package a release archive for the current platform.
# Cross-compile targets are produced by CI; this script packages a local build.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-0.0.1-dev}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64) ARCH=arm64 ;;
  arm64) ARCH=arm64 ;;
esac

BIN_NAME="openthaiid-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BIN_NAME="${BIN_NAME}.exe"
fi

DIST="$ROOT/dist"
STAGE="$DIST/stage-$VERSION"
ARCHIVE_BASE="openthaiid-${VERSION}-${OS}-${ARCH}"
ARCHIVE="$DIST/${ARCHIVE_BASE}.tar.gz"
if [ "$OS" = "windows" ]; then
  ARCHIVE="$DIST/${ARCHIVE_BASE}.zip"
fi

echo "Building $BIN_NAME ..."
mkdir -p "$DIST"
CGO_ENABLED="${CGO_ENABLED:-1}"
if [ "$OS" = "windows" ]; then
  CGO_ENABLED=0
fi
CGO_ENABLED="$CGO_ENABLED" go build -trimpath -ldflags "-s -w" -o "$DIST/$BIN_NAME" ./cmd/openthaiid

rm -rf "$STAGE"
mkdir -p "$STAGE"
cp "$DIST/$BIN_NAME" "$STAGE/"
cp LICENSE NOTICE THIRD_PARTY_NOTICES.md README.md "$STAGE/"
cp -R docs "$STAGE/docs"
# SHA256 of the binary
if command -v shasum >/dev/null 2>&1; then
  (cd "$STAGE" && shasum -a 256 "$BIN_NAME" > SHA256SUMS)
elif command -v sha256sum >/dev/null 2>&1; then
  (cd "$STAGE" && sha256sum "$BIN_NAME" > SHA256SUMS)
fi

echo "Creating $ARCHIVE ..."
if [ "$OS" = "windows" ]; then
  (cd "$STAGE" && zip -qr "$ARCHIVE" .)
else
  tar -czf "$ARCHIVE" -C "$STAGE" .
fi

echo "OK $ARCHIVE"
ls -la "$ARCHIVE"
