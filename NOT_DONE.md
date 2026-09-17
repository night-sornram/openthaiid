# Friday readiness (pre-hardware)

Last updated after **v0.1.0-rc.9** prep: SDK version align + HTTPS ERP
sample (#40). Physical ACR39U execution (including Friday E4–E6 on
hardware) is still blocked. This tree is **tag-ready** for
`v0.1.0-rc.9` after review. Do not retag `v0.1.0-rc.8`.
This file is the **Friday morning playbook**, not a software backlog.

Do **not** retag `v0.1.0-rc.8`. Do **not** enable GitHub Actions (included
minutes until ~1 Oct 2026). Do **not** implement Laser-id / NHSO.

Local evidence until Actions returns: `make test`, `make qualify-smoke`,
`make bind-stress-smoke`, `make friday-dry-run`, `make smoke-package`, and
`make acceptance-firefox`.

---

## Done in software (rc.9)

Already shipped. Do not re-open these as Friday software todos.

- Reader ownership, cancellation, stall recovery, request saturation
- Cross-platform acceptance harness (synthetic) + `make friday-dry-run`
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
- Staff Friday **desk card** (`docs/friday-desk-card.md`)
- Technician **`openthaiid doctor`** desk snapshot (`GET /v1/helper/doctor`)
- Friday playbook (`NOT_DONE.md` split: software done / ACR39U / APDU / certs / Actions)
- PC/SC **bind-stress** harness (`openthaiid bind-stress`, `make bind-stress-smoke`)
- SDK packages aligned at `0.1.0-rc.9` (`@openthaiid/web` / `@openthaiid/react`, unpublished)
- HTTPS clinic ERP sample (`examples/clinic-patient/clinic-https.json`)
- Published GitHub Release **v0.1.0-rc.8** (linux-amd64 + windows-amd64 archives)

Authoritative software matrix: [docs/acceptance.md](docs/acceptance.md).
Release notes: [CHANGELOG.md](CHANGELOG.md) `[0.1.0-rc.9]`.

---

## Do Friday with ACR39U

Reader on the desk. Never `-synthetic` for a hardware claim. No personal
field values in reports, tickets, or screenshots.
Kit: [docs/hardware-qualification.md](docs/hardware-qualification.md).
One-page: [docs/friday-checklist.md](docs/friday-checklist.md).
Staff print: [docs/friday-desk-card.md](docs/friday-desk-card.md).

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
| P2–P6, P10 | `ebfe/scard` binding audit (layouts, WinSCard load, cgo stress) | Cross-build + **run the harness** on the Friday hosts. Today: `make bind-stress-smoke`. Friday: `./scripts/bind-stress.sh --report ./openthaiid-bind-stress.json` (never `-synthetic`). [docs/bind-stress.md](docs/bind-stress.md) |
| P7–P9 | Concurrent context / Transmit lengths / `Cancel` vs `GetStatusChange` | Code is Partial; **harness is runnable**. Synthetic smoke is not a close. Hardware: same script, card seated for P8. [docs/bind-stress.md](docs/bind-stress.md) |

Untested configurations stay unverified.

Darwin clinic archive: produce with `make package-macos` **on a Mac** and
attach to the **v0.1.0-rc.9** release after it is tagged (unsigned / not
notarized). Linux builders cannot cross-compile `ebfe/scard` + Cocoa. That
is a Mac build step, not a reason to retag `v0.1.0-rc.8`.

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
make friday-dry-run
make smoke-package
make acceptance-firefox   # Linux
# On macOS: make package-macos && make acceptance-webkit
```

Go toolchain / `ebfe/scard` pin: **held** (`go1.27.1`; no newer `ebfe/scard`
tag). Not a Friday software task.

---

## Coordinator — next tag only

Tree is tag-ready on `dev` after this PR. A human should tag
`v0.1.0-rc.9` after review. This PR does **not** create the git tag.

- Do **not** retag `v0.1.0-rc.8`.
- Do **not** enable GitHub Actions.

```bash
git checkout dev && git pull origin dev
git tag -a v0.1.0-rc.9 -m "OpenThaiID v0.1.0-rc.9"
git push origin v0.1.0-rc.9

# Linux + Windows archives (Linux builder). Darwin must be on a Mac.
VERSION=0.1.0-rc.9 OS=linux ARCH=amd64 ./scripts/package.sh
VERSION=0.1.0-rc.9 OS=windows ARCH=amd64 ./scripts/package.sh
# On a Mac only:
#   VERSION=0.1.0-rc.9 make package-macos

# Checksums of the archives (package.sh already writes SHA256SUMS *inside*
# each archive for the binary):
(
  cd dist
  sha256sum openthaiid-0.1.0-rc.9-linux-amd64.tar.gz \
    openthaiid-0.1.0-rc.9-windows-amd64.zip \
    > SHA256SUMS
)
# append the darwin archive SHA when produced on a Mac

# Release notes: the [0.1.0-rc.9] section only (not the whole CHANGELOG).
sed -n '/^## \[0.1.0-rc.9\]/,/^## \[0.1.0-rc.8\]/{ /^## \[0.1.0-rc.8\]/d; p; }' \
  CHANGELOG.md > /tmp/openthaiid-0.1.0-rc.9-notes.md

gh release create v0.1.0-rc.9 \
  --repo night-sornram/openthaiid \
  --prerelease \
  --target dev \
  --title "v0.1.0-rc.9" \
  --notes-file /tmp/openthaiid-0.1.0-rc.9-notes.md \
  dist/openthaiid-0.1.0-rc.9-linux-amd64.tar.gz \
  dist/openthaiid-0.1.0-rc.9-windows-amd64.zip \
  dist/SHA256SUMS
# add dist/openthaiid-0.1.0-rc.9-darwin-<arch>.tar.gz when available
```
