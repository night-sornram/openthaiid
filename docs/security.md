# Security

## Trust boundary

OpenThaiID runs as the signed-in desktop user and talks only to `127.0.0.1`.
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
| Slow client delaying others | Per-connection output queue; overflow closes with WS 1013 |
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
- Operator-initiated `openthaiid update check|download|apply` (and the tray
  “Check for updates” item) query GitHub Releases and may download an
  **unsigned** asset. No origin, grant, token, or card field is sent.
- No card-data files, databases, result caches, or diagnostic uploads.
- Logs carry only error codes, durations, counts, and random request ids.
- Never log citizen IDs, names, addresses, dates, portraits (including
  Base64 / data URLs), raw APDUs, tokens, or hashes of personal fields.
- The helper page and technician **Copy diagnostics** omit personal card
  field values. Last errors store protocol codes and generic messages only.
- `GET /v1/helper/summary` (loopback) returns helper/reader/card lights for
  tray and `openthaiid status`. It does not include origins, field tokens, or
  personal values.
- `POST /v1/helper/shutdown` is loopback-only. Requests with a non-manager
  `Origin` are rejected so a website cannot stop the helper via the browser.
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
- Never require disabled browser security, public tunnels, bundled TLS private
  keys, or installed root certificates.
- Firefox mixed-content: `ws://127.0.0.1` from an HTTPS origin is blocked
  unless the browser allows insecure WebSockets from HTTPS (harness sets
  `network.websocket.allowInsecureFromHTTPS`). This does not weaken origin
  approval or loopback bind.
- Portrait `<img>` data URLs need CSP `img-src data:` (or `blob:` with
  `portraitBlob()`). That is display-only; grants and loopback bind are
  unchanged. Never log photo bytes.
- The local Manager (`GET /`, `GET/POST /v1/manager/*`) is loopback HTML and
  JSON for the operator on this PC. Mutating calls require the manager origin
  and a CSRF token embedded in the page (not a cookie). Personal card fields
  are not included in Manager responses.

## Residual risk

An authorized origin retains access to the approved field set until grant
expiry or revocation. Display scope and lifetime clearly to the operator.
