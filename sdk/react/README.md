# `@openthaiid/react`

React hooks wrapping the `@openthaiid/web` **library**. Drop them into an
existing ERP screen. This package does not talk to the WebSocket itself.

Call `connect()` from a click so the browser can prompt for local-network
access. Importing the module is SSR-safe; `connect()` still requires a browser.

## Install

```bash
npm install @openthaiid/react @openthaiid/web react
```

Until the packages are published, depend on this repo path (or pack locally):

```bash
cd sdk/web && npm ci && npm run build
cd ../react && npm ci && npm test && npm run build
```

## Usage

```tsx
import { useThaiID } from "@openthaiid/react";

export function ThaiIDPanel() {
  const {
    connection,
    status,
    fields,
    approvalRequired,
    error,
    connect,
    read,
    disconnect,
  } = useThaiID({
    fields: ["citizenId", "thaiName", "birthDate"],
    // Extra fields each need a new grant on the PC:
    // "englishName", "gender", "issuer", "issueDate", "expireDate", "address", "portrait"
  });

  return (
    <section>
      <p>Status: {connection}</p>
      {approvalRequired && (
        <p>
          Approve request {approvalRequired.approvalId} in the terminal (code{" "}
          {approvalRequired.comparisonCode}).
        </p>
      )}
      {error && <p>{error.code}</p>}
      <button type="button" onClick={() => void connect()} disabled={connection !== "disconnected"}>
        Connect
      </button>
      <button type="button" onClick={() => void read()} disabled={connection !== "connected"}>
        Read
      </button>
      <button type="button" onClick={() => void disconnect()} disabled={connection === "disconnected"}>
        Disconnect
      </button>
      <pre>{fields ? JSON.stringify({ status, fields }, null, 2) : "No card data yet."}</pre>
    </section>
  );
}
```

### AbortSignal

`connect` and `read` accept the same `{ signal?: AbortSignal }` options as
`ThaiID`. A pre-aborted or in-flight abort becomes `ThaiIDError` `CANCELLED`.

```tsx
const ac = new AbortController();
await connect({ signal: ac.signal });
await read({ signal: ac.signal });
ac.abort();
```

### Reuse a client

Pass `client` to subscribe to an existing `ThaiID` instance (tests, or sharing
one connection across components):

```tsx
const client = useMemo(() => new ThaiID({ fields: ["citizenId"] }), []);
const { connect, fields } = useThaiID({ client });
```

The hook disconnects that client when the component unmounts.

## API

| Return | Meaning |
|---|---|
| `connection` | `disconnected` / `connecting` / `awaiting-approval` / `connected` / `reconnecting` |
| `status` | Last `BridgeStatus` (backend + readers) |
| `fields` | Last successful `read()` values; cleared on disconnect, card removal, or revoke |
| `approvalRequired` | Terminal comparison code while waiting; `null` once connected |
| `error` | Last `ThaiIDError` from connect/read or bridge events |
| `connect` / `read` / `disconnect` | AbortSignal-friendly wrappers around `@openthaiid/web` |

Default field grant remains `citizenId`, `thaiName`, `birthDate`. Portrait is
optional and off by default. Fill errors from `autofillThaiID` are Thai staff
guidance by default; `error.code` is unchanged (`formatStaffError`, `{ lang: "en" }`).

## Form autofill (same config as `@openthaiid/web`)

Clinic ERP screens should keep a JSON config (`request` + `map`) and call it
from a button — do not connect on render. Parse with
`parseAutofillClinicConfig`; attach callbacks in code. Samples live in
`examples/clinic-patient/` (`kyc-minimal.json`, `clinic-full.json`).

```tsx
import { autofillThaiID, parseAutofillClinicConfig, staffApprovalGuidance } from "@openthaiid/react";
import clinicJson from "./openthaiid-clinic-config.json";

const clinicPatientAutofill = parseAutofillClinicConfig(clinicJson);

<button
  type="button"
  onClick={() =>
    void autofillThaiID({
      ...clinicPatientAutofill,
      onApprovalRequired: (info) => {
        const g = staffApprovalGuidance(info);
        setBanner(`${g.message} ${g.next}`);
      },
    })
  }
>
  กรอกข้อมูล
</button>
```

`useThaiIDAutofill(config)` is a thin wrapper around that same object if you
want `filling` / `approvalRequired` / `error` state. It does not read until
`autofill()` runs from a click. Drop-in ERP steps: `docs/erp-integration.md`.
