#!/usr/bin/env bash
# Software smoke for the PC/SC bind-stress harness (synthetic only).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/dist/openthaiid}"
if [ ! -x "$BIN" ]; then
  echo "missing $BIN (run make build)" >&2
  exit 1
fi

chmod +x "$ROOT/scripts/bind-stress.sh"
REPORT="$(mktemp)"
cleanup() { rm -f "$REPORT"; }
trap cleanup EXIT

"$ROOT/scripts/bind-stress.sh" --synthetic --cycles 3 --report "$REPORT"

test -s "$REPORT"
grep -q '"schema": "openthaiid.bindstress.v1"' "$REPORT"
grep -q '"pii"' "$REPORT"
python3 - "$REPORT" <<'PY'
import json, sys
rep = json.load(open(sys.argv[1]))
by = {c["id"]: c for c in rep["checks"]}
for i in ("B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8", "B9"):
    if by[i]["result"] != "PASS":
        raise SystemExit(f"{i} {by[i]['result']}")
if rep.get("mode") != "synthetic-smoke":
    raise SystemExit("mode must be synthetic-smoke")
if not rep.get("evidence", {}).get("needsACR39U"):
    raise SystemExit("synthetic must still need ACR39U")
if not rep.get("evidence", {}).get("syntheticDoesNotClose"):
    raise SystemExit("synthetic must list P2–P10 as not closed")
blob = json.dumps(rep)
for s in ("1234567890121", "สมชาย", "Somchai", "/9j/", "data:image",
          "00A4040008A000000054480001", "A000000054480001"):
    if s in blob:
        raise SystemExit(f"report leaked {s!r}")
print("OK bind-stress-smoke overall=%s" % rep["overall"])
PY
