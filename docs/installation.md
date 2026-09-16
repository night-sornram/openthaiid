# Installation

Clinic PCs need the **local helper** next to the card reader. There is no
OpenThaiID website or backend to deploy. The helper listens only on
`127.0.0.1`.

## Clinic IT (happy path)

1. Get a per-OS archive from `scripts/package.sh` (or an internal share of
   `dist/openthaiid-<version>-<os>-<arch>.tar.gz` / `.zip`).
2. Follow **START-HERE.txt** in that archive.
   Friday-morning index in the same archive: `docs/getting-started.md`.

| OS | What staff run | Notes |
|---|---|---|
| Windows | `Start-OpenThaiID.bat`, or `install.ps1` for Start Menu | Tray icon. Unsigned. `install.ps1 -Autostart` is opt-in. |
| macOS | `OpenThaiID.app` | Menu-bar helper (`LSUIElement`). Unsigned — right-click Open or remove quarantine. |
| Linux | `./install-linux.sh` then the application menu, or `./start-openthaiid.sh` | Often **no** tray icon on GNOME; helper still runs in the background. Ready page is the UI. |

Start at login is **off** until someone runs `openthaiid autostart enable` or
checks **Start at login** in the tray. Login must not pop a browser tab
(the login item starts `openthaiid tray` without `-open`).

Shortcuts that staff click should use `openthaiid tray -open` so the Ready
page appears once.

Start at login is enough when someone signs in to the clinic PC. For a
kiosk or shared PC that must keep the helper up **without a logged-in
tray**, clinic IT can opt into the always-on path below.

### Always-on (no logged-in tray)

This is **opt-in** and separate from Start Menu / XDG autostart. The process
is `openthaiid serve`: loopback helper, no notification-area icon. Staff still
use `http://127.0.0.1:17891/` after they sign in. Approvals are unchanged.

| OS | Maintainable path | Fallback |
|---|---|---|
| **Windows** | Administrator: `openthaiid service install` or `Install-OpenThaiID-Service.ps1` (SCM, Automatic, LocalSystem). | `Register-OpenThaiID-Task.ps1` (At logon) or `-AtStartup` (admin). Documented Task Scheduler wrapper; prefer the SCM service. |
| **Linux** | `openthaiid service install` writes a systemd **--user** unit and tries `enable --now`. Survive logout: `loginctl enable-linger $USER`. | Package file `openthaiid-system.service` for `/etc/systemd/system/` on kiosk images (`User=` must be able to use the reader). |
| **macOS** | `openthaiid service install` writes a LaunchDaemon plist (`KeepAlive`). Needs root to load. | Login-item tray (`openthaiid autostart enable`) if a signed-in session is enough. |

Do not enable a service by default. Unsigned packages still trigger
SmartScreen / Gatekeeper.

Windows Smart Card service and Linux `pcscd` stay as they are. Session 0 /
a user unit talks to the OS PC/SC daemon; there is still no tray in that
mode. Technician **Operator audit trail** works the same.

```bash
openthaiid tray            # background helper
openthaiid tray -open      # same, and open the Ready page
openthaiid serve           # headless (no tray) for a service / always-on helper
openthaiid status          # helper / reader / card (no personal data)
openthaiid quit
openthaiid autostart enable|disable|status
openthaiid service install|uninstall|start|stop|status
openthaiid audit           # redacted operator trail JSON
```

Loopback and Allow/Deny are unchanged. See [user.md](user.md) and
[clinic.md](clinic.md).

### Windows (clinic)

- Smart Card service running; vendor reader driver only if needed.
- Portable: unzip, double-click `Start-OpenThaiID.bat`.
- Optional: `powershell -ExecutionPolicy Bypass -File install.ps1`
  copies to `%LOCALAPPDATA%\OpenThaiID` and adds a Start Menu shortcut.
  `-DesktopShortcut` and `-Autostart` are extra. `uninstall.ps1` reverses that.
- Always-on without a tray (Administrator):
  `powershell -ExecutionPolicy Bypass -File Install-OpenThaiID-Service.ps1`
  (`openthaiid service install`). Remove with
  `Uninstall-OpenThaiID-Service.ps1`. Task Scheduler fallback:
  `Register-OpenThaiID-Task.ps1` (see always-on table above).
- SmartScreen may warn on an unsigned exe. That is expected. Signing
  checklist: [signing.md](signing.md). Auto-update: `openthaiid update check`.

### Linux (clinic)

```bash
sudo apt update
sudo apt install -y pcscd libpcsclite1
sudo systemctl enable --now pcscd
```

`install-linux.sh` copies the binary to `~/.local/bin` and writes an
application menu entry. Optional `openthaiid.service` is a systemd --user
unit; **do not enable it by default**. Prefer XDG autostart via
`openthaiid autostart enable` when a signed-in session is enough. For
always-on without a tray: `openthaiid service install` then
`loginctl enable-linger $USER`. A system-unit template is
`openthaiid-system.service`.

### macOS (clinic)

Built-in PC/SC is sufficient. The `.app` is **not** notarized. Staff may need:

```bash
xattr -dr com.apple.quarantine OpenThaiID.app
open OpenThaiID.app
```

or right-click → Open. Autostart writes
`~/Library/LaunchAgents/org.openthaiid.helper.plist` (`RunAtLoad`,
`KeepAlive` false so tray **Quit** stays quit until the next login).

## Prerequisites (building from source)

### All platforms

- Go 1.27+ (CI pins `go1.27.1`; `go.mod` matches).
- Node.js 20+ for the web SDK and examples.

### Windows

- Smart Card service running.
- Reader driver if WinSCard cannot bind the device.

### Linux

```bash
sudo apt update
sudo apt install -y pcscd libpcsclite-dev
sudo systemctl enable --now pcscd
```

### macOS

Built-in PC/SC is sufficient. Xcode CLT required for cgo builds.

## Build from source

```bash
go build -o dist/openthaiid ./cmd/openthaiid
./dist/openthaiid version
./dist/openthaiid doctor
```

Package a clinic folder (binary + START-HERE + OS launch files):

```bash
make package
# dist/openthaiid-<version>-<os>-<arch>.tar.gz (zip on Windows)
```

### macOS packaging (Mac host required)

`ebfe/scard` and the Cocoa tray use **cgo**. Darwin archives cannot be
cross-compiled from Linux or Windows (that is why RC linux/windows
artifacts exist and darwin is missing from a Linux builder).

On a Mac:

```bash
make package-macos
# dist/openthaiid-<version>-darwin-<arch>.tar.gz
# contains openthaiid, unsigned OpenThaiID.app, start-openthaiid.sh, START-HERE.txt
```

`make package-macos` builds the **host architecture** (arm64 on Apple
Silicon, amd64 on Intel). It does **not** sign or notarize. Gatekeeper
warnings are expected. See [signing.md](signing.md).

`scripts/package.sh` exits with a clear error if `OS=darwin` is set on a
non-Mac host.

Cross-compilation notes:

- Windows: `CGO_ENABLED=0` is supported by the pinned binding (tray uses
  Win32 syscalls, not cgo).
- Linux/macOS: cgo is required for PC/SC. macOS tray uses Cocoa (cgo).
  Darwin **must** be packaged on a Mac (`make package-macos`).
- Linux tray icon is not shipped (no GTK/AppIndicator dependency). Use the
  Ready page, `.desktop` actions, and `openthaiid status`.

## Run

```bash
./dist/openthaiid              # terminal helper (technicians)
./dist/openthaiid tray         # clinic background helper
./dist/openthaiid tray -open   # same, open Ready page
./dist/openthaiid serve        # headless always-on helper
./dist/openthaiid service status
./dist/openthaiid qualify -report ./openthaiid-qualify.json   # Friday kit
./dist/openthaiid audit        # redacted operator trail JSON
# One-page sheet: docs/friday-checklist.md  Practice: docs/training.md
# Technician: docs/technician.md
```

Default WebSocket endpoint: `ws://127.0.0.1:17891/v1/ws`

Local helper page (device status, Allow/Deny; technician tools hidden):
`http://127.0.0.1:17891/`

```bash
./dist/openthaiid manager   # run and try to open the helper page
./dist/openthaiid -open     # same
```

The default view is for clinic staff. Technicians open
**Technician / developer** (or `#technician`). Approvals are the same
decision as the terminal:

```text
allow <request-id>
deny <request-id>
revoke <origin>
revoke-all
trust
status
help
quit
```

## Web SDK (local)

```bash
cd sdk/web
npm ci
npm test
npm run build
```

## Example

```bash
cd examples/vanilla
npm install
npm run dev
```

Serve the example from an HTTPS origin for real browser testing. HTTP on
`localhost` is limited to development and still requires Manager (or terminal)
approval.

## Clinic ERP

Add `@openthaiid/web` to the existing app — that is the install for the
website. See [clinic.md](clinic.md).

The PC with the reader still needs the local helper running (`openthaiid tray`
or `openthaiid -synthetic`). That is hardware access, not a backend you deploy.

## Privacy reminder

This project is independent and is not affiliated with or endorsed by the
Department of Provincial Administration. Card data returned to an application
is that application’s responsibility.
