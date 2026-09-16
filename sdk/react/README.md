# `@openthaiid/react`

React hooks for the OpenThaiID local bridge. This package wraps
[`@openthaiid/web`](../web) and does not talk to the WebSocket itself.

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
    // Extra textual fields each need a new terminal grant:
    // "englishName", "gender", "issuer", "issueDate", "expireDate", "address"
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
not implemented.

## Form autofill (same config as `@openthaiid/web`)

Clinic ERP screens should keep a settings object and call it from a button —
do not connect on render:

```tsx
import { autofillThaiID } from "@openthaiid/web";
// or: import { autofillThaiID, useThaiIDAutofill } from "@openthaiid/react";

const clinicPatientAutofill = {
  request: ["citizenId", "thaiName", "birthDate", "gender", "address"],
  map: {
    citizenId: "#patient-cid",
    "thaiName.givenName": "#patient-given",
    "thaiName.familyName": "#patient-family",
    birthDate: "#patient-dob",
    gender: "#patient-gender",
    address: "#patient-address",
  },
  onApprovalRequired: ({ comparisonCode }) => {
    setBanner(`Approve in the terminal, code ${comparisonCode}`);
  },
};

<button type="button" onClick={() => void autofillThaiID(clinicPatientAutofill)}>
  Fill from Thai ID
</button>
```

`useThaiIDAutofill(config)` is a thin wrapper around that same object if you
want `filling` / `approvalRequired` / `error` state. It does not read until
`autofill()` runs from a click. Copy-paste clinic example:
`examples/clinic-patient/`.
