#!/usr/bin/env bash
# Package-script smoke: native archive + darwin cross-compile refusal.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
chmod +x "$ROOT/scripts/package.sh"

HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
VERSION="${VERSION:-0.1.0-rc.2}"

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

echo "== START-HERE Friday qualify + audit JSON + service install =="
for f in packaging/START-HERE.linux.txt packaging/START-HERE.windows.txt packaging/START-HERE.darwin.txt; do
  grep -q "qualify" "$f"
  grep -qi "audit" "$f"
  grep -q "service install" "$f"
done
test -f packaging/windows/Qualify-OpenThaiID.bat
test -f packaging/windows/Qualify-OpenThaiID.ps1
test -f packaging/windows/Install-OpenThaiID-Service.ps1
test -f packaging/windows/Register-OpenThaiID-Task.ps1
grep -q "serve" packaging/linux/openthaiid.service
grep -q "serve" packaging/linux/openthaiid-system.service

echo "== friday-checklist / technician cross-links =="
grep -q "qualify" docs/friday-checklist.md
grep -qi "audit" docs/friday-checklist.md
grep -q "service install" docs/friday-checklist.md
grep -q "qualify" docs/technician.md
grep -qi "audit" docs/technician.md
grep -q "service install" docs/technician.md
grep -q "0.1.0-rc.2" cmd/openthaiid/main.go
grep -q "0.1.0-rc.2" Makefile

echo "OK smoke-package"
