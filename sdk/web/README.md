# `@openthaiid/web`

JavaScript library for Thai ID autofill in an existing ERP page. Importing
the module is SSR-safe. There is no OpenThaiID frontend or backend to
deploy — add this package to the app you already have.

`connect()` / `autofillThaiID()` / `probeThaiID()` still need a browser.
Autofill and connect must be called from a visible user gesture (so the
browser can prompt for local-network access).

Origin + field Allow is still required before personal fields are returned.
This package does not bypass the local helper. Portrait is optional: request
`portrait` and Allow it on the helper, then use `portraitDataUrl()` or
`portraitBlob()`. Do not log photo bytes.

## Install

```bash
cd sdk/web && npm ci && npm test && npm run build
```

Until the package is published, import the built `dist/index.js` (as the vanilla
examples do) or depend on this folder.

## Check the reader (no personal data)

```js
import { probeThaiID, mountThaiIDStatus } from "@openthaiid/web";

mountThaiIDStatus(document.getElementById("reader-status"));

const probe = await probeThaiID();
// probe.reachable, probe.backend.status, probe.readers[].presence
// summarizeProbe(probe) → Ready / not ready in plain language
```

This talks to the local helper with `bridge.probe`. It never returns citizen
ID or names. Card reads still need Allow. Default status copy is for clinic
staff; `{ mode: "advanced" }` adds technician details.

## Form autofill (plain JavaScript)

Mark inputs and call `autofillThaiID` from a button click:

```html
<form id="kyc">
  <input data-openthaiid="citizenId" />
  <input data-openthaiid="thaiName" />
  <input type="date" data-openthaiid="birthDate" />
</form>
<button type="button" id="fill">Autofill from Thai ID</button>
```

```js
import { autofillThaiID } from "@openthaiid/web";

document.getElementById("fill").addEventListener("click", async () => {
  await autofillThaiID({
    root: document.getElementById("kyc"),
    onApprovalRequired: ({ comparisonCode }) => {
      console.log("Approve in the terminal, code", comparisonCode);
    },
  });
});
```

Or pass CSS selectors (no attributes needed):

```js
await autofillThaiID({
  citizenId: "#citizen-id",
  thaiName: "#full-name",
  birthDate: "#dob",
  gender: "#gender",
  address: "#address",
    onApprovalRequired: ({ comparisonCode }) => {
      status.textContent = `Allow this origin on the PC, code ${comparisonCode}`;
    },
});
```

The helper connects, waits for approval, reads, writes string values into the
inputs, then disconnects (pass `disconnect: false` or `client` to keep the
session). It requests only the fields implied by the slots (least privilege).

Default formatting: citizen ID digits, `fullName` / `fullAddress`, ISO dates
(`YYYY-MM-DD`; `input[type=date]` always gets a complete ISO date), gender
labels `male` / `female`. Nested slots work (`thaiName.givenName`,
`address.province`, `gender.code`, `birthDate.year`).

See `examples/vanilla/autofill.html` for a copy-paste page. With
`./dist/openthaiid -synthetic`, the seeded fictional card is citizen ID
`1234567890121`, Thai name `นาย สมชาย ใจดี`, birth `1990-04-12`.

## Clinic / ERP settings object

For a Create Patient screen, keep one config object (clinic admin can persist
`request`) and call it from the button:

```js
import { autofillThaiID } from "@openthaiid/web";

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
  dateFormat: "iso",
  genderLabels: { male: "male", female: "female" },
  onApprovalRequired: ({ comparisonCode, approvalId }) => {
    showBanner(`Approve in the terminal — request ${approvalId}, code ${comparisonCode}`);
  },
  onError: (err) => {
    showBanner(`${err.code ?? "ERROR"}: ${err.message ?? err}`);
  },
};

fillBtn.addEventListener("click", async () => {
  await autofillThaiID(clinicPatientAutofill);
});
```

- `request` — allowlist of protocol fields (not nested slots). The session
  asks only for fields that also have a mapped or `data-openthaiid` target.
- `map` — alias of `fields` (slot → selector). Phone / HN stay off the map.
- `onApprovalRequired` / `onError` — ERP chrome; the promise still rejects.
- Portrait is optional. Request `portrait` and map it to an `<img>` (data URL)
  or consume `result.fields.portrait` via `portraitDataUrl` / `portraitBlob`.
  Do not call this on page load; do not log photo bytes.

Worked example: `examples/clinic-patient/`.

## Full connect / read / status

```ts
import { ThaiID } from "@openthaiid/web";

const thaiID = new ThaiID({
  fields: ["citizenId", "thaiName", "birthDate"],
  // Optional extra textual fields (each must be granted):
  // "englishName", "gender", "issuer", "issueDate", "expireDate", "address"
});

thaiID.on("approvalRequired", ({ comparisonCode }) => {
  console.log("Approve in terminal, code", comparisonCode);
});

await thaiID.connect();
const result = await thaiID.read();
await thaiID.disconnect();
```

Default grant remains `citizenId`, `thaiName`, `birthDate`. Add `"portrait"`
only when the screen needs the photo.
