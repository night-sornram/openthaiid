#!/usr/bin/env bash
# Package-script smoke: native archive + darwin cross-compile refusal.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
chmod +x "$ROOT/scripts/package.sh"

HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
VERSION="${VERSION:-0.1.0-rc.7}"

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

echo "== friday-checklist / technician / user cross-links =="
grep -q "qualify" docs/friday-checklist.md
grep -qi "audit" docs/friday-checklist.md
grep -q "service install" docs/friday-checklist.md
grep -q "ใช้เครื่องนี้" docs/friday-checklist.md
grep -q "CARD_REMOVED" docs/friday-checklist.md
grep -q "PCSC_UNAVAILABLE" docs/friday-checklist.md
grep -q "technician.md" docs/friday-checklist.md
grep -q "user.md" docs/friday-checklist.md
grep -q "qualify" docs/technician.md
grep -qi "audit" docs/technician.md
grep -q "service install" docs/technician.md
grep -q "ใช้เครื่องนี้" docs/technician.md
grep -q "CARD_REMOVED" docs/technician.md
grep -q "PCSC_UNAVAILABLE" docs/technician.md
grep -q "friday-checklist.md" docs/technician.md
grep -q "user.md" docs/technician.md
grep -q "ใช้เครื่องนี้" docs/user.md
grep -q "CARD_REMOVED" docs/user.md
grep -q "PCSC_UNAVAILABLE" docs/user.md
grep -q "friday-checklist.md" docs/user.md
grep -q "technician.md" docs/user.md
grep -q "getting-started.md" README.md
grep -q "user.md" docs/getting-started.md
grep -q "training.md" docs/getting-started.md
grep -q "friday-checklist.md" docs/getting-started.md
grep -q "erp-integration.md" docs/getting-started.md
grep -q "technician.md" docs/getting-started.md
grep -q "getting-started.md" docs/friday-checklist.md
grep -q "getting-started.md" docs/user.md
grep -q "getting-started.md" docs/training.md
grep -q "friday-dry-run" docs/getting-started.md
grep -q "ซ้อมก่อนได้เครื่อง" docs/getting-started.md
grep -q "friday-dry-run" docs/training.md
grep -q "ซ้อมก่อนได้เครื่อง" docs/training.md
grep -q "friday-dry-run" Makefile
test -f scripts/friday-dry-run.sh
test -f scripts/friday-dry-run.ps1
grep -q "getting-started.md" docs/erp-integration.md
grep -q "getting-started.md" docs/technician.md
grep -q "getting-started.md" packaging/START-HERE.linux.txt
grep -q "getting-started.md" packaging/START-HERE.windows.txt
grep -q "getting-started.md" packaging/START-HERE.darwin.txt
grep -q "security.md" docs/getting-started.md
grep -q "getting-started.md" docs/security.md
grep -q "Loopback HTTP" docs/security.md
grep -q "friday-dry-run" docs/security.md
grep -q "security.md" docs/technician.md
grep -q "getting-started.md" docs/hardware-qualification.md
grep -q "friday-dry-run" docs/hardware-qualification.md
grep -q "security.md" docs/friday-checklist.md
grep -q "friday-dry-run" packaging/START-HERE.linux.txt
grep -q "friday-dry-run" packaging/START-HERE.windows.txt
grep -q "friday-dry-run" packaging/START-HERE.darwin.txt
grep -q "loopback-tls.md" docs/getting-started.md
grep -q "compatibility.md" docs/getting-started.md
grep -q "getting-started.md" docs/loopback-tls.md
grep -q "getting-started.md" docs/compatibility.md
grep -q "loopback-tls.md" docs/compatibility.md
grep -q "loopback-tls.md" docs/hardware-qualification.md
grep -q "loopback-tls.md" docs/friday-checklist.md
grep -q "compatibility.md" docs/friday-checklist.md
grep -q "loopback-tls.md" docs/training.md
grep -q "compatibility.md" docs/training.md
grep -q "loopback-tls.md" docs/security.md
grep -q "compatibility.md" docs/security.md
grep -q "loopback-tls.md" docs/protocol.md
grep -q "loopback-tls.md" docs/architecture.md
grep -q "loopback-tls.md" packaging/START-HERE.linux.txt
grep -q "loopback-tls.md" packaging/START-HERE.windows.txt
grep -q "loopback-tls.md" packaging/START-HERE.darwin.txt
grep -q "compatibility.md" packaging/START-HERE.linux.txt
grep -q "compatibility.md" packaging/START-HERE.windows.txt
grep -q "compatibility.md" packaging/START-HERE.darwin.txt
grep -q "0.1.0-rc.7" cmd/openthaiid/main.go
grep -q "0.1.0-rc.7" Makefile

echo "== openthaiid doctor technician desk snapshot links =="
grep -q "openthaiid doctor" docs/technician.md
grep -q "GET /v1/helper/doctor" docs/technician.md
grep -q "Desk snapshot" docs/technician.md
grep -q "openthaiid doctor" docs/friday-checklist.md
grep -q "openthaiid doctor" docs/getting-started.md
grep -q "doctor -port" scripts/friday-dry-run.sh
grep -q "doctor -port" scripts/friday-dry-run.ps1
grep -q "openthaiid doctor" packaging/START-HERE.linux.txt
grep -q "openthaiid doctor" packaging/START-HERE.darwin.txt
grep -q "openthaiid.exe doctor" packaging/START-HERE.windows.txt

echo "== friday-desk-card staff-only (not the qualify kit) =="
test -f docs/friday-desk-card.md
test -f docs/friday-desk-card.html
grep -q "เสียบเครื่องอ่าน" docs/friday-desk-card.md
grep -q "ทดสอบการเชื่อมต่อ" docs/friday-desk-card.md
grep -q "กรอกข้อมูล" docs/friday-desk-card.md
grep -q "เรียกช่าง" docs/friday-desk-card.md
grep -q "ทดสอบการเชื่อมต่อ" docs/friday-desk-card.html
grep -q "เรียกช่าง" docs/friday-desk-card.html
grep -q "@media print" docs/friday-desk-card.html
grep -q "friday-desk-card.html" docs/friday-desk-card.md
grep -q "friday-desk-card.md" docs/friday-checklist.md
grep -q "friday-desk-card.md" docs/getting-started.md
grep -q "friday-desk-card.md" docs/training.md
grep -q "friday-desk-card.md" docs/user.md
grep -q "friday-desk-card.md" docs/hardware-qualification.md
grep -q "friday-desk-card.md" docs/clinic.md
grep -q "friday-desk-card.md" docs/installation.md
grep -q "friday-desk-card.md" packaging/START-HERE.linux.txt
grep -q "friday-desk-card.md" packaging/START-HERE.windows.txt
grep -q "friday-desk-card.md" packaging/START-HERE.darwin.txt
grep -q "friday-desk-card.md" README.md
grep -q "friday-desk-card.md" scripts/friday-dry-run.sh
grep -q "friday-desk-card.md" scripts/friday-dry-run.ps1
if grep -q "openthaiid qualify" docs/friday-desk-card.md docs/friday-desk-card.html; then
  echo "ERROR: staff desk card must not include qualify" >&2
  exit 1
fi
if grep -qiE "laser-id|NHSO" docs/friday-desk-card.md docs/friday-desk-card.html; then
  echo "ERROR: staff desk card must not mention Laser-id/NHSO" >&2
  exit 1
fi

echo "OK smoke-package"
