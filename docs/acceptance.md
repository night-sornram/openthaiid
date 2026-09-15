# MVP acceptance matrix

This matrix is the gate for calling an MVP “complete.” Synthetic CI evidence
never substitutes for physical-card evidence where hardware is claimed.

Status values: **Done** (local evidence), **CI pending** (code ready, needs a
green five-target run on GitHub), **Open** (hardware or environments required).

## Build artifacts

| # | Criterion | Evidence | Status |
|---|---|---|---|
| A1 | All five OS/arch artifacts build and start | CI + local smoke | CI pending |
| A2 | `openthaiid doctor` reports PC/SC status without reading personal fields | CLI output | Done |
| A3 | `openthaiid version` prints module version (injected at release build) | CLI + package.sh verify | Done |

## Protocol and authorization

| # | Criterion | Evidence | Status |
|---|---|---|---|
| B1 | Unapproved origin receives no reader inventory, events, or card data | Security integration tests | Done |
| B2 | Missing/null Origin, lookalikes, wrong ports, rebinding-style Host rejected | Auth origin tests | Done |
| B3 | Approval, denial, expiry, revocation behave as specified | Auth unit + integration | Done |
| B4 | Field-set expansion requires a new approval | Auth tests | Done |
| B5 | Pending approval cannot be claimed by another connection | Connection-bound approvals | Done |
| B6 | Tokens fail across origins and bridge restarts | Auth tests | Done |
| B7 | Grant expiry/revocation during a read suppresses the result | Auth Session() + recheck | Done |
| B8 | Competing clients never receive each other’s results | Unicast write path | Done |

## Reading

| # | Criterion | Evidence | Status |
|---|---|---|---|
| C1 | `citizenId`, `thaiName`, `birthDate` normalize correctly | Parser fixtures + APDU byte tests | Done (synthetic) |
| C2 | Field selection limits returned data and necessary reads | Integration + APDU script | Done |
| C3 | Removal/replacement cannot produce stale results | Lifecycle tests | Done |
| C4 | Unsupported/malformed cards yield typed errors and monitoring continues | Synthetic + engine tests | Done |
| C5 | No personal-data logging (sentinel log tests) | Privacy tests | Done |
| C6 | Published APDU bytes/offsets match the cited table | `TestThaiProfileCommandPlanMatchesPublishedAPDU` | Done |
| C7 | BE→Gregorian dates, leap-year validation, null fields | Parser tests | Done |
| C8 | Synthetic CLI demonstration reads a fictional card | `TestSyntheticDemoReadAllFields` | Done |

## SDK and example

| # | Criterion | Evidence | Status |
|---|---|---|---|
| D1 | Connect → approve → read works through published SDK API | Bridge integration + SDK unit | Done (synthetic) |
| D2 | Vanilla example clears displayed data on removal/disconnect/revocation | SDK personal-data clear | Done (SDK); example needs manual check |
| D3 | SSR import is safe; connect requires a browser | SDK unit tests | Done |
| D4 | Browser refresh within an active grant reuses trust | In-memory grant + session.open | Done (logic); Playwright Open |
| D5 | Cancellation cancels the real request id | SDK abort test | Done |
| D6 | Session renewal and reconnect backoff | SDK unit paths | Done |

## Hardware qualification (manual)

For each claimed configuration, record OS, arch, reader variant, driver,
bridge revision, and non-personal outcome only.

| # | Criterion | Status |
|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | Open |
| E2 | 100 successful repeated MVP reads | Open |
| E3 | ≥20 insertion/removal cycles | Open |
| E4 | Removal during reading and reader unplug | Open |
| E5 | Browser refresh, bridge restart, sleep/wake | Open |
| E6 | Missing/stopped PC/SC service recovery where applicable | Open |
| E7 | Local visual verification without storing field values in reports | Open |

Available first target: macOS + ACR39U. Windows/Linux physical qualification
remains required for the original three-platform release claim. Untested
configurations stay explicitly unverified.

## Phase 0 / Phase 1 exit

| # | Criterion | Status |
|---|---|---|
| F1 | Architecture, threat model, protocol, provenance, compatibility docs committed | Done |
| F2 | Go module builds with CLI skeleton (`version`, `doctor`, default run) | Done |
| F3 | Backend/Card interfaces defined; synthetic backend implements them | Done |
| F4 | Real PC/SC adapter compiles on darwin with pinned `ebfe/scard` | Done (compile) |
| F5 | Five-target build matrix documented; CI workflow present | Done |
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
| G6 | Vanilla example | Done |
| G7 | Privacy log sentinel test | Done |
| G8 | Release packaging script + third-party notices | Done |
| G9 | Responsive dispatch + real cancellation + duplicate-ID rejection | Done |
| G10 | Grant-lifetime and revocation enforcement | Done |
| G11 | PC/SC StateUnaware retention + recovery | Done |
| G12 | Shared synthetic demo card end-to-end | Done |
| G13 | Green five-target CI with version smoke + npm ci | CI pending |
| G14 | Playwright browser acceptance (Chrome/Edge × 3 OS) | Open |
| G15 | Hardware qualification evidence | Open |

Hardware acceptance (E*) and Playwright matrix (G14) remain open. Successful
compilation alone does not satisfy hardware or multi-browser claims.
