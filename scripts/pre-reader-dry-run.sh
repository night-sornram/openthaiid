#!/usr/bin/env bash
# Pre-reader software dry-run (synthetic). No ACR39U required.
# Rehearses: build → qualify-smoke → -synthetic helper → Ready/summary/status
# → Test connection (loopback summary + CSRF status + bridge.probe).
# Never prints citizen ID, names, address, or photo bytes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/dist/openthaiid}"
PORT="${PORT:-17981}"
TWO_READERS=0
SKIP_QUALIFY=0

usage() {
  cat <<'EOF'
Usage: scripts/pre-reader-dry-run.sh [--two-readers] [--port N] [--skip-qualify]

  Rehearse the clinic software path without a real reader.
  Does not claim an ACR39U / E1–E7 hardware pass.

  --two-readers   Seed OPENTHAIID_SYNTHETIC_READERS=2 and pick ใช้เครื่องนี้
  --port N        Helper port (default 17981; does not clash with 17891)
  --skip-qualify  Skip make qualify-smoke (helper/Ready checks still run)

  make pre-reader-dry-run
  ./scripts/pre-reader-dry-run.sh --two-readers

  Never prints citizen ID, names, address, or photo bytes.
  See docs/getting-started.md and docs/training.md (ซ้อมก่อนได้เครื่อง).
  When hardware arrives: docs/hardware-checklist.md
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --two-readers|-two-readers) TWO_READERS=1 ;;
    --port) PORT="${2:-}"; shift ;;
    --skip-qualify) SKIP_QUALIFY=1 ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
  echo "invalid --port" >&2
  exit 2
fi

# Fictional-card / photo sentinels. Fail closed; never echo the match.
FORBIDDEN_PY='[
  "1234567890121", "1101701234567",
  "สมชาย", "ใจดี", "Somchai", "Jaidee",
  "เทศบาลนครนิทาน", "ถนนนิทาน", "ตำบลบ้านใหม่", "อำเภอเมือง", "จังหวัดนิทาน",
  "25330412", "25600115", "25700114",
  "FICTIONAL-OPENTHAIID-PORTRAIT",
  "/9j/", "data:image", "image/jpeg;base64,", "iVBORw0KGgo", "FFD8FF",
]'

refuse_pii_file() {
  local path="$1"
  python3 - "$path" <<PY
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
blob = p.read_bytes()
needles = ${FORBIDDEN_PY}
for s in needles:
    if s.encode() in blob:
        raise SystemExit("ERROR: output leaked personal or photo data; refusing to print")
if re.search(br"(?<![0-9])[0-9]{13}(?![0-9])", blob):
    raise SystemExit("ERROR: output leaked personal or photo data; refusing to print")
if b"\\xff\\xd8\\xff" in blob:
    raise SystemExit("ERROR: output leaked personal or photo data; refusing to print")
PY
}

assert_json() {
  python3 - "$@" <<'PY'
import json, pathlib, sys
needles = [
  "1234567890121", "1101701234567",
  "สมชาย", "ใจดี", "Somchai", "Jaidee",
  "เทศบาลนครนิทาน", "ถนนนิทาน", "ตำบลบ้านใหม่", "อำเภอเมือง", "จังหวัดนิทาน",
  "25330412", "25600115", "25700114",
  "FICTIONAL-OPENTHAIID-PORTRAIT",
  "/9j/", "data:image", "image/jpeg;base64,", "iVBORw0KGgo", "FFD8FF",
]
path = pathlib.Path(sys.argv[1])
mode = sys.argv[2]
blob = path.read_text(encoding="utf-8")
for s in needles:
    if s in blob:
        raise SystemExit("ERROR: output leaked personal or photo data; refusing to print")
data = json.loads(blob)

def require(cond, msg):
    if not cond:
        raise SystemExit(msg)

if mode == "summary":
    two = sys.argv[3] == "1"
    require(data.get("helper") is True, "summary: helper is not up")
    require(data.get("practice") is True, "summary: expected practice/synthetic")
    require(data.get("reader") is True, "summary: expected a practice reader")
    headline = str(data.get("headline") or "")
    nxt = str(data.get("next") or "")
    if two:
        require(data.get("card") is not True, "summary: two readers must not look Ready until picked")
        require("ยังไม่พร้อม" in headline or "Not ready" in headline, "summary: expected Not ready until reader pick")
        require("มากกว่าหนึ่ง" in nxt or "more than one" in nxt.lower() or "เลือกเครื่อง" in nxt, "summary: expected choose-reader next step")
    else:
        require(data.get("card") is True, "summary: expected practice card present")
        require("พร้อม" in headline or "Ready" in headline, "summary: expected Ready (practice)")
    print("OK helper-summary practice=true")
elif mode == "manager-status":
    two = sys.argv[3] == "1"
    require(data.get("running") is True, "manager status: helper not running")
    require(data.get("synthetic") is True, "manager status: expected synthetic")
    backend = (data.get("backend") or {}).get("status")
    require(backend == "available", "manager status: backend not available")
    readers = data.get("readers") or []
    require(isinstance(readers, list) and readers, "manager status: no readers")
    if two:
        require(len(readers) == 2, "manager status: expected two practice readers")
        require(not any(r.get("selected") for r in readers), "manager status: no reader should be selected yet")
    else:
        require(any(r.get("selected") for r in readers), "manager status: solo reader should be selected")
        require(any(r.get("presence") == "present" for r in readers), "manager status: expected present practice card")
    for r in readers:
        require("cardSessionId" not in r, "manager status must not include cardSessionId")
        require("citizenId" not in r, "manager status must not include citizenId")
    print("OK manager-status probe-only")
elif mode == "probe":
    two = sys.argv[3] == "1"
    if "error" in data and data["error"]:
        raise SystemExit("bridge.probe returned an error")
    res = data.get("result") or {}
    require(res.get("synthetic") is True, "probe: expected synthetic")
    backend = (res.get("backend") or {}).get("status")
    require(backend == "available", "probe: backend not available")
    readers = res.get("readers") or []
    require(isinstance(readers, list) and readers, "probe: no readers")
    if two:
        require(len(readers) == 2, "probe: expected two practice readers")
        require(not any(r.get("selected") for r in readers), "probe: no reader should be selected yet")
    else:
        require(any(r.get("selected") for r in readers), "probe: solo reader should be selected")
        require(any(r.get("presence") == "present" for r in readers), "probe: expected present practice card")
    for r in readers:
        require("cardSessionId" not in r, "probe must not include cardSessionId")
        require("citizenId" not in r, "probe must not include citizenId")
    print("OK bridge.probe metadata-only")
else:
    raise SystemExit("unknown assert mode")
PY
}

ws_probe() {
  local host="$1" port="$2" out="$3"
  python3 - "$host" "$port" "$out" <<'PY'
import base64, json, os, socket, sys

host, port, out_path = sys.argv[1], int(sys.argv[2]), sys.argv[3]
origin = f"http://{host}:{port}"
key = base64.b64encode(os.urandom(16)).decode("ascii")
req = (
    f"GET /v1/ws HTTP/1.1\r\n"
    f"Host: {host}:{port}\r\n"
    f"Upgrade: websocket\r\n"
    f"Connection: Upgrade\r\n"
    f"Sec-WebSocket-Key: {key}\r\n"
    f"Sec-WebSocket-Version: 13\r\n"
    f"Sec-WebSocket-Protocol: openthaiid.v1\r\n"
    f"Origin: {origin}\r\n"
    f"\r\n"
).encode("ascii")

def recvn(sock, n):
    buf = b""
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise SystemExit("websocket closed")
        buf += chunk
    return buf

def mask_frame(opcode: int, payload: bytes) -> bytes:
    mk = os.urandom(4)
    masked = bytes(b ^ mk[i % 4] for i, b in enumerate(payload))
    n = len(payload)
    hdr = bytearray([0x80 | (opcode & 0x0F)])
    if n < 126:
        hdr.append(0x80 | n)
    elif n < 65536:
        hdr.append(0x80 | 126)
        hdr.extend(n.to_bytes(2, "big"))
    else:
        hdr.append(0x80 | 127)
        hdr.extend(n.to_bytes(8, "big"))
    hdr.extend(mk)
    return bytes(hdr) + masked

def read_text(sock) -> str:
    while True:
        hdr = recvn(sock, 2)
        opcode = hdr[0] & 0x0F
        ln = hdr[1] & 0x7F
        masked = bool(hdr[1] & 0x80)
        if ln == 126:
            ln = int.from_bytes(recvn(sock, 2), "big")
        elif ln == 127:
            ln = int.from_bytes(recvn(sock, 8), "big")
        mask = recvn(sock, 4) if masked else b""
        data = recvn(sock, ln)
        if masked:
            data = bytes(b ^ mask[i % 4] for i, b in enumerate(data))
        if opcode == 0x1:
            return data.decode("utf-8")
        if opcode == 0x8:
            raise SystemExit("websocket close")
        if opcode == 0x9:
            sock.sendall(mask_frame(0xA, data))
            continue
        if opcode in (0x0, 0xA):
            continue
        raise SystemExit("unexpected websocket opcode")

sock = socket.create_connection((host, port), timeout=5)
sock.settimeout(5)
try:
    sock.sendall(req)
    header = b""
    while b"\r\n\r\n" not in header:
        chunk = sock.recv(4096)
        if not chunk:
            raise SystemExit("no websocket handshake")
        header += chunk
        if len(header) > 8192:
            raise SystemExit("websocket handshake too large")
    head, rest = header.split(b"\r\n\r\n", 1)
    status = head.split(b"\r\n", 1)[0]
    if b"101" not in status:
        raise SystemExit("websocket upgrade failed")
    payload = json.dumps({"v": 1, "id": 1, "type": "bridge.probe"}).encode("utf-8")
    sock.sendall(mask_frame(0x1, payload))
    # If the server coalesced bytes after headers, we still need a frame.
    # Put leftover in a small buffer by prepending via makefile-style: send then read.
    # Leftover after handshake should be empty for this server.
    if rest:
        raise SystemExit("unexpected bytes after websocket handshake")
    msg = read_text(sock)
finally:
    try:
        sock.close()
    except OSError:
        pass

open(out_path, "w", encoding="utf-8").write(msg)
PY
}

csrf_from_ready() {
  python3 - "$1" <<'PY'
import pathlib, re, sys
html = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
m = re.search(r'name="openthaiid-csrf"\s+content="([a-f0-9]+)"', html)
if not m:
    raise SystemExit("Ready page missing CSRF token")
print(m.group(1))
PY
}

echo "== OpenThaiID pre-reader dry-run (synthetic; not a hardware pass) =="
echo "docs: getting-started.md · training.md · staff-desk-card.md · hardware-checklist.md"
echo "never print citizen ID, names, address, or photo bytes"

if [ ! -x "$BIN" ]; then
  echo "== build helper =="
  make -C "$ROOT" build
fi
if [ ! -x "$BIN" ]; then
  echo "missing $BIN (run make build)" >&2
  exit 1
fi

ver="$("$BIN" version)"
case "$ver" in
  *สมชาย*|*1234567890121*|*Somchai*|*data:image*)
    echo "ERROR: version output refused" >&2
    exit 1
    ;;
esac
echo "version: $ver"

if [ "$SKIP_QUALIFY" != 1 ]; then
  echo "== qualify-smoke (synthetic kit; not ACR39U) =="
  chmod +x "$ROOT/scripts/smoke-qualify.sh" "$ROOT/scripts/hardware-qualify.sh"
  BIN="$BIN" "$ROOT/scripts/smoke-qualify.sh"
fi

HOME_DIR="$(mktemp -d)"
export OPENTHAIID_HOME="$HOME_DIR"
export HOME="$HOME_DIR"
export XDG_CONFIG_HOME="$HOME_DIR/.config"
export OPENTHAIID_LANG="${OPENTHAIID_LANG:-th}"
if [ "$TWO_READERS" = 1 ]; then
  export OPENTHAIID_SYNTHETIC_READERS=2
  echo "== helper -synthetic (two practice readers) port=$PORT =="
else
  unset OPENTHAIID_SYNTHETIC_READERS || true
  echo "== helper -synthetic port=$PORT =="
fi

READY="http://127.0.0.1:${PORT}/"
SUM_URL="${READY}v1/helper/summary"
HEALTH="${READY}healthz"
STATUS_URL="${READY}v1/manager/status"

cleanup() {
  "$BIN" quit -port "$PORT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

"$BIN" -synthetic -port "$PORT" -open=false >"$HOME_DIR/helper.out" 2>"$HOME_DIR/helper.err" &
helper_pid=$!

ok=0
for _ in $(seq 1 80); do
  if curl -sf "$HEALTH" >/dev/null; then
    ok=1
    break
  fi
  if ! kill -0 "$helper_pid" 2>/dev/null; then
    break
  fi
  sleep 0.1
done
if [ "$ok" != 1 ]; then
  echo "helper did not start on $READY" >&2
  refuse_pii_file "$HOME_DIR/helper.out" || true
  refuse_pii_file "$HOME_DIR/helper.err" || true
  exit 1
fi

echo "== status + Ready + summary =="
st="$("$BIN" status -port "$PORT")"
printf '%s\n' "$st" >"$HOME_DIR/status.txt"
refuse_pii_file "$HOME_DIR/status.txt"
if [ "$TWO_READERS" = 1 ]; then
  grep -E -q "ยังไม่พร้อม|Not ready" "$HOME_DIR/status.txt"
else
  grep -E -q "บัตรฝึกซ้อม|practice card" "$HOME_DIR/status.txt"
fi
grep -vq '1234567890121' "$HOME_DIR/status.txt"
echo "OK status (no personal fields)"

echo "== doctor desk snapshot =="
doc="$("$BIN" doctor -port "$PORT")"
printf '%s\n' "$doc" >"$HOME_DIR/doctor.txt"
refuse_pii_file "$HOME_DIR/doctor.txt"
grep -q "pcsc: OK" "$HOME_DIR/doctor.txt"
grep -q "personal fields: not read" "$HOME_DIR/doctor.txt"
grep -q "helper: running" "$HOME_DIR/doctor.txt"
grep -q "probe: OK" "$HOME_DIR/doctor.txt"
grep -q "tls=" "$HOME_DIR/doctor.txt"
grep -vq "BEGIN PRIVATE" "$HOME_DIR/doctor.txt"
grep -vq "tlsKeyFile" "$HOME_DIR/doctor.txt"
echo "OK doctor (redacted desk snapshot)"

curl -sf "$READY" -o "$HOME_DIR/ready.html"
refuse_pii_file "$HOME_DIR/ready.html"
grep -q 'id="test-connection"' "$HOME_DIR/ready.html"
grep -q "ทดสอบการเชื่อมต่อ" "$HOME_DIR/ready.html"
grep -q "ฝึกใช้ก่อนได้เครื่องจริง" "$HOME_DIR/ready.html"
grep -q "openthaiid-csrf" "$HOME_DIR/ready.html"
grep -vq '1234567890121' "$HOME_DIR/ready.html"
grep -vq 'สมชาย' "$HOME_DIR/ready.html"
echo "OK Ready page (test-connection present; no personal fields)"

curl -sf "$SUM_URL" -o "$HOME_DIR/summary.json"
refuse_pii_file "$HOME_DIR/summary.json"
assert_json "$HOME_DIR/summary.json" summary "$TWO_READERS"

echo "== connection test (summary already fetched; CSRF status + bridge.probe) =="
token="$(csrf_from_ready "$HOME_DIR/ready.html")"
curl -sf -H "X-OpenThaiID-Manager: $token" "$STATUS_URL" -o "$HOME_DIR/manager-status.json"
refuse_pii_file "$HOME_DIR/manager-status.json"
assert_json "$HOME_DIR/manager-status.json" manager-status "$TWO_READERS"

ws_probe "127.0.0.1" "$PORT" "$HOME_DIR/probe.json"
refuse_pii_file "$HOME_DIR/probe.json"
assert_json "$HOME_DIR/probe.json" probe "$TWO_READERS"

if [ "$TWO_READERS" = 1 ]; then
  echo "== pick practice reader (ใช้เครื่องนี้) =="
  curl -sf -X POST \
    -H "X-OpenThaiID-Manager: $token" \
    -H "Origin: http://127.0.0.1:${PORT}" \
    -H "Content-Type: application/json" \
    --data '{"readerId":"synthetic-acr39u"}' \
    "${READY}v1/manager/reader/select" -o "$HOME_DIR/select.json"
  refuse_pii_file "$HOME_DIR/select.json"
  curl -sf "$SUM_URL" -o "$HOME_DIR/summary-selected.json"
  refuse_pii_file "$HOME_DIR/summary-selected.json"
  assert_json "$HOME_DIR/summary-selected.json" summary 0
  echo "OK reader selected → Ready (practice)"
fi

cat <<EOF

======== ผ่าน / PASS — ซ้อมก่อนได้เครื่อง ========
ซอฟต์แวร์ก่อนได้เครื่องพร้อมบนโหมดฝึกซ้อม (ยังไม่ใช่ผ่าน ACR39U)
pre-reader software path rehearsed on -synthetic (not a hardware pass).

หน้า พร้อม / Ready: ${READY}
ดัชนีเช้านี้: docs/getting-started.md
ฝึกกรอกข้อมูลต่อ: docs/training.md  (examples/clinic-patient/ แล้วกด อนุญาต)
เช้าที่มีเครื่องจริง: พิมพ์ docs/staff-desk-card.md ที่โต๊ะ
ชุด qualify: docs/hardware-checklist.md  (เปิดตัวช่วยโดยไม่ใส่ -synthetic)

ห้ามส่งเลขบัตร ชื่อ ที่อยู่ วันเกิด รูปถ่าย ภาพหน้าจอฟอร์ม หรือ HAR ที่มียอด card.read
Do not send citizen ID, names, address, date of birth, photos, filled-form
screenshots, or HARs that contain card.read payloads.

EOF
