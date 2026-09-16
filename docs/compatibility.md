# Compatibility

## Release OS targets (initial)

| OS | Version | Arch | Notes |
|---|---|---|---|
| Windows | 11 | amd64 | WinSCard; cgo disabled |
| Ubuntu | 24.04 LTS | amd64, arm64 | `pcscd`, `libpcsclite`, CCID drivers |
| macOS | 15 | arm64, amd64 | Apple `PCSC.framework` |

Other versions are unverified until tested.

## Browser targets (initial)

| Browser | Platforms | How to run |
|---|---|---|
| Firefox | Linux (preferred in CI/cloud), also Windows/macOS | `OPENTHAIID_BROWSER=firefox` / `make acceptance-firefox` |
| Chrome | Windows, Linux, macOS | `OPENTHAIID_BROWSER=chrome` (default local `make acceptance`) |
| Edge | Windows | `OPENTHAIID_BROWSER=msedge` |
| Safari / WebKit | **macOS only** for real Safari. Playwright `webkit` is the automatable engine (`OPENTHAIID_BROWSER=webkit` / `make acceptance-webkit`). Linux WebKit is not Safari. | Headed Safari remains a manual check |

The GitHub Actions matrix is **not** expanded: pull requests (when the
workflow is enabled) run Firefox on ubuntu only; the full lane adds Chrome
on macOS/Windows and Edge on Windows. Do not add Safari rows (macOS
multiplier).

### Clinic Fill / Ready (software)

SDK and Ready page assume a current desktop engine: **Firefox 115+**,
**Chrome / Edge 120+**, **Safari 16+** (macOS). No IE. No iOS Safari as a
clinic desk target.

| Surface | Chrome / Edge | Firefox | Safari 16+ (macOS) |
|---|---|---|---|
| Ready page `http://127.0.0.1:17891/` | Works | Works | Works |
| Fill / `probeThaiID` from **http** ERP (intranet / localhost) | Works | Works | Works |
| Fill / `probeThaiID` from **https** ERP | Works (may prompt Local Network Access). Optional `wss://` if helper TLS is on. | **Blocked** on `ws://` (mixed content). Works with technician-managed loopback TLS (`wss://127.0.0.1:17892`). | Same as Firefox (macOS Sequoia may also prompt Local Network) |
| Technician Copy diagnostics on Ready | Clipboard API | Clipboard API, then `execCommand`, then prompt | Same fallbacks |

Firefox and Safari treat `ws://127.0.0.1` from an `https://` clinic page as
mixed active content. Chromium still allows loopback mixed WebSockets.
OpenThaiID does **not** ship a helper TLS private key or a Firefox pref.
Technicians may enable optional loopback HTTPS/WSS with a **local** cert
([loopback-tls.md](loopback-tls.md)). The library then uses
`wss://127.0.0.1:17892`. Without that, it returns `HTTPS_WS_BLOCKED` (not
“helper is not running”). An untrusted local cert is `TLS_UNTRUSTED`. Do not
tell staff to disable mixed-content checks.

Workable clinic paths on Firefox/Safari today:

1. Keep the ERP on `http://` on that PC (typical intranet), **or**
2. Use Chrome/Edge for the HTTPS ERP, **or**
3. Enable optional helper TLS once per PC ([loopback-tls.md](loopback-tls.md)), **or**
4. Use the Ready page (always `http://127.0.0.1`) for lights / Allow / version check; Fill still needs a browser that can open the loopback WebSocket.

Do not tell clinics to set `network.websocket.allowInsecureFromHTTPS` — that
pref is **acceptance-harness only**.

Automated coverage without GitHub Actions: `sdk/web` vitest (jsdom UA
matrix + `replaceChildren` fallback) and `make acceptance-firefox` locally.
Real Safari UI stays a headed macOS check (`NOT_DONE.md`).

## Reader

- Primary physical reader: **ACS ACR39U**.
- ACS documents PC/SC, CCID, Windows, Linux, and macOS support. That supports
  reader selection; it does not prove OpenThaiID compatibility.
- Exact ACR39U variant and driver versions must be recorded in hardware
  acceptance reports.

## Artifacts

| Artifact | Build environment | Native dependency |
|---|---|---|
| `openthaiid-windows-amd64.exe` | Windows amd64 | System WinSCard; cgo disabled (tray is Win32) |
| `openthaiid-linux-amd64` | Linux amd64 | cgo + libpcsclite headers (no GTK tray) |
| `openthaiid-linux-arm64` | Linux arm64 | cgo + libpcsclite headers (no GTK tray) |
| `openthaiid-darwin-arm64` | macOS arm64 **host** (`make package-macos`) | cgo + PCSC.framework + Cocoa tray |
| `openthaiid-darwin-amd64` | macOS Intel **host** | cgo + PCSC.framework + Cocoa tray |

Clinic archives from `scripts/package.sh` rename the binary to `openthaiid` /
`openthaiid.exe` and add START-HERE + launch files. Unsigned. Not notarized.
`OS=darwin` on Linux/Windows exits with an error: darwin cannot be
cross-compiled (`ebfe/scard` + Cocoa cgo).

Linux binaries are **not** advertised as fully static or universally compatible
across libc variants.

## Runtime prerequisites

### Windows

- Smart Card service enabled.
- Vendor driver only when the generic CCID/WinSCard path is insufficient.

### Linux

```bash
sudo apt install pcscd libpcsclite-dev
sudo systemctl enable --now pcscd
# CCID driver as required by the reader
```

Document distro-specific permissions if a user cannot access the PC/SC daemon.

### macOS

- Built-in PC/SC framework.
- Test Apple Silicon and Intel builds separately.

## Browser / network notes

- Bridge binds `127.0.0.1:17891` by default (HTTP/WS). Optional HTTPS/WSS is
  `127.0.0.1:17892` when a technician configures a local cert.
- Web pages should use `ws://127.0.0.1:17891/v1/ws` from an HTTP origin.
  HTTPS Firefox/Safari pages should use `wss://127.0.0.1:17892/v1/ws` after
  local TLS is enabled.
- Local-network permission may be required (see Chrome 147 notes in
  `docs/security.md`).
- Application CSP must allow `connect-src` to the loopback WebSocket origin
  (`ws:` and, when TLS is on, `wss:`).
- Firefox treats `ws://127.0.0.1` from an `https://` page as mixed active
  content. The acceptance harness sets
  `network.websocket.allowInsecureFromHTTPS` (**tests only**). Real Firefox
  and Safari clinic pages on HTTPS need helper TLS or Chrome/Edge / http ERP.
  `HTTPS_WS_BLOCKED` / `TLS_UNTRUSTED` are the honest errors. Chrome is more
  lenient for localhost mixed content.
- Firefox is strict about ES module MIME types (`text/javascript`). Serve SDK
  `.js` with that type; an empty Content-Type (including a 404) is a hard
  block. Chrome often still executes the graph.

## Versioning

- Bridge and SDK use SemVer.
- Protocol compatibility is negotiated separately (`openthaiid.v1`).
- Publish prereleases until the required OS/browser/hardware matrix passes.
- Platform code signing and notarization belong to a later installer milestone.
  This tree ships an unsigned portable folder, Windows Start Menu script, Linux
  `.desktop`, and an unsigned macOS `.app`.
