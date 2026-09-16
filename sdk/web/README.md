# `@openthaiid/web`

Browser SDK for the OpenThaiID local bridge. Importing the module is SSR-safe;
`connect()` and `autofillThaiID()` still require a browser and must be called
from a visible user gesture (so the browser can prompt for local-network access).

Terminal origin + field approval is always required. This package does not
bypass the bridge, and it does not read portraits/photos.

## Install

```bash
cd sdk/web && npm ci && npm test && npm run build
```

Until the package is published, import the built `dist/index.js` (as the vanilla
examples do) or depend on this folder.

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
    status.textContent = `Approve in the terminal, code ${comparisonCode}`;
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

Default grant remains `citizenId`, `thaiName`, `birthDate`. Portrait is not
implemented.
