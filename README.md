# OpenThaiID

Open-source Thai National ID card reader and secure web bridge built with Go
and PC/SC.

> This is an independent open-source project and is not affiliated with or
> endorsed by the Department of Provincial Administration.

## What it does

A local Go process talks to a PC/SC smart-card reader and exposes selected
Thai ID fields (`citizenId`, `thaiName`, `birthDate` by default; additional
textual fields such as `englishName`, `gender`, `issuer`, `issueDate`,
`expireDate`, and `address` when requested and granted) to browser apps over a
loopback WebSocket — only after the operator approves the website origin and
field set in the terminal.

## Quick start

```bash
# Build
make build

# Check PC/SC (never reads personal fields)
./dist/openthaiid doctor

# Run with real hardware
./dist/openthaiid

# Or run without hardware (synthetic backend + fictional demo card)
./dist/openthaiid -synthetic
```

The synthetic backend seeds one clearly labeled fictional reader and Thai ID
card so the documented connect → approve → read path works from a clean
checkout without hardware.
Default WebSocket endpoint: `ws://127.0.0.1:17891/v1/ws`

### Terminal commands

```text
status | trust | allow <id> | deny <id> | revoke <origin> | revoke-all | quit
```

When a website first connects, the terminal prints the exact origin, requested
fields, request id, and comparison code. Type `allow <id>` to grant in-memory
trust (8 hours or until the bridge stops).

## Browser SDK

```bash
cd sdk/web && npm install && npm test && npm run build
```

```ts
import { ThaiID } from "@openthaiid/web";

const thaiID = new ThaiID({
  fields: ["thaiName", "birthDate"],
  // Optional extra textual fields (each must be granted):
  // "englishName", "gender", "issuer", "issueDate", "expireDate", "address"
});

thaiID.on("approvalRequired", ({ comparisonCode }) => {
  console.log("Approve in terminal, code", comparisonCode);
});

// Call connect() from a click so the browser can prompt for local network access.
await thaiID.connect();

const result = await thaiID.read();
console.log(result.fields);

await thaiID.disconnect();
```

Open `examples/vanilla/index.html` via the built SDK for a minimal demo
(serve from HTTPS for real browser tests; localhost HTTP is development-only).

## Architecture

```text
Browser (HTTPS page)
  └─ WebSocket ws://127.0.0.1:17891/v1/ws
       └─ OpenThaiID bridge (Go)
            ├─ Origin + field grants (terminal approval)
            ├─ Reader manager (card generations)
            └─ PC/SC adapter → Thai ID card
```

| Package | Role |
|---|---|
| `cmd/openthaiid` | CLI, signals, wiring |
| `internal/card` | Thai profile, APDU engine, TIS-620 parsers |
| `internal/reader` | Inventory, generations, single-read slot |
| `internal/pcsc` | OS PC/SC + synthetic backends |
| `internal/auth` | Grants, sessions, pending approvals |
| `internal/protocol` | Protocol v1 envelopes |
| `internal/bridge` | Loopback HTTP + WebSocket dispatch |
| `sdk/web` | `@openthaiid/web` |

## Security

- Listens only on `127.0.0.1`; rejects non-loopback peers and Host rebinding.
- Validates browser `Origin` (HTTPS always; HTTP loopback for development).
- Terminal approval per origin + exact field set; trust is memory-only.
- One active read; results are unicast; card-session generations prevent stale data.
- Never logs citizen IDs, names, dates, raw APDUs, or tokens.

See [docs/security.md](docs/security.md) and [docs/protocol.md](docs/protocol.md).

## Documentation

| Document | Contents |
|---|---|
| [docs/architecture.md](docs/architecture.md) | Process model and concurrency |
| [docs/security.md](docs/security.md) | Threat model |
| [docs/protocol.md](docs/protocol.md) | Protocol v1 |
| [docs/provenance.md](docs/provenance.md) | APDU sources and licenses |
| [docs/compatibility.md](docs/compatibility.md) | OS/browser/reader matrix |
| [docs/acceptance.md](docs/acceptance.md) | Acceptance gates |
| [docs/installation.md](docs/installation.md) | Runtime prerequisites |
| [docs/dependency-audit.md](docs/dependency-audit.md) | Binding audit checklist |

## Tests

```bash
make test          # Go unit + integration (synthetic hardware)
make race          # race detector
make test-repeat   # bridge/reader tests ×50 (local determinism; full CI too)
cd sdk/web && npm test
make acceptance    # node-pty + Playwright (Chrome/Edge) against real bridge
```

GitHub Actions is split to conserve included minutes: **pull requests** run
Ubuntu Go tests (`-count=5` on formerly flaky packages), linux/amd64 build,
the Web SDK, and Chrome acceptance (allow/deny/unanswered). The **full**
five-target build + Chrome×Linux/macOS/Windows + Edge×Windows matrix, and
`-count=50`, run on push to `dev`/`main` and via **workflow_dispatch**.
Docs-only changes skip CI. See [docs/acceptance.md](docs/acceptance.md).

Browser acceptance uses Playwright channels (`chrome`, `msedge`) and a real
PTY via node-pty for terminal approval. Approval policies: allow, deny,
unanswered. See [acceptance/README.md](acceptance/README.md).

Hardware acceptance (ACR39U on Windows/Linux/macOS) is still required before
a production release claim. See
[docs/acceptance.md](docs/acceptance.md) for the current evidence status.

## License

Apache-2.0. See [LICENSE](LICENSE).
