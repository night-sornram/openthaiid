#!/usr/bin/env bash
# PC/SC / ebfe/scard binding-audit stress wrapper.
# Never prints or stores citizen ID, names, address, portrait bytes, or raw APDUs.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/dist/openthaiid}"
SYNTHETIC=0
CYCLES="${CYCLES:-20}"
REPORT="${REPORT:-}"
EXTRA=()

usage() {
  cat <<'EOF'
Usage: scripts/bind-stress.sh [--synthetic] [--cycles N] [--report PATH]

  Repeatedly exercises PC/SC context connect/disconnect, reader list,
  presence probe, and (with --synthetic) card present/absent.

  --synthetic   Practice backend. Proves the harness. Does not close P2–P10.
  --cycles N    Establish/list/probe/release loops (default 20; use 3 for smoke).
  --report PATH Write PII-free JSON (default: dist/openthaiid-bind-stress.json).

Today (no ACR39U):  scripts/bind-stress.sh --synthetic --cycles 3
Hardware (ACR39U):    scripts/bind-stress.sh --report ./openthaiid-bind-stress.json
                    Never -synthetic for a hardware claim.

Never prints citizen ID, names, address, photo bytes, or raw APDUs.
See docs/bind-stress.md.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --synthetic|-synthetic) SYNTHETIC=1 ;;
    --cycles) CYCLES="${2:-}"; shift ;;
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
  REPORT="$ROOT/dist/openthaiid-bind-stress.json"
fi
REPORT="$(cd "$(dirname "$REPORT")" && pwd)/$(basename "$REPORT")"

args=(bind-stress -cycles "$CYCLES" -report "$REPORT")
if [ "$SYNTHETIC" = 1 ]; then
  args+=(-synthetic)
  echo "OpenThaiID bind-stress: SYNTHETIC harness smoke (does not close P2–P10)"
else
  echo "OpenThaiID bind-stress: hardware PC/SC (no personal fields, no APDUs)"
  echo "A pass on this OS/arch does not close other hosts. See docs/bind-stress.md"
fi

echo "report will be: $REPORT"
set +e
"$BIN" "${args[@]}" "${EXTRA[@]}"
status=$?
set -e

if [ -f "$REPORT" ]; then
  if grep -E '1234567890121|data:image|/9j/|สมชาย|Somchai|00A4040008A000000054480001' "$REPORT" >/dev/null 2>&1; then
    echo "ERROR: report looks like it contains personal data or an APDU dump; deleting" >&2
    rm -f "$REPORT"
    exit 1
  fi
  echo "report: $REPORT"
fi

exit "$status"
