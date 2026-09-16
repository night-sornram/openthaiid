# Technician / developer guide

Clinic staff should stay on the simple Ready view (Thai **พร้อม** lights by
default; English toggle on that page). This page is the debug checklist when
**Fill fails**. Technician chrome on this page stays English.

Open technician tools on the PC: `http://127.0.0.1:17891/#technician`
(or the **Technician / developer** button). Defaults stay quiet; turn on
**Detailed logs** only while you are debugging. Copy diagnostics never
includes citizen ID, names, dates, or address values.

ERP pages can pass `mountThaiIDStatus(el, { mode: "advanced" })` for the
same technician details in-page. Approval security is unchanged: the ERP
cannot allow itself. Reader selection also stays on this PC (Ready page or
the **Readers** list below) — the ERP cannot choose a reader over CORS.

## When Fill fails

Work top to bottom. Most failures are the first four. Clinic ERP pages should
show `formatStaffError(err)` (Thai by default) and keep the protocol **code**
for tickets (`BRIDGE_UNAVAILABLE`, `NO_READER`, `NO_CARD`, `APPROVAL_DENIED`,
`FIELD_NOT_ALLOWED`, …).

### 1. Helper not running

The ERP status says OpenThaiID is not running, or `probeThaiID()` returns
`reachable: false`.

- Start `openthaiid tray` (clinic) or `openthaiid` on **this** PC (the one with the reader).
  Always-on kiosk: `openthaiid service status` — if installed, skip the tray
  ([installation.md](installation.md)).
- Confirm `http://127.0.0.1:17891/` loads, or use tray **Open Ready page**.
- Ready page **ทดสอบการเชื่อมต่อ / Test connection** is `bridge.probe` +
  status only — if that fails, the ERP Fill path will fail too.
- After a helper restart, leave the Ready page open: it refreshes the CSRF
  token and shows **ตัวช่วยหลุดการเชื่อมต่อ** until the helper is back.
- `openthaiid status` prints helper / reader / card without personal fields.
- There is no cloud service to restart.

```bash
./openthaiid tray         # background helper
./openthaiid              # terminal helper
./openthaiid serve        # headless (Windows service / systemd)
./openthaiid -synthetic   # practice reader (no hardware)
./openthaiid doctor       # card service only; never reads personal fields
./openthaiid qualify -report ./openthaiid-qualify.json   # Friday kit
./openthaiid audit        # redacted operator trail JSON
```

### 2. No reader / card service down

Lights: helper running, reader not connected.

- Unplug/replug the reader, then **Refresh readers**.
- **Reconnect card service** if the status is stalled or unavailable.
- Linux: `pcscd` running. Windows: Smart Card service. macOS: built-in PC/SC.
- Two readers and none selected: **Use this reader** (Ready page for staff,
  English **Readers** list here). Probe, status, and Fill then use that reader.
  Audit events may include `readerId` (device id, not card data).

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
- Phone / HN are not on the card. Portrait is optional and **off by
  default**; if the ERP adds `"portrait"` to `request`, staff must Allow
  that expanded set. Preview belongs in the example/ERP UI only — never
  in logs or tickets.

### 6. Practice vs real hardware

The helper started with `-synthetic` shows **practice reader**. Fill still
needs Allow. For a real card, restart without `-synthetic`.

## Copy diagnostics

**Copy diagnostics** gathers version, OS, card-service state, listener,
WebSocket URL, log level, reader presence (not card contents), pending and
approved origins (field *names* only), last operational errors, and the
redacted **operator audit trail**.

Paste that text into a ticket. Do not paste form field values.

## Operator audit trail

Technician tools show a local ring of recent events (also a JSONL file under
the user config dir, `audit.jsonl`). **Copy JSON** / **Download JSON** export
the same payload as `openthaiid audit`.

Recorded:

- helper start / stop (mode: console, tray, or serve)
- reader connected / disconnected (device name + opaque `readerId`, not card contents)
- reader selected (which PC/SC reader Fill will use)
- card present / absent
- approval allow / deny / revoke (website origin + field *tokens* only)
- Fill failures by **error code only** (`NO_CARD`, `CARD_REMOVED`, …)

Never stored: citizen ID, names, address, portrait / photo bytes, raw APDUs,
or Fill error *messages*. Forbidden payloads are dropped rather than written.

Use this JSON with Copy diagnostics on Friday tickets, next to the
redacted `openthaiid qualify` report. See
[friday-checklist.md](friday-checklist.md) and
[hardware-qualification.md](hardware-qualification.md).

## Multi-reader desks

One reader stays automatic. When PC/SC lists two or more:

1. Ready page (Thai): **เลือกเครื่องอ่าน** → **ใช้เครื่องนี้**.
2. Technician tools (English): **Readers** → **Use this reader** (shows opaque `id`).
3. Confirm Ready / `probeThaiID()` names the selected reader, then Fill.

Selection is in-memory on this helper (cleared if that reader is unplugged or
the helper restarts). The ERP cannot select a reader. Synthetic tests add a
second reader with `AddReader`. Default `openthaiid -synthetic` still seeds
one. To practice the picker without hardware:

```bash
OPENTHAIID_SYNTHETIC_READERS=2 ./dist/openthaiid -synthetic
```

## Friday qualify

When the ACR39U is on the desk (never `-synthetic` for a hardware claim):

```bash
./openthaiid qualify -operator -report ./openthaiid-qualify.json
# Windows package: Qualify-OpenThaiID.bat
```

Type only `pass` / `fail` / `skip`. Attach the printed `report:` JSON and
the audit JSON. One-page: [friday-checklist.md](friday-checklist.md).
Full kit: [hardware-qualification.md](hardware-qualification.md).

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

Always-on without a logged-in tray (clinic IT): `openthaiid service install`
(Windows SCM or Linux systemd --user + linger). See
[installation.md](installation.md).

See [clinic.md](clinic.md), [erp-integration.md](erp-integration.md),
[user.md](user.md), [training.md](training.md),
[friday-checklist.md](friday-checklist.md), and
[hardware-qualification.md](hardware-qualification.md)
(`openthaiid qualify`).
