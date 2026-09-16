#!/usr/bin/env bash
# Friday hardware qualification wrapper.
# Never prints or stores citizen ID, names, address, or portrait bytes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/dist/openthaiid}"
SYNTHETIC=0
E2="${E2_COUNT:-100}"
E3="${E3_COUNT:-20}"
REPORT="${REPORT:-}"
EXTRA=()

usage() {
  cat <<'EOF'
Usage: scripts/hardware-qualify.sh [--synthetic] [--e2 N] [--e3 N] [--report PATH]

  --synthetic   Practice reader (kit smoke). Does not claim ACR39U pass.
  --e2 N        Repeated reads (default 100; use 3 for smoke).
  --e3 N        Insert/remove cycles on --synthetic (default 20).
  --report PATH Write PII-free JSON (default: dist/openthaiid-qualify.json).

Hardware: plug ACR39U, insert a Thai ID, then run without --synthetic.
Operator E4–E7 and M1 stay on the checklist:
  docs/hardware-qualification.md

macOS first, then Windows, then Linux. Packages are unsigned.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --synthetic|-synthetic) SYNTHETIC=1 ;;
    --e2) E2="${2:-}"; shift ;;
    --e3) E3="${2:-}"; shift ;;
    --report) REPORT="${2:-}"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) EXTRA+=("$1") ;;
  esac
  shift
done

if [ ! -x "$BIN" ]; then
  echo "missing $BIN (run make build)" >&2
  exit 1
fi

mkdir -p "$ROOT/dist"
if [ -z "$REPORT" ]; then
  REPORT="$ROOT/dist/openthaiid-qualify.json"
fi

args=(qualify -skip-operator -e2-count "$E2" -e3-count "$E3" -report "$REPORT")
if [ "$SYNTHETIC" = 1 ]; then
  args+=(-synthetic)
  echo "OpenThaiID qualify: SYNTHETIC kit smoke (not a hardware pass)"
else
  echo "OpenThaiID qualify: hardware (no personal fields in output)"
  echo "Platform order: macOS + ACR39U first, then Windows, then Linux."
fi

echo "See docs/hardware-qualification.md for E4–E7 and M1 (HTTPS local-network)."
set +e
"$BIN" "${args[@]}" "${EXTRA[@]}"
status=$?
set -e

if [ -f "$REPORT" ]; then
  if grep -E '1234567890121|data:image|/9j/|สมชาย|Somchai' "$REPORT" >/dev/null 2>&1; then
    echo "ERROR: report looks like it contains personal or photo data; deleting" >&2
    rm -f "$REPORT"
    exit 1
  fi
  echo "report: $REPORT"
fi

exit "$status"
