# Clinic ERP — create patient

The ERP is a normal web page. OpenThaiID is a **library** you call from that
page — not a product to install, and not a backend/frontend to deploy.

```js
import { autofillThaiID, mountThaiIDStatus } from "@openthaiid/web";
import { clinicPatientAutofill } from "./settings.js";

mountThaiIDStatus(document.getElementById("thaiid-status"));

fillBtn.addEventListener("click", async () => {
  await autofillThaiID(clinicPatientAutofill());
});
```

The flow stays **manual**: a user clicks **กรอกข้อมูล / Fill from Thai ID**.
Staff copy on this example is Thai by default (`?lang=en` for English).
Practice without hardware: [docs/training.md](../../docs/training.md)
(`openthaiid -synthetic` + this page).

The helper connects from that gesture. Origin + field Allow still happens on this PC
(the page cannot approve itself). There is no silent background read.
Portrait is optional and **off by default**. Tick `portrait` in **request**
to include it in the grant; the example shows an `<img>` preview only.
Production ERPs should keep the default grant without portrait unless the
screen needs the photo. Do not `console.log` photo bytes.

## Integrator snippet

On Create Patient (or a dedicated fill button):

```js
import { autofillThaiID, formatStaffError, mountThaiIDStatus, staffApprovalGuidance } from "@openthaiid/web";

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
  discover: false,
  onApprovalRequired: (info) => {
    const g = staffApprovalGuidance(info);
    showBanner(`${g.message} ${g.next}`);
  },
  onError: (err) => {
    showBanner(`${formatStaffError(err)} (${err.code ?? "ERROR"})`);
  },
};

mountThaiIDStatus(document.getElementById("thaiid-status"));
fillBtn.addEventListener("click", async () => {
  await autofillThaiID(clinicPatientAutofill);
});
```

`request` is an **allowlist**. The session asks only for fields that also have
a mapped (or `data-openthaiid`) slot — least privilege. Nested slots such as
`thaiName.givenName` belong in `map`, not in `request`.

`map` is an alias of the existing `fields` option. Attribute discovery still
works if you omit `map` and tag inputs with `data-openthaiid`.

This page’s copy-paste config lives in `settings.js`
(`clinicPatientAutofill()`) and as JSON samples:

- `kyc-minimal.json` — `citizenId` + `thaiName` + `birthDate`
- `clinic-full.json` — clinic allowlist (`gender` + `address` too)

Schema: `sdk/web/schema/autofill-clinic-config.schema.json`.
Hospital/clinic ERP steps: [docs/erp-integration.md](../../docs/erp-integration.md).

The right-hand **Developer settings** panel mutates `request`, and can
**download / upload** that JSON (or load the samples) so integrators iterate
without editing code. Callbacks stay in `app.js`.

`probeThaiID()` / `mountThaiIDStatus()` show helper up/down, reader, and card
present/absent in clinic-staff Thai (พร้อม). Pass `{ lang: "en" }` for English.
Fill failures from `autofillThaiID` use the same Thai guidance by default
(`formatStaffError` / `{ lang: "en" }`); protocol **codes** do not change.
They never display personal field values. Pass `{ mode: "advanced" }` for
technician details.

## Run

```bash
# Terminal 1: local helper (fictional demo card, no hardware)
./dist/openthaiid -synthetic

# Terminal 2: build the SDK, then serve the repo root
cd sdk/web && npm ci && npm run build && cd ../..
python3 -m http.server 8000
```

Open `http://127.0.0.1:8000/examples/clinic-patient/`. Prefer HTTPS for real
browser tests; localhost HTTP is development-only.

With `-synthetic`, expect citizen ID `1234567890121`, given name `สมชาย`,
family name `ใจดี`, birth `1990-04-12`, gender `male`, and the seeded
fictional address. Phone and HN stay empty. The default `request` does
**not** include `portrait`. Tick **portrait** in the settings panel to
receive the fictional checkerboard JPEG in `#patient-photo` (example
preview only). `clinicPatientAutofill()` always maps `portrait` to that
`<img>`; the session still asks for it only when it is in `request`.

Uncheck a field in **request** and fill again: that column is left alone and
the grant does not include it.

## React

The same config object works from a React click handler:

```tsx
<button type="button" onClick={() => void autofillThaiID(clinicPatientAutofill)}>
  Fill from Thai ID
</button>
```

`@openthaiid/react` also exports `useThaiIDAutofill(config)` and
`probeThaiID`. See `sdk/react/README.md`.
