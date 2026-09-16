# Not done

Last updated after pairing-only handshake, mixed-case revoke, H9 process
restart, and H11 two-browser isolation coverage.

Software ownership, cancellation, stall recovery, request saturation, and the
cross-platform acceptance harness are implemented and pass locally. This file
lists everything that is **not** complete for a qualified MVP release candidate.

---

## Blocked on hardware / target environments

| ID | Item | Needs |
|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | ACR39U + OS |
| E2 | 100 successful repeated MVP reads (no field values stored in reports) | ACR39U |
| E3 | ≥20 insertion/removal cycles | ACR39U |
| E4 | Removal during reading and reader unplug | ACR39U |
| E5 | Browser refresh, bridge restart, sleep/wake | ACR39U + browser |
| E6 | Missing/stopped PC/SC service recovery where applicable | OS PC/SC |
| E7 | Local visual verification without storing field contents | Operator |
| — | Windows physical qualification | Windows host + reader |
| — | Linux physical qualification | Linux host + reader |
| M1 | Headed manual check from a public HTTPS origin (local-network permission) | Operator HTTPS origin |
| P2–P6, P10 | `ebfe/scard` binding audit items (layouts, WinSCard load, cgo stress) | Cross-build + review |

**Order:** macOS + ACR39U first, then Windows/Linux. Untested configurations stay unverified.

---

## Written but not proven in CI

| Item | Status |
|---|---|
| Green five-target build + version smoke on GitHub Actions | Workflow updated; needs a green run |
| `npm ci` only (no `npm install` fallback) in SDK/acceptance jobs | Ready; needs CI run |
| Artifact upload (5 binaries + SDK dist + allowlisted acceptance reports) | Ready; needs CI run |
| Browser acceptance: Chrome × Linux/macOS/Windows + Edge × Windows | Jobs present; allow+deny+unanswered policies |
| node-pty harness on Windows | Code ready; needs Windows CI run |
| Repeat bridge/reader tests `-count=50` | Step present; needs CI run |
| Resolved Go 1.27.1 / Node 22 versions in CI reports | Job prints them; needs a green run |

---

## Explicitly out of this milestone

Do not treat these as MVP gaps:

- Remaining textual fields (address, issuer, issue date, expire date, gender, …)
- Portrait / photo support
- React hooks
- Installers, tray UI, auto-update
- Official Firefox / Safari support
- Published GitHub Release / release candidate tag

---

## Outstanding software items

| Item | Notes |
|---|---|
| `go.mod` still declares `go 1.22` | CI validates on 1.27.1; bump optional |
| Binding audit P7–P9 marked Partial | Code ownership/limits implemented; hardware stress still open |

---

## How to close the remaining work

1. Push `dev` and confirm five-target + browser CI jobs (Chrome ×3, Edge ×1)
   are green with allow/deny/unanswered policies.
2. Run `make acceptance` on Windows (Chrome + Edge) and Linux.
3. Connect an ACR39U on macOS; complete E1–E7 without logging field values.
4. Repeat physical checks on Windows and Linux.
5. Complete the headed public-HTTPS local-network permission check (M1).
6. Finish the binding audit checklist in `docs/dependency-audit.md` /
   `docs/acceptance.md` (P2–P6, P10).
7. Only then tag a release candidate.

Authoritative matrix: [docs/acceptance.md](docs/acceptance.md).
