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
- Synthetic backend for CI and hardware-free tests
- Thai citizen MVP profile: `citizenId`, `thaiName`, `birthDate`
- Browser SDK `@openthaiid/web`
- Vanilla example page
- Contract docs (architecture, security, protocol, provenance, acceptance)
- GitHub Actions CI: Go test/race, five-target build matrix, SDK

### Security

- Origin and Host validation; pairing-only until operator approval
- In-memory grants and sessions (8h trust / 15m session)
- Unicast read results; no card-data logging
- Card-session generations prevent stale results after removal

### Not yet claimed

- Physical ACR39U acceptance on Windows/Linux/macOS
- Official Firefox/Safari support
- Portraits, remaining fields, React hooks, installers, tray UI
