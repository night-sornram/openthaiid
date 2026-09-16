#!/usr/bin/env bash
# Install the helper for the current user (no root). Does not enable autostart.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
BIN_SRC="$HERE/openthaiid"
if [ ! -x "$BIN_SRC" ]; then
  echo "openthaiid binary not found next to install-linux.sh" >&2
  exit 1
fi
BIN_DIR="${HOME}/.local/bin"
APP_DIR="${HOME}/.local/share/applications"
mkdir -p "$BIN_DIR" "$APP_DIR"
cp "$BIN_SRC" "$BIN_DIR/openthaiid"
chmod +x "$BIN_DIR/openthaiid"

cat > "$APP_DIR/openthaiid.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=OpenThaiID
Comment=Local Thai National ID helper (loopback only)
Exec="$BIN_DIR/openthaiid" tray -open
Terminal=false
Categories=Utility;
Actions=Ready;Technician;Status;Quit;

[Desktop Action Ready]
Name=Open Ready page
Exec="$BIN_DIR/openthaiid" open

[Desktop Action Technician]
Name=Open Technician tools
Exec="$BIN_DIR/openthaiid" open technician

[Desktop Action Status]
Name=Show status
Exec="$BIN_DIR/openthaiid" status

[Desktop Action Quit]
Name=Quit helper
Exec="$BIN_DIR/openthaiid" quit
EOF

echo "Installed $BIN_DIR/openthaiid"
echo "Application menu: OpenThaiID"
echo "Start at login is off. Opt in with: $BIN_DIR/openthaiid autostart enable"
echo "Always-on without a tray: $BIN_DIR/openthaiid service install"
echo "  (systemd --user; survive logout with: loginctl enable-linger \$USER)"
echo "The helper listens only on 127.0.0.1."
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  echo "Note: add $BIN_DIR to PATH if the command is not found."
fi
