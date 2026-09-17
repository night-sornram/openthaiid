# Hardware-arrival readiness (pre-hardware)

Last updated after **v0.1.0-rc.9** (tagged + GitHub Release) plus merged
SDK reconnect flake fix (#42). Physical ACR39U execution (including E4–E6 on hardware) is still blocked.

**Pre-hardware software is complete as of post-#42 / rc.9+.** There is no
remaining software backlog that can be finished without ACR39U, without
user signing certs, without cited Laser-id/NHSO APDU, and without enabling
Actions.

The only open software-ish leftovers are:

- optional Darwin clinic archive (`make package-macos` on a Mac host)
  attached to the existing **v0.1.0-rc.9** release (unsigned / not notarized)
- headed browser checks that need the clinic PC (real Safari UI; loopback
  TLS Firefox/Safari **https** ERP Fill with a locally trusted helper cert)

Do **not** retag `v0.1.0-rc.9`. Do **not** bump to `v0.1.0-rc.10` for the
#42 test-only flake fix. Do **not** enable GitHub Actions (included
minutes until ~1 Oct 2026). Do **not** implement Laser-id / NHSO.

This file is the **hardware-arrival playbook**, not a software backlog.

Local evidence until Actions returns: `make test`, `make qualify-smoke`,
`make bind-stress-smoke`, `make pre-reader-dry-run`, `make smoke-package`, and
`make acceptance-firefox`.

---

## Done in software (rc.9+)

Already shipped. Do not re-open these as software todos.

- Reader ownership, cancellation, stall recovery, request saturation
- Cross-platform acceptance harness (synthetic) + `make pre-reader-dry-run`
- Remaining textual fields + optional portrait (synthetic)
- Qualification *kit* (`openthaiid qualify`, no personal fields in reports)
- Unsigned clinic packaging + macOS-host-only `make package-macos` instructions
- Reconnect / **ทดสอบการเชื่อมต่อ**, multi-reader **ใช้เครื่องนี้**
- Card-removed / reader-unplug clinic UX (`CARD_REMOVED` / `READER_REMOVED`)
- Sleep/wake + PC/SC recovery mapping (`BRIDGE_UNAVAILABLE` / `PCSC_UNAVAILABLE`)
- Operator GitHub Releases **check** (`openthaiid update check`, Ready **ตรวจเวอร์ชัน**)
- Honest Firefox/Safari HTTPS Fill errors (`HTTPS_WS_BLOCKED` / `TLS_UNTRUSTED`)
- Optional technician-managed loopback HTTPS/WSS (`docs/loopback-tls.md`)
- Go module hygiene (rc.6); helper version string `0.1.0-rc.9`
- Staff **desk card** (`docs/staff-desk-card.md`)
- Technician **`openthaiid doctor`** desk snapshot (`GET /v1/helper/doctor`)
- hardware playbook (`NOT_DONE.md` split: software done / ACR39U / APDU / certs / Actions)
- PC/SC **bind-stress** harness (`openthaiid bind-stress`, `make bind-stress-smoke`)
- SDK packages aligned at `0.1.0-rc.9` (`@openthaiid/web` / `@openthaiid/react`, unpublished)
- HTTPS clinic ERP sample (`examples/clinic-patient/clinic-https.json`)
- SDK reconnect unit-test flake (#42; fake timers; no production reconnect change)
- Published GitHub Release **v0.1.0-rc.9** (linux-amd64 + windows-amd64 archives;
  Darwin still a Mac-host attach)

Authoritative software matrix: [docs/acceptance.md](docs/acceptance.md).
Release notes: [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` + `[0.1.0-rc.9]`.

---

## When the ACR39U arrives

Reader on the desk. Never `-synthetic` for a hardware claim. No personal
field values in reports, tickets, or screenshots.
Kit: [docs/hardware-qualification.md](docs/hardware-qualification.md).
One-page: [docs/hardware-checklist.md](docs/hardware-checklist.md).
Staff print: [docs/staff-desk-card.md](docs/staff-desk-card.md).

**Order:** macOS + ACR39U first, then Windows, then Linux. A macOS pass
does not close the three-platform claim.

| ID | Item | Needs |
|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | ACR39U + OS. `openthaiid qualify` |
| E2 | 100 successful repeated MVP reads (no field values stored) | ACR39U. Pass/fail + timing + codes only |
| E3 | ≥20 insertion/removal cycles | ACR39U |
| E4 | Removal during reading and reader unplug | ACR39U (software mapped `CARD_REMOVED` / `READER_REMOVED`) |
| E5 | Browser refresh, bridge restart, sleep/wake | ACR39U + browser (software mapped; physical still required) |
| E6 | Missing/stopped PC/SC service recovery where applicable | OS PC/SC (synthetic unavailable→restore done; physical still required; macOS may SKIP) |
| E7 | Local visual verification without storing field contents | Operator + real card |
| M1 | Headed manual check from a public HTTPS origin (local-network permission) | Operator HTTPS origin. Kit section in `docs/hardware-qualification.md` |
| — | Windows physical qualification | Windows host + reader (after macOS) |
| — | Linux physical qualification | Linux host + reader (after Windows) |
| — | Real Safari UI (not Playwright WebKit) | macOS + Safari |
| — | Loopback TLS headed: Firefox/Safari **https** ERP Fill with a **locally trusted** helper cert (`wss://127.0.0.1:17892`) | Clinic PC + mkcert (or imported self-signed). Software listener + SDK selection are in this tree. |
| — | Real-card layout checks (community hypotheses) | Gender `1`/`2`; English-name `#` layout; 7-part address `#` order; portrait JPEG SOI/EOI trim — see `docs/provenance.md` |
| P2–P6, P10 | `ebfe/scard` binding audit (layouts, WinSCard load, cgo stress) | Cross-build + **run the harness** on the hardware hosts. Today: `make bind-stress-smoke`. Hardware: `./scripts/bind-stress.sh --report ./openthaiid-bind-stress.json` (never `-synthetic`). [docs/bind-stress.md](docs/bind-stress.md) |
| P7–P9 | Concurrent context / Transmit lengths / `Cancel` vs `GetStatusChange` | Code is Partial; **harness is runnable**. Synthetic smoke is not a close. Hardware: same script, card seated for P8. [docs/bind-stress.md](docs/bind-stress.md) |

Untested configurations stay unverified.

Darwin clinic archive: produce with `make package-macos` **on a Mac** and
attach to the existing **v0.1.0-rc.9** release (unsigned / not
notarized). Linux builders cannot cross-compile `ebfe/scard` + Cocoa. That
is a Mac build step, not a reason to retag `v0.1.0-rc.9` or bump to rc.10.

---

## Blocked without cited APDU (Laser-id / NHSO)

Do **not** implement `laserId`, NHSO, or other DFs. The cited APDU.md
revision for `thai-citizen-mvp` does not list them. Community source
constants are not a citable command table.

See [docs/provenance.md](docs/provenance.md) (Laser-id / NHSO section).
Stay document-only until a URL + immutable revision + license note is
reviewed the same way as APDU.md `4d803243715254cc93efe9040d8b1ecd85f520aa`.

---

## Blocked without user certs (Store / notarize)

Cannot fake:

- Apple notarization / Developer ID signing (no certificate)
- Microsoft Authenticode / Store signing (no certificate)

Unsigned packages + [docs/signing.md](docs/signing.md) are the path.
A technician can **check** GitHub Releases without auto-download.
CLI `update apply --yes` stays explicit and unsigned. There is no
Store-signed auto-update.

---

## Paused until Actions minutes (~1 Oct 2026)

The workflow file stays in the tree and stays **disabled** at the repo
until included minutes reset (~1 Oct 2026). Do not enable it to “prove”
this PR or to build an RC.

Docs-only / `.cursor/`-only changes already skip CI (`paths-ignore`).
When minutes return: PR lane is Ubuntu Firefox; full lane is
`workflow_dispatch` / push to `dev`/`main`. See
[docs/acceptance.md](docs/acceptance.md).

Until then, run locally:

```bash
make test
make qualify-smoke
make bind-stress-smoke
make pre-reader-dry-run
make smoke-package
make acceptance-firefox   # Linux
# On macOS: make package-macos && make acceptance-webkit
```

Go toolchain / `ebfe/scard` pin: **held** (`go1.27.1`; no newer `ebfe/scard`
tag). Not a software task.

---

## Coordinator — Darwin attach only (no next tag)

`v0.1.0-rc.9` is already tagged and published (linux-amd64 + windows-amd64).
This PR does **not** create a git tag and does **not** bump to rc.10.
Wait for ACR39U for E1–E7 / M1 / P2–P10 hardware closes.

- Do **not** retag `v0.1.0-rc.9`.
- Do **not** enable GitHub Actions.
- Optional leftover: Darwin archive on a Mac, then attach to the existing
  release (replace SHA256SUMS so it includes the darwin line).

```bash
# On a Mac only. Do not retag. Do not create v0.1.0-rc.10.
git checkout dev && git pull origin dev
VERSION=0.1.0-rc.9 make package-macos

(
  cd dist
  curl -fsSL -o SHA256SUMS \
    https://github.com/night-sornram/openthaiid/releases/download/v0.1.0-rc.9/SHA256SUMS
  sha256sum openthaiid-0.1.0-rc.9-darwin-*.tar.gz >> SHA256SUMS
)

gh release upload v0.1.0-rc.9 \
  --repo night-sornram/openthaiid \
  dist/openthaiid-0.1.0-rc.9-darwin-*.tar.gz \
  dist/SHA256SUMS
```
