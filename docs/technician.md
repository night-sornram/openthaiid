# Technician / developer guide

Clinic staff should stay on the simple Ready view. This page is the debug
checklist when **Fill fails**.

Open technician tools on the PC: `http://127.0.0.1:17891/#technician`
(or the **Technician / developer** button). Defaults stay quiet; turn on
**Detailed logs** only while you are debugging. Copy diagnostics never
includes citizen ID, names, dates, or address values.

ERP pages can pass `mountThaiIDStatus(el, { mode: "advanced" })` for the
same technician details in-page. Approval security is unchanged: the ERP
cannot allow itself.

## When Fill fails

Work top to bottom. Most failures are the first four.

### 1. Helper not running

The ERP status says OpenThaiID is not running, or `probeThaiID()` returns
`reachable: false`.

- Start `openthaiid tray` (clinic) or `openthaiid` on **this** PC (the one with the reader).
- Confirm `http://127.0.0.1:17891/` loads, or use tray **Open Ready page**.
- `openthaiid status` prints helper / reader / card without personal fields.
- There is no cloud service to restart.

```bash
./openthaiid tray         # background helper
./openthaiid              # terminal helper
./openthaiid -synthetic   # practice reader (no hardware)
./openthaiid doctor       # card service only; never reads personal fields
```

### 2. No reader / card service down

Lights: helper running, reader not connected.

- Unplug/replug the reader, then **Refresh readers**.
- **Reconnect card service** if the status is stalled or unavailable.
- Linux: `pcscd` running. Windows: Smart Card service. macOS: built-in PC/SC.
- Two readers and none selected: **Use this reader**.

### 3. Approval pending

ERP is waiting; helper shows **A website wants to read the card**.

- **Allow** that origin (match the comparison code).
- **Don’t allow** if the origin is not the clinic ERP.
- Pending list with request id, field tokens, and expiry is in technician
  tools. Same decision as `allow <id>` / `deny <id>` in the terminal.

Do not weaken this. The website still cannot approve itself.

### 4. Wrong origin

Grant is for a different site than the page staff opened
(`http://localhost:8000` vs `https://erp.clinic.example`).

- Compare the pending/approved origin to the browser address bar.
- **Revoke** the stale origin and Allow the one staff actually use.
- HTTPS origins are always eligible; HTTP is loopback-only for development.

### 5. Field grants

`FIELD_NOT_ALLOWED`, or some inputs stay empty while others fill.

- The ERP `request` allowlist and `map` must include the slot.
- The PC grant must include that field. Expanding fields prompts a **new**
  Allow (the live grant is an exact set, not a silent superset).
- Phone / HN are not on the card. Portrait is out of scope.

### 6. Practice vs real hardware

The helper started with `-synthetic` shows **practice reader**. Fill still
needs Allow. For a real card, restart without `-synthetic`.

## Copy diagnostics

**Copy diagnostics** gathers version, OS, card-service state, listener,
WebSocket URL, log level, reader presence (not card contents), pending and
approved origins (field *names* only), and last operational errors.

Paste that text into a ticket. Do not paste form field values.

## Verbose logging

Technician checkbox, or `openthaiid -log-level debug`. Default is `info`.
The switch lasts until restart and does not persist into config. Logs still
must not contain citizen IDs, names, dates, raw APDUs, or tokens.

## ERP library (no cloud deploy)

Keep integration as `@openthaiid/web` (or `@openthaiid/react`) plus this
local helper:

```js
import { autofillThaiID, mountThaiIDStatus, probeThaiID } from "@openthaiid/web";

mountThaiIDStatus(statusEl); // user copy; { mode: "advanced" } for techs
await autofillThaiID(config); // from a click
```

See [clinic.md](clinic.md) and [user.md](user.md).
