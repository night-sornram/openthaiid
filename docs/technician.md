# Technician / developer guide

Friday-morning index (Thai-first): [getting-started.md](getting-started.md).

Clinic staff should stay on the simple Ready view (Thai **พร้อม** lights by
default; English toggle on that page). Lights include **OK / Waiting / Not
ready** text, not color alone. This page is the debug checklist when
**Fill fails**. Technician chrome on this page stays English.

Open technician tools on the PC: `http://127.0.0.1:17891/#technician`
(or the **Technician / developer** button). Defaults stay quiet; turn on
**Detailed logs** only while you are debugging. Copy diagnostics never
includes citizen ID, names, dates, or address values.

The helper listens only on `127.0.0.1` (never the LAN). What is on that
port: Ready HTML (`/`), liveness (`/healthz`), staff lights
(`GET /v1/helper/summary`, no personal fields), quit
(`POST /v1/helper/shutdown`, websites cannot call it), technician JSON
under `/v1/manager/*` (CSRF + this PC’s origin; refresh/reconnect are
rate-limited), and the WebSocket (`/v1/ws`) for the ERP after Allow.
Loopback HTTP inventory: [security.md](security.md) (foreign `Origin`
cannot read summary or shut the helper down).

ERP pages can pass `mountThaiIDStatus(el, { mode: "advanced" })` for the
same technician details in-page. Approval security is unchanged: the ERP
cannot allow itself. Reader selection also stays on this PC (Ready page or
the **Readers** list below) — the ERP cannot choose a reader over CORS.

## When Fill fails

Work top to bottom. Most failures are the first four. Clinic ERP pages should
show `formatStaffError(err)` (Thai by default) and keep the protocol **code**
for tickets (`BRIDGE_UNAVAILABLE`, `NO_READER`, `NO_CARD`, `CARD_REMOVED`,
`READER_REMOVED`, `PCSC_UNAVAILABLE`, `APPROVAL_DENIED`, `FIELD_NOT_ALLOWED`,
…). Staff Thai tables: [user.md](user.md). Friday one-page (E4–E6):
[friday-checklist.md](friday-checklist.md).

### 1. Helper not running

The ERP status says OpenThaiID is not running, or `probeThaiID()` returns
`reachable: false`.

- Start `openthaiid tray` (clinic) or `openthaiid` on **this** PC (the one with the reader).
  Always-on kiosk: `openthaiid service status` — if installed, skip the tray
  ([installation.md](installation.md)).
- Confirm `http://127.0.0.1:17891/` loads, or use tray **Open Ready page**.
- Ready page **ทดสอบการเชื่อมต่อ / Test connection** is `bridge.probe` +
  status only — if that fails, the ERP Fill path will fail too
  (`BRIDGE_UNAVAILABLE`).
- After a helper restart, leave the Ready page open: it refreshes the CSRF
  token and shows **ตัวช่วยหลุดการเชื่อมต่อ** until the helper is back.
- After OS sleep/wake the Ready page re-probes (same `bridge.probe` / status
  path — never `card.read`). Do not reboot the PC.
- `openthaiid status` prints helper / reader / card without personal fields.
- There is no cloud service to restart.

```bash
./openthaiid tray         # background helper
./openthaiid              # terminal helper
./openthaiid serve        # headless (Windows service / systemd)
./openthaiid -synthetic   # practice reader (no hardware)
make friday-dry-run       # ซ้อมก่อนได้เครื่อง (qualify-smoke + Ready + probe)
./openthaiid doctor       # card service only; never reads personal fields
./openthaiid qualify -report ./openthaiid-qualify.json   # Friday kit
./openthaiid audit        # redacted operator trail JSON
```

### 2. No reader / card service down

Lights: helper running, reader not connected — or **บริการอ่านบัตรยังไม่พร้อม**.

- Unplug/replug the reader, then **Refresh readers**.
- **Reconnect card service** if the status is stalled or unavailable
  (`PCSC_UNAVAILABLE` / `BACKEND_STALLED`). Ready / `probeThaiID()` must
  leave **พร้อม** while the service is down and return to **พร้อม** after
  recovery (card seated). Do not reboot.
- Linux: start `pcscd`, then Reconnect if lights stay down. Windows: start
  the Smart Card service, then Reconnect. macOS: built-in PC/SC (Friday E6
  is often SKIP).
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

Staff copy: [user.md](user.md). Friday plug-in step:
[friday-checklist.md](friday-checklist.md).

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

Friday **E4** (physical): pull the card during Fill → `CARD_REMOVED`; unplug
the reader during Fill → `READER_REMOVED`; opaque native I/O → `READ_FAILED`.
Staff Thai on the clinic page and in [user.md](user.md); this page stays
English. One-page recovery table: [friday-checklist.md](friday-checklist.md).
Synthetic tests already cover the codes — hardware still has to tick E4.

Friday **E5** (physical): browser refresh, helper quit/start, OS sleep/wake.
Helper down → `BRIDGE_UNAVAILABLE` and Test connection fails. After wake,
Ready re-probes without reading card fields. Do not reboot. Staff: do not
reboot, **ทดสอบการเชื่อมต่อ** ([user.md](user.md)). Friday table:
[friday-checklist.md](friday-checklist.md).

Friday **E6** (physical): stop `pcscd` / Smart Card service if safe →
`PCSC_UNAVAILABLE`, Test connection stays probe-only, **Reconnect card
service**, Ready says **พร้อม** again. macOS SKIP if you cannot stop PC/SC.
Synthetic tests cover unavailable → restored without simulating sleep.
Staff copy: [user.md](user.md). Friday E5/E6 table:
[friday-checklist.md](friday-checklist.md).

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
