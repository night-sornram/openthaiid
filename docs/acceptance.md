# MVP acceptance matrix

This matrix is the gate for calling an MVP “complete.” Synthetic CI evidence
never substitutes for physical-card evidence where hardware is claimed.

Status values:
- **Done** — local evidence on this machine
- **CI ready** — code present; needs a green GitHub Actions run
- **Open** — hardware, target environments, or operator action required

Record for each claim: commit, platform, browser version, command, result,
and evidence location.

## GitHub Actions lanes

Included Actions minutes are billed with OS multipliers (Linux 1×, Windows
~2×, macOS ~10×). Every PR push used to run the full 11-job matrix; macOS
builds/acceptance and hung Windows jobs dominated the pool.

| Lane | When | What runs |
|---|---|---|
| **PR** | `pull_request` (code changes) | Ubuntu: Go fmt/vet/test/race, bridge/reader `-count=5`, linux/amd64 build + version smoke, Web + React SDKs, **Firefox** acceptance (allow/deny/unanswered). **4 jobs.** |
| **Full** | `push` to `main`/`dev`, `workflow_dispatch` | PR jobs with `-count=50`, plus linux/arm64, windows/amd64, darwin/arm64, darwin/amd64 builds; Chrome on macOS and Windows; Edge on Windows; Linux stays Firefox. **11 jobs.** |

Docs-only / `.cursor/`-only changes skip the workflow (`paths-ignore`).
In-progress runs still cancel on a new SHA. Local `make test`,
`make test-repeat` (`-count=50`), and `make acceptance` are unchanged.

A calendar nightly is not enabled: GitHub `schedule` workflows run from the
default branch (`main`), which is still the initial commit. Dispatch this
workflow on `dev` for an on-demand full matrix.

## Build artifacts

| # | Criterion | Evidence | Status |
|---|---|---|---|
| A1 | All five OS/arch artifacts build and start | Full-lane CI matrix + version smoke (PRs build linux/amd64) | Done (PR #1 run 35054867099); re-confirm on full lane |
| A2 | `openthaiid doctor` reports PC/SC status without reading personal fields | CLI output | Done |
| A3 | `openthaiid version` prints module version (injected at release build) | CLI + package.sh verify | Done |

## Protocol and authorization

| # | Criterion | Evidence | Status |
|---|---|---|---|
| B1 | Unapproved origin receives no reader inventory, events, or card data | Pairing-only: no pre-auth snapshot; `TestIntegrationPairingOnlyNoInventory` | Done |
| B2 | Missing/null Origin, lookalikes, wrong ports, rebinding-style Host rejected | Auth origin tests | Done |
| B3 | Approval, denial, expiry, revocation behave as specified | Auth unit + integration | Done |
| B4 | Field-set expansion requires a new approval | Auth tests + `TestIntegrationSessionOpenFieldExpansionPrompts`; Playwright extra-fields allow scenario | Done |
| B5 | Pending approval cannot be claimed by another connection | Connection-bound approvals | Done |
| B6 | Tokens fail across origins and bridge restarts | Auth tests | Done |
| B7 | Grant expiry/revocation during a read suppresses the result | Auth Session() + recheck | Done |
| B8 | Competing clients never receive each other’s results | Unicast write path + two-connection integration test | Done |
| B9 | Eight concurrent async ops per connection; excess is `RATE_LIMITED` | Lifecycle saturation test | Done |
| B10 | Event-queue overflow closes with WS 1013 and requires a fresh snapshot | Bridge enqueue + overflow path | Done (code); browser reconnect case CI ready |

## Reading

| # | Criterion | Evidence | Status |
|---|---|---|---|
| C1 | `citizenId`, `thaiName`, `birthDate` plus remaining textual fields and optional `portrait` | Parser fixtures + APDU byte tests + `TestEngineTextualFields` + `TestEnginePortraitJPEG` | Done (synthetic) |
| C2 | Field selection limits returned data and necessary reads | Integration + APDU script; extra textual fields omitted unless granted (`TestIntegrationUngrantedTextualFieldRejected`) | Done |
| C3 | Removal/replacement cannot produce stale results | Lifecycle tests | Done |
| C4 | Unsupported/malformed cards yield typed errors and monitoring continues | Synthetic + engine tests | Done |
| C5 | No personal-data logging (sentinel log tests) | Privacy tests | Done |
| C6 | Published APDU bytes/offsets match the cited table | `TestThaiProfileCommandPlanMatchesPublishedAPDU` | Done |
| C7 | BE→Gregorian dates, leap-year validation, null fields | Parser tests | Done |
| C8 | Synthetic CLI demonstration reads a fictional card | `TestSyntheticDemoReadAllFields` + `TestSyntheticDemoReadTextualFields` | Done |
| C9 | Exact synthetic field values (full Thai name + 1990-04-12) asserted | Seeded integration tests | Done |
| C10 | Native worker owns open→tx→read→end→close; exactly-once cleanup | Reader ownership tests | Done |
| C11 | Prompt `CANCELLED`/`READ_TIMEOUT`; slot retained; late results discarded | Reader + bridge cancel tests | Done |
| C12 | `BUSY` on overlapping reads (no queueing) | Overlap tests | Done |
| C13 | `BACKEND_STALLED` after 2s post-cancel; recovery after worker finishes | Stall + recovery tests | Done |
| C14 | Shutdown returns at deadline without releasing a context still in use | `TestStopRefusesReleaseWhileWorkerBlocked` | Done |
| C15 | Non-cooperative backend (blocked open/begin/transmit/cleanup) | Synthetic `BlockPhase` tests | Done |
| C16 | Formerly flaky integration cases ×50 | Full CI `go test -count=50`; PRs use `-count=5` | CI ready (PR + full lanes) |

## SDK and example

| # | Criterion | Evidence | Status |
|---|---|---|---|
| D1 | Connect → approve → read works through published SDK API | Playwright + synthetic | Done (local); multi-OS CI ready |
| D2 | Vanilla/acceptance page clears displayed data on removal/disconnect/revocation | Playwright disconnect scenario | Done |
| D3 | SSR import is safe; connect requires a browser | SDK unit tests | Done |
| D4 | Browser refresh within an active grant reuses trust | Playwright full-page-refresh scenario | Done |
| D5 | Cancellation cancels the real request id | SDK abort test | Done |
| D6 | Session renewal and reconnect backoff | SDK unit paths | Done |
| D7 | HTTPS page + CSP + loopback WS | Playwright harness (acceptance page only) | Done |
| D8 | Reconnect never automatically repeats a card read | Playwright reconnect scenario | Done |
| D9 | React hooks wrap connect/read/disconnect with AbortSignal, approvalRequired, status, fields | `@openthaiid/react` unit tests | Done |

## Browser qualification (automated)

Channels: Playwright `firefox` (Linux CI/cloud), `chrome` (macOS/Windows full
lane), `msedge` (Windows). `webkit` (Safari engine) is **local / macOS**.
Do not add Safari rows to GitHub Actions. Every GitHub Actions job has a hard
`timeout-minutes` (acceptance 25m, 5m per policy step); the harness also
exits via `process.exit` at 150s. A new push cancels in-progress runs.

PR CI runs H1 + H5–H7 (ubuntu **Firefox**, three policies). H2–H4 (macOS Chrome,
Windows Chrome, Windows Edge) run on the full lane only (merge to `dev`/`main`
or `workflow_dispatch`). The workflow may be disabled to conserve minutes;
run `make acceptance-firefox` locally.

| # | Criterion | Status |
|---|---|---|
| H1 | Stable Firefox on ubuntu-24.04 | CI ready (PR + full; workflow may be disabled) |
| H1b | Stable Chrome on ubuntu-24.04 | Local `OPENTHAIID_BROWSER=chrome` |
| H2 | Stable Chrome on macos-15 | CI ready (full lane) |
| H3 | Stable Chrome on windows-2025 | CI ready (full lane) |
| H4 | Stable Edge on windows-2025 | CI ready (full lane) |
| H15 | Playwright WebKit (Safari engine) on macOS | Local `make acceptance-webkit`; not in CI |
| H16 | Optional portrait on synthetic (allow policy; new grant; photo bytes redacted in reports) | Local / harness |
| H5 | Approval + read (allow policy) | CI ready |
| H6 | Terminal denial (deny policy) | CI ready |
| H7 | Approval stays pending (unanswered policy) | CI ready |
| H8 | Full-page refresh | CI ready |
| H9 | Bridge restart / fresh snapshot after new process | Done (local Playwright process restart); multi-OS CI ready |
| H10 | Disconnect clears fields | CI ready |
| H11 | Two-client isolation | Done (local Playwright two-context + Go unicast); multi-OS CI ready |
| H12 | Origin rejection independent of terminal denial | Done (Go `TestIntegrationBadOriginRejected`) |
| H13 | Connection failure on closed port (renamed scenario) | CI ready |
| H14 | Extra textual fields requested and asserted on synthetic (allow policy; new grant) | CI ready |

## Browser qualification (manual)

| # | Criterion | Status |
|---|---|---|
| M1 | Headed manual check from a **public HTTPS** test origin for local-network permission allow/deny | Open — operator supplies HTTPS origin; synthetic data; normal browser security settings; recorded separately from localhost automation |

## Hardware qualification (manual)

Operator kit (checklist + `openthaiid qualify`): [hardware-qualification.md](hardware-qualification.md).
The kit never stores citizen ID, names, address, or portrait bytes.

For each claimed configuration, record OS, arch, reader variant, driver,
bridge revision, and non-personal outcome only.

**Order:** macOS + ACR39U first, then Windows, then Linux.

| # | Criterion | Status |
|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | Open — kit ready |
| E2 | 100 successful repeated MVP reads | Open — kit ready |
| E3 | ≥20 insertion/removal cycles | Open — kit ready |
| E4 | Removal during reading and reader unplug | Open — software mapped (`CARD_REMOVED` / `READER_REMOVED` / `READ_FAILED`); physical ACR39U still required |
| E5 | Browser refresh, bridge restart, sleep/wake | Open — software mapped (`BRIDGE_UNAVAILABLE`, Ready re-probe, Test connection); physical ACR39U + sleep still required |
| E6 | Missing/stopped PC/SC service recovery where applicable | Open — software mapped (`PCSC_UNAVAILABLE` → Reconnect card service → พร้อม); synthetic unavailable→restore covered; physical PC/SC still required |
| E7 | Local visual verification without storing field values in reports | Open |

Available first target: macOS + ACR39U. Windows/Linux physical qualification
remains required for the original three-platform release claim. Untested
configurations stay explicitly unverified. Darwin clinic archives must be
built on a Mac (`make package-macos`); they are unsigned / not notarized.

## PC/SC binding audit (`ebfe/scard`)

Successful compilation alone does not close these checks.

| # | Check | Status |
|---|---|---|
| P1 | Pin to immutable git revision + license | Done |
| P2 | Native layouts / pointer lifetimes on darwin/arm64 | Open |
| P3 | Native layouts / pointer lifetimes on darwin/amd64 | Open |
| P4 | Native layouts / pointer lifetimes on linux/amd64 | Open |
| P5 | Native layouts / pointer lifetimes on linux/arm64 | Open |
| P6 | Windows WinSCard system-library loading | Open |
| P7 | Concurrent context and reader-state handling | Partial (separate monitor/read backends; stress open) |
| P8 | Returned buffer lengths from `Transmit` | Partial (profile MaxResponse bounds in engine) |
| P9 | Shutdown and blocked-call cleanup (`Cancel` unblocks `GetStatusChange`) | Partial (no context release while worker holds slot) |
| P10 | GC interaction with cgo pointers under stress | Open |

## Phase 0 / Phase 1 exit

| # | Criterion | Status |
|---|---|---|
| F1 | Architecture, threat model, protocol, provenance, compatibility docs committed | Done |
| F2 | Go module builds with CLI skeleton (`version`, `doctor`, default run) | Done |
| F3 | Backend/Card interfaces defined; synthetic backend implements them | Done |
| F4 | Real PC/SC adapter compiles on darwin with pinned `ebfe/scard` | Done (compile) |
| F5 | Five-target build matrix documented; CI workflow present | Done (full lane; PRs build linux/amd64) |
| F6 | Provenance ledger records MVP profile sources and independent implementation | Done |
| F7 | Dependency audit checklist present | Done |

## Software MVP exit

| # | Criterion | Status |
|---|---|---|
| G1 | Thai profile + APDU engine + Windows-874 parsers with unit tests | Done |
| G2 | Protocol v1 WebSocket dispatch (open/read/cancel/status/events) | Done |
| G3 | Origin approval, deny, revoke, field-subset enforcement | Done |
| G4 | Card-session generations; no stale results on removal | Done |
| G5 | SDK connect → approve → read | Done |
| G6 | Acceptance page (vanilla example retained) | Done |
| G7 | Privacy log sentinel test | Done |
| G8 | Release packaging script + third-party notices | Done |
| G9 | Responsive dispatch + real cancellation + duplicate-ID rejection | Done |
| G10 | Grant-lifetime and revocation enforcement | Done |
| G11 | PC/SC StateUnaware retention + recovery; stall + ownership | Done |
| G12 | Shared synthetic demo card end-to-end | Done |
| G13 | Green five-target CI with version smoke + npm ci | Done (PR #1); full lane after merge / dispatch |
| G14 | Playwright browser acceptance (Firefox on Linux; Chrome × macOS/Windows + Edge on Windows) | Code ready; run locally while Actions is disabled |
| G15 | Hardware qualification evidence | Open |
| G16 | node-pty cross-platform terminal approval harness | Done (local macOS); multi-OS CI ready |

A qualified release candidate requires both green software checks and
completed hardware/binding evidence. Publication is out of scope for this
milestone.
