# Use OpenThaiID as a library in your ERP

OpenThaiID is a **JavaScript library** you add to the clinic’s existing web
app (Create Patient, KYC, registration). You do **not** install an OpenThaiID
website, and you do **not** deploy an OpenThaiID backend.

```js
import { autofillThaiID, probeThaiID, mountThaiIDStatus } from "@openthaiid/web";

mountThaiIDStatus(document.getElementById("reader-status"));

fillBtn.addEventListener("click", async () => {
  await autofillThaiID({
    request: ["citizenId", "thaiName", "birthDate"],
    map: {
      citizenId: "#patient-cid",
      "thaiName.givenName": "#patient-given",
      "thaiName.familyName": "#patient-family",
      birthDate: "#patient-dob",
    },
  });
});
```

That is the product for clinic ERP teams: one helper in the page you already
have.

Clinic staff (start helper, plug reader, Fill): [user.md](user.md).
Practice without a reader: [training.md](training.md).
Friday with ACR39U: [friday-checklist.md](friday-checklist.md).
Technicians (Fill failed): [technician.md](technician.md).

## What you add (and what you do not)

| In the ERP (your existing app) | Not an OpenThaiID product |
|---|---|
| `@openthaiid/web` (or `@openthaiid/react`) | A hosted OpenThaiID frontend |
| `probeThaiID` / `mountThaiIDStatus` for connection | A clinic “manager app” to install |
| `autofillThaiID(config)` on a button click | An OpenThaiID server in the data center |

The PC with the reader still needs the local helper running (`openthaiid tray`,
`openthaiid serve` / `openthaiid service install` for always-on, or
`openthaiid` / `openthaiid -synthetic`). That helper is not a stack you deploy;
it is local hardware access. Clinic IT: unzip the OS archive and follow
START-HERE.txt ([installation.md](installation.md)). If it is not running, `probeThaiID()` returns
`reachable: false` and the status panel says so.

## ERP developers

1. Depend on `@openthaiid/web` (or import the built `dist/index.js`).
2. Put a status element on the screen if you want a connection check:

```js
mountThaiIDStatus(document.getElementById("reader-status"));
```

3. On **Create Patient**, call `autofillThaiID(config)` from the click.
   Origin + field Allow still happens on the PC (helper page or terminal).
   The library never silent-reads. Portrait is returned only when `portrait`
   is in `request` **and** the helper Allows that field. Keep portrait out
   of the default clinic `request`. Map `portrait` to an `<img>` (data URL)
   or use `portraitBlob()` / `portraitDataUrl()` from the read result. Do
   not `console.log` photo bytes. If the ERP sets a CSP, allow `img-src
   data:` (or `blob:` when using `URL.createObjectURL`). The
   `examples/clinic-patient/` screen shows an opt-in preview only.

`mountThaiIDStatus` defaults to Thai clinic-staff copy (พร้อม / ยังไม่พร้อม).
Pass `{ lang: "en" }` for English, or `{ mode: "advanced" }` for technician
details (English) on the same page. `autofillThaiID` / `probeThaiID` use the
same default: human messages in Thai, stable protocol **codes**. Display Fill
failures with `formatStaffError(err)` (or `{ lang: "en" }`).

`request` is an allowlist. `map` is slot → selector. Phone / HN stay off the
map. JSON Schema + samples + import/export:
[erp-integration.md](erp-integration.md). Worked example:
`examples/clinic-patient/`.

React:

```tsx
import { autofillThaiID, probeThaiID } from "@openthaiid/react";
```

Same config object. `useThaiIDAutofill(config)` if you want `filling` state.

## Check connection from the page

`probeThaiID()` opens a short WebSocket, sends `bridge.probe`, and closes.
It reports helper up/down, card service, readers, and card present/absent in
machine fields. `mountThaiIDStatus` turns that into **Ready** / plug reader /
insert card. It does **not** return citizen ID, names, or other personal
fields, and it does **not** skip Allow for a later `autofillThaiID` / `read()`.

## Operator Allow (security)

The ERP page cannot approve itself. The first time that origin asks for
fields, someone at the PC clicks Allow on the helper Ready page
(`http://127.0.0.1:17891/`, tray **Open Ready page**, or `allow <id>` in
the helper terminal). After that, the grant is reused until it expires or
is revoked. Technician tools on that page can list pending/approved origins
and revoke them; they do not bypass Allow.

## Local helper (hardware only)

Clinic staff should not need a developer terminal. Clinic IT installs the
per-OS package ([installation.md](installation.md)); staff start **OpenThaiID**
(tray / application menu) or use **Start at login** (opt-in).

```bash
./openthaiid tray         # background helper (clinic)
./openthaiid              # terminal helper (technicians)
./openthaiid -synthetic   # fictional demo card, no hardware
```

No cloud, no extra frontend. The helper still binds `127.0.0.1` only.
