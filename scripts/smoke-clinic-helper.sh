#!/usr/bin/env bash
# Linux smoke: tray/background helper, staff status, autostart files, quit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/dist/openthaiid}"
if [ ! -x "$BIN" ]; then
  echo "missing $BIN (run make build)" >&2
  exit 1
fi

HOME_DIR="$(mktemp -d)"
export OPENTHAIID_HOME="$HOME_DIR"
export HOME="$HOME_DIR"
export XDG_CONFIG_HOME="$HOME_DIR/.config"
PORT="${PORT:-17991}"
log="$HOME_DIR/smoke.log"

cleanup() {
  "$BIN" quit -port "$PORT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

"$BIN" version | tee -a "$log"
"$BIN" help | grep -q "openthaiid tray"
"$BIN" help | grep -q "openthaiid serve"
"$BIN" help | grep -q "openthaiid service"
"$BIN" help | grep -q "openthaiid audit"

echo "== autostart opt-in (off by default) =="
out="$("$BIN" autostart status)"
echo "$out"
echo "$out" | grep -q "off"
"$BIN" autostart enable
"$BIN" autostart status | grep -q "on"
test -f "$HOME_DIR/.config/autostart/openthaiid.desktop"
grep -q "tray" "$HOME_DIR/.config/autostart/openthaiid.desktop"
! grep -q -- "-open" "$HOME_DIR/.config/autostart/openthaiid.desktop"
"$BIN" autostart disable
"$BIN" autostart status | grep -q "off"

echo "== always-on service unit (file only; enable is opt-in) =="
"$BIN" service status | tee -a "$log"
"$BIN" service install
"$BIN" service status | grep -q "Always-on helper: installed"
unit="$HOME_DIR/.config/systemd/user/openthaiid.service"
test -f "$unit"
grep -E -q "ExecStart=.+ serve" "$unit"
! grep -E "ExecStart=.+tray" "$unit"
! grep -q -- "-open" "$unit"
"$BIN" service uninstall
"$BIN" service status | grep -q "not installed"

echo "== tray / background helper =="
"$BIN" tray -synthetic -port "$PORT" -open=false >"$HOME_DIR/helper.out" 2>"$HOME_DIR/helper.err" &
pid=$!
ok=0
for _ in $(seq 1 50); do
  if curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null; then
    ok=1
    break
  fi
  sleep 0.1
done
if [ "$ok" != 1 ]; then
  echo "helper did not start" >&2
  cat "$HOME_DIR/helper.out" "$HOME_DIR/helper.err" >&2 || true
  exit 1
fi

sum="$(curl -sf "http://127.0.0.1:${PORT}/v1/helper/summary")"
echo "$sum"
echo "$sum" | grep -q '"helper":true'
echo "$sum" | grep -q '"headline"'
echo "$sum" | grep -vq '1234567890121'
echo "$sum" | grep -vq 'สมชาย'
echo "$sum" | grep -vq 'pending'

st="$("$BIN" status -port "$PORT")"
echo "$st"
echo "$st" | grep -q "OpenThaiID"
echo "$st" | grep -E -q "พร้อม|Ready|กำลังทำงาน|is running"
echo "$st" | grep -vq '1234567890121'

# Foreign Origin must not be able to quit the helper.
code="$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Origin: https://evil.example' "http://127.0.0.1:${PORT}/v1/helper/shutdown")"
test "$code" = "403"

"$BIN" quit -port "$PORT"
for _ in $(seq 1 50); do
  if ! kill -0 "$pid" 2>/dev/null; then
    break
  fi
  sleep 0.1
done

echo "== headless serve + redacted audit =="
"$BIN" serve -synthetic -port "$PORT" >"$HOME_DIR/serve.out" 2>"$HOME_DIR/serve.err" &
pid=$!
ok=0
for _ in $(seq 1 50); do
  if curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null; then
    ok=1
    break
  fi
  sleep 0.1
done
if [ "$ok" != 1 ]; then
  echo "serve did not start" >&2
  cat "$HOME_DIR/serve.out" "$HOME_DIR/serve.err" >&2 || true
  exit 1
fi
page="$(curl -sf "http://127.0.0.1:${PORT}/")"
echo "$page" | grep -q "Operator audit trail"
echo "$page" | grep -vq '1234567890121'
echo "$page" | grep -vq 'สมชาย'
audit_out="$("$BIN" audit)"
echo "$audit_out"
echo "$audit_out" | grep -q "openthaiid.audit.v1"
echo "$audit_out" | grep -q "never stored"
echo "$audit_out" | grep -q "helper.start"
echo "$audit_out" | grep -vq '1234567890121'
echo "$audit_out" | grep -vq 'สมชาย'
"$BIN" quit -port "$PORT"
for _ in $(seq 1 50); do
  if ! kill -0 "$pid" 2>/dev/null; then
    break
  fi
  sleep 0.1
done

echo "OK smoke-clinic-helper"
