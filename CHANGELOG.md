# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Thai Fill-error guidance, staff training, and a one-page Friday runbook on
`dev` while ACR39U hardware is in transit. **Do not enable GitHub Actions.**
Physical E1–E7 execution still waits on the reader. Packages remain
**unsigned** (not notarized).

### Added

- Clinic Fill / probe errors default to **Thai** staff guidance
  (`formatStaffError`, `staffErrorGuidance`) while protocol **codes** stay
  stable (`BRIDGE_UNAVAILABLE`, `NO_READER`, `NO_CARD`, `APPROVAL_REQUIRED` /
  `APPROVAL_DENIED`, `FIELD_NOT_ALLOWED`, …). English: `{ lang: "en" }`.
  `autofillThaiID` and `probeThaiID` localize `ThaiIDError.message` for ERP
  chrome. Wire/technician messages remain English.
- Staff training: Ready-page **ฝึกใช้ก่อนได้เครื่องจริง** card in
  `-synthetic`, Thai clinic example flow, [docs/training.md](docs/training.md).
- One-page Friday runbook: [docs/friday-checklist.md](docs/friday-checklist.md)
  (plug reader → Ready → qualify → Fill → diagnostics only for tickets).

### Changed

- Practice-mode Ready copy no longer hides a pending Allow prompt.
- Clinic example staff chrome is Thai-first (`?lang=en` fallback).

### Added (Thai staff UX)

- Clinic-staff Ready copy defaults to **Thai** (พร้อม / ยังไม่พร้อม) on the
  helper page, `mountThaiIDStatus` / `summarizeProbe`, tray **แสดงสถานะ**,
  and `openthaiid status`. English fallback: Ready page ไทย/English toggle,
  `{ lang: "en" }` in the SDK, or `OPENTHAIID_LANG=en`. Technician / debug
  chrome stays English (bilingual toggle label). Staff user doc is Thai-first.
- `openthaiid qualify -operator` records E4–E7 / M1 as pass/fail/skip from
  the keyboard (those words only; pasted card data is refused). Hardware
  mode prints the bilingual operator prompts even without `-operator`.
  Hardware E3 can be driven with Enter (remove / seat) instead of only
  `-synthetic`. Always writes **one** redacted JSON report (default
  `./openthaiid-qualify.json`) and prints `report: <absolute path>`.
- Windows clinic wrappers: `Qualify-OpenThaiID.bat` /
  `Qualify-OpenThaiID.ps1`, plus `scripts/hardware-qualify.ps1`.
  `scripts/hardware-qualify.sh --operator`.

### Changed

- JPEG SOI bytes, `FFD8FF`, and PNG Base64 prefixes are refused in qualify
  output in addition to the existing citizen-ID / synthetic / `data:image`
  blockers.
- Windows/Linux START-HERE document Friday `qualify`; Mac START-HERE keeps
  `make package-macos` accuracy and the same qualify command.

### Added (clinic-prep)

- Friday hardware qualification kit: `docs/hardware-qualification.md`,
  `openthaiid qualify`, `scripts/hardware-qualify.sh`, `make qualify-smoke`.
  Reports are pass/fail + timing + error codes + technician diagnostics.
  Citizen ID, names, address text, and portrait bytes are never stored;
  the writer refuses output that contains those payloads.
- Clinic / vanilla example portrait **opt-in**: default `request` stays
  without `portrait`. The example UI can show an `<img>` preview when the
  integrator ticks portrait. Production grant is unchanged unless requested.
- `make package-macos` and a hard error from `scripts/package.sh` when
  `OS=darwin` is set on a non-Mac host (`ebfe/scard` + Cocoa cgo). Mac
  START-HERE documents the unsigned `.app` and `start-openthaiid.sh`.

### Changed

- Technician docs no longer call portrait “out of scope”; it is optional
  and off by default.
- macOS `Info.plist` version is injected from the package `VERSION`.

## [0.1.0-rc.1] — software release candidate (unreleased tag)

Software RC against `dev`. **Do not enable GitHub Actions** on this change
(minutes exhausted). Tag `v0.1.0-rc.1` only after human review. Physical
ACR39U qualification (E1–E7) and Apple/Microsoft code signing are **not**
in this candidate.

### Added

- Optional `portrait` field (cited 20×255-byte JPEG parts at `0x017B`,
  Photo_Part1/20 … Photo_Part20/20). Protocol minor 3. Default grant is
  unchanged. SDK: `portraitDataUrl()` / `portraitBlob()`; autofill can set
  `<img src>` from a data URL. Synthetic fixture is a clearly fictional
  16×16 checkerboard JPEG (`FICTIONAL-OPENTHAIID-PORTRAIT`). Photo bytes
  are omitted from logs and acceptance reports.
- Official Firefox support in the Playwright harness
  (`OPENTHAIID_BROWSER=firefox`). Linux CI/cloud lane uses Firefox instead
  of Chrome **without** adding matrix rows. The harness serves unbundled
  SDK ESM (`autofill.js`, `probe.js`) as `text/javascript` (Firefox rejects
  empty MIME) and sets `network.websocket.allowInsecureFromHTTPS` for
  loopback `ws://` from the HTTPS fixture. Safari: Playwright `webkit` on
  macOS (`OPENTHAIID_BROWSER=webkit`); real Safari UI is macOS-only and not
  in CI.
- `openthaiid update check|download|apply` against GitHub Releases (version
  compare, unsigned asset download, SHA256SUMS verify on apply). Tray:
  Check for updates / Open GitHub Releases. Apply requires `--yes`.
- `docs/signing.md` checklist for future Authenticode / Developer ID /
  notarization. This tree still ships **unsigned** packages.

### Changed

- `go.mod` `go` directive is **1.27** (toolchain `go1.27.1`, matching the
  existing CI pin).
- Helper version string defaults to `0.1.0-rc.1`.

### Documented, not implemented

- Laser-id and NHSO / other DFs: not in the cited APDU.md revision.
  Community source offsets are recorded in `docs/provenance.md` as blocked.

### Security

- Auto-update is the only operator-initiated outbound call (GitHub API +
  release assets). No card fields, tokens, or photo bytes are sent.
- Portrait Base64 is redacted from acceptance reports (`[portrait-omitted]`).

### Not claimed

- Physical ACR39U acceptance
- Signed / notarized store installers
- A published GitHub Release until a human tags this RC

## [0.0.1-dev] — previous development line

### Added

- Clinic-staff vs technician split on the local helper page: default view is
  helper / reader / card in plain language; **Technician / developer** tools
  (refresh, reconnect, pending/approved origins, copy diagnostics, verbose
  logs) stay hidden (`#technician`)
- Background helper UX: `openthaiid tray` (Windows/macOS notification-area
  menu: Open Ready page, Open Technician tools, Show status without PII,
  opt-in Start at login, Quit). Linux runs the same command in the background
  and uses the Ready page / `.desktop` actions when there is no tray
- Opt-in start at login: `openthaiid autostart enable|disable|status`
  (Windows Startup folder, macOS LaunchAgent, Linux XDG autostart). Off by
  default; login does not open a browser
- Clinic packaging: `scripts/package.sh` ships a runnable folder (`openthaiid`
  / `openthaiid.exe`, START-HERE.txt, Windows bat/ps1, Linux desktop/service,
  unsigned macOS `.app`)
- `openthaiid status` / `quit` / `open` for staff-safe control without a TTY
- Loopback `GET /v1/helper/summary` (no personal fields) and
  `POST /v1/helper/shutdown` (foreign `Origin` rejected)
- `docs/user.md` (clinic staff happy path) and `docs/technician.md` (Fill
  failure checklist)
- Manager APIs `POST /v1/manager/reconnect` and `POST /v1/manager/log-level`
  (CSRF + manager origin; verbose logging is not persisted)
- Status diagnostics: OS, log level, ERP connection count, last operational
  errors (codes/messages only — no personal card field values)
- `@openthaiid/web` `summarizeProbe()` and `mountThaiIDStatus({ mode })`
  (`user` default, `advanced` for technicians)
- Local loopback WebSocket bridge (`openthaiid.v1`) on `127.0.0.1:17891`
- `@openthaiid/web` library helpers `probeThaiID()` and `mountThaiIDStatus()`
  so an existing ERP page can check helper/PC/SC/readers without installing
  a separate OpenThaiID app (no personal field values)
- Protocol `bridge.probe` (minor 2): unauthenticated metadata for the library
- Loopback operator page (`GET /`) for Allow/Deny when a TTY is unavailable;
  CSRF-gated `/v1/manager/*` (no CORS)
- Clinic docs: add the JS library to the ERP; no OpenThaiID stack to deploy
  (`docs/clinic.md`)
- CLI: `openthaiid`, `openthaiid doctor`, `openthaiid version`
- Terminal origin/field approval (`allow` / `deny` / `revoke` / `revoke-all`)
- PC/SC backend for Windows, Linux, macOS (`ebfe/scard` pinned)
- Synthetic backend for CI and hardware-free tests, including a seeded
  fictional demo card readable by `openthaiid -synthetic`
- Thai citizen profile textual fields: `citizenId`, `thaiName`, `englishName`,
  `birthDate`, `gender`, `issuer`, `issueDate`, `expireDate`, `address`
  (portrait/photo remains out of scope; default grant stays the original three)
- Browser SDK `@openthaiid/web` with cancellation, session renewal, and reconnect
- React hooks `@openthaiid/react` wrapping `@openthaiid/web` (connect/read/disconnect,
  approvalRequired, status, fields, AbortSignal)
- Vanilla example page (external module script, CSP-friendly)
- `@openthaiid/web` `autofillThaiID()` helper: one user-gesture call that
  connects, waits for terminal approval, reads, maps structured fields to
  input strings, fills `[data-openthaiid]` / selector targets, then
  disconnects (existing `ThaiID` class API unchanged)
- Vanilla form example (`examples/vanilla/autofill.html`) with a single
  “Autofill from Thai ID” button
- Clinic ERP example (`examples/clinic-patient/`): New patient form plus a
  developer settings object (`request` allowlist, `map` of slots to
  selectors, format tweaks, `onApprovalRequired` / `onError`)
- `autofillThaiID` `request` (field allowlist), `map` (alias of `fields`),
  and `onError` chrome hook; existing selector / `data-openthaiid` calls
  stay unchanged
- `@openthaiid/react` `useThaiIDAutofill(config)` thin wrapper around the
  same settings object (does not connect on mount)
- Playwright browser acceptance harness (`acceptance/`) against the real
  bridge + synthetic card, using node-pty for real terminal approval (no bypass)
- Cross-platform acceptance: Playwright `chrome` / `msedge` channels,
  approval policies (allow / deny / unanswered), allowlisted reports
- Allow-policy scenario that requests extra textual fields (`englishName`,
  `gender`, `issuer`, `issueDate`, `expireDate`, `address`) via a new
  terminal grant; default pages still request the original three fields
- `session.open` with a field set that is not a subset of the live grant
  prompts for a new terminal approval (card.read of ungranted fields stays
  `FIELD_NOT_ALLOWED`)
- Playwright two-browser-context isolation (H11): a card read is unicast
  to the requester; the idle peer page and WebSocket frames stay free of
  personal fields; the peer can still read on its own connection
- Go integration coverage that two authenticated WebSocket clients never
  share `card.read` results or session tokens
- Pairing-only handshake: unauthenticated connections receive no reader
  inventory or card events until `session.open` succeeds
- Playwright bridge process restart (H9): personal data is cleared, reconnect
  does not auto-read, a fresh read works against the new process
- Acceptance harness hard deadline (`process.exit` after teardown): Windows
  Chrome/Edge no longer hang on `https.close` keep-alive / ConPTY handles
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
- Hard `timeout-minutes` on every CI job and long step (Go test 35m, builds 15m,
  SDK 10m, acceptance 25m / 5m per policy) plus `concurrency: cancel-in-progress`
  so a new SHA does not leave hung Windows jobs occupying runners for 6h

### Changed

- Helper page and ERP status default copy is clinic-staff language (helper /
  reader / card). Protocol details live under Technician / developer.
- CLI startup points staff at the simple helper page; `-log-level` default
  stays `info`
- GitHub Actions CI is split to conserve included minutes: **pull requests**
  run 4 ubuntu jobs (Go test/race with bridge/reader `-count=5`, linux/amd64
  build, Web + React SDKs, Chrome allow/deny/unanswered). The **full** 11-job matrix
  (five-target build, Chrome × Linux/macOS/Windows + Edge × Windows,
  `-count=50`) runs on push to `dev`/`main` and `workflow_dispatch`. Docs-only
  changes skip the workflow. Local `make test-repeat` stays `-count=50`.
  The PR SDK job now also typechecks, tests, and builds `@openthaiid/react`
  on the same ubuntu runner (no extra OS matrix rows).

### Changed (protocol / card — pre-release corrections)

- READ BINARY command form corrected to the cited APDU table:
  `CLA=80 INS=B0 P1 P2 Lc=02 Data=00 Le=len`
- Thai name is one `#`-delimited 100-byte block at `0x0011` (not four 100-byte
  blocks)
- Birth date offset is `0x00D9` (not `0x0331`)
- Protocol minor 1 advertises remaining textual fields in `fieldsSupported`
- Requested unavailable fields are explicit JSON `null` with warnings;
  unrequested fields are omitted
- Oversized APDU responses are rejected instead of truncated
- Thai name decoding uses `golang.org/x/text` Windows-874 (not a hand map)
- Provenance ledger records immutable source revision
  `4d803243715254cc93efe9040d8b1ecd85f520aa`

### Fixed

- SDK `connect({ signal })` reports `CANCELLED` when the AbortSignal is already
  aborted, even if `WebSocket` is unavailable (Node/SSR). Abort during connect
  also stays `CANCELLED` rather than racing `CONNECTION_FAILED`.

- Sessions are capped at the live grant expiry; an expired grant invalidates
  an otherwise unexpired token
- Terminal `revoke <origin>` canonicalizes the origin (mixed-case hosts match
  the handshake grant)
- Windows acceptance: `https.Server.close()` waited forever for Chromium
  keep-alive sockets; teardown now drops connections and `process.exit`s
- Acceptance `/healthz` polls abort per attempt so a hung TCP cannot stall
  startup; H9 fails fast if the previous bridge is still serving after stop
- CI jobs no longer inherit GitHub's 6h default: Go test/race/build/SDK now
  have job- and step-level timeouts; `go test -timeout` is explicit so a
  deadlock dumps running tests instead of sitting until the runner is cancelled
- Approvals are bound to their originating connection and cancelled on disconnect
- Trust revocation delivers `authorization.revoked` and closes affected clients
- Duplicate / non-increasing request IDs are rejected
- Unauthenticated clients no longer receive reader inventory in a
  pre-auth `status.snapshot` (pairing-only until `session.open` succeeds)
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
- Official Firefox/Safari support (see `[0.1.0-rc.1]`)
- Portraits, signed store installers, auto-update (portrait + unsigned
  auto-update: see `[0.1.0-rc.1]`; signing still blocked)
