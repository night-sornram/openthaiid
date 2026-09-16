# React example

Copy-paste starter that uses `@openthaiid/react`. There is no bundler in this
folder; see `sdk/react/README.md` for install and build commands.

```tsx
import { useThaiID } from "@openthaiid/react";

export function App() {
  const {
    connection,
    fields,
    approvalRequired,
    error,
    connect,
    read,
    disconnect,
  } = useThaiID({
    fields: ["citizenId", "thaiName", "birthDate"],
  });

  return (
    <main>
      <p>Status: {connection}</p>
      {approvalRequired && (
        <p>
          Approve in the terminal — request {approvalRequired.approvalId}, code{" "}
          {approvalRequired.comparisonCode}
        </p>
      )}
      {error && <p>{error.code}: {error.message}</p>}
      <p>
        <button type="button" onClick={() => void connect()}>
          Connect
        </button>
        <button type="button" onClick={() => void read()} disabled={connection !== "connected"}>
          Read
        </button>
        <button type="button" onClick={() => void disconnect()}>
          Disconnect
        </button>
      </p>
      <pre>{fields ? JSON.stringify(fields, null, 2) : "No card data yet."}</pre>
    </main>
  );
}
```

Start the bridge (`./dist/openthaiid -synthetic` or real PC/SC), call Connect
from a click, then allow the origin in the terminal. Extra textual fields must
be listed in `useThaiID({ fields })` and granted; they are not part of the
default set.

For a Create Patient form, pass the same `request` + `map` config used by
`autofillThaiID` from a button (`examples/clinic-patient/`). You do not need
a React-specific SDK fork; `useThaiIDAutofill(config)` is optional.
