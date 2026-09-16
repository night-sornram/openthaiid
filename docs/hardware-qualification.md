# Friday hardware qualification kit

Operator checklist for **ACR39U** physical qualification (acceptance E1–E7
plus M1). This is the work that waits on the reader. Software can only
prepare the kit.

**Never record, paste, screenshot, or log:** citizen ID, names, address
text, dates of birth, or portrait / photo bytes (including Base64 and
`data:image` URLs). Tickets get **pass/fail**, **milliseconds**, **error
codes**, and the technician diagnostics copy below.

Synthetic `openthaiid qualify -synthetic` proves the kit, not the reader.

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
| B2 | `openthaiid version` | Prints the package version (for example `0.1.0-rc.1`). |
| B3 | `openthaiid doctor` | `pcsc: OK` and the **reader name** only. Line `personal fields: not read`. |
| B4 | Plug ACR39U, insert a Thai ID | Doctor lists the reader. Do not run a dump tool that prints fields. |
| B5 | Helper running | `openthaiid tray` or OpenThaiID.app / Start menu. Ready page `http://127.0.0.1:17891/`. |

Record: OS, arch, reader variant, driver/version if any, helper version,
commit SHA. No card field values.

## Automated slice (this repo)

```bash
# Kit smoke (no hardware; does not claim E1–E7)
make qualify-smoke

# Friday, with the reader + card already inserted:
./dist/openthaiid qualify -report /tmp/openthaiid-qualify.json
# or:
./scripts/hardware-qualify.sh
```

`openthaiid qualify` (and the script) print **PASS/FAIL**, timings, protocol
error codes (`NO_READER`, `NO_CARD`, `PCSC_UNAVAILABLE`, `CARD_REMOVED`, …),
field *names* present, JPEG SOI/EOI flags, and technician diagnostics.
They **refuse to write** a report that would contain the synthetic
sentinels, a 13-digit citizen ID, JPEG Base64, or a `data:image` URL.

Default: 100 E2 reads, 20 E3 cycles on `-synthetic`, E4–E7/M1 **SKIP**
(`-skip-operator`) so the operator fills those rows from the table below.

## E1–E7 (physical)

Use one Thai ID you are allowed to handle. Look at the ERP / example
preview if you need to confirm a layout. **Do not save the preview.**

| ID | Criterion | How | Record |
|---|---|---|---|
| E1 | Reader/card detection within 2s under normal operation | Card inserted, helper started (or `qualify`). Ready lights go on. | PASS/FAIL, `durationMs`, `NO_READER` / `NO_CARD` / `PCSC_UNAVAILABLE` if fail |
| E2 | 100 successful repeated MVP reads | `qualify` E2, or 100 Fill clicks. Report stores field *names* and JPEG flags only. | PASS/FAIL, min/avg/max ms, error code counts |
| E3 | ≥20 insertion/removal cycles | Remove until status says insert card, seat fully, repeat. Synthetic kit toggles a practice card. | PASS/FAIL, cycles ok, max cycle ms |
| E4 | Removal during reading and reader unplug | Start Fill / a qualify read, pull the card, then unplug the reader. | PASS if helper stays up and the error is `CARD_REMOVED` / `READER_REMOVED` / `READ_FAILED` — not a crash |
| E5 | Browser refresh, bridge restart, sleep/wake | Fill, refresh ERP, `openthaiid quit` then start again, sleep/wake. | PASS if reconnect works; no field values in logs |
| E6 | Missing/stopped PC/SC recovery | Linux: stop `pcscd` briefly. Windows: stop Smart Card service if safe. macOS: note built-in PC/SC; record SKIP if you cannot disable it. | `PCSC_UNAVAILABLE` then recovery, or SKIP + reason code |
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

Firefox: `https://` pages may block `ws://127.0.0.1` as mixed content.
That is a browser policy, not an OpenThaiID bypass. See
[compatibility.md](compatibility.md).

## Technician diagnostics copy

From the helper: **Technician / developer → Copy diagnostics**.

Or the bottom of `openthaiid qualify` output / `openthaiid status` /
`openthaiid doctor`.

Paste that text into the ticket. It is versions, OS, card-service state,
reader *names*, presence, error **codes**. It is not card content.

Do not paste ERP form dumps, HAR files with `card.read` payloads, or
photos of the card.

## Report attachment

Attach `openthaiid-qualify.json` when the script wrote one. Confirm the
`"pii"` field says values are omitted. If a tool ever printed a citizen
ID, **delete that file** and re-run; do not redact by hand in chat.

## After macOS

Repeat B1–B5 and E1–E7 on Windows, then Linux. Keep three rows in
[acceptance.md](acceptance.md). A macOS pass does not close the
three-platform claim.
