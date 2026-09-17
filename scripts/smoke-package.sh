#!/usr/bin/env bash
# Package-script smoke: native archive + darwin cross-compile refusal.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
chmod +x "$ROOT/scripts/package.sh"

HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
VERSION="${VERSION:-0.1.0-rc.9}"

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

echo "== START-HERE Hardware qualify + audit JSON + service install =="
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

echo "== hardware-checklist / technician / user cross-links =="
grep -q "qualify" docs/hardware-checklist.md
grep -qi "audit" docs/hardware-checklist.md
grep -q "service install" docs/hardware-checklist.md
grep -q "ใช้เครื่องนี้" docs/hardware-checklist.md
grep -q "CARD_REMOVED" docs/hardware-checklist.md
grep -q "PCSC_UNAVAILABLE" docs/hardware-checklist.md
grep -q "technician.md" docs/hardware-checklist.md
grep -q "user.md" docs/hardware-checklist.md
grep -q "qualify" docs/technician.md
grep -qi "audit" docs/technician.md
grep -q "service install" docs/technician.md
grep -q "ใช้เครื่องนี้" docs/technician.md
grep -q "CARD_REMOVED" docs/technician.md
grep -q "PCSC_UNAVAILABLE" docs/technician.md
grep -q "hardware-checklist.md" docs/technician.md
grep -q "user.md" docs/technician.md
grep -q "ใช้เครื่องนี้" docs/user.md
grep -q "CARD_REMOVED" docs/user.md
grep -q "PCSC_UNAVAILABLE" docs/user.md
grep -q "hardware-checklist.md" docs/user.md
grep -q "technician.md" docs/user.md
grep -q "getting-started.md" README.md
grep -q "user.md" docs/getting-started.md
grep -q "training.md" docs/getting-started.md
grep -q "hardware-checklist.md" docs/getting-started.md
grep -q "erp-integration.md" docs/getting-started.md
grep -q "technician.md" docs/getting-started.md
grep -q "getting-started.md" docs/hardware-checklist.md
grep -q "getting-started.md" docs/user.md
grep -q "getting-started.md" docs/training.md
grep -q "pre-reader-dry-run" docs/getting-started.md
grep -q "ซ้อมก่อนได้เครื่อง" docs/getting-started.md
grep -q "pre-reader-dry-run" docs/training.md
grep -q "ซ้อมก่อนได้เครื่อง" docs/training.md
grep -q "pre-reader-dry-run" Makefile
test -f scripts/pre-reader-dry-run.sh
test -f scripts/pre-reader-dry-run.ps1
grep -q "getting-started.md" docs/erp-integration.md
grep -q "getting-started.md" docs/technician.md
grep -q "getting-started.md" packaging/START-HERE.linux.txt
grep -q "getting-started.md" packaging/START-HERE.windows.txt
grep -q "getting-started.md" packaging/START-HERE.darwin.txt
grep -q "security.md" docs/getting-started.md
grep -q "getting-started.md" docs/security.md
grep -q "Loopback HTTP" docs/security.md
grep -q "pre-reader-dry-run" docs/security.md
grep -q "security.md" docs/technician.md
grep -q "getting-started.md" docs/hardware-qualification.md
grep -q "pre-reader-dry-run" docs/hardware-qualification.md
grep -q "bind-stress.md" docs/hardware-qualification.md
grep -q "bind-stress" NOT_DONE.md
grep -q "bind-stress.md" docs/technician.md
grep -q "bind-stress.md" docs/hardware-checklist.md
grep -q "bind-stress.md" docs/acceptance.md
grep -q "bind-stress.md" docs/dependency-audit.md
grep -q "bind-stress-smoke" Makefile
test -f scripts/bind-stress.sh
test -f scripts/bind-stress.ps1
test -f scripts/smoke-bind-stress.sh
test -f docs/bind-stress.md
grep -q "openthaiid.bindstress.v1" docs/bind-stress.md
grep -q "bind-stress.md" docs/getting-started.md
grep -q "bind-stress.md" README.md
grep -q "security.md" docs/hardware-checklist.md
grep -q "pre-reader-dry-run" packaging/START-HERE.linux.txt
grep -q "pre-reader-dry-run" packaging/START-HERE.windows.txt
grep -q "pre-reader-dry-run" packaging/START-HERE.darwin.txt
grep -q "loopback-tls.md" docs/getting-started.md
grep -q "compatibility.md" docs/getting-started.md
grep -q "getting-started.md" docs/loopback-tls.md
grep -q "getting-started.md" docs/compatibility.md
grep -q "loopback-tls.md" docs/compatibility.md
grep -q "loopback-tls.md" docs/hardware-qualification.md
grep -q "loopback-tls.md" docs/hardware-checklist.md
grep -q "compatibility.md" docs/hardware-checklist.md
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
grep -q "0.1.0-rc.9" cmd/openthaiid/main.go
grep -q "0.1.0-rc.9" Makefile

echo "== SDK package versions + https ERP clinic sample =="
grep -q '"version": "0.1.0-rc.9"' sdk/web/package.json
grep -q '"version": "0.1.0-rc.9"' sdk/react/package.json
grep -q '"@openthaiid/web": "0.1.0-rc.9"' sdk/react/package.json
test -f examples/clinic-patient/clinic-https.json
grep -q '"preferLoopbackTls": true' examples/clinic-patient/clinic-https.json
grep -q "wss://127.0.0.1:17892/v1/ws" examples/clinic-patient/clinic-https.json
grep -q "loopback-tls.md" examples/clinic-patient/clinic-https.json
grep -q "compatibility.md" examples/clinic-patient/clinic-https.json
grep -q "clinic-https.json" docs/erp-integration.md
grep -q "clinic-https.json" docs/loopback-tls.md
grep -q "clinic-https.json" docs/compatibility.md
grep -q '## \[0.1.0-rc.9\]' CHANGELOG.md
grep -q "clinic-https.json" CHANGELOG.md
grep -q "@openthaiid/web" CHANGELOG.md
grep -q "preferLoopbackTls" CHANGELOG.md
grep -q '## \[Unreleased\]' CHANGELOG.md
grep -q "#42" CHANGELOG.md
grep -q "pre-hardware software is complete" CHANGELOG.md
if awk '/^## \[0.1.0-rc.9\]/{p=1} /^## \[0.1.0-rc.8\]/{p=0} p && /unreleased tag/{found=1} END{exit found?0:1}' CHANGELOG.md; then
  echo "ERROR: [0.1.0-rc.9] must not still say unreleased tag (already tagged)" >&2
  exit 1
fi
if ! awk '/^## \[Unreleased\]/{u=1} /^## \[0.1.0-rc.9\]/{u=0} u && /#42/{found=1} END{exit found?0:1}' CHANGELOG.md; then
  echo "ERROR: #42 flake note must live under [Unreleased], not folded into tagged rc.9" >&2
  exit 1
fi
if grep -qE "1234567890121|สมชาย|BEGIN PRIVATE KEY" examples/clinic-patient/clinic-https.json; then
  echo "ERROR: clinic-https.json must not contain PII or private keys" >&2
  exit 1
fi

echo "== openthaiid doctor technician desk snapshot links =="
grep -q "openthaiid doctor" docs/technician.md
grep -q "GET /v1/helper/doctor" docs/technician.md
grep -q "Desk snapshot" docs/technician.md
grep -q "openthaiid doctor" docs/hardware-checklist.md
grep -q "openthaiid doctor" docs/getting-started.md
grep -q "doctor -port" scripts/pre-reader-dry-run.sh
grep -q "doctor -port" scripts/pre-reader-dry-run.ps1
grep -q "openthaiid doctor" packaging/START-HERE.linux.txt
grep -q "openthaiid doctor" packaging/START-HERE.darwin.txt
grep -q "openthaiid.exe doctor" packaging/START-HERE.windows.txt

echo "== staff-desk-card staff-only (not the qualify kit) =="
test -f docs/staff-desk-card.md
test -f docs/staff-desk-card.html
grep -q "เสียบเครื่องอ่าน" docs/staff-desk-card.md
grep -q "ทดสอบการเชื่อมต่อ" docs/staff-desk-card.md
grep -q "กรอกข้อมูล" docs/staff-desk-card.md
grep -q "เรียกช่าง" docs/staff-desk-card.md
grep -q "ทดสอบการเชื่อมต่อ" docs/staff-desk-card.html
grep -q "เรียกช่าง" docs/staff-desk-card.html
grep -q "@media print" docs/staff-desk-card.html
grep -q "staff-desk-card.html" docs/staff-desk-card.md
grep -q "staff-desk-card.md" docs/hardware-checklist.md
grep -q "staff-desk-card.md" docs/getting-started.md
grep -q "staff-desk-card.md" docs/training.md
grep -q "staff-desk-card.md" docs/user.md
grep -q "staff-desk-card.md" docs/hardware-qualification.md
grep -q "staff-desk-card.md" docs/clinic.md
grep -q "staff-desk-card.md" docs/installation.md
grep -q "staff-desk-card.md" packaging/START-HERE.linux.txt
grep -q "staff-desk-card.md" packaging/START-HERE.windows.txt
grep -q "staff-desk-card.md" packaging/START-HERE.darwin.txt
grep -q "staff-desk-card.md" README.md
grep -q "staff-desk-card.md" scripts/pre-reader-dry-run.sh
grep -q "staff-desk-card.md" scripts/pre-reader-dry-run.ps1
if grep -q "openthaiid qualify" docs/staff-desk-card.md docs/staff-desk-card.html; then
  echo "ERROR: staff desk card must not include qualify" >&2
  exit 1
fi
if grep -qiE "laser-id|NHSO" docs/staff-desk-card.md docs/staff-desk-card.html; then
  echo "ERROR: staff desk card must not mention Laser-id/NHSO" >&2
  exit 1
fi

echo "== NOT_DONE hardware playbook (not a stale software backlog) =="
grep -q "Done in software (rc.9" NOT_DONE.md
grep -q "When the ACR39U arrives" NOT_DONE.md
grep -q "Blocked without cited APDU" NOT_DONE.md
grep -q "Blocked without user certs" NOT_DONE.md
grep -q "Paused until Actions minutes" NOT_DONE.md
grep -q "Darwin attach only" NOT_DONE.md
grep -q "software is complete as of post-#42" NOT_DONE.md
grep -q "v0.1.0-rc.9" NOT_DONE.md
grep -q "E1" NOT_DONE.md
grep -q "M1" NOT_DONE.md
grep -q "1 Oct 2026" NOT_DONE.md
grep -q "#42" NOT_DONE.md
if grep -q "git tag -a v0.1.0-rc.9" NOT_DONE.md; then
  echo "ERROR: NOT_DONE must not retag v0.1.0-rc.9" >&2
  exit 1
fi
if grep -qE "tag-ready.*v0\\.1\\.0-rc\\.9|v0\\.1\\.0-rc\\.9.*tag-ready" NOT_DONE.md; then
  echo "ERROR: NOT_DONE must not still call rc.9 tag-ready (already tagged)" >&2
  exit 1
fi
if grep -q "git tag -a v0.1.0-rc.8" NOT_DONE.md; then
  echo "ERROR: NOT_DONE must not retag v0.1.0-rc.8" >&2
  exit 1
fi
if grep -qE "tag-ready.*v0\\.1\\.0-rc\\.8|v0\\.1\\.0-rc\\.8.*tag-ready" NOT_DONE.md; then
  echo "ERROR: NOT_DONE must not still call rc.8 tag-ready (already tagged)" >&2
  exit 1
fi
if grep -q "git tag -a v0.1.0-rc.7" NOT_DONE.md; then
  echo "ERROR: NOT_DONE must not retag v0.1.0-rc.7" >&2
  exit 1
fi
if grep -qE "tag-ready.*v0\\.1\\.0-rc\\.7|v0\\.1\\.0-rc\\.7.*tag-ready" NOT_DONE.md; then
  echo "ERROR: NOT_DONE must not still call rc.7 tag-ready (already tagged)" >&2
  exit 1
fi
grep -q "NOT_DONE.md" README.md
grep -q "NOT_DONE.md" docs/getting-started.md
grep -q "NOT_DONE.md" docs/hardware-checklist.md
grep -q "NOT_DONE.md" docs/hardware-qualification.md
grep -q "post-#42 / rc.9+" docs/hardware-qualification.md
if grep -q "rc.7 is done in" docs/hardware-qualification.md; then
  echo "ERROR: hardware-qualification must not still say rc.7 is done in software" >&2
  exit 1
fi
grep -q '## \[0.1.0-rc.9\]' CHANGELOG.md
grep -q '## \[0.1.0-rc.8\]' CHANGELOG.md
grep -q "SDK version align" CHANGELOG.md
grep -q "clinic-https.json" CHANGELOG.md
grep -q "bind-stress" CHANGELOG.md
grep -q "NOT_DONE hardware playbook" CHANGELOG.md
grep -q "bind-stress.md" CHANGELOG.md
grep -q "NOT_DONE.md" CHANGELOG.md

echo "OK smoke-package"
