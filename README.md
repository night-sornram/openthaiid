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
fields such as `englishName`, `gender`, `issuer`, `issueDate`,
`expireDate`, `address`, and optional `portrait` when requested and granted) over a loopback
WebSocket — only after the operator Allows the website origin and field set.

See [docs/clinic.md](docs/clinic.md): add the library to the ERP, check the
reader from the same page, call `autofillThaiID(config)`. Drop-in JSON
(`request` + `map`, schema + samples): [docs/erp-integration.md](docs/erp-integration.md).

Clinic staff: [docs/user.md](docs/user.md) (Thai-first Ready lights; tray /
Start menu, not a developer terminal). Practice without a reader:
[docs/training.md](docs/training.md). Friday with ACR39U:
[docs/friday-checklist.md](docs/friday-checklist.md).
Technicians: [docs/technician.md](docs/technician.md).
Clinic IT packaging: [docs/installation.md](docs/installation.md).
Always-on without a tray: `openthaiid service install`.

## Quick start

```bash
# Build
make build

# Check PC/SC (never reads personal fields)
./dist/openthaiid doctor

# Friday qualification kit (no personal fields; synthetic smoke or real reader)
./dist/openthaiid qualify -synthetic -e2-count 3 -e3-count 2 -report dist/openthaiid-qualify.json
# With ACR39U + card inserted:
./scripts/hardware-qualify.sh
./scripts/hardware-qualify.sh --operator   # E3 / E4–E7 / M1 prompts; pass|fail|skip only
# Windows package: Qualify-OpenThaiID.bat

# Run with real hardware (terminal; technicians)
./dist/openthaiid

# Clinic background helper (tray on Windows/macOS; background on Linux)
./dist/openthaiid tray -open

# Always-on headless helper (Windows service / systemd / Task Scheduler)
./dist/openthaiid serve
./dist/openthaiid service status

# Run and open the local helper page (clinic-staff view; technician tools hidden)
./dist/openthaiid manager

# Or run without hardware (practice reader + fictional demo card)
./dist/openthaiid -synthetic
```

The synthetic backend seeds one clearly labeled fictional reader and Thai ID
card so the documented connect → approve → read path works from a clean
checkout without hardware. Tests add a second synthetic reader when they need
the multi-reader picker; default `-synthetic` stays the single-reader auto
path. Practice the picker with `OPENTHAIID_SYNTHETIC_READERS=2`.

Local helper (hardware): `ws://127.0.0.1:17891/v1/ws`
Optional helper page: `http://127.0.0.1:17891/` — default view is Thai
Ready lights (พร้อม) for clinic staff. English toggle on that page.
Technician / developer tools stay English (`#technician`).

### Terminal commands

```text
status | trust | allow <id> | deny <id> | revoke <origin> | revoke-all | quit
```

When a website first connects, the helper page shows the origin, requested
fields, and comparison code. Allow only the clinic page staff are using
(the website still cannot approve itself). Terminal `allow <id>` remains
available for technicians.

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
// Thai Ready lights by default. { lang: "en" } for English. { mode: "advanced" } for technicians.

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
SDK fork. Copy-paste: `examples/clinic-patient/`. JSON Schema and samples:
[docs/erp-integration.md](docs/erp-integration.md).

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
            ├─ Tray / background helper (optional; default for clinic packages)
            ├─ Helper page http://127.0.0.1:17891/ (staff view + hidden technician tools)
            ├─ Origin + field grants (Manager or terminal approval)
            ├─ Reader manager (card generations)
            └─ PC/SC adapter → Thai ID card
```

| Package | Role |
|---|---|
| `cmd/openthaiid` | CLI, tray, serve/service, audit, qualify, autostart |
| `internal/desktop` | Tray host, login items, staff-safe status |
| `internal/card` | Thai profile, APDU engine, TIS-620 parsers |
| `internal/reader` | Inventory, generations, single-read slot |
| `internal/pcsc` | OS PC/SC + synthetic backends |
| `internal/auth` | Grants, sessions, pending approvals |
| `internal/audit` | Redacted operator trail (codes, origins, presence) |
| `internal/protocol` | Protocol v1 envelopes |
| `internal/bridge` | Loopback HTTP + WebSocket dispatch |
| `internal/qualify` | Friday E1–E7 / M1 kit (redacted JSON report) |
| `internal/update` | GitHub Releases unsigned auto-update |
| `sdk/web` | `@openthaiid/web` |
| `sdk/react` | `@openthaiid/react` hooks wrapping the web SDK |

## Security

- Listens only on `127.0.0.1`; rejects non-loopback peers and Host rebinding.
- Validates browser `Origin` (HTTPS always; HTTP loopback for development).
- Operator approval per origin + exact field set (Manager or terminal); trust is memory-only.
- Local Manager is loopback HTML; mutating APIs require a CSRF token and the manager origin. No CORS.
- `POST /v1/helper/shutdown` is loopback-only and ignores foreign website `Origin` headers.
- One active read; results are unicast; card-session generations prevent stale data.
- Never logs citizen IDs, names, dates, raw APDUs, or tokens. The helper page
  does not display personal field values. Technician diagnostics copy the same
  rule (versions, OS, card-service state, last error *codes* — not card data).

See [docs/security.md](docs/security.md) and [docs/protocol.md](docs/protocol.md).

## Documentation

| Document | Contents |
|---|---|
| [docs/clinic.md](docs/clinic.md) | Clinic IT + ERP: install on the PC, no cloud stack |
| [docs/erp-integration.md](docs/erp-integration.md) | Drop-in JSON config + schema for an existing hospital/clinic ERP |
| [docs/user.md](docs/user.md) | Clinic staff: start helper (tray), plug reader, Fill |
| [docs/training.md](docs/training.md) | Practice Fill with `-synthetic` before ACR39U arrives |
| [docs/friday-checklist.md](docs/friday-checklist.md) | One-page Friday: Ready → qualify → Fill → what to send |
| [docs/technician.md](docs/technician.md) | Debug checklist when Fill fails (audit JSON, service, qualify) |
| [docs/architecture.md](docs/architecture.md) | Process model and concurrency |
| [docs/security.md](docs/security.md) | Threat model |
| [docs/protocol.md](docs/protocol.md) | Protocol v1 |
| [docs/provenance.md](docs/provenance.md) | APDU sources and licenses |
| [docs/compatibility.md](docs/compatibility.md) | OS/browser/reader matrix |
| [docs/acceptance.md](docs/acceptance.md) | Acceptance gates |
| [docs/hardware-qualification.md](docs/hardware-qualification.md) | Friday E1–E7 / M1 operator kit (no personal data) |
| [docs/installation.md](docs/installation.md) | Clinic package + source build (`make package-macos` on a Mac) |
| [docs/signing.md](docs/signing.md) | Unsigned packages; future Authenticode / Developer ID checklist |
| [docs/dependency-audit.md](docs/dependency-audit.md) | Binding audit checklist |

## Tests

```bash
make test          # Go unit + integration (synthetic hardware)
make smoke-helper  # Linux tray/background + package-free status/quit/autostart
make qualify-smoke # synthetic E1–E3 kit (no personal fields in the report)
make smoke-package # native archive + darwin cross-compile refusal on Linux
make package       # clinic archive (START-HERE + OS launch files)
make package-macos # Darwin archive; Mac host only; unsigned / not notarized
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
a production release claim. The operator kit is
[docs/hardware-qualification.md](docs/hardware-qualification.md). See
[docs/acceptance.md](docs/acceptance.md) for the current evidence status.

## License

Apache-2.0. See [LICENSE](LICENSE).
