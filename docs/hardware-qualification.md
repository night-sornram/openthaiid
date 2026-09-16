# Friday hardware qualification kit

Operator checklist for **ACR39U** physical qualification (acceptance E1–E7
plus M1). This is the work that waits on the reader. Software can only
prepare the kit.

**Never record, paste, screenshot, or log:** citizen ID, names, address
text, dates of birth, or portrait / photo bytes (including Base64 and
`data:image` URLs). Tickets get **pass/fail**, **milliseconds**, **error
codes**, and the technician diagnostics copy below.

Synthetic `openthaiid qualify -synthetic` proves the kit, not the reader.
Friday-morning index: [getting-started.md](getting-started.md).
One-command software rehearsal: `make friday-dry-run` (ซ้อมก่อนได้เครื่อง).
One-page Friday sheet: [friday-checklist.md](friday-checklist.md).
Practice Fill without hardware: [training.md](training.md).
Loopback helper HTTP inventory (foreign Origin cannot stop the helper):
[security.md](security.md).

## Order of platforms

1. **macOS + ACR39U first** (Apple Silicon preferred; Intel is a separate
   archive). Built-in PC/SC. Unsigned `OpenThaiID.app` — see
   [installation.md](installation.md) (`make package-macos` on a Mac host).
2. **Windows 11** (Smart Card service + vendor driver only if needed).
3. **Linux** (`pcscd` + `libpcsclite`; CCID as required).

Untested OS/arch stays unverified. Do not copy a macOS pass onto Windows
or Linux.

## Before you start (every OS)

| Step | What | Pass looks like |
|---|---|---|
| B1 | Unsigned clinic package on this PC | START-HERE.txt followed; helper starts. Not signed / not notarized. |
| B2 | `openthaiid version` | Prints the package version (for example `0.1.0-rc.4`). |
| B3 | `openthaiid doctor` | `pcsc: OK` and the **reader name** only. Line `personal fields: not read`. |
| B4 | Plug ACR39U, insert a Thai ID | Doctor lists the reader. Do not run a dump tool that prints fields. |
| B5 | Helper running | `openthaiid tray`, OpenThaiID.app / Start menu, or always-on `openthaiid serve` / `openthaiid service status`. Ready page `http://127.0.0.1:17891/`. |

Record: OS, arch, reader variant, driver/version if any, helper version,
commit SHA. No card field values.

## Automated slice (this repo)

```bash
# Friday-morning software rehearsal (no hardware; does not claim E1–E7)
make friday-dry-run
./scripts/friday-dry-run.sh --two-readers   # optional practice picker

# Kit smoke only (no helper / Ready page)
make qualify-smoke

# Friday, with the reader + card already inserted (one JSON report):
./dist/openthaiid qualify -report ./openthaiid-qualify.json
# Interactive E3 + E4–E7/M1 (type pass, fail, or skip only — never card data):
./dist/openthaiid qualify -operator -report ./openthaiid-qualify.json
# Wrappers:
./scripts/hardware-qualify.sh
./scripts/hardware-qualify.sh --operator
# Windows package: Qualify-OpenThaiID.bat / Qualify-OpenThaiID.ps1
```

`openthaiid qualify` always writes **one** redacted JSON file (default
`./openthaiid-qualify.json`) and prints `report: <absolute path>`. It prints
**PASS/FAIL**, timings, protocol error codes (`NO_READER`, `NO_CARD`,
`PCSC_UNAVAILABLE`, `CARD_REMOVED`, …), field *names* present, JPEG SOI/EOI
flags, and technician diagnostics. It **refuses to print or write** output
that would contain the synthetic sentinels, a 13-digit citizen ID, JPEG
bytes / Base64, a PNG Base64 prefix, or a `data:image` URL. Operator
answers other than `pass` / `fail` / `skip` are refused (including pasted
names or photo data).

Default: 100 E2 reads. E3 is automatic on `-synthetic`; on hardware, run
`-operator` and follow the remove/insert Enter prompts. E4–E7/M1 prompts
are **printed in hardware mode** even without `-operator`. `-skip-operator`
(the default) leaves those rows SKIP; `-operator` records pass/fail/skip.

## E1–E7 (physical)

Use one Thai ID you are allowed to handle. Look at the ERP / example
preview if you need to confirm a layout. **Do not save the preview.**

| ID | Criterion | How | Record |
|---|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | Card inserted, helper started (or `qualify`). Ready lights go on. | PASS/FAIL, `durationMs`, `NO_READER` / `NO_CARD` / `PCSC_UNAVAILABLE` if fail |
| E2 | 100 successful repeated MVP reads | `qualify` E2, or 100 Fill clicks. Report stores field *names* and JPEG flags only. | PASS/FAIL, min/avg/max ms, error code counts |
| E3 | ≥20 insertion/removal cycles | Remove until status says insert card, seat fully, repeat. Synthetic kit toggles a practice card. | PASS/FAIL, cycles ok, max cycle ms |
| E4 | Removal during reading and reader unplug | Start Fill / a qualify read, pull the card, then unplug the reader. Ready must leave **พร้อม**. Staff: [user.md](user.md). Technician: [technician.md](technician.md). One-page: [friday-checklist.md](friday-checklist.md). | PASS if helper stays up and the clinic error is `CARD_REMOVED` (card pulled) / `READER_REMOVED` (USB unplug) / `READ_FAILED` (native I/O that cannot tell) — not a crash. Staff Thai tells them to reinsert, wait for Ready, Fill again. Codes stay stable. |
| E5 | Browser refresh, bridge restart, sleep/wake | Fill, refresh ERP, `openthaiid quit` then start again, sleep/wake. Ready re-probes on wake (`bridge.probe` / status only — never `card.read`). Staff: [user.md](user.md). Technician: [technician.md](technician.md). One-page: [friday-checklist.md](friday-checklist.md). | PASS if reconnect works without a reboot; helper-down is `BRIDGE_UNAVAILABLE` (Test connection fails); after restart Ready returns to **พร้อม**. No field values in logs. SKIP sleep/wake only if the PC cannot sleep, and record which steps you did. |
| E6 | Missing/stopped PC/SC recovery | Linux: stop `pcscd` briefly. Windows: stop Smart Card service if safe. macOS: note built-in PC/SC; record SKIP if you cannot disable it. Ready must leave **พร้อม**. Test connection stays probe-only. Staff: [user.md](user.md). Technician: [technician.md](technician.md). One-page: [friday-checklist.md](friday-checklist.md). | PASS if clinic/Ready/doctor show `PCSC_UNAVAILABLE`, then recovery: start the service, Technician **Reconnect card service** if lights stay down, Ready says **พร้อม** again. SKIP + reason on macOS when PC/SC cannot be stopped. Do not reboot. |
| E7 | Local visual verification without storing field contents | Look at clinic example / ERP. Confirm name split, address, gender, photo *on screen only*. | PASS = looked right **and** nothing was copied/saved |

Community layout hypotheses (still not a government spec) — confirm with
**eyes only**, then tick:

- Gender card byte `1` = male, `2` = female (`qualify` E2 sample may list
  `gender.code` / `gender.value` as that 1-byte flag, not a citizen ID).
- English name `#` parts: title / given / (empty middle) / family.
- Address `#` order: house, village, alley, street, subdistrict, district,
  province (empty alley is consecutive `##`).
- Portrait JPEG starts `FFD8` and ends `FFD9` (`jpegSOI` / `jpegEOI` in the
  report). Do not export the photo.

## M1 — public HTTPS local-network permission

Localhost HTTP is **not** this check.

1. Serve the ERP (or `examples/clinic-patient/`) from a **public `https://`
   origin** the clinic PC can open (real cert; not a disabled-security
   browser profile).
2. Helper still on `ws://127.0.0.1:17891/v1/ws`.
3. Click Fill from that page. Allow **local network** access if the
   browser asks (Chrome local WebSocket checks; Safari/macOS may also
   prompt).
4. Allow the origin on the helper Ready page (comparison code). Default
   grant stays without `portrait` unless the page opts in.

Record: browser + version, origin host (not a query string with PII),
allow vs deny of the local-network prompt, helper grant allow vs deny.
Synthetic data is fine if you do not yet want a real card on a public
origin; a real card is still required for E7.

Firefox and Safari: `https://` pages block `ws://127.0.0.1` as mixed content
(`HTTPS_WS_BLOCKED` — helper is still running). That is a browser policy,
not an OpenThaiID bypass. See [compatibility.md](compatibility.md).

## Technician diagnostics copy

From the helper: **Technician / developer → Copy diagnostics**.

Or the bottom of `openthaiid qualify` output / `openthaiid status` /
`openthaiid doctor`.

Also attach the redacted **operator audit trail**: Technician tools
**Copy JSON** / **Download JSON**, or `openthaiid audit`. Error codes and
origins only — never card values. See [technician.md](technician.md).

Paste that text into the ticket. It is versions, OS, card-service state,
reader *names*, presence, error **codes**. It is not card content.

Do not paste ERP form dumps, HAR files with `card.read` payloads, or
photos of the card.

Always-on clinic PCs (no tray): [installation.md](installation.md)
(`openthaiid service install`).

## Report attachment

Attach the single `report: …` JSON path when the command wrote one. Confirm
the `"pii"` field says values are omitted. Attach `openthaiid audit` JSON
the same way. If a tool ever printed a citizen ID, **delete that file**
and re-run; do not redact by hand in chat.

## After macOS

Repeat B1–B5 and E1–E7 on Windows, then Linux. Keep three rows in
[acceptance.md](acceptance.md). A macOS pass does not close the
three-platform claim.
