# Browser acceptance harness

Runs Playwright against the **real** `openthaiid` bridge with the synthetic
backend and a seeded fictional card.

## How approvals work

The bridge fails closed when stdin is not a TTY. This harness starts the
bridge under a real PTY via **node-pty** (cross-platform: Windows, macOS,
Linux) and types `allow <id>` / `deny <id>` when it sees a terminal approval
prompt — the same path a human operator uses. There is no production
approval bypass.

Policies (`OPENTHAIID_APPROVAL_POLICY`):

| Policy | Behavior |
|---|---|
| `allow` (default) | Auto-allow every approval once |
| `deny` | Auto-deny every approval once |
| `unanswered` | Leave approvals pending |

## Prerequisites

```bash
# From the repo root
make build sdk
cd acceptance
npm ci
npx playwright install chrome msedge
```

## Run

```bash
npm test                 # allow policy, default channel chrome
npm run test:all         # allow + deny + unanswered
OPENTHAIID_BROWSER=msedge npm test
# or from the repo root
make acceptance
```

## Scenarios (allow policy)

1. Connect → terminal allow → Read → assert citizenId, full Thai name,
   Gregorian birth date
2. Disconnect clears displayed personal data
3. Reconnect does not automatically repeat a card read
4. Full-page refresh reuses the active grant
5. Connection failure on a closed port
6. CSP allows `ws://127.0.0.1`
7. Two browser contexts: requester read is unicast (peer page and
   WebSocket frames stay free of card fields); peer can still read on its
   own connection
8. Bridge process restart: displayed data is cleared, reconnect does not
   auto-read, and a fresh read works against the new process

Origin rejection (HTTP evil origin) is covered by Go integration tests
(`TestIntegrationBadOriginRejected`), not the browser harness: browsers
control the `Origin` header.

A headed manual check from a **public HTTPS** test origin is required for
browser local-network permission allow/deny behavior. That is separate from
this localhost HTTPS automation. See `docs/acceptance.md`.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `OPENTHAIID_BIN` | `../dist/openthaiid` | Bridge binary (`.exe` on Windows) |
| `OPENTHAIID_PORT` | `17899` | Bridge listen port |
| `OPENTHAIID_HTTPS_PORT` | `8443` | Static HTTPS server |
| `OPENTHAIID_BROWSER` | `chrome` | Playwright channel (`chrome`, `msedge`, `chromium`) |
| `OPENTHAIID_APPROVAL_POLICY` | `allow` | `allow` / `deny` / `unanswered` |
| `OPENTHAIID_REPORT_DIR` | `./reports` | Allowlisted JSON report |
| `OPENTHAIID_LOG_DIR` | `./logs` | Harness diagnostics |

Certificates are generated in the OS temp directory and never served.
Only the acceptance page and `sdk/web/dist/index.js` are served.

## Artifacts

- `reports/acceptance-report.json` — commit, platform, browser version,
  command, per-scenario results (synthetic values redacted)
- `logs/harness.log` — console diagnostics (no personal field values)
