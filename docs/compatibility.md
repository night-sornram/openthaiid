# Compatibility

## Release OS targets (initial)

| OS | Version | Arch | Notes |
|---|---|---|---|
| Windows | 11 | amd64 | WinSCard; cgo disabled |
| Ubuntu | 24.04 LTS | amd64, arm64 | `pcscd`, `libpcsclite`, CCID drivers |
| macOS | 15 | arm64, amd64 | Apple `PCSC.framework` |

Other versions are unverified until tested.

## Browser targets (initial)

| Browser | Platforms |
|---|---|
| Chrome | Windows, Linux, macOS |
| Edge | Windows |

Firefox and Safari are out of MVP scope.

## Reader

- Primary physical reader: **ACS ACR39U**.
- ACS documents PC/SC, CCID, Windows, Linux, and macOS support. That supports
  reader selection; it does not prove OpenThaiID compatibility.
- Exact ACR39U variant and driver versions must be recorded in hardware
  acceptance reports.

## Artifacts

| Artifact | Build environment | Native dependency |
|---|---|---|
| `openthaiid-windows-amd64.exe` | Windows amd64 | System WinSCard; cgo disabled |
| `openthaiid-linux-amd64` | Linux amd64 | cgo + libpcsclite headers |
| `openthaiid-linux-arm64` | Linux arm64 | cgo + libpcsclite headers |
| `openthaiid-darwin-arm64` | macOS arm64 | cgo + PCSC.framework |
| `openthaiid-darwin-amd64` | macOS Intel | cgo + PCSC.framework |

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

- Bridge binds `127.0.0.1:17891` by default.
- Web pages should use `ws://127.0.0.1:17891/v1/ws` from an HTTPS origin.
- Local-network permission may be required (see Chrome 147 notes in
  `docs/security.md`).
- Application CSP must allow `connect-src` to the loopback WebSocket origin.

## Versioning

- Bridge and SDK use SemVer.
- Protocol compatibility is negotiated separately (`openthaiid.v1`).
- Publish prereleases until the required OS/browser/hardware matrix passes.
- Platform code signing and notarization belong to a later installer milestone.
