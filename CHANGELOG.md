# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-rc.8] — software release candidate (unreleased tag)

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Physical E1–E7 and
Apple/Microsoft code signing are **not** in this candidate. Tag
`v0.1.0-rc.8` only after human review. Do not retag `v0.1.0-rc.7`.

Summarizes merges since `[0.1.0-rc.7]`: Friday NOT_DONE playbook (#37),
PC/SC bind-stress harness (#38). Earlier technician `openthaiid doctor`
desk snapshot stays in `[0.1.0-rc.7]` (not repeated here).

### Added

- Binding-audit stress helper: `openthaiid bind-stress`,
  `scripts/bind-stress.sh` / `.ps1`, `make bind-stress-smoke`. Repeated
  context connect/disconnect, reader list, presence probe, and synthetic
  card present/absent. Redacted JSON
  (`openthaiid.bindstress.v1`) refuses PII and raw APDUs. Synthetic smoke
  does **not** close P2–P10. Friday: omit `-synthetic` on each host.
  [docs/bind-stress.md](docs/bind-stress.md). smoke-package asserts those
  bind-stress links.

### Changed

- Helper version string defaults to `0.1.0-rc.8`.
- Friday playbook: [NOT_DONE.md](NOT_DONE.md) is the ACR39U morning split
  (done in software / do Friday with hardware / blocked APDU / blocked
  certs / paused Actions), not a software backlog. Technician/qualify
  docs point at the bind-stress harness so P2–P10 is run-the-script, not
  invent-the-script.
  [docs/hardware-qualification.md](docs/hardware-qualification.md),
  [docs/technician.md](docs/technician.md). smoke-package asserts the
  playbook headings.

### Security

- Bind-stress reports never store citizen ID, names, address, photo
  bytes, or raw APDUs. The writer refuses those payloads.

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- A published GitHub Release until a human tags this RC
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)
- Binding audit P2–P10 closed (synthetic smoke is not a close)

## [0.1.0-rc.7] — software release candidate

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Physical E1–E7 and
Apple/Microsoft code signing are **not** in this candidate. Tagged
`v0.1.0-rc.7`. Do not retag `v0.1.0-rc.6`.

Summarizes merges since `[0.1.0-rc.6]`: technician `openthaiid doctor`
desk snapshot (#35). Earlier Go module hygiene / Friday staff desk card
stay in `[0.1.0-rc.6]` (not repeated here).

### Added

- `openthaiid doctor` technician **desk snapshot** (pre-hardware): helper
  version, HTTP/TLS listen ports, PC/SC reader list / selected reader,
  `bridge.probe`, Ready URL, TLS yes/no (**no** private-key paths), and the
  last GitHub version-check if one already ran (otherwise `not checked`,
  offline-safe). Next step is Thai + English. Exit 1 only when the helper is
  down or PC/SC is unavailable; soft update-check failures stay soft. Loopback
  `GET /v1/helper/doctor` (same Origin rule as summary; does not call GitHub)
  and Technician tools **Desk snapshot**. Tests cover redaction and exit codes
  with fakes. `make friday-dry-run` runs the snapshot and refuses PII.
  Pointers: [technician.md](docs/technician.md),
  [friday-checklist.md](docs/friday-checklist.md),
  [getting-started.md](docs/getting-started.md). smoke-package asserts
  those three links.

### Changed

- Helper version string defaults to `0.1.0-rc.7`.

### Security

- `openthaiid doctor` / `GET /v1/helper/doctor` never print citizen ID,
  names, address, photo bytes, tokens, or TLS private-key paths. Foreign
  `Origin` is rejected. The endpoint does not call GitHub.

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)

## [0.1.0-rc.6] — software release candidate

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Physical E1–E7 and Apple/Microsoft
code signing are **not** in this candidate. Tagged `v0.1.0-rc.6`.

Summarizes merges since `[0.1.0-rc.5]`: Go module hygiene (#32), Friday
staff desk card (#33). Earlier version check / Firefox/Safari honesty /
loopback helper TLS stay in `[0.1.0-rc.5]` (not repeated here).

### Added

- Staff-only Friday morning **printable desk card** (Thai-first, one page):
  [docs/friday-desk-card.md](docs/friday-desk-card.md) with a print-ready
  sibling [friday-desk-card.html](docs/friday-desk-card.html) (large type,
  A4). Plug reader → open Ready → **ทดสอบการเชื่อมต่อ** → open ERP → Fill
  → if stuck call a technician. Tiny tips for card-removed, sleep/wake,
  and https ERP TLS (ask tech — do not list browser options). Linked from
  getting-started, friday-checklist, training, user, START-HERE,
  hardware-qualification, clinic, installation, and friday-dry-run.
  Qualify / diagnostics stay on
  [friday-checklist.md](docs/friday-checklist.md). No Ready
  print-stylesheet change.

### Changed

- Helper version string defaults to `0.1.0-rc.6`.
- Go module hygiene (`go mod tidy`) after bumping **direct** dependencies
  already on a compatible major. No public SDK API change.
  - `github.com/coder/websocket` `v1.8.12` → `v1.8.15` (1.8.x: handshake
    after close, Origin scheme matching, single-frame compression, fewer
    per-frame allocs). LICENSE file is now Coder ISC (was MIT / Anmol
    Sethi); `THIRD_PARTY_NOTICES.md` matches.
  - `golang.org/x/sys` `v0.34.0` → `v0.48.0` (Windows `svc` / event log)
  - `golang.org/x/text` `v0.21.0` → `v0.42.0` (Windows-874 decoder only)

### Held

- `go` **1.27** / toolchain **go1.27.1** — current stable (`go.dev/dl`
  has no `go1.27.2`). CI pin stays `1.27.1`. Not a Go major bump.
- `github.com/ebfe/scard` remains
  `v0.0.0-20241214075232-7af069cabc25` (`7af069cabc25`). No newer
  tagged revision; binding audit P2–P6 / P10 still open. Do not switch
  PC/SC libraries.
- GitHub Actions stay disabled (minutes exhausted).
- No Laser-id / NHSO. No extra Go modules.

### Security

- `govulncheck ./...` reports no known vulnerabilities on this module
  graph (Go 1.27.1 toolchain). Desk card and friday-dry-run never print
  citizen ID, names, address, or photo bytes.

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)

## [0.1.0-rc.5] — software release candidate (unreleased tag)

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Tag `v0.1.0-rc.5` only after
human review. Physical E1–E7 and Apple/Microsoft code signing are **not**
in this candidate.

Summarizes merges since `[0.1.0-rc.4]`: operator GitHub Releases version
check (#28), Firefox/Safari clinic Fill honesty (#29), optional loopback
helper TLS (#30). Earlier getting-started hub / Ready a11y / friday-dry-run
/ helper HTTP hardening stay in `[0.1.0-rc.4]` (not repeated here).

### Added

- Operator-initiated GitHub Releases **version check** (no auto-download or
  install): Ready page **ตรวจเวอร์ชัน**, technician **Check GitHub Releases**,
  tray **ตรวจเวอร์ชัน**, and `openthaiid update check`. Compares the running
  helper to `night-sornram/openthaiid` (override `OPENTHAIID_UPDATE_REPO`).
  Shows up to date vs a newer prerelease/release plus the release URL.
  Offline and GitHub rate limits fail soft — Ready / Fill never wait on
  GitHub. `POST /v1/manager/update/check` is CSRF + manager origin, not
  called on page load or `GET /v1/helper/summary`.
- Firefox / Safari clinic Fill/Ready honesty: `@openthaiid/web` reports
  `HTTPS_WS_BLOCKED` when an **https://** ERP cannot open `ws://127.0.0.1`
  (Firefox/Safari mixed content) and helper TLS is not trusted/enabled.
  Ready (`http://127.0.0.1:17891/`) still works. No bundled TLS private key
  and no “turn off mixed content” staff copy.
  Ready page clipboard falls back to `execCommand` / prompt; `replaceChildren`
  is guarded. Matrix: [docs/compatibility.md](docs/compatibility.md).
- Optional loopback **HTTPS + WSS** on `127.0.0.1:17892` (technician-managed
  cert + key; never a shared private key). Plain HTTP/WS on 17891 stays the
  default for Ready and http ERP. `@openthaiid/web` prefers
  `wss://127.0.0.1` on https Firefox/Safari pages and when
  `preferLoopbackTls` / helper TLS advertisement is set. Untrusted certs
  surface as `TLS_UNTRUSTED`. Docs: [docs/loopback-tls.md](docs/loopback-tls.md).

### Changed

- Helper version string defaults to `0.1.0-rc.5`.
- Getting-started, compatibility, loopback-tls, Friday checklist, training,
  START-HERE, architecture, protocol, and security docs cross-link the
  Friday hub, Firefox/Safari Fill matrix, and optional helper TLS.
- `openthaiid update check` lists `/releases` (not `/releases/latest`) so
  prereleases `v0.1.0-rc.1`–`rc.5` count unless `--stable`. GitHub HTTP 404
  (private repo, unauthenticated) is **unavailable**, not “no releases”.
  Optional `OPENTHAIID_GITHUB_TOKEN` / `GITHUB_TOKEN` on the helper.
- Check copy is Thai for staff (and English fallback). It does not tell
  clinics to `update apply`. Existing `update download` / `update apply --yes`
  stay technician CLI only.

### Security

- Version check is operator-initiated only (Ready button, tray, technician
  tools, CLI). It never runs on Ready load or Fill. No card fields are sent
  to GitHub. Optional helper TLS never ships a private key; certs stay on
  the clinic PC ([docs/loopback-tls.md](docs/loopback-tls.md)).

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- A published GitHub Release until a human tags this RC
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)

## [0.1.0-rc.4] — software release candidate (unreleased tag)

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Tag `v0.1.0-rc.4` only after
human review. Physical E1–E7 and Apple/Microsoft code signing are **not**
in this candidate.

Summarizes merges since `[0.1.0-rc.3]`: getting-started hub + Ready a11y
(#24), Friday synthetic dry-run (#25), helper HTTP hardening (#26). Earlier
clinic-day reconnect / multi-reader / card-removed / sleep-wake waves stay
in `[0.1.0-rc.3]` (not repeated here).

### Added

- Thai-first Friday-morning hub: [getting-started.md](docs/getting-started.md)
  indexes clinic staff, training, Friday checklist, ERP drop-in, and
  technician guides in one place.
- Ready page accessibility for clinic staff: **ใช้เครื่องนี้**,
  **ทดสอบการเชื่อมต่อ**, and the ไทย / English toggle are at least 44px tap
  targets; `:focus-visible` rings; the three lights include **ปกติ / รอ /
  ยังไม่พร้อม** text (not color alone).
- Friday-morning **ซ้อมก่อนได้เครื่อง** rehearsal: `make friday-dry-run`
  (`scripts/friday-dry-run.sh`, Windows `scripts/friday-dry-run.ps1`) builds
  the helper if needed, runs qualify-smoke, starts `-synthetic` (optional
  two practice readers), curls Ready / `GET /v1/helper/summary` and the
  Test connection path (`GET /v1/manager/status` + `bridge.probe`), and
  prints a Thai/English pass summary pointing at
  [getting-started.md](docs/getting-started.md) and
  [friday-checklist.md](docs/friday-checklist.md). Never prints citizen ID,
  names, address, or photo bytes. Not an ACR39U hardware pass.

### Changed

- Helper version string defaults to `0.1.0-rc.4`.
- Getting-started, training, Friday checklist, hardware-qualification,
  technician, and security docs cross-link the hub, `make friday-dry-run`,
  and the loopback helper HTTP inventory.

### Security

- Loopback helper/manager HTTP: foreign `Origin` and `Sec-Fetch-Site:
  cross-site` cannot read summary or mutate shutdown, reader select,
  reconnect, audit export, or other `/v1/manager/*` APIs. Browser shutdown
  also requires the CSRF header (`openthaiid quit` still sends no Origin).
  Refresh/reconnect are bounded (one in flight, four per two seconds).
  Last-error JSON drops forbidden payloads. Go tests cover Origin/CSRF/PII.
  Inventory in [docs/security.md](docs/security.md); technician notes in
  [docs/technician.md](docs/technician.md).

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- A published GitHub Release until a human tags this RC
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)

## [0.1.0-rc.3] — software release candidate (unreleased tag)

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Tag `v0.1.0-rc.3` only after
human review. Physical E1–E7 and Apple/Microsoft code signing are **not**
in this candidate.

Summarizes merges since `[0.1.0-rc.2]`: clinic-day reconnect +
test-connection (#19), multi-reader select (#20), card-removed /
reader-unplug mid-Fill recovery (#21), sleep/wake + PC/SC recovery UX
(#22). Earlier qualify / Thai UI / ERP schema / audit / serve waves stay
in `[0.1.0-rc.2]` (not repeated here).

### Added

- Clinic-day reconnect UX: `mountThaiIDStatus` / `watchThaiID` auto-detect
  the helper coming back after a restart, with Thai staff copy
  (**ตัวช่วยหลุดการเชื่อมต่อ** / **ตัวช่วยกลับมาแล้ว**). Faster poll while
  waiting. `waitForThaiID()` is the same loop without DOM.
- Ready page **ทดสอบการเชื่อมต่อ** (and the SDK status widget button):
  `bridge.probe` + status only, no personal fields.
- `autofillThaiID` waits for the helper before `session.open`. It still
  never auto-repeats a card read.
- Multi-reader desks: Ready page lists PC/SC readers in Thai and lets staff
  pick **ใช้เครื่องนี้**. Technician tools keep English **Use this reader**
  plus opaque `readerId`. One reader stays auto-selected.
- Protocol minor 4: `bridge.probe` / `bridge.status` / `status.snapshot`
  mark `readers[].selected`. `card.read` without `readerId` uses that
  reader; two readers with no choice return `READER_SELECTION_REQUIRED`.
- Operator audit may record `readerId` and `reader.selected` (device id,
  never card values). `OPENTHAIID_SYNTHETIC_READERS=2` seeds a second
  practice reader so the picker can be exercised without hardware.
- Mid-read recovery: pulling the card or unplugging the reader during Fill
  returns promptly with stable `CARD_REMOVED` / `READER_REMOVED` (or
  `READ_FAILED` when native I/O cannot tell). Thai staff copy tells them
  what happened, to reinsert, wait for **พร้อม**, and Fill again.
  Ready / `watchThaiID` poll faster while Ready so **พร้อม** cannot linger
  after `card.removed` / reader gone. `autofillThaiID` clears mapped
  sensitive fields on those aborts (phone / HN stay). Technician English
  remains on `{ lang: "en" }` and qualify prompts. Friday E4 maps to these
  codes in [friday-checklist.md](docs/friday-checklist.md),
  [technician.md](docs/technician.md), and
  [user.md](docs/user.md).
- Sleep/wake and PC/SC recovery UX: Ready re-probes on `pageshow` /
  visibility / `online` (`bridge.probe` + status, never `card.read`).
  Helper down stays `BRIDGE_UNAVAILABLE`; card service down is
  `PCSC_UNAVAILABLE` with Thai “do not reboot / Test connection /
  Reconnect card service / wait for **พร้อม**”. `watchThaiID` and Ready
  detect backend unavailable → restored and say **พร้อม** again when the
  card is seated. Synthetic `SetServiceDown` covers that path without
  simulating OS sleep. Friday E5/E6 mapped in the same docs.

### Changed

- Helper version string defaults to `0.1.0-rc.3`.
- Friday checklist, technician, and staff user docs cross-link
  multi-reader selection and card-removed / sleep-wake / PC/SC recovery.

### Fixed

- SDK reconnect after helper restart: clear the previous `instanceId` so
  the new helper snapshot is accepted, and reset the reconnect budget after
  a successful recovery.
- Ready page recovers from a helper restart without a manual reload (CSRF
  token refresh + offline lights while waiting).

### Security

- Wake / **ทดสอบการเชื่อมต่อ** stay `bridge.probe` + status only — never
  `card.read`, citizen ID, names, address, or portrait.
- Operator audit `readerId` / `reader.selected` are device ids, never card
  values.
- Auto-update remains the only operator-initiated outbound call (GitHub
  API + unsigned release assets). No card fields are sent.

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- A published GitHub Release until a human tags this RC
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)

## [0.1.0-rc.2] — software release candidate (unreleased tag)

Software RC against `dev` while ACR39U hardware is in transit. **Do not
enable GitHub Actions** (minutes exhausted). Tag `v0.1.0-rc.2` only after
human review. Physical E1–E7 and Apple/Microsoft code signing are **not**
in this candidate.

Summarizes merges since `[0.1.0-rc.1]`: Friday qualify kit (#13–#14), Thai
staff UI (#14), Thai Fill errors + training + Friday runbook (#15), ERP
autofill JSON schema (#16), operator audit trail + always-on serve/service
(#17).

### Added

- Friday hardware qualification kit: [docs/hardware-qualification.md](docs/hardware-qualification.md),
  `openthaiid qualify`, `scripts/hardware-qualify.sh`, `make qualify-smoke`.
  Reports are pass/fail + timing + error codes + technician diagnostics.
  Citizen ID, names, address text, and portrait bytes are never stored;
  the writer refuses output that contains those payloads.
- `openthaiid qualify -operator` records E4–E7 / M1 as pass/fail/skip from
  the keyboard (those words only; pasted card data is refused). Hardware
  mode prints the bilingual operator prompts even without `-operator`.
  Hardware E3 can be driven with Enter (remove / seat) instead of only
  `-synthetic`. Always writes **one** redacted JSON report (default
  `./openthaiid-qualify.json`) and prints `report: <absolute path>`.
- Windows clinic wrappers: `Qualify-OpenThaiID.bat` /
  `Qualify-OpenThaiID.ps1`, plus `scripts/hardware-qualify.ps1`.
  `scripts/hardware-qualify.sh --operator`.
- Clinic-staff Ready copy defaults to **Thai** (พร้อม / ยังไม่พร้อม) on the
  helper page, `mountThaiIDStatus` / `summarizeProbe`, tray **แสดงสถานะ**,
  and `openthaiid status`. English fallback: Ready page ไทย/English toggle,
  `{ lang: "en" }` in the SDK, or `OPENTHAIID_LANG=en`. Technician / debug
  chrome stays English (bilingual toggle label). Staff user doc is Thai-first.
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
- `sdk/web/schema/autofill-clinic-config.schema.json` (export
  `@openthaiid/web/schema/autofill-clinic-config.json`), TypeScript
  `AutofillClinicConfig`, and `parseAutofillClinicConfig` /
  `serializeAutofillClinicConfig`. Callbacks stay code-only. Samples:
  `examples/clinic-patient/kyc-minimal.json` (3-field KYC) and
  `clinic-full.json`. Drop-in steps (Thai + English):
  [docs/erp-integration.md](docs/erp-integration.md).
- Clinic example settings panel **download / upload** of that JSON (plus
  load-sample buttons) so integrators can iterate without editing code.
- Redacted **operator audit trail** for technicians: local ring +
  `audit.jsonl` of helper start/stop, reader/card presence, approval
  allow/deny/revoke, and Fill failures by **error code only**. Never stores
  citizen ID, names, address, portrait, or raw APDUs. Technician UI
  (Copy JSON / Download JSON), `GET /v1/manager/audit`, and
  `openthaiid audit`. See [docs/technician.md](docs/technician.md).
- Always-on helper without a logged-in tray: `openthaiid serve` plus
  `openthaiid service install|uninstall|start|stop|status`. Windows SCM
  (`Install-OpenThaiID-Service.ps1`, Administrator). Linux systemd --user
  unit + linger (`loginctl enable-linger`); system-unit template
  `openthaiid-system.service`. Task Scheduler fallback
  `Register-OpenThaiID-Task.ps1`. Docs: [installation.md](docs/installation.md).
- Clinic / vanilla example portrait **opt-in**: default `request` stays
  without `portrait`. The example UI can show an `<img>` preview when the
  integrator ticks portrait. Production grant is unchanged unless requested.
- `make package-macos` and a hard error from `scripts/package.sh` when
  `OS=darwin` is set on a non-Mac host (`ebfe/scard` + Cocoa cgo). Mac
  START-HERE documents the unsigned `.app` and `start-openthaiid.sh`.

### Changed

- Helper version string defaults to `0.1.0-rc.2`.
- Practice-mode Ready copy no longer hides a pending Allow prompt.
- Clinic example staff chrome is Thai-first (`?lang=en` fallback).
- JPEG SOI bytes, `FFD8FF`, and PNG Base64 prefixes are refused in qualify
  output in addition to the existing citizen-ID / synthetic / `data:image`
  blockers.
- Package START-HERE / Friday checklist / technician docs cross-link
  qualify, redacted audit JSON, and `openthaiid service install`.
- Technician docs no longer call portrait “out of scope”; it is optional
  and off by default.
- macOS `Info.plist` version is injected from the package `VERSION`.

### Security

- Operator audit never stores citizen ID, names, address, portrait, raw
  APDUs, or Fill error *messages*. Forbidden payloads are dropped rather
  than written.
- Auto-update remains the only operator-initiated outbound call (GitHub
  API + unsigned release assets). No card fields are sent.

### Not claimed

- Physical ACR39U acceptance (E1–E7 / M1)
- Signed / notarized store installers
- A published GitHub Release until a human tags this RC
- Laser-id / NHSO / other DFs (not in the cited APDU.md revision)

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
