# `@openthaiid/web`

JavaScript library for Thai ID autofill in an existing ERP page. Importing
the module is SSR-safe. There is no OpenThaiID frontend or backend to
deploy — add this package to the app you already have.

`connect()` / `autofillThaiID()` / `probeThaiID()` still need a browser.
Autofill and connect must be called from a visible user gesture (so the
browser can prompt for local-network access).

**Browsers:** Ready (`http://127.0.0.1:17891/`) works in current Firefox,
Safari 16+, and Chrome/Edge. Fill from an **http** clinic page works in all
three. Fill from an **https** clinic page works in Chrome/Edge; Firefox and
Safari block `ws://127.0.0.1` (mixed content) unless optional helper TLS is
on (`wss://127.0.0.1:17892`). Without a trusted local cert the library
returns `HTTPS_WS_BLOCKED` or `TLS_UNTRUSTED` instead of “helper not
running”. See `docs/compatibility.md` and `docs/loopback-tls.md`. Do not
disable mixed-content checks.

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
// probe.reachable, probe.backend.status, probe.readers[].presence, probe.readers[].selected
// summarizeProbe(probe) → พร้อม / ยังไม่พร้อม (Thai default; pass "en" for English)
// Two readers with none selected: not Ready until staff pick on the Ready page.
```

This talks to the local helper with `bridge.probe`. It never returns citizen
ID or names. Card reads still need Allow. Default status copy is for clinic
staff; `{ lang: "en" }` for English; `{ mode: "advanced" }` adds technician details.

`mountThaiIDStatus` keeps probing and, if the helper restarts mid-session,
shows Thai **ตัวช่วยหลุดการเชื่อมต่อ** then **ตัวช่วยกลับมาแล้ว** without a
page reload. After sleep/wake it re-probes (`pageshow` / visibility /
`online`) without reading card fields. While Ready (card present) it polls
faster so a pulled card or unplug does not leave **พร้อม**. If the card
service drops (`PCSC_UNAVAILABLE`) it leaves Ready and says **พร้อม** again
after restore. There is an optional **ทดสอบการเชื่อมต่อ** button (probe only).
`waitForThaiID()` / `watchThaiID()` are the same loop without DOM. `autofillThaiID`
waits for the helper to come back before opening a session; it never auto-repeats
a card read.

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
Values are written only after a complete successful `card.read`. If the card
is pulled or the reader is unplugged mid-read (`CARD_REMOVED` /
`READER_REMOVED`, also `READ_FAILED` / `READ_TIMEOUT`), mapped fields are
cleared so a previous patient’s values are not mixed with a failed Fill.
Phone / HN and other unmapped controls stay. Thai staff `message` tells
them to wait for Ready and Fill again (`{ lang: "en" }` for English). Codes
do not change.

Default formatting: citizen ID digits, `fullName` / `fullAddress`, ISO dates
(`YYYY-MM-DD`; `input[type=date]` always gets a complete ISO date), gender
labels `male` / `female`. Nested slots work (`thaiName.givenName`,
`address.province`, `gender.code`, `birthDate.year`).

See `examples/vanilla/autofill.html` for a copy-paste page. With
`./dist/openthaiid -synthetic`, the seeded fictional card is citizen ID
`1234567890121`, Thai name `นาย สมชาย ใจดี`, birth `1990-04-12`.

## Clinic / ERP settings object

For a Create Patient screen, keep one JSON config (`request` + `map`) and call
it from the button. Schema:
`schema/autofill-clinic-config.schema.json` (package export
`@openthaiid/web/schema/autofill-clinic-config.json`). Samples:
`examples/clinic-patient/kyc-minimal.json` and `clinic-full.json`.
Hospital ERP drop-in: `docs/erp-integration.md`.

```js
import { autofillThaiID, formatStaffError, parseAutofillClinicConfig, staffApprovalGuidance } from "@openthaiid/web";
import clinicJson from "./openthaiid-clinic-config.json";

const clinicPatientAutofill = parseAutofillClinicConfig(clinicJson);

fillBtn.addEventListener("click", async () => {
  await autofillThaiID({
    ...clinicPatientAutofill,
    onApprovalRequired: (info) => {
      const g = staffApprovalGuidance(info); // Thai default; pass "en" for English
      showBanner(`${g.message} ${g.next}`);
    },
    onError: (err) => {
      showBanner(`${formatStaffError(err)} (${err.code ?? "ERROR"})`);
    },
  });
});
```

- `request` — allowlist of protocol fields (not nested slots). The session
  asks only for fields that also have a mapped or `data-openthaiid` target.
- `map` — alias of `fields` (slot → selector). Phone / HN stay off the map.
  JSON configs use `map` with CSS selector strings only.
- `onApprovalRequired` / `onError` — **code-only** ERP chrome; they must not
  appear in the JSON file. The promise still rejects.
  Default Fill error **messages** are Thai staff guidance; `err.code` is
  unchanged. Pass `{ lang: "en" }` or `formatStaffError(err, "en")`.
- Type export: `AutofillClinicConfig`. Runtime: `parseAutofillClinicConfig`.
- Portrait is optional. Request `portrait` and map it to an `<img>` (data URL)
  or consume `result.fields.portrait` via `portraitDataUrl` / `portraitBlob`.
  Do not call this on page load; do not log photo bytes.

Worked example: `examples/clinic-patient/` (settings panel download/upload).

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
