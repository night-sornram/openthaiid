#!/bin/sh
# Unsigned clinic helper launcher for the macOS archive.
# Not notarized. Prefer OpenThaiID.app; this script is a fallback.
DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
if [ -d "$DIR/OpenThaiID.app" ]; then
  exec open "$DIR/OpenThaiID.app"
fi
exec "$DIR/openthaiid" tray -open
