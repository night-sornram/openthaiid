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
| `thai-citizen-mvp` | `citizenId`, `thaiName`, `englishName`, `birthDate`, `gender`, `issuer`, `issueDate`, `expireDate`, `address`, `portrait` | https://github.com/dotnetthailand/ThaiNationalIDCard/blob/main/APDU.md | [`4d803243715254cc93efe9040d8b1ecd85f520aa`](https://github.com/dotnetthailand/ThaiNationalIDCard/blob/4d803243715254cc93efe9040d8b1ecd85f520aa/APDU.md) | Community command table; treat as requiring independent implementation | Credit community lineage (chakphanu / Manoi / dotnetthailand) in docs | Not an official government spec; layout may vary by card generation; gender encoding and address component order are community hypotheses (see open concerns); JPEG assembly (SOI/EOI trim) is a community hypothesis; real-card validation still required | **Implemented independently.** SELECT `00 A4 04 00 08 A0 00 00 00 54 48 00 01`. READ BINARY form `CLA=80 INS=B0 P1=off_hi P2=off_lo Lc=02 Data=00 Le=len`. Offsets: CID `0x0004`/13; Thai full name `0x0011`/100; English full name `0x0075`/100; birth `0x00D9`/8; gender `0x00E1`/1; issuer `0x00F6`/100; issue `0x0167`/8; expire `0x016F`/8; **portrait 20×255-byte parts from `0x017B` (Photo_Part1/20 … Photo_Part20/20, last `0x1468`)**; address `0x1579`/100. Dates are Buddhist Era on card, Gregorian after parse. Portrait is optional (not in the default grant). **Release still requires physical ACR39U acceptance.** |

### Command facts used by `thai-citizen-mvp`

Independently reconstructed from the cited table (not copied from source code):

| Step | Bytes | Notes |
|---|---|---|
| SELECT | `00 A4 04 00 08 A0 00 00 00 54 48 00 01` | Direct DF name selection |
| READ CID | `80 B0 00 04 02 00 0D` | 13 ASCII digits |
| READ Thai name | `80 B0 00 11 02 00 64` | 100-byte block, `#` between title/given/middle/family |
| READ English name | `80 B0 00 75 02 00 64` | 100-byte block, same `#` layout as Thai name (typically ASCII) |
| READ birth date | `80 B0 00 D9 02 00 08` | `YYYYMMDD` Buddhist Era |
| READ gender | `80 B0 00 E1 02 00 01` | 1 byte; **hypothesis:** ASCII `1`=male, `2`=female |
| READ issuer | `80 B0 00 F6 02 00 64` | 100-byte TIS-620 / Windows-874 string |
| READ issue date | `80 B0 01 67 02 00 08` | `YYYYMMDD` Buddhist Era |
| READ expire date | `80 B0 01 6F 02 00 08` | `YYYYMMDD` Buddhist Era |
| READ address | `80 B0 15 79 02 00 64` | 100-byte TIS-620 block; **hypothesis:** `#` between house/village/alley/street/subdistrict/district/province |
| READ photo part 1/20 | `80 B0 01 7B 02 00 FF` | 255 bytes at `0x017B` |
| READ photo part N/20 | `80 B0` + `0x017B+(N-1)*0xFF` | 255 bytes; part 20 is `80 B0 14 68 02 00 FF` |
| GET RESPONSE | `00 C0 00 00 <Le>` | Used after `61xx` |

Open concerns retained for qualification:

1. The community table is not a government specification.
2. Card generations may differ; physical ACR39U acceptance is required before release claims.
3. Name delimiter (`#`) and empty-middle handling must be confirmed on real cards during Phase G.
4. Birth-date century edge cases (cards issued before BE 2400) are out of MVP scope and must remain unverified until observed.
5. Gender byte encoding (`1`/`2`) is **not** stated in APDU.md. It is a community convention used by independent readers; confirm on real cards. Unrecognized codes are returned with `value: null`.
6. Address 7-part `#` component order is hypothesized from the same delimiter convention as names. Prefixes such as `จังหวัด` are preserved, not stripped. Confirm on real cards.
7. English name `#` layout is assumed identical to the Thai name block; confirm on real cards.
8. Lifetime expire sentinels (for example `99999999`) are unverified; no special-case mapping is implemented.
9. Portrait JPEG assembly: concatenate 20×255 bytes, keep `FFD8`…`FFD9`, treat trailing bytes as padding. Confirm on real cards.

### Laser-id / NHSO / other DFs — not implemented

The cited APDU.md revision used for `thai-citizen-mvp` **does not** list laser-id or NHSO. Independent community implementations encode those as **source constants**, not as a published command table with an immutable reviewed revision in this project’s ledger:

| Claim | Where it appears | Why it is blocked here |
|---|---|---|
| NHSO DF AID `A0 00 00 00 54 48 00 83` and field READ BINARY offsets | `somprasongd/go-thai-smartcard` `pkg/apdu/nhso.go` (Go source); also referenced by `phakhawatchu/libthai-idcard` README | Not in the cited APDU.md table. Copying offsets from another implementation would violate independent-reconstruction. Layout/AID may vary by card generation. No license-reviewed command table with a pinned revision was adopted. |
| Laser-engraved serial | Same community libraries (`laser.go` / `show_laser_data`) | Not in APDU.md. Offsets and SELECT DF disagree across readers; treating any one gist as fact would be invention. |

Status: **document-only.** Do not ship `laserId` or NHSO fields until a citable table (URL + immutable revision + license note) is reviewed the same way as APDU.md `4d803243715254cc93efe9040d8b1ecd85f520aa`.

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
