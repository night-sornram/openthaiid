# Not done

Last updated after the software RC items (portrait, Firefox/WebKit harness,
unsigned GitHub Releases auto-update, go 1.27, laser/NHSO provenance block).

Software ownership, cancellation, stall recovery, request saturation, the
cross-platform acceptance harness, remaining textual fields, and optional
portrait (synthetic) are implemented. This file lists everything that is
**not** complete for a **hardware-qualified** release.

Do **not** treat GitHub Actions as green on this PR: the workflow stays
disabled (included minutes exhausted). Run `make test` and
`make acceptance-firefox` locally.

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
| Safari | Real Safari UI (not Playwright WebKit) | macOS + Safari |

**Order:** macOS + ACR39U first, then Windows/Linux. Untested configurations stay unverified.

Physical-card confirmation is also still required for community hypotheses on
this branch: gender `1`/`2` encoding, English-name `#` layout, 7-part
address `#` component order, and portrait JPEG SOI/EOI trim
(see `docs/provenance.md`).

---

## Explicitly out of this milestone (cannot fake)

- Apple notarization / Developer ID signing (no certificate)
- Microsoft Authenticode / Store signing (no certificate)
- Physical E1–E7 ACR39U qualification
- Laser-id / NHSO / other DFs not in the cited APDU table (see provenance)

Unsigned packages + `docs/signing.md` are the path toward signed installers.
Auto-update against GitHub Releases is implemented for **unsigned** assets.

---

## Outstanding software items

| Item | Notes |
|---|---|
| Binding audit P7–P9 marked Partial | Code ownership/limits implemented; hardware stress still open |
| Published GitHub Release / `v0.1.0-rc.1` tag | Prepare from this PR; a human should tag after review. Do not enable Actions to build the RC. |

---

## How to close the remaining work

1. Local: `make test` and `make acceptance-firefox` (Linux). On macOS,
   `make acceptance-webkit` exercises Playwright WebKit; real Safari is
   still a headed check.
2. Connect an ACR39U on macOS; complete E1–E7 without logging field values
   (including photo bytes). Confirm gender, English-name, address order,
   and portrait JPEG on a real card.
3. Repeat physical checks on Windows and Linux.
4. Complete the headed public-HTTPS local-network permission check (M1).
5. Finish the binding audit checklist in `docs/dependency-audit.md` /
   `docs/acceptance.md` (P2–P6, P10).
6. Human/coordinator: tag `v0.1.0-rc.1` from `dev` after review, attach
   `scripts/package.sh` archives + SHA256SUMS as a GitHub **prerelease**.
   Do not enable GitHub Actions until minutes are available.

Authoritative matrix: [docs/acceptance.md](docs/acceptance.md).
Release notes: [CHANGELOG.md](CHANGELOG.md) `[0.1.0-rc.1]`.
