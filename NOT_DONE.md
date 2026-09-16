# Not done

Last updated after **optional loopback helper TLS** (pre-hardware) so https ERP
Fill can use `wss://127.0.0.1` on Firefox/Safari without weakening mixed-content
checks. Physical ACR39U execution (including Friday E4–E6 on hardware) is still
blocked. Published **v0.1.0-rc.4** is on GitHub; this tree stays on that helper
version.

Software ownership, cancellation, stall recovery, request saturation, the
cross-platform acceptance harness, remaining textual fields, optional
portrait (synthetic), the qualification *kit*, unsigned macOS packaging
instructions, reconnect/test-connection staff UX, multi-reader select,
card-removed / unplug clinic UX, sleep/wake + PC/SC recovery mapping,
operator GitHub Releases **check**, honest Firefox/Safari HTTPS Fill
errors, and optional technician-managed loopback HTTPS/WSS are implemented.
This file lists everything that is **not** complete for a **hardware-qualified**
release.

Do **not** treat GitHub Actions as green on this PR: the workflow stays
disabled (included minutes exhausted). Run `make test`,
`make qualify-smoke`, `make friday-dry-run`, `make smoke-package`, and
`make acceptance-firefox` locally.

---

## Blocked on hardware / target environments

| ID | Item | Needs |
|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | ACR39U + OS. Kit: `docs/hardware-qualification.md` + `openthaiid qualify` |
| E2 | 100 successful repeated MVP reads (no field values stored in reports) | ACR39U. Kit records pass/fail + timing + codes only |
| E3 | ≥20 insertion/removal cycles | ACR39U |
| E4 | Removal during reading and reader unplug | ACR39U |
| E5 | Browser refresh, bridge restart, sleep/wake | ACR39U + browser (software mapped; physical still required) |
| E6 | Missing/stopped PC/SC service recovery where applicable | OS PC/SC (synthetic unavailable→restore done; physical still required) |
| E7 | Local visual verification without storing field contents | Operator |
| — | Windows physical qualification | Windows host + reader |
| — | Linux physical qualification | Linux host + reader |
| M1 | Headed manual check from a public HTTPS origin (local-network permission) | Operator HTTPS origin. Kit section in `docs/hardware-qualification.md` |
| P2–P6, P10 | `ebfe/scard` binding audit items (layouts, WinSCard load, cgo stress) | Cross-build + review |
| Safari | Real Safari UI (not Playwright WebKit) | macOS + Safari |
| Loopback TLS headed | Firefox/Safari **https** ERP Fill with a **locally trusted** helper cert (`wss://127.0.0.1:17892`) | Clinic PC + mkcert (or imported self-signed). Software listener + SDK selection are in this tree. |

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
A technician can **check** GitHub Releases (`openthaiid update check`, Ready
**ตรวจเวอร์ชัน**) without auto-download. CLI `update apply --yes` stays
explicit and unsigned. There is no Store-signed auto-update.

---

## Outstanding software items

| Item | Notes |
|---|---|
| Binding audit P7–P9 marked Partial | Code ownership/limits implemented; hardware stress still open |
| Published GitHub Release / `v0.1.0-rc.4` tag | Tree is tag-ready on `dev` after this PR. A human should tag after review. Do not enable Actions to build the RC. |
| Darwin clinic archive on GitHub | Produce with `make package-macos` **on a Mac**. Linux builders cannot cross-compile `ebfe/scard` + Cocoa. Unsigned / not notarized. |

---

## How to close the remaining work

1. Local: `make test`, `make qualify-smoke`, `make friday-dry-run`,
   `make smoke-package`, and
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
6. Human/coordinator: tag `v0.1.0-rc.4` from `dev` after review. Do **not**
   enable GitHub Actions until minutes are available. Exact commands after
   this PR is merged:

```bash
git checkout dev && git pull origin dev
git tag -a v0.1.0-rc.4 -m "OpenThaiID v0.1.0-rc.4"
git push origin v0.1.0-rc.4

# Linux + Windows archives (Linux builder). Darwin must be on a Mac.
VERSION=0.1.0-rc.4 OS=linux ARCH=amd64 ./scripts/package.sh
VERSION=0.1.0-rc.4 OS=windows ARCH=amd64 ./scripts/package.sh
# On a Mac only:
#   VERSION=0.1.0-rc.4 make package-macos

# Checksums of the archives (package.sh already writes SHA256SUMS *inside*
# each archive for the binary):
(
  cd dist
  sha256sum openthaiid-0.1.0-rc.4-linux-amd64.tar.gz \
    openthaiid-0.1.0-rc.4-windows-amd64.zip \
    > SHA256SUMS
)
# append the darwin archive SHA when produced on a Mac

# Release notes: the [0.1.0-rc.4] section only (not the whole CHANGELOG).
sed -n '/^## \[0.1.0-rc.4\]/,/^## \[0.1.0-rc.3\]/{ /^## \[0.1.0-rc.3\]/d; p; }' \
  CHANGELOG.md > /tmp/openthaiid-0.1.0-rc.4-notes.md

gh release create v0.1.0-rc.4 \
  --repo night-sornram/openthaiid \
  --prerelease \
  --target dev \
  --title "v0.1.0-rc.4" \
  --notes-file /tmp/openthaiid-0.1.0-rc.4-notes.md \
  dist/openthaiid-0.1.0-rc.4-linux-amd64.tar.gz \
  dist/openthaiid-0.1.0-rc.4-windows-amd64.zip \
  dist/SHA256SUMS
# add dist/openthaiid-0.1.0-rc.4-darwin-<arch>.tar.gz when available
```

Authoritative matrix: [docs/acceptance.md](docs/acceptance.md).
Release notes: [CHANGELOG.md](CHANGELOG.md) `[0.1.0-rc.4]`.
