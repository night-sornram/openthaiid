# Technician / developer guide

Hardware-arrival index (Thai-first): [getting-started.md](getting-started.md).

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
(`GET /v1/helper/summary`, no personal fields), desk snapshot
(`GET /v1/helper/doctor`, same loopback rule; no GitHub call), quit
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
`HTTPS_WS_BLOCKED`, `TLS_UNTRUSTED`). Staff Thai tables: [user.md](user.md). Operator one-page (E4–E6):
[hardware-checklist.md](hardware-checklist.md).

### 1. Helper not running

The ERP status says OpenThaiID is not running, or `probeThaiID()` returns
`reachable: false`. If the code is `HTTPS_WS_BLOCKED` or `TLS_UNTRUSTED`, the helper **is**
running — Firefox/Safari blocked insecure loopback or the local certificate
is not trusted (see below). Do not restart the helper for that.

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
- `openthaiid doctor` prints one redacted desk snapshot (helper version, HTTP/TLS
  ports, readers, `bridge.probe`, Ready URL, TLS yes/no with **no key paths**,
  last version-check if one already ran). Exit 1 only if the helper is down or
  PC/SC is unavailable. Soft GitHub failures stay soft. Technician tools
  **Desk snapshot** copies the same text.
- There is no cloud service to restart.

```bash
./openthaiid tray         # background helper
./openthaiid              # terminal helper
./openthaiid serve        # headless (Windows service / systemd)
./openthaiid -synthetic   # practice reader (no hardware)
make pre-reader-dry-run       # ซ้อมก่อนได้เครื่อง (qualify-smoke + Ready + probe)
./openthaiid doctor       # desk snapshot (helper, ports, PC/SC, probe, TLS yes/no)
./openthaiid qualify -report ./openthaiid-qualify.json   # Qualify kit
./openthaiid bind-stress -synthetic -cycles 3            # binding harness smoke (not P2–P10 close)
./openthaiid audit        # redacted operator trail JSON
./openthaiid trust add https://erp.clinic.example  # IT pre-approve (staff skip Allow)
./openthaiid update check # GitHub Releases lookup; no download
```

### 2. No reader / card service down

Lights: helper running, reader not connected — or **บริการอ่านบัตรยังไม่พร้อม**.

- Unplug/replug the reader, then **Refresh readers**.
- **Reconnect card service** if the status is stalled or unavailable
  (`PCSC_UNAVAILABLE` / `BACKEND_STALLED`). Ready / `probeThaiID()` must
  leave **พร้อม** while the service is down and return to **พร้อม** after
  recovery (card seated). Do not reboot.
- Linux: start `pcscd`, then Reconnect if lights stay down. Windows: start
  the Smart Card service, then Reconnect. macOS: built-in PC/SC (E6
  is often SKIP).
- Two readers and none selected: **Use this reader** (Ready page for staff,
  English **Readers** list here). Probe, status, and Fill then use that reader.
  Audit events may include `readerId` (device id, not card data).

### 3. Approval pending

ERP is waiting; helper staff view shows **เว็บคลินิกขอใช้บัตรประชาชน**
(hostname, not the raw Origin). Technician pending list still shows Origin.

- Staff **อนุญาต / Allow** that clinic page (match the comparison code).
- **ไม่อนุญาต / Don't allow** if it is not the clinic page they use.
- Pending list with request id, field tokens, and expiry is in technician
  tools. Same decision as `allow <id>` / `deny <id>` in the terminal.

Do not weaken this. The website still cannot approve itself.

Clinic IT should pre-approve the ERP origin **once** so staff rarely see
Allow: `openthaiid trust add https://erp.clinic.example`, `install.ps1
-ClinicOrigin`, `config.json` `preapprovedOrigins`, env
`OPENTHAIID_PREAPPROVED_ORIGINS`, or Technician **Trusted clinic websites**.
Explicit origins only — not “allow all”. Extra fields still prompt Allow.
[installation.md](installation.md).

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

## Version check (no auto-install)

Clinic PCs stay on the installed unsigned package until a technician
replaces it. To see if GitHub has a newer Release:

- Ready page **ตรวจเวอร์ชัน** (English: **Check version**)
- Technician tools **Check GitHub Releases**
- Tray **ตรวจเวอร์ชัน**
- `openthaiid update check`

The helper compares the running version to
[GitHub Releases](https://github.com/night-sornram/openthaiid/releases)
(`OPENTHAIID_UPDATE_REPO=owner/name` to fork). Prereleases (`v0.1.0-rc.*`)
count unless `openthaiid update check --stable`. It prints up to date vs a
newer prerelease/release and the release URL. It does **not** download or
install. If the PC is offline, GitHub rate-limits the request, or the repo
is private without a token, the message says so and **กรอกข้อมูล / Fill
still works**.

Private repository: unauthenticated GitHub API returns HTTP 404 (not an
empty catalog). Set `OPENTHAIID_GITHUB_TOKEN` (or `GITHUB_TOKEN`) in the
**helper** environment — never in the ERP page. `/releases/latest` is not
used; that endpoint 404s when every published release is a prerelease.

Do not enable GitHub Actions for this (minutes until ~1 Oct 2026).
Store-signed auto-update is not implemented ([signing.md](signing.md)).

## Firefox / Safari clinic pages

Ready (`http://127.0.0.1:17891/`) works in Firefox and Safari. Fill /
`probeThaiID` from an **https://** ERP needs optional loopback **HTTPS + WSS**
on this PC (`wss://127.0.0.1:17892/v1/ws`). Without that, those browsers
block `ws://127.0.0.1` as mixed content (`HTTPS_WS_BLOCKED`). Do not tell
staff to flip Firefox prefs. Generate and trust a **local** cert once —
never a shared private key: [loopback-tls.md](loopback-tls.md). If the cert
is present but untrusted, the page shows `TLS_UNTRUSTED`. Chrome/Edge still
work with plain `ws://`. Matrix: [compatibility.md](compatibility.md).

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

Use this JSON with Copy diagnostics on hardware tickets, next to the
redacted `openthaiid qualify` report. See
[hardware-checklist.md](hardware-checklist.md) and
[hardware-qualification.md](hardware-qualification.md).

## Multi-reader desks

One reader stays automatic. When PC/SC lists two or more:

1. Ready page (Thai): **เลือกเครื่องอ่าน** → **ใช้เครื่องนี้**.
2. Technician tools (English): **Readers** → **Use this reader** (shows opaque `id`).
3. Confirm Ready / `probeThaiID()` names the selected reader, then Fill.

Staff copy: [user.md](user.md). Reader plug-in step:
[hardware-checklist.md](hardware-checklist.md).

Selection is in-memory on this helper (cleared if that reader is unplugged or
the helper restarts). The ERP cannot select a reader. Synthetic tests add a
second reader with `AddReader`. Default `openthaiid -synthetic` still seeds
one. To practice the picker without hardware:

```bash
OPENTHAIID_SYNTHETIC_READERS=2 ./dist/openthaiid -synthetic
```

## Hardware qualify

When the ACR39U is on the desk (never `-synthetic` for a hardware claim):

```bash
./openthaiid qualify -operator -report ./openthaiid-qualify.json
# Windows package: Qualify-OpenThaiID.bat
```

Type only `pass` / `fail` / `skip`. Attach the printed `report:` JSON and
the audit JSON. One-page: [hardware-checklist.md](hardware-checklist.md).
Full kit: [hardware-qualification.md](hardware-qualification.md).

## Bind-stress (P2–P10)

Separate from qualify. Repeated PC/SC connect/list/probe without field
reads or APDU logs. Today on synthetic; with hardware omit `-synthetic`.

```bash
make bind-stress-smoke
./scripts/bind-stress.sh --report ./openthaiid-bind-stress.json   # ACR39U; never -synthetic
```

Sheet: [bind-stress.md](bind-stress.md). Attach the redacted JSON next to
the qualify report. A pass on this OS/arch does not close other hosts.

**E4** (physical): pull the card during Fill → `CARD_REMOVED`; unplug
the reader during Fill → `READER_REMOVED`; opaque native I/O → `READ_FAILED`.
Staff Thai on the clinic page and in [user.md](user.md); this page stays
English. One-page recovery table: [hardware-checklist.md](hardware-checklist.md).
Synthetic tests already cover the codes — hardware still has to tick E4.

**E5** (physical): browser refresh, helper quit/start, OS sleep/wake.
Helper down → `BRIDGE_UNAVAILABLE` and Test connection fails. After wake,
Ready re-probes without reading card fields. Do not reboot. Staff: do not
reboot, **ทดสอบการเชื่อมต่อ** ([user.md](user.md)). Hardware checklist:
[hardware-checklist.md](hardware-checklist.md).

**E6** (physical): stop `pcscd` / Smart Card service if safe →
`PCSC_UNAVAILABLE`, Test connection stays probe-only, **Reconnect card
service**, Ready says **พร้อม** again. macOS SKIP if you cannot stop PC/SC.
Synthetic tests cover unavailable → restored without simulating sleep.
Staff copy: [user.md](user.md). E5/E6 table:
[hardware-checklist.md](hardware-checklist.md).

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
[hardware-checklist.md](hardware-checklist.md), and
[hardware-qualification.md](hardware-qualification.md)
(`openthaiid qualify`). Binding-audit stress (P2–P10):
[bind-stress.md](bind-stress.md) (`make bind-stress-smoke`).
