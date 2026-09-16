#!/usr/bin/env bash
# Package a release archive. OS/arch are explicit inputs (never rely on
# Windows uname). Version is injected into the binary and verified after.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-0.0.1-dev}"
OS="${OS:-}"
ARCH="${ARCH:-}"
if [ -z "$OS" ] || [ -z "$ARCH" ]; then
  # Local convenience fallback (POSIX hosts only).
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64) ARCH=amd64 ;;
    aarch64) ARCH=arm64 ;;
    arm64) ARCH=arm64 ;;
  esac
fi

CGO_ENABLED="${CGO_ENABLED:-1}"
if [ "$OS" = "windows" ]; then
  CGO_ENABLED=0
fi

BIN_NAME="openthaiid-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BIN_NAME="${BIN_NAME}.exe"
fi

DIST="$ROOT/dist"
STAGE="$DIST/stage-$VERSION-$OS-$ARCH"
ARCHIVE_BASE="openthaiid-${VERSION}-${OS}-${ARCH}"
ARCHIVE="$DIST/${ARCHIVE_BASE}.tar.gz"
if [ "$OS" = "windows" ]; then
  ARCHIVE="$DIST/${ARCHIVE_BASE}.zip"
fi

echo "Building $BIN_NAME (version=$VERSION) ..."
mkdir -p "$DIST"
GOOS="$OS" GOARCH="$ARCH" CGO_ENABLED="$CGO_ENABLED" \
  go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o "$DIST/$BIN_NAME" ./cmd/openthaiid

rm -rf "$STAGE"
mkdir -p "$STAGE"
cp "$DIST/$BIN_NAME" "$STAGE/"
cp LICENSE NOTICE THIRD_PARTY_NOTICES.md README.md CHANGELOG.md "$STAGE/"
cp -R docs "$STAGE/docs"
# Dependency inventory (best-effort; empty on failure).
(go list -m all > "$STAGE/go-modules.txt" 2>/dev/null || true)
if [ -f sdk/web/package-lock.json ]; then
  cp sdk/web/package-lock.json "$STAGE/sdk-web-package-lock.json"
fi
if [ -f sdk/react/package-lock.json ]; then
  cp sdk/react/package-lock.json "$STAGE/sdk-react-package-lock.json"
fi

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

# Verify the extracted binary reports the injected version when it is the
# host platform binary.
HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
  x86_64) HOST_ARCH=amd64 ;;
  aarch64|arm64) HOST_ARCH=arm64 ;;
esac
if [ "$OS" = "$HOST_OS" ] && [ "$ARCH" = "$HOST_ARCH" ] && [ "$OS" != "windows" ]; then
  VERIFY="$DIST/verify-$VERSION"
  rm -rf "$VERIFY"
  mkdir -p "$VERIFY"
  tar -xzf "$ARCHIVE" -C "$VERIFY"
  got="$("$VERIFY/$BIN_NAME" version)"
  echo "verified version: $got"
  if [ "$got" != "$VERSION" ]; then
    echo "ERROR: version mismatch" >&2
    exit 1
  fi
fi

echo "OK $ARCHIVE"
ls -la "$ARCHIVE"
