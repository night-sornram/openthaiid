#!/usr/bin/env bash
# Package-script smoke: native archive + darwin cross-compile refusal.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
chmod +x "$ROOT/scripts/package.sh"

HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
VERSION="${VERSION:-0.1.0-rc.1}"

echo "== native package =="
VERSION="$VERSION" ./scripts/package.sh
ls dist/openthaiid-"$VERSION"-* >/dev/null

if [ "$HOST_OS" != "darwin" ]; then
  echo "== darwin cross-compile must fail on $HOST_OS =="
  set +e
  out="$(OS=darwin ARCH=arm64 VERSION="$VERSION" ./scripts/package.sh 2>&1)"
  status=$?
  set -e
  echo "$out"
  test "$status" -ne 0
  echo "$out" | grep -q "cannot be cross-compiled"
  echo "$out" | grep -q "make package-macos"
  echo "$out" | grep -q "not produce a signed"
fi

echo "== make package-macos is Mac-host only =="
if [ "$HOST_OS" != "darwin" ]; then
  set +e
  out="$(make package-macos 2>&1)"
  status=$?
  set -e
  echo "$out"
  test "$status" -ne 0
  echo "$out" | grep -q "must run on a Mac"
fi

echo "== START-HERE Friday qualify command =="
grep -q "qualify" packaging/START-HERE.linux.txt
grep -q "qualify" packaging/START-HERE.windows.txt
grep -q "qualify" packaging/START-HERE.darwin.txt
test -f packaging/windows/Qualify-OpenThaiID.bat
test -f packaging/windows/Qualify-OpenThaiID.ps1

echo "OK smoke-package"
