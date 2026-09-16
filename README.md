# OpenThaiID

Open-source Thai National ID card reader and secure web bridge built with Go
and PC/SC.

> This is an independent open-source project and is not affiliated with or
> endorsed by the Department of Provincial Administration.

## What it does

A **JavaScript library** (`@openthaiid/web`) that an existing clinic ERP page
calls to autofill Thai National ID fields. There is no OpenThaiID website or
backend to deploy.

A small local helper on the PC talks to a PC/SC reader and exposes selected
Thai ID fields (`citizenId`, `thaiName`, `birthDate` by default; additional
textual fields such as `englishName`, `gender`, `issuer`, `issueDate`,
`expireDate`, and `address` when requested and granted) over a loopback
WebSocket — only after the operator Allows the website origin and field set.

See [docs/clinic.md](docs/clinic.md): add the library to the ERP, check the
reader from the same page, call `autofillThaiID(config)`.

## Quick start

```bash
# Build
make build

# Check PC/SC (never reads personal fields)
./dist/openthaiid doctor

# Run with real hardware
./dist/openthaiid

# Run and open the local Manager in a browser
./dist/openthaiid manager

# Or run without hardware (synthetic backend + fictional demo card)
./dist/openthaiid -synthetic
```

The synthetic backend seeds one clearly labeled fictional reader and Thai ID
card so the documented connect → approve → read path works from a clean
checkout without hardware.

Local helper (hardware): `ws://127.0.0.1:17891/v1/ws`
Optional operator Allow UI: `http://127.0.0.1:17891/`

### Terminal commands

```text
status | trust | allow <id> | deny <id> | revoke <origin> | revoke-all | quit
```

When a website first connects, the terminal prints the exact origin, requested
fields, request id, and comparison code. Type `allow <id>` to grant in-memory
trust (8 hours or until the bridge stops).

## Browser SDK (the library)

Add `@openthaiid/web` to the existing ERP. Check the reader and autofill from
the same page — nothing extra to install for the website:

For a HTML form, one click is enough — `autofillThaiID` connects (from that
gesture), waits for terminal approval, reads, fills inputs, then disconnects:

```html
<form id="kyc">
  <input data-openthaiid="citizenId" />
  <input data-openthaiid="thaiName" />
  <input type="date" data-openthaiid="birthDate" />
</form>
<button type="button" id="fill">Autofill from Thai ID</button>
```

```js
import { autofillThaiID, mountThaiIDStatus } from "@openthaiid/web";

mountThaiIDStatus(document.getElementById("reader-status"));

document.getElementById("fill").addEventListener("click", async () => {
  await autofillThaiID({
    root: document.getElementById("kyc"),
  });
});
```

Selectors work too: `await autofillThaiID({ citizenId: "#cid", thaiName: "#name", birthDate: "#dob" })`.
Call it from a user gesture. Origin + field Allow still happens on this PC
(the ERP cannot approve itself). See `sdk/web/README.md` and
`examples/vanilla/autofill.html`.

Clinic / ERP screens should pass a **settings object** (`request` allowlist +
`map` of slots to selectors) from the Create Patient click — same helper, no
SDK fork. Copy-paste: `examples/clinic-patient/`.

The `ThaiID` class is unchanged when you need connect / read / status yourself:

```ts
import { ThaiID } from "@openthaiid/web";

const thaiID = new ThaiID({
  fields: ["thaiName", "birthDate"],
  // Optional extra textual fields (each must be granted):
  // "englishName", "gender", "issuer", "issueDate", "expireDate", "address"
});

thaiID.on("approvalRequired", ({ comparisonCode }) => {
  console.log("Allow this origin on the PC, code", comparisonCode);
});

// Call connect() from a click so the browser can prompt for local network access.
await thaiID.connect();

const result = await thaiID.read();
console.log(result.fields);

await thaiID.disconnect();
```

React apps can use `@openthaiid/react` instead of wiring events by hand:

```tsx
import { useThaiID } from "@openthaiid/react";

function Panel() {
  const { connection, fields, approvalRequired, connect, read, disconnect } = useThaiID({
    fields: ["citizenId", "thaiName", "birthDate"],
  });
  return (
    <>
      {approvalRequired && <p>Approve code {approvalRequired.comparisonCode}</p>}
      <button type="button" onClick={() => void connect()}>Connect</button>
      <button type="button" onClick={() => void read()}>Read</button>
      <button type="button" onClick={() => void disconnect()}>Disconnect</button>
      <pre>{fields ? JSON.stringify(fields, null, 2) : connection}</pre>
    </>
  );
}
```

```bash
cd sdk/web && npm install && npm test && npm run build
cd sdk/react && npm install && npm test && npm run build
```

Open `examples/clinic-patient/` for a New patient form with configurable
fields, `examples/vanilla/autofill.html` for the generic form helper, or
`examples/vanilla/index.html` for Connect / Read / Disconnect
(serve from HTTPS for real browser tests; localhost HTTP is development-only).
Copy-paste React usage is in `examples/react/README.md` and `sdk/react/README.md`.

## Architecture

```text
Browser (HTTPS ERP page)
  └─ WebSocket ws://127.0.0.1:17891/v1/ws
       └─ OpenThaiID on the clinic PC (Go)
            ├─ Local Manager http://127.0.0.1:17891/
            ├─ Origin + field grants (Manager or terminal approval)
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
| `sdk/react` | `@openthaiid/react` hooks wrapping the web SDK |

## Security

- Listens only on `127.0.0.1`; rejects non-loopback peers and Host rebinding.
- Validates browser `Origin` (HTTPS always; HTTP loopback for development).
- Operator approval per origin + exact field set (Manager or terminal); trust is memory-only.
- Local Manager is loopback HTML; mutating APIs require a CSRF token and the manager origin. No CORS.
- One active read; results are unicast; card-session generations prevent stale data.
- Never logs citizen IDs, names, dates, raw APDUs, or tokens. The Manager does not display personal field values.

See [docs/security.md](docs/security.md) and [docs/protocol.md](docs/protocol.md).

## Documentation

| Document | Contents |
|---|---|
| [docs/clinic.md](docs/clinic.md) | Clinic IT + ERP: install on the PC, no cloud stack |
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
cd sdk/react && npm test
make acceptance    # node-pty + Playwright (Chrome/Edge) against real bridge
```

GitHub Actions is split to conserve included minutes: **pull requests** run
Ubuntu Go tests (`-count=5` on formerly flaky packages), linux/amd64 build,
the Web + React SDKs, and Chrome acceptance (allow/deny/unanswered). The **full**
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
