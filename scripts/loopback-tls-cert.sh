#!/usr/bin/env bash
# Generate a 127.0.0.1 loopback certificate for optional helper HTTPS/WSS.
# Never commit the private key. Prefer mkcert when it is available
# (see docs/loopback-tls.md). This openssl fallback is for machines without mkcert.
set -euo pipefail

DEST="${1:-}"
if [[ -z "$DEST" ]]; then
  if [[ -n "${OPENTHAIID_HOME:-}" ]]; then
    DEST="$OPENTHAIID_HOME"
  elif [[ -n "${XDG_CONFIG_HOME:-}" ]]; then
    DEST="$XDG_CONFIG_HOME/openthaiid"
  elif [[ "$(uname -s)" == "Darwin" ]]; then
    DEST="$HOME/Library/Application Support/openthaiid"
  else
    DEST="$HOME/.config/openthaiid"
  fi
fi

mkdir -p "$DEST"
chmod 700 "$DEST" 2>/dev/null || true
CERT="$DEST/loopback.pem"
KEY="$DEST/loopback-key.pem"
CNF="$(mktemp)"
trap 'rm -f "$CNF"' EXIT

cat >"$CNF" <<'EOF'
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no
[req_distinguished_name]
CN = 127.0.0.1
[v3_req]
subjectAltName = @alt_names
keyUsage = digitalSignature
extendedKeyUsage = serverAuth
[alt_names]
IP.1 = 127.0.0.1
DNS.1 = localhost
EOF

openssl req -x509 -newkey rsa:2048 -sha256 -days 825 -nodes \
  -keyout "$KEY" -out "$CERT" -config "$CNF"
chmod 600 "$KEY" "$CERT"

echo "Wrote $CERT"
echo "Wrote $KEY"
echo "Do not commit these files. Trust the certificate in the clinic browser."
echo "Then start the helper with:"
echo "  openthaiid tray -tls-cert \"$CERT\" -tls-key \"$KEY\""
echo "Docs: docs/loopback-tls.md"
