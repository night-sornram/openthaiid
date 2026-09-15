# Installation (development)

## Prerequisites

### All platforms

- Go 1.22+ (CI pins the toolchain version).
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

Cross-compilation notes:

- Windows: `CGO_ENABLED=0` is supported by the pinned binding.
- Linux/macOS: cgo is required for PC/SC.

## Run

```bash
./dist/openthaiid
```

Default WebSocket endpoint: `ws://127.0.0.1:17891/v1/ws`

Approvals are handled in the terminal:

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
npm install
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
`localhost` is limited to development and still requires terminal approval.

## Privacy reminder

This project is independent and is not affiliated with or endorsed by the
Department of Provincial Administration. Card data returned to an application
is that application’s responsibility.
