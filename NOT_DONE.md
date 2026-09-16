# Not done

Last updated after GitHub Actions was split into a cheap PR lane and a
full post-merge / `workflow_dispatch` matrix (on top of remaining textual
ID fields, pairing-only handshake, mixed-case revoke, H9 restart, H11
isolation, and green CI on PR #1).

Software ownership, cancellation, stall recovery, request saturation, the
cross-platform acceptance harness, and remaining textual fields (synthetic)
are implemented. This file lists everything that is **not** complete for a
qualified MVP release candidate.

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

Physical-card confirmation is also still required for community hypotheses on
this branch: gender `1`/`2` encoding, English-name `#` layout, and 7-part
address `#` component order (see `docs/provenance.md`).

---

## Proven in CI (PR #1)

On `3ca134d` (PR #1, Actions run [35054867099](https://github.com/night-sornram/openthaiid/actions/runs/35054867099)): 11/11 jobs green, including five-target build + version smoke, `npm ci` SDK, Chrome × Linux/macOS/Windows + Edge × Windows (allow/deny/unanswered), artifact-style steps, node-pty Windows, and `go test -count=50`.

After the CI split, **pull requests** only re-run the ubuntu lane (Go tests
with `-count=5`, linux/amd64, SDK, Chrome × allow/deny/unanswered). Confirm
the full 11-job matrix on merge to `dev`/`main` or via **workflow_dispatch**
(select branch `dev` until `main` tracks releases).

---

## Explicitly out of this milestone

Do not treat these as MVP gaps:

- Portrait / photo support
- React hooks
- Installers, tray UI, auto-update
- Official Firefox / Safari support
- Published GitHub Release / release candidate tag
- Laser-id / NHSO / other DFs not in the cited APDU table

---

## Outstanding software items

| Item | Notes |
|---|---|
| `go.mod` still declares `go 1.22` | CI validates on 1.27.1; bump optional |
| Binding audit P7–P9 marked Partial | Code ownership/limits implemented; hardware stress still open |

---

## How to close the remaining work

1. Push this branch and confirm PR CI (ubuntu Go + Chrome, 4 jobs) is green.
   Confirm the full five-target + Chrome×3/Edge matrix on merge to `dev` or
   via workflow_dispatch on `dev`.
2. Run `make acceptance` on Windows (Chrome + Edge) and Linux.
3. Connect an ACR39U on macOS; complete E1–E7 without logging field values.
   While doing so, confirm gender encoding, English-name delimiters, and
   address component order against a real card.
4. Repeat physical checks on Windows and Linux.
5. Complete the headed public-HTTPS local-network permission check (M1).
6. Finish the binding audit checklist in `docs/dependency-audit.md` /
   `docs/acceptance.md` (P2–P6, P10).
7. Only then tag a release candidate.

Authoritative matrix: [docs/acceptance.md](docs/acceptance.md).
