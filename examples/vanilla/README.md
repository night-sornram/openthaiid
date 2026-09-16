# Vanilla example

Minimal pages that talk to a local OpenThaiID bridge.

## Form autofill (recommended)

`autofill.html` is a KYC-style form plus one **Autofill from Thai ID** button.
Inputs are tagged with `data-openthaiid`; the click handler is:

```js
import { autofillThaiID } from "@openthaiid/web";

fillBtn.addEventListener("click", async () => {
  await autofillThaiID({
    root: document.getElementById("kyc"),
    onApprovalRequired: ({ comparisonCode, approvalId }) => {
      status.textContent = `Approve in the terminal — request ${approvalId}, code ${comparisonCode}`;
    },
  });
});
```

Or map selectors explicitly:

```js
await autofillThaiID({
  citizenId: "#citizen-id",
  thaiName: "#full-name",
  birthDate: "#dob",
});
```

`connect()` still runs from that click (local-network permission). The operator
must `allow <id>` in the bridge terminal. The helper does not bypass the
bridge and does not read portraits.

## Run

```bash
# Terminal 1: bridge (fictional demo card, no hardware)
./dist/openthaiid -synthetic

# Terminal 2: build the SDK, then serve the repo root
cd sdk/web && npm ci && npm run build && cd ../..
python3 -m http.server 8000
```

Open `http://127.0.0.1:8000/examples/vanilla/autofill.html`. Prefer HTTPS for
real browser tests; localhost HTTP is development-only.

With `-synthetic`, the seeded fictional card fills roughly:

| Slot | Value |
|---|---|
| `citizenId` | `1234567890121` |
| `thaiName` | `นาย สมชาย ใจดี` |
| `birthDate` | `1990-04-12` |
| `gender` | `male` |
| `address` | `99 หมู่ที่ 1 ถนนนิทาน ตำบลบ้านใหม่ อำเภอเมือง จังหวัดนิทาน` |

This page also requests `gender` and `address`, so the terminal grant must
include them (not only the default three fields).

## Full ceremony

`index.html` still shows Connect / Read / Disconnect plus raw JSON. Use that
when you need the `ThaiID` class events yourself.

The page must call `connect()` (or `autofillThaiID()`) from a visible user
gesture.

A clinic-oriented Create Patient screen (configurable `request` + `map`,
phone left blank) lives in `examples/clinic-patient/`.

## Data handling

Clear any displayed card fields on card removal, disconnect, or revocation.
Do not store results in localStorage.
