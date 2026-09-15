# MVP acceptance matrix

This matrix is the gate for calling an MVP “complete.” Synthetic CI evidence
never substitutes for physical-card evidence where hardware is claimed.

## Build artifacts

| # | Criterion | Evidence |
|---|---|---|
| A1 | All five OS/arch artifacts build and start | CI + local smoke |
| A2 | `openthaiid doctor` reports PC/SC status without reading personal fields | CLI output |
| A3 | `openthaiid version` prints module version | CLI output |

## Protocol and authorization

| # | Criterion | Evidence |
|---|---|---|
| B1 | Unapproved origin receives no reader inventory, events, or card data | Security integration tests |
| B2 | Missing/null Origin, lookalikes, wrong ports, rebinding-style Host rejected | Security integration tests |
| B3 | Approval, denial, expiry, revocation behave as specified | Auth unit + integration |
| B4 | Field-set expansion requires a new approval | Auth tests |
| B5 | Pending approval cannot be claimed by another connection | Auth tests |
| B6 | Tokens fail across origins and bridge restarts | Auth tests |
| B7 | Grant expiry/revocation during a read suppresses the result | Integration |
| B8 | Competing clients never receive each other’s results | Integration |

## Reading

| # | Criterion | Evidence |
|---|---|---|
| C1 | `citizenId`, `thaiName`, `birthDate` normalize correctly | Parser fixtures + hardware |
| C2 | Field selection limits returned data and necessary reads | Integration + APDU script |
| C3 | Removal/replacement cannot produce stale results | Lifecycle tests |
| C4 | Unsupported/malformed cards yield typed errors and monitoring continues | Synthetic + hardware |
| C5 | No personal-data logging (sentinel log tests) | Privacy tests |

## SDK and example

| # | Criterion | Evidence |
|---|---|---|
| D1 | Connect → approve → read works through published SDK API | Playwright |
| D2 | Vanilla example clears displayed data on removal/disconnect/revocation | Example + test |
| D3 | SSR import is safe; connect requires a browser | SDK unit tests |
| D4 | Browser refresh within an active grant reuses trust | Playwright |

## Hardware qualification (manual)

For each claimed configuration, record OS, arch, reader variant, driver,
bridge revision, and non-personal outcome only.

| # | Criterion |
|---|---|
| E1 | Reader/card detection within 2s under normal operation |
| E2 | 100 successful repeated MVP reads |
| E3 | ≥20 insertion/removal cycles |
| E4 | Removal during reading and reader unplug |
| E5 | Browser refresh, bridge restart, sleep/wake |
| E6 | Missing/stopped PC/SC service recovery where applicable |
| E7 | Local visual verification without storing field values in reports |

## Phase 0 / Phase 1 exit (this milestone)

| # | Criterion | Status target |
|---|---|---|
| F1 | Architecture, threat model, protocol, provenance, compatibility docs committed | Required |
| F2 | Go module builds with CLI skeleton (`version`, `doctor`, default run) | Required |
| F3 | Backend/Card interfaces defined; synthetic backend implements them | Required |
| F4 | Real PC/SC adapter compiles on darwin with pinned `ebfe/scard` | Required on this machine |
| F5 | Five-target build matrix documented; CI workflow present | Required |
| F6 | Provenance ledger lists MVP profile as blocked pending source review | Required |
| F7 | Dependency audit checklist present | Required |

Hardware acceptance (E*) remains open until Windows/Linux test access exists.
Successful compilation alone does not satisfy hardware claims.
