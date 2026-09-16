#!/usr/bin/env bash
# Software smoke for the Friday qualification kit (synthetic only).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/dist/openthaiid}"
if [ ! -x "$BIN" ]; then
  echo "missing $BIN (run make build)" >&2
  exit 1
fi

chmod +x "$ROOT/scripts/hardware-qualify.sh"
REPORT="$(mktemp)"
cleanup() { rm -f "$REPORT"; }
trap cleanup EXIT

"$ROOT/scripts/hardware-qualify.sh" --synthetic --e2 3 --e3 2 --report "$REPORT"

test -s "$REPORT"
grep -q '"schema": "openthaiid.qualify.v1"' "$REPORT"
grep -q '"pii"' "$REPORT"
# Automated slice must pass on the practice card.
python3 - "$REPORT" <<'PY'
import json, sys
rep = json.load(open(sys.argv[1]))
by = {c["id"]: c for c in rep["checks"]}
for i in ("E1", "E2", "E3"):
    if by[i]["result"] != "PASS":
        raise SystemExit(f"{i} {by[i]['result']}")
if by["E7"]["result"] != "SKIP":
    raise SystemExit("E7 should be SKIP in smoke")
blob = json.dumps(rep)
for s in ("1234567890121", "สมชาย", "Somchai", "/9j/", "data:image", "FICTIONAL-OPENTHAIID-PORTRAIT"):
    if s in blob:
        raise SystemExit(f"report leaked {s!r}")
print("OK qualify-smoke overall=%s" % rep["overall"])
PY
