# Provenance

This document records the evidence basis for Thai ID card command knowledge
and dependency licensing. It is not legal advice.

## Project licensing

- Repository license: **Apache-2.0** (`LICENSE`).
- Rationale: explicit patent grant, already present in the repository, compatible
  with permissive dependencies.
- Preserve required dependency notices in release archives.
- Apache-2.0 has compatibility implications for GPLv2-only combinations. See
  [Apache/GPL compatibility](https://www.apache.org/licenses/GPL-compatibility).

README disclaimer (required):

> This is an independent open-source project and is not affiliated with or
> endorsed by the Department of Provincial Administration.

## APDU / card-profile sources

The original community lineage for Thai National ID memory layout and APDU
sequences is documented in:

| Source | URL | Role |
|---|---|---|
| `chakphanu/ThaiNationalIDCard` | https://github.com/chakphanu/ThaiNationalIDCard | Credits APDU commands to Manoi’s HOSxP forum post |
| `dotnetthailand/ThaiNationalIDCard` | https://github.com/dotnetthailand/ThaiNationalIDCard | Successor describing Delphi example lineage and USB observation; publishes an [APDU.md](https://github.com/dotnetthailand/ThaiNationalIDCard/blob/main/APDU.md) command table |

### Classification

- These are **community references**, not an official DOPA specification.
- This research did **not** establish a publicly licensed DOPA specification
  for the memory layout.
- Code must be **independently implemented**. Do not copy forum code, unlicensed
  gists, or proprietary implementations.
- Provenance of protocol facts is separate from permission to reuse source code.
- Required profile facts used in shipped code must appear in the table below
  with source URL, immutable revision, license note, and open concerns.
- Unresolved optional profiles stay out of the release.

### Profile ledger

| Profile ID | Fields | Source URL | Immutable revision | Source license | Attribution | Concerns | Status |
|---|---|---|---|---|---|---|---|
| `thai-citizen-mvp` | `citizenId`, `thaiName`, `birthDate` | https://github.com/dotnetthailand/ThaiNationalIDCard/blob/main/APDU.md | [`4d803243715254cc93efe9040d8b1ecd85f520aa`](https://github.com/dotnetthailand/ThaiNationalIDCard/blob/4d803243715254cc93efe9040d8b1ecd85f520aa/APDU.md) | Community command table; treat as requiring independent implementation | Credit community lineage (chakphanu / Manoi / dotnetthailand) in docs | Not an official government spec; layout may vary by card generation; real-card validation still required | **Implemented independently.** SELECT `00 A4 04 00 08 A0 00 00 00 54 48 00 01`. READ BINARY form `CLA=80 INS=B0 P1=off_hi P2=off_lo Lc=02 Data=00 Le=len`. Offsets: CID `0x0004`/13; Thai full name `0x0011`/100 (single `#`-delimited block); birth `0x00D9`/8 (Buddhist Era on card, Gregorian after parse). **Release still requires physical ACR39U acceptance.** |

### Command facts used by `thai-citizen-mvp`

Independently reconstructed from the cited table (not copied from source code):

| Step | Bytes | Notes |
|---|---|---|
| SELECT | `00 A4 04 00 08 A0 00 00 00 54 48 00 01` | Direct DF name selection |
| READ CID | `80 B0 00 04 02 00 0D` | 13 ASCII digits |
| READ Thai name | `80 B0 00 11 02 00 64` | 100-byte block, `#` between title/given/middle/family |
| READ birth date | `80 B0 00 D9 02 00 08` | `YYYYMMDD` Buddhist Era |
| GET RESPONSE | `00 C0 00 00 <Le>` | Used after `61xx` |

Open concerns retained for qualification:

1. The community table is not a government specification.
2. Card generations may differ; physical ACR39U acceptance is required before release claims.
3. Name delimiter (`#`) and empty-middle handling must be confirmed on real cards during Phase G.
4. Birth-date century edge cases (cards issued before BE 2400) are out of MVP scope and must remain unverified until observed.

Before merging real-card reading for a profile:

1. Inspect and record the source license for the command table.
2. Confirm the implementation is independent (no copied source).
3. Record which offsets/commands are used and why each is required for the
   requested field set.
4. Classify remaining unknowns; do not “probe until something works.”

## Synthetic fixtures

All automated tests use synthetic fixtures under `testdata/synthetic/`.
Fixtures specify command bytes, offsets, and expected parse results
independently of implementation constants. CI must never contain captured
real-card data.

## Dependency policy

Preferred candidates (subject to actual license terms):

| Dependency | Use | License |
|---|---|---|
| [ebfe/scard](https://github.com/ebfe/scard) @ `7af069cabc25c2ba8051ea6216c0eb0d376f95fb` | PC/SC bindings | BSD-2-Clause |
| [coder/websocket](https://github.com/coder/websocket) | Loopback WebSocket | MIT-like permissive |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text/encoding/charmap) | Windows-874 decoder for Thai name fields | BSD-3-Clause |

Rejected:

| Candidate | Reason |
|---|---|
| `sf1/go-card` | Archived, unmaintained, no macOS |

Alternative (not default): `ElMostafaIdrassi/goscard` (Apache-2.0, purego ABI)
— credible, but requires extra manual native-FFI review.

### Binding audit checklist

Before production use of `ebfe/scard`:

- [ ] Native structure sizes and pointer lifetimes on target architectures
- [ ] Concurrent context and reader-state handling
- [ ] Returned buffer lengths
- [ ] Windows system-library loading
- [ ] Shutdown and blocked-call behavior

If a confirmed defect exists and upstream will not fix promptly, maintain a
narrow, attributed fork. Do not silently switch libraries or disable safety
checks.

## Personal data

Readable card content is personal data even when the operator approved the
session. Bridge approval is not a substitute for the consuming application’s
lawful collection, storage, or transfer practices.
