# Browser acceptance harness

Runs Playwright Chromium against the **real** `openthaiid` bridge with the
synthetic backend and a seeded fictional card.

## How approvals work

The bridge fails closed when stdin is not a TTY (`NonInteractive`). This
harness starts the bridge under a PTY (`pty-run.py`) and types `allow <id>`
when it sees a terminal approval prompt — the same path a human operator
uses. There is no production approval bypass.

## Prerequisites

```bash
# From the repo root
make build sdk
cd acceptance
npm ci
npx playwright install chromium
```

## Run

```bash
npm test
# or from the repo root
make acceptance
```

## Scenarios

1. HTTPS example page → Connect → terminal allow → Read → assert citizenId,
   Thai name, Gregorian birth year
2. Disconnect clears displayed personal data
3. Connection to a closed/unsafe port fails closed
4. Page CSP allows `ws://127.0.0.1` loopback WebSocket

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `OPENTHAIID_BIN` | `../dist/openthaiid` | Bridge binary |
| `OPENTHAIID_PORT` | `17899` | Bridge listen port |
| `OPENTHAIID_HTTPS_PORT` | `8443` | Static HTTPS server |
| `PLAYWRIGHT_CHROMIUM_EXECUTABLE` | auto | Override browser binary |
| `PYTHON` | `python3` | Used for the PTY wrapper |
