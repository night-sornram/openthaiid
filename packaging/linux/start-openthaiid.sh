#!/bin/sh
# Start the local helper and open the Ready page. No cloud service.
set -eu
cd "$(dirname "$0")"
exec ./openthaiid tray -open
