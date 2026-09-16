# Not done

Last updated after the ERP autofill config package (schema, samples,
`docs/erp-integration.md`, clinic JSON import/export). Thai Fill-error
guidance, staff training, and the Friday runbook remain on `dev`. Physical
ACR39U execution is still blocked on hardware.

Software ownership, cancellation, stall recovery, request saturation, the
cross-platform acceptance harness, remaining textual fields, optional
portrait (synthetic), the qualification *kit*, and unsigned macOS packaging
instructions are implemented. This file lists everything that is
**not** complete for a **hardware-qualified** release.

Do **not** treat GitHub Actions as green on this PR: the workflow stays
disabled (included minutes exhausted). Run `make test`,
`make qualify-smoke`, `make smoke-package`, and `make acceptance-firefox`
locally.

---

## Blocked on hardware / target environments

| ID | Item | Needs |
|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | ACR39U + OS. Kit: `docs/hardware-qualification.md` + `openthaiid qualify` |
| E2 | 100 successful repeated MVP reads (no field values stored in reports) | ACR39U. Kit records pass/fail + timing + codes only |
| E3 | ≥20 insertion/removal cycles | ACR39U |
| E4 | Removal during reading and reader unplug | ACR39U |
| E5 | Browser refresh, bridge restart, sleep/wake | ACR39U + browser |
| E6 | Missing/stopped PC/SC service recovery where applicable | OS PC/SC |
| E7 | Local visual verification without storing field contents | Operator |
| — | Windows physical qualification | Windows host + reader |
| — | Linux physical qualification | Linux host + reader |
| M1 | Headed manual check from a public HTTPS origin (local-network permission) | Operator HTTPS origin. Kit section in `docs/hardware-qualification.md` |
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
| Published GitHub Release / `v0.1.0-rc.1` tag | Prepare from `dev`; a human should tag after review. Do not enable Actions to build the RC. |
| Darwin clinic archive on GitHub | Produce with `make package-macos` **on a Mac**. Linux builders cannot cross-compile `ebfe/scard` + Cocoa. Unsigned / not notarized. |

---

## How to close the remaining work

1. Local: `make test`, `make qualify-smoke`, `make smoke-package`, and
   `make acceptance-firefox` (Linux). On macOS, `make package-macos` and
   `make acceptance-webkit`. Real Safari is still a headed check.
2. Connect an ACR39U on macOS; complete
   [docs/hardware-qualification.md](docs/hardware-qualification.md)
   (E1–E7) without logging field values (including photo bytes). Confirm
   gender, English-name, address order, and portrait JPEG on a real card.
3. Repeat physical checks on Windows and Linux.
4. Complete the headed public-HTTPS local-network permission check (M1).
5. Finish the binding audit checklist in `docs/dependency-audit.md` /
   `docs/acceptance.md` (P2–P6, P10).
6. Human/coordinator: tag `v0.1.0-rc.1` from `dev` after review, attach
   `scripts/package.sh` archives (linux/windows from Linux; darwin from a
   Mac via `make package-macos`) + SHA256SUMS as a GitHub **prerelease**.
   Do not enable GitHub Actions until minutes are available.

Authoritative matrix: [docs/acceptance.md](docs/acceptance.md).
Release notes: [CHANGELOG.md](CHANGELOG.md) `[0.1.0-rc.1]`.
