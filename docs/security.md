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

## Handshake policy

- Bind only to `127.0.0.1`.
- Validate the exact configured `Host` authority and loopback peer address.
- Ignore forwarded-host and proxy headers.
- Require a valid browser `Origin`.
- Accept HTTPS origins and explicit HTTP loopback development origins.
- Reject missing, `null`, malformed, wildcard, and insecure non-loopback
  origins.
- Match origin by scheme, normalized hostname, and effective port.
- An eligible but untrusted origin is pairing-only: no inventory, events, or
  card data until approved.

## Authorization

1. Application calls `connect()` from a visible user action.
2. SDK sends `session.open` with the intended field set.
3. Bridge derives origin from the handshake and creates an expiring pending
   request.
4. Terminal shows exact origin, fields, request id, and comparison code.
5. Operator runs `allow <id>` or `deny <id>`.
6. Approval creates an in-memory origin grant and authenticates the connection.

- Origin grants hold the exact permitted field set.
- Expanding the field set requires a new approval.
- Grants expire after eight hours or process shutdown.
- Session tokens are 256-bit cryptographic randomness, expire after 15 minutes,
  remain in SDK memory, and travel only in protocol messages.
- Sessions bind to origin, grant, and bridge instance.
- Revocation cancels pending reads, closes sessions, and blocks delivery.
- Authorization is rechecked before delivering a completed read.
- Noninteractive processes fail closed on new approvals.

## Privacy and logging

- No telemetry or outbound network calls.
- No card-data files, databases, result caches, or diagnostic uploads.
- Logs carry only error codes, durations, counts, and random request ids.
- Never log citizen IDs, names, addresses, dates, portraits, raw APDUs, tokens,
  or hashes of personal fields.
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
- HTTP endpoints expose no cross-origin data interface and no wildcard CORS.

## Residual risk

An authorized origin retains access to the approved field set until grant
expiry or revocation. Display scope and lifetime clearly to the operator.
