# Clinic ERP — create patient

Vanilla example of a clinic “New patient” screen that autofills from Thai ID
through OpenThaiID. Clinics configure **which fields** are requested and how
they map to this form via a settings object — not a forked SDK.

The flow stays **manual**: a user clicks **Fill from Thai ID**. The helper
still connects from that gesture and waits for **terminal origin + field
approval**. There is no silent background read and no portrait/photo.

## Integrator snippet

On Create Patient (or a dedicated fill button):

```js
import { autofillThaiID } from "@openthaiid/web";

const clinicPatientAutofill = {
  // Protocol fields to request (subset of supported textual fields).
  request: ["citizenId", "thaiName", "birthDate", "gender", "address"],
  // Slot → selector on this screen. Phone / HN stay unmapped.
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

`request` is an **allowlist**. The session asks only for fields that also have
a mapped (or `data-openthaiid`) slot — least privilege. Nested slots such as
`thaiName.givenName` belong in `map`, not in `request`.

`map` is an alias of the existing `fields` option. Attribute discovery still
works if you omit `map` and tag inputs with `data-openthaiid`.

This page’s copy-paste config lives in `settings.js`
(`clinicPatientAutofill()`). The right-hand panel mutates `request` so a
clinic can drop gender or opt into `englishName` without changing selectors.

## Run

```bash
# Terminal 1: bridge (fictional demo card, no hardware)
./dist/openthaiid -synthetic

# Terminal 2: build the SDK, then serve the repo root
cd sdk/web && npm ci && npm run build && cd ../..
python3 -m http.server 8000
```

Open `http://127.0.0.1:8000/examples/clinic-patient/`. Prefer HTTPS for real
browser tests; localhost HTTP is development-only.

With `-synthetic`, expect citizen ID `1234567890121`, given name `สมชาย`,
family name `ใจดี`, birth `1990-04-12`, gender `male`, and the seeded
fictional address. Phone and HN stay empty.

Uncheck a field in **request** and fill again: that column is left alone and
the terminal grant does not include it.

## React

The same config object works from a React click handler:

```tsx
<button type="button" onClick={() => void autofillThaiID(clinicPatientAutofill)}>
  Fill from Thai ID
</button>
```

`@openthaiid/react` also exports `useThaiIDAutofill(config)` if you want
`filling` / `approvalRequired` / `error` state. See `sdk/react/README.md`.
