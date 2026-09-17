# PC/SC bind-stress (P2–P10)

**Binding-audit stress** for `ebfe/scard`. This is a runnable harness,
not a blank checklist. It is **not** the E1–E7 qualify kit (no MVP field
reads, no operator pass/fail of clinic UX).

**Never record, paste, screenshot, or log:** citizen ID, names, address,
dates of birth, portrait / photo bytes, or **raw APDUs**. The report stores
pass/fail, timings, protocol **codes**, reader *names*, presence booleans,
and SELECT response *length* + SW-ok. Tickets get the JSON path and
technician copy below.

Qualify kit (E1–E7 / M1): [hardware-qualification.md](hardware-qualification.md).
Acceptance P-rows: [acceptance.md](acceptance.md).
Pin / license: [dependency-audit.md](dependency-audit.md).
Playbook: [../NOT_DONE.md](../NOT_DONE.md).

## What runs **today** on synthetic

No ACR39U. Proves the *script*, not native layouts.

```bash
make bind-stress-smoke
# or:
./scripts/bind-stress.sh --synthetic --cycles 3
# Windows:
# powershell -ExecutionPolicy Bypass -File scripts/bind-stress.ps1 -Synthetic -Cycles 3
```

| ID | Loop | Synthetic today |
|---|---|---|
| B1 | Context Establish / Release (connect/disconnect) | Practice backend |
| B2 | List readers (names only) | Seeded fictional reader name |
| B3 | Probe `CardPresent` (status flags, not `card.read`) | Yes |
| B4 | Card present/absent toggle | Auto InsertCard / RemoveCard |
| B5 | Card handle Open / Close | Practice card |
| B6 | SELECT AID Transmit — **length + SW-ok only** (P8) | Scripted `90 00` (2 bytes) |
| B7 | Concurrent monitor + read `Readers()` (P7) | Two synthetic contexts |
| B8 | `CancelWait` vs `WaitForChange` (P9) | Synthetic notify |
| B9 | `runtime.GC` between cycles (P10) | Go heap only — **not cgo** |

This **does not** close P2–P10. The report `evidence.syntheticDoesNotClose`
lists all of them and `needsACR39U` is true.

## What becomes **real only with ACR39U**

Same command, **no** `-synthetic`. Plug the reader (card seated for B5/B6).

```bash
make build
./scripts/bind-stress.sh --report ./openthaiid-bind-stress.json
# 50 loops on the clinic desk:
./scripts/bind-stress.sh --cycles 50 --report ./openthaiid-bind-stress.json
```

| P-id | Real only when | This host |
|---|---|---|
| P2 | darwin/arm64 + OS PC/SC | macOS Apple Silicon |
| P3 | darwin/amd64 + OS PC/SC | macOS Intel |
| P4 | linux/amd64 + `pcscd` | Linux x86_64 |
| P5 | linux/arm64 + `pcscd` | Linux ARM |
| P6 | windows + WinSCard load | Windows |
| P7 | Two native contexts listing / probing together | Every hardware host |
| P8 | Native `Transmit` length of SELECT (not field reads) | Hardware + card |
| P9 | `Cancel` unblocks `GetStatusChange` | Hardware reader present (empty-reader wait is a short timeout) |
| P10 | GC while cgo pointers exist | darwin / linux hardware (Windows WinSCard is not cgo) |

A macOS pass does **not** close Windows or Linux. Run the same script on
each hardware host. Order stays macOS first, then Windows, then Linux
([hardware-qualification.md](hardware-qualification.md)).

Without a card, B5/B6 SKIP (`NO_CARD` / `NO_READER`); B1–B3, B7–B9 still
exercise context + list + probe. Seat a Thai ID for P8.

B4 stays SKIP on hardware (the harness will not yank a real card). Presence
is B3. Insertion/removal cycles remain qualify **E3**.

## Capture a redacted report

The command writes **one** JSON file and prints `report: <absolute path>`.

```bash
./dist/openthaiid bind-stress -report ./openthaiid-bind-stress.json
# smoke:
./dist/openthaiid bind-stress -synthetic -cycles 3 -report ./openthaiid-bind-stress.json
```

Confirm:

1. `"schema": "openthaiid.bindstress.v1"`
2. `"pii"` says values are omitted
3. `"mode"` is `hardware` when the reader is on the desk (`synthetic-smoke` is not a binding pass)
4. No citizen ID, names, `/9j/`, `data:image`, or SELECT hex dumps
5. Attach the file. Do not paste ERP dumps or HAR with `card.read`

If a tool printed a citizen ID or APDU hex, **delete that file** and re-run.
Do not redact by hand in chat.

Technician copy is the stdout block (checks + `evidence.thisHostMayEvidence`).
Same privacy rule as `openthaiid qualify` / Copy diagnostics.

## Not this harness

- Clinic Fill / Ready / E1–E7 — [hardware-qualification.md](hardware-qualification.md)
- Laser-id / NHSO — not in the cited APDU.md revision
- Enabling GitHub Actions — still disabled until ~1 Oct 2026
