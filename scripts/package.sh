#!/usr/bin/env bash
# Package a release archive. OS/arch are explicit inputs (never rely on
# Windows uname). Version is injected into the binary and verified after.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-0.1.0-rc.1}"
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

HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [ "$OS" = "darwin" ] && [ "$HOST_OS" != "darwin" ]; then
  echo "ERROR: macOS packages cannot be cross-compiled from $HOST_OS." >&2
  echo "ebfe/scard (PC/SC) and the Cocoa tray require cgo on a Mac host." >&2
  echo "On a Mac:  make package-macos" >&2
  echo "This does not produce a signed or notarized build." >&2
  exit 1
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
CLINIC_BIN="openthaiid"
if [ "$OS" = "windows" ]; then
  CLINIC_BIN="openthaiid.exe"
fi
cp "$DIST/$BIN_NAME" "$STAGE/$CLINIC_BIN"
cp LICENSE NOTICE THIRD_PARTY_NOTICES.md README.md CHANGELOG.md "$STAGE/"
cp -R docs "$STAGE/docs"
if [ -f "packaging/START-HERE.${OS}.txt" ]; then
  cp "packaging/START-HERE.${OS}.txt" "$STAGE/START-HERE.txt"
fi
case "$OS" in
  windows)
    cp packaging/windows/Start-OpenThaiID.bat "$STAGE/"
    cp packaging/windows/install.ps1 "$STAGE/"
    cp packaging/windows/uninstall.ps1 "$STAGE/"
    cp packaging/windows/Qualify-OpenThaiID.bat "$STAGE/"
    cp packaging/windows/Qualify-OpenThaiID.ps1 "$STAGE/"
    cp packaging/windows/Install-OpenThaiID-Service.ps1 "$STAGE/"
    cp packaging/windows/Uninstall-OpenThaiID-Service.ps1 "$STAGE/"
    cp packaging/windows/Register-OpenThaiID-Task.ps1 "$STAGE/"
    ;;
  linux)
    cp packaging/linux/start-openthaiid.sh "$STAGE/"
    cp packaging/linux/install-linux.sh "$STAGE/"
    cp packaging/linux/openthaiid.desktop "$STAGE/"
    cp packaging/linux/openthaiid.service "$STAGE/"
    cp packaging/linux/openthaiid-system.service "$STAGE/"
    chmod +x "$STAGE/start-openthaiid.sh" "$STAGE/install-linux.sh" "$STAGE/$CLINIC_BIN"
    ;;
  darwin)
    chmod +x "$STAGE/$CLINIC_BIN"
    if [ -f packaging/macos/start-openthaiid.sh ]; then
      cp packaging/macos/start-openthaiid.sh "$STAGE/start-openthaiid.sh"
      chmod +x "$STAGE/start-openthaiid.sh"
    fi
    APP="$STAGE/OpenThaiID.app"
    mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
    cp "$STAGE/$CLINIC_BIN" "$APP/Contents/MacOS/openthaiid"
    # Inject the same version as the binary. Unsigned; not notarized.
    sed "s/__OPENTHAIID_VERSION__/${VERSION}/g" packaging/macos/Info.plist \
      > "$APP/Contents/Info.plist"
    if grep -q "__OPENTHAIID_VERSION__" "$APP/Contents/Info.plist"; then
      echo "ERROR: Info.plist version placeholder not replaced" >&2
      exit 1
    fi
    chmod +x "$APP/Contents/MacOS/openthaiid"
    ;;
esac
# Dependency inventory (best-effort; empty on failure).
(go list -m all > "$STAGE/go-modules.txt" 2>/dev/null || true)
if [ -f sdk/web/package-lock.json ]; then
  cp sdk/web/package-lock.json "$STAGE/sdk-web-package-lock.json"
fi
if [ -f sdk/react/package-lock.json ]; then
  cp sdk/react/package-lock.json "$STAGE/sdk-react-package-lock.json"
fi

if command -v shasum >/dev/null 2>&1; then
  (cd "$STAGE" && shasum -a 256 "$CLINIC_BIN" > SHA256SUMS)
elif command -v sha256sum >/dev/null 2>&1; then
  (cd "$STAGE" && sha256sum "$CLINIC_BIN" > SHA256SUMS)
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
  got="$("$VERIFY/$CLINIC_BIN" version)"
  echo "verified version: $got"
  if [ "$got" != "$VERSION" ]; then
    echo "ERROR: version mismatch" >&2
    exit 1
  fi
  help="$("$VERIFY/$CLINIC_BIN" help)"
  echo "$help" | grep -q "openthaiid tray" || { echo "ERROR: help missing tray" >&2; exit 1; }
  echo "$help" | grep -q "openthaiid serve" || { echo "ERROR: help missing serve" >&2; exit 1; }
  echo "$help" | grep -q "openthaiid service" || { echo "ERROR: help missing service" >&2; exit 1; }
  echo "$help" | grep -q "openthaiid audit" || { echo "ERROR: help missing audit" >&2; exit 1; }
  test -f "$VERIFY/START-HERE.txt" || { echo "ERROR: missing START-HERE.txt" >&2; exit 1; }
  grep -q "qualify" "$VERIFY/START-HERE.txt" || { echo "ERROR: START-HERE missing qualify" >&2; exit 1; }
  if [ "$OS" = "darwin" ]; then
    test -x "$VERIFY/OpenThaiID.app/Contents/MacOS/openthaiid" || { echo "ERROR: missing .app executable" >&2; exit 1; }
    grep -q "$VERSION" "$VERIFY/OpenThaiID.app/Contents/Info.plist" || { echo "ERROR: Info.plist version" >&2; exit 1; }
    test -x "$VERIFY/start-openthaiid.sh" || { echo "ERROR: missing start-openthaiid.sh" >&2; exit 1; }
  fi
fi

echo "OK $ARCHIVE"
ls -la "$ARCHIVE"
