# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1-dev] — unreleased

### Added

- Local loopback WebSocket bridge (`openthaiid.v1`) on `127.0.0.1:17891`
- CLI: `openthaiid`, `openthaiid doctor`, `openthaiid version`
- Terminal origin/field approval (`allow` / `deny` / `revoke` / `revoke-all`)
- PC/SC backend for Windows, Linux, macOS (`ebfe/scard` pinned)
- Synthetic backend for CI and hardware-free tests, including a seeded
  fictional demo card readable by `openthaiid -synthetic`
- Thai citizen MVP profile: `citizenId`, `thaiName`, `birthDate`
- Browser SDK `@openthaiid/web` with cancellation, session renewal, and reconnect
- Vanilla example page (external module script, CSP-friendly)
- Playwright browser acceptance harness (`acceptance/`) against the real
  bridge + synthetic card, using node-pty for real terminal approval (no bypass)
- Cross-platform acceptance: Playwright `chrome` / `msedge` channels,
  approval policies (allow / deny / unanswered), allowlisted reports
- Eight concurrent async operations per connection (`RATE_LIMITED` on excess)
- Event-queue overflow closes clients with WebSocket 1013 and requires a
  fresh snapshot after reconnect
- Native read worker ownership: open → transaction → transmit → end → close
  under one owner; prompt `CANCELLED`/`READ_TIMEOUT`; `BUSY` on overlap;
  `BACKEND_STALLED` after 2s post-cancel; bounded shutdown without releasing
  a PC/SC context still in use
- Deterministic integration tests: seed synthetic state before startup,
  deadline-bounded card-session waits, exact synthetic field assertions
- Contract docs (architecture, security, protocol, provenance, acceptance)
- GitHub Actions CI: Go test/race, five-target build matrix with version smoke,
  SDK (`npm ci` only), Chrome×3 + Edge×1 browser acceptance with
  allow/deny/unanswered policies, and repeated bridge/reader tests

### Changed (protocol / card — pre-release corrections)

- READ BINARY command form corrected to the cited APDU table:
  `CLA=80 INS=B0 P1 P2 Lc=02 Data=00 Le=len`
- Thai name is one `#`-delimited 100-byte block at `0x0011` (not four 100-byte
  blocks)
- Birth date offset is `0x00D9` (not `0x0331`)
- Birth-date years on the card are Buddhist Era; the bridge returns Gregorian
  years and validates leap years
- Requested unavailable fields are explicit JSON `null` with warnings;
  unrequested fields are omitted
- Oversized APDU responses are rejected instead of truncated
- Thai name decoding uses `golang.org/x/text` Windows-874 (not a hand map)
- Provenance ledger records immutable source revision
  `4d803243715254cc93efe9040d8b1ecd85f520aa`

### Fixed

- Sessions are capped at the live grant expiry; an expired grant invalidates
  an otherwise unexpired token
- Approvals are bound to their originating connection and cancelled on disconnect
- Trust revocation delivers `authorization.revoked` and closes affected clients
- Duplicate / non-increasing request IDs are rejected
- Cancellation targets the real in-flight request id (SDK no longer sends `0`)
- Receive loop stays responsive while approval and card operations run
- Event sequences are contiguous per connection
- PC/SC `GetStatusChange` no longer busy-loops on `StateUnaware`
- Card generations are invalidated on reader loss and service recovery
- Synthetic monitor/read contexts share one device state
- Windows CI shell steps use explicit Bash; `npm ci` is required

### Security

- Origin and Host validation; pairing-only until operator approval
- Origins with paths, queries, fragments, or invalid ports are rejected
- IPv6 origins keep bracketed serialization
- In-memory grants and sessions (8h trust / 15m session)
- Unicast read results; no card-data logging
- Terminal approval output is escaped
- Card-session generations prevent stale results after removal

### Not yet claimed

- Physical ACR39U acceptance on Windows/Linux/macOS
- Playwright browser acceptance matrix (Chrome/Edge across three OSes)
- Official Firefox/Safari support
- Portraits, remaining fields, React hooks, installers, tray UI
