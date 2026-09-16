#!/usr/bin/env bash
# Idempotent Cloud Agent bootstrap for OpenThaiID.
# Runs after the repository is checked out. Safe to run repeatedly.
set -euo pipefail

echo "==> Installing PC/SC system dependencies"
# libpcsclite-dev provides the cgo headers the PC/SC binding links against;
# pcscd is the daemon `openthaiid doctor` talks to with real hardware.
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update
sudo apt-get install -y --no-install-recommends libpcsclite-dev pcscd

echo "==> Fetching Go modules and building the bridge"
go mod download
make build
./dist/openthaiid version

echo "==> Installing and building the web SDK"
(cd sdk/web && npm ci && npm run build)

echo "==> Installing browser acceptance dependencies"
(cd acceptance && npm ci)
# Chrome ships in the base image; this resolves the Playwright channel and is a
# no-op when Chrome is already present.
(cd acceptance && npx playwright install chrome) || true

echo "==> Bootstrap complete"
