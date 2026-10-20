# Security

## Trust boundary

OpenThaiID runs as the signed-in desktop user (tray / autostart) or, when
clinic IT opts in, as an always-on helper (`openthaiid serve`: Windows
service, Linux systemd --user, or a documented Task Scheduler task). It
talks only to `127.0.0.1`.
It is not an OS-user authenticator. Any native process on the machine can
construct its own `Origin` header; origin checks protect websites operating
within browser security rules only.

## Threat model

| Threat | Control |
|---|---|
| Random website reads an inserted card | Explicit origin-and-field approval before device access |
| Cross-site WebSocket hijacking | Validate browser `Origin`; authenticate before accepting operations |
| DNS rebinding | Bind literal IPv4 loopback; validate exact configured authority |
| Forged approval details | Derive origin from the handshake, never from request JSON |
| Approval spam | Bounded pending requests, rate limits, expiry, denial cooldown |
| Token leakage | Memory-only tokens; no URL, cookie, log, or browser-storage transport |
| Stale result after card replacement | Card-insertion generation checked throughout the operation |
| Cross-client data mix-up | Unicast read results; metadata-only event subscriptions |
| Malformed card or oversized frames | Strict lengths, bounded command sequences, parser validation |
| Compromised authorized site | Narrow field grants, expiry, revocation |
| Local malware impersonating an origin | Outside browser-origin trust guarantee |
| Slow client delaying others | Per-connection output queue; overflow closes with WS 1013. The single writer never holds the queue lock during a socket write, so a stalled socket cannot block events to other clients |
| Idle sockets exhausting `maxConnections` | A socket with no session and nothing in flight is closed after 30s (WS 1008 `authentication timeout`). A `session.open` waiting on staff Allow is in flight, so `approvalTimeout` still governs that wait |
| Slow or idle HTTP clients | `ReadHeaderTimeout` 5s, `IdleTimeout` 60s, 16 KiB header cap on HTTP and HTTPS listeners |
| Request flooding a connection | Eight concurrent async operations max; excess is `RATE_LIMITED` |
| Stuck native PC/SC call | Slot retained; `BACKEND_STALLED` after 2s post-cancel; no unsafe context release |

## Handshake policy

- Bind only to `127.0.0.1`.
- Validate the exact configured `Host` authority and loopback peer address.
- Ignore forwarded-host and proxy headers.
- Require a valid browser `Origin`.
- Accept HTTPS origins and explicit HTTP loopback development origins.
- Reject missing, `null`, malformed, wildcard, and insecure non-loopback
  origins.
- Match origin by scheme, normalized hostname, and effective port.
- An eligible but untrusted origin is pairing-only for **unsolicited**
  events: no inventory push, and no card data, until approved. The origin
  may call `bridge.probe` (metadata only: helper/PC/SC/readers/presence) so
  the JS library can check connection without a separate install.

## Authorization

1. Application calls `connect()` from a visible user action.
2. SDK sends `session.open` with the intended field set.
3. Bridge derives origin from the handshake and creates an expiring pending
   request.
4. The local helper shows the origin, fields, and comparison code on the
   clinic-staff view (request id and field tokens also appear in technician
   tools, or in the terminal).
5. Operator clicks Allow/Deny (the ERP page cannot approve itself).
6. Approval creates an in-memory origin grant and authenticates the connection.

- Origin grants hold the exact permitted field set.
- Expanding the field set requires a new approval.
- Grants expire after eight hours or process shutdown.
- Session tokens are 256-bit cryptographic randomness, expire after 15 minutes,
  remain in SDK memory, and travel only in protocol messages.
- Sessions bind to origin, grant, and bridge instance.
- Revocation cancels pending reads, closes sessions, and blocks delivery.
- Authorization is rechecked before delivering a completed read.
- Noninteractive processes may still create pending approvals because the
  helper page is the operator UI (stdin need not be a TTY). Set
  `NonInteractive` only for explicit fail-closed tests.
- At most eight asynchronous operations may be in flight per connection.
- Revocation during a blocked read suppresses the result and closes the socket.

## Privacy and logging

- No telemetry. The helper does not phone home during card reads.
- Operator-initiated GitHub Releases **check** (`openthaiid update check`,
  Ready **ตรวจเวอร์ชัน**, tray **ตรวจเวอร์ชัน**,
  `POST /v1/manager/update/check`) queries the GitHub API only. It does not
  download. Offline / rate-limit fails soft. `update download` /
  `update apply --yes` remain explicit technician CLI. No origin, grant,
  token, or card field is sent.
- No card-data files, databases, result caches, or diagnostic uploads.
- Logs carry only error codes, durations, counts, and random request ids.
- Never log citizen IDs, names, addresses, dates, portraits (including
  Base64 / data URLs), raw APDUs, tokens, or hashes of personal fields.
- The helper page and technician **Copy diagnostics** omit personal card
  field values. Last errors store protocol codes and generic messages only.
- **Operator audit trail** (technician UI, `GET /v1/manager/audit`,
  `openthaiid audit`, user-dir `audit.jsonl`) records helper start/stop,
  reader/card presence, approval allow/deny/revoke, and Fill failures by
  **error code only**. It never stores citizen ID, names, address, portrait,
  or raw APDUs. Poisoned lines are dropped and rewritten, not redacted in
  place with leftover values.
- `GET /v1/helper/summary` (loopback) returns helper/reader/card lights for
  tray and `openthaiid status`. It does not include origins, field tokens, or
  personal values. A website `Origin` is rejected (CLI/tray send none).
- `POST /v1/helper/shutdown` is loopback-only. Foreign `Origin` and
  `Sec-Fetch-Site: cross-site` are rejected so a website cannot stop the
  helper. A browser `Origin` also needs the manager CSRF header.
  `openthaiid quit` sends no `Origin` and still works.
- Verbose logging (`-log-level debug` or the technician checkbox) is optional
  and defaults off; it still must not include personal fields.
- Terminal approval output is separate from persistent logs.
- Response buffers are released promptly; guaranteed memory erasure is not
  promised in Go or JavaScript.

## Browser integration notes

- WebSocket does not bypass local-network permission prompts. Chrome 147
  extended checks to local WebSocket connections.
- Applications need a real HTTPS origin talking to loopback.
- CSP must allow the bridge WebSocket; document recovery when permission is
  denied.
- Never require disabled browser security, public tunnels, or a **bundled**
  TLS private key. Optional loopback HTTPS/WSS uses a **technician-managed**
  cert on that PC only ([loopback-tls.md](loopback-tls.md)).
- Firefox mixed-content: `ws://127.0.0.1` from an HTTPS origin is blocked
  on Firefox and Safari. The SDK reports `HTTPS_WS_BLOCKED` instead of
  helper-down unless loopback TLS is enabled. With a trusted local cert the
  SDK uses `wss://127.0.0.1:17892`. An untrusted cert is `TLS_UNTRUSTED`.
  The acceptance harness may set
  `network.websocket.allowInsecureFromHTTPS` (**tests only**). This does not
  weaken origin approval or loopback bind. Chromium still allows localhost
  mixed WebSockets and may prompt for Local Network Access.
- Portrait `<img>` data URLs need CSP `img-src data:` (or `blob:` with
  `portraitBlob()`). That is display-only; grants and loopback bind are
  unchanged. Never log photo bytes.
- The local Manager (`GET /`, `GET/POST /v1/manager/*`) is loopback HTML and
  JSON for the operator on this PC. Mutating calls require the manager origin
  and a CSRF token embedded in the page (not a cookie). `Sec-Fetch-Site:
  cross-site` is rejected. Refresh/reconnect are bounded (one in flight, four
  per two seconds). GitHub Releases **check** is a separate operator POST
  (one in flight, two per 20 seconds) and is never invoked from Ready load
  or Fill. Personal card fields are not included in Manager responses.

## Loopback HTTP on `127.0.0.1`

The helper binds only IPv4 loopback. Any local process can still call these
APIs (origin checks protect browsers, not malware on the PC). There is no
CORS. Hardware-arrival hub: [getting-started.md](getting-started.md).
Technician debug: [technician.md](technician.md). Software rehearsal
without a reader: `make pre-reader-dry-run` ([training.md](training.md)).
Firefox/Safari https Fill: [compatibility.md](compatibility.md). Optional
helper TLS: [loopback-tls.md](loopback-tls.md).
Endpoints:

| Path | Used by | Mutates | Browser rule |
|---|---|---|---|
| `GET /` | Ready / technician page | no | HTML; CSRF token in page |
| `GET /healthz` | probes | no | liveness only; no readers or card state |
| `GET /v1/helper/summary` | tray, `openthaiid status` | no | no PII; foreign `Origin` rejected |
| `GET /v1/helper/doctor` | `openthaiid doctor`, Technician **Desk snapshot** | no | no PII / no key paths; foreign `Origin` rejected; does not call GitHub |
| `POST /v1/helper/shutdown` | `openthaiid quit` | yes | foreign `Origin` rejected; browser `Origin` needs CSRF |
| `GET /v1/helper/trust` | `openthaiid trust list` | no | foreign `Origin` rejected; origins only, no PII |
| `POST /v1/helper/trust` | `openthaiid trust add\|remove` | yes | foreign `Origin` rejected; browser `Origin` needs CSRF; explicit origin only |
| `GET /v1/manager/status` | Ready page | no | CSRF + manager origin; no PII |
| `GET /v1/manager/audit` | technician export | no | CSRF + manager origin; redacted events |
| `POST /v1/manager/refresh` | technician | yes | CSRF + manager origin; rate-limited |
| `POST /v1/manager/reconnect` | technician | yes | CSRF + manager origin; rate-limited |
| `POST /v1/manager/reader/select` | Ready page | yes | CSRF + manager origin; ERP cannot select |
| `POST /v1/manager/approvals/{id}/allow\|deny` | operator | yes | CSRF + manager origin; ERP cannot allow itself |
| `POST /v1/manager/trust/revoke` `revoke-all` | operator | yes | CSRF + manager origin; revoke-all keeps IT pre-approved sites |
| `POST /v1/manager/trust/preapprove` `unpreapprove` | technician / IT | yes | CSRF + manager origin; explicit origin only; not “allow all” |
| `POST /v1/manager/log-level` | technician | yes | CSRF + manager origin |
| `POST /v1/manager/update/check` | Ready / technician / tray-adjacent | no (lookup only) | CSRF + manager origin; rate-limited; never on page load; no download |
| `GET /v1/ws` | ERP JavaScript | via protocol | eligible `Origin`; approval before card fields |

Optional second listener when `tlsCertFile` + `tlsKeyFile` are set: same
paths on `127.0.0.1:17892` over HTTPS/WSS. Ready stays on HTTP. Healthz
advertises `"tls": true` and `wssUrl`. No CORS. No bundled key.

## Residual risk

An authorized origin retains access to the approved field set until grant
expiry or revocation. Display scope and lifetime clearly to the operator.
