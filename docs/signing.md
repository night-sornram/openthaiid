# Signing and store packaging (future)

This tree **ships unsigned packages**. There is no Apple Developer ID,
notarization ticket, or Microsoft Authenticode / Store certificate in
this repository or in CI secrets.

`scripts/package.sh` produces a portable folder plus:

- Windows: `Start-OpenThaiID.bat`, optional `install.ps1` (Start Menu)
- macOS: unsigned `OpenThaiID.app` (`LSUIElement`)
- Linux: `.desktop` + optional user systemd unit (`openthaiid serve`) +
  system-unit template

SmartScreen, Gatekeeper, and “unidentified developer” warnings are
**expected** until a human with certificates signs the bits.

Auto-update (`openthaiid update check|download|apply`) reads **GitHub
Releases** only. It does not claim the asset is signed. Prefer
`SHA256SUMS` inside the archive; `update apply --yes` verifies that
file when present.

## Checklist when certificates exist

Do not check any of these off until the named credential is in the
operator’s possession.

### Windows (Authenticode / Microsoft Store)

1. Obtain an Authenticode certificate (or Store publisher account).
2. Sign `openthaiid.exe` after `go build` (`signtool sign /fd SHA256 …`).
3. Optionally wrap in an MSI/MSIX. Store listing is a separate publisher
   process — this repo does not submit to the Store.
4. Publish the signed artifact as a GitHub Release asset. Keep SHA256SUMS.
5. SmartScreen reputation still needs download volume; signing is necessary
   but not sufficient on day one.

### macOS (Developer ID / notarization)

1. Obtain an Apple Developer ID Application certificate.
2. Sign `OpenThaiID.app` (`codesign --deep --force --options runtime`).
3. Notarize (`notarytool submit`) and staple.
4. Gatekeeper then allows a downloaded `.app` without right-click Open.
5. Mac App Store is a different provisioning profile; not started here.

### Linux

1. Optional: a distro package (`.deb`/Flathub) with the maintainer’s key.
2. The portable tarball remains the default clinic path.

Until then: ship the unsigned archive, document quarantine/`xattr` on
macOS, and keep auto-update explicit (`--yes`) so an unsigned binary is
never silently replaced.
