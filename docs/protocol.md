# Protocol v1

## Transport

- Default endpoint: `ws://127.0.0.1:17891/v1/ws`
- Subprotocol: `openthaiid.v1`
- Health: `GET /healthz` — liveness only. No readers, grants, or card state.
- Local Manager: `GET /` — HTML device UI. JSON under `/v1/manager/*`
  (CSRF header + manager origin; no CORS). Status includes reader names and
  card present/absent only — never personal field values.
- Frame limit: 64 KiB text-only. Compression disabled.
- Port is explicit configuration. No port scanning or silent fallback.

## Envelope

Client → server request:

```json
{
  "v": 1,
  "id": 3,
  "type": "card.read",
  "payload": {
    "readerId": "reader-opaque-id",
    "cardSessionId": "insertion-opaque-id",
    "fields": ["thaiName", "birthDate"]
  }
}
```

Server → client response: same `id`, plus either `result` or `error`.

Server → client event:

```json
{
  "v": 1,
  "type": "card.inserted",
  "instanceId": "bridge-instance",
  "sequence": 12,
  "payload": {
    "readerId": "reader-opaque-id",
    "cardSessionId": "insertion-opaque-id"
  }
}
```

Rules:

- `id` increases monotonically per connection. Reuse is rejected.
- After authentication the server sends a fresh ordered status snapshot.
- Event `sequence` detects gaps; reconnect obtains a new snapshot.
- Historical events and read responses are not replayed or cached.
- Card results are unicast to the requester only.
- Supported fields and capabilities are advertised after authentication.
- Protocol **minor 4** adds `selected` on reader views in `bridge.probe`,
  `bridge.status`, and `status.snapshot` (operator-chosen reader, or the only
  reader). Protocol **minor 3** adds optional `portrait` (JPEG Base64) to `fieldsSupported`.
  Protocol **minor 2** added `bridge.probe` without changing the envelope.
- Unknown operations and unsupported protocol majors are rejected.
- Each connection allows at most **eight** concurrent asynchronous operations
  (`session.open`, `session.renew`, `bridge.status`, `bridge.probe`, `card.read`). Excess
  requests receive `RATE_LIMITED`. `session.close` and `request.cancel` stay
  synchronous and remain responsive under the limit.
- If a client is too slow to drain events, or the internal reader-event queue
  overflows, the connection is closed with WebSocket status **1013**
  (Try Again Later). Reconnect obtains a fresh snapshot; events are not
  replayed. Reconnect never automatically repeats a card read.

## Operations

| Type | Direction | Purpose |
|---|---|---|
| `session.open` | client | Request authentication with intended fields |
| `session.renew` | client | Refresh an active session token |
| `session.close` | client | End the session |
| `bridge.status` | client | Request current status snapshot (authenticated) |
| `bridge.probe` | client | Metadata only, no session: helper version, PC/SC, readers (`selected` when the operator chose one, or when only one reader exists), card present/absent. No personal fields. Used by `probeThaiID()` |
| `card.read` | client | Read selected fields from the current card session |
| `request.cancel` | client | Cancel an in-flight operation by request id |

### `session.open` payload

```json
{
  "fields": ["citizenId", "thaiName", "birthDate"]
}
```

Success result:

```json
{
  "token": "<opaque>",
  "expiresAt": "2026-01-01T00:15:00Z",
  "fields": ["citizenId", "thaiName", "birthDate"],
  "instanceId": "bridge-instance",
  "protocol": { "major": 1, "minor": 1 },
  "fieldsSupported": ["citizenId", "thaiName", "englishName", "birthDate", "gender", "issuer", "issueDate", "expireDate", "address", "portrait"]
}
```

Pairing-only (`APPROVAL_REQUIRED`) result carries a pending approval id the
application may display while the operator acts in the terminal. The approval
id is not a capability.

### `card.read` payload

```json
{
  "readerId": "reader-opaque-id",
  "cardSessionId": "insertion-opaque-id",
  "fields": ["thaiName", "birthDate"]
}
```

Success result:

```json
{
  "readerId": "reader-opaque-id",
  "cardSessionId": "insertion-opaque-id",
  "fields": {
    "thaiName": {
      "title": "นาย",
      "givenName": "สมชาย",
      "middleName": null,
      "familyName": "ใจดี",
      "fullName": "นาย สมชาย ใจดี"
    },
    "birthDate": { "year": 1990, "month": 4, "day": 12 }
  },
  "warnings": []
}
```

- Requested fields must be a subset of the approved grant.
- Invalid or unavailable individual fields become `null` plus a
  field-specific warning. Unrequested fields are omitted.
- Transport loss, unsupported card, or generation change fails the whole read.
- Omit `readerId` to use the operator-selected reader (or the only reader).
  Two or more readers with no selection return `READER_SELECTION_REQUIRED`.

## Events

| Type | Payload sketch |
|---|---|
| `approval.required` | `approvalId`, requested fields, comparison code (metadata only) |
| `status.snapshot` | connection/backend/reader/card view (`readers[].selected`) |
| `reader.connected` | `readerId`, label |
| `reader.disconnected` | `readerId` |
| `card.inserted` | `readerId`, `cardSessionId` |
| `card.removed` | `readerId`, `cardSessionId` |
| `authorization.revoked` | reason code |

An initial snapshot may report an already-inserted card without fabricating a
new insertion event.

## Errors

```ts
type ThaiIDError = {
  code: string;
  message: string;
  retryable: boolean;
  requestId?: number;
};
```

| Group | Codes |
|---|---|
| Connection | `BRIDGE_UNAVAILABLE`, `CONNECTION_FAILED`, `PROTOCOL_MISMATCH` |
| Authorization | `APPROVAL_REQUIRED`, `APPROVAL_DENIED`, `APPROVAL_TIMEOUT`, `AUTH_EXPIRED`, `FIELD_NOT_ALLOWED` |
| Hardware | `PCSC_UNAVAILABLE`, `NO_READER`, `READER_SELECTION_REQUIRED`, `NO_CARD`, `CARD_REMOVED`, `READER_REMOVED`, `BUSY` |
| Reading | `UNSUPPORTED_CARD`, `READ_FAILED`, `READ_TIMEOUT`, `CANCELLED`, `BACKEND_STALLED` |
| Validation | `INVALID_REQUEST`, `UNSUPPORTED_FIELD`, `RATE_LIMITED` |

### Read ownership and recovery

- One native worker owns open → transaction → transmit → end → close.
- `CANCELLED` and `READ_TIMEOUT` return promptly; blocked native cleanup does
  not delay the client.
- The global read slot stays occupied until that worker finishes. Overlapping
  reads receive `BUSY`. Late results are discarded.
- If native work is still blocked two seconds after cancellation, the backend
  is marked `BACKEND_STALLED`. Further reads fail with `BACKEND_STALLED` until
  the worker finishes and monitoring recovery completes.
- Shutdown never releases a PC/SC context still owned by a worker; it returns
  at its deadline instead.

Messages never contain raw field contents, command responses, or native error
dumps. Wire `message` text is English for technicians. Clinic ERP helpers
(`autofillThaiID`, `probeThaiID`, `formatStaffError`) replace the human
message with Thai staff guidance by default; **`code` does not change**.
Pass `{ lang: "en" }` for English staff guidance.

## Normalized textual fields

```ts
type CardDate = {
  year: number | null;
  month: number | null;
  day: number | null;
};

type PersonName = {
  title: string | null;
  givenName: string | null;
  middleName: string | null;
  familyName: string | null;
  fullName: string;
};

type Gender = {
  code: string;                 // raw card digit, typically "1" or "2"
  value: "male" | "female" | null;
};

type ThaiAddress = {
  houseNo: string | null;
  villageNo: string | null;     // หมู่ที่
  alley: string | null;         // ซอย
  street: string | null;
  subdistrict: string | null;
  district: string | null;
  province: string | null;
  fullAddress: string;
};
```

Default `session.open` / SDK field set remains `citizenId`, `thaiName`, `birthDate`. Extra fields are returned only when requested **and** granted.

| Field | Representation |
|---|---|
| `citizenId` | String preserving all 13 digits |
| `thaiName` | `PersonName` (single `#`-delimited 100-byte block) |
| `englishName` | `PersonName` (same `#` layout; typically ASCII) |
| `birthDate` | `CardDate` with **Gregorian** year (card stores Buddhist Era; bridge converts BE→CE and validates leap years) |
| `gender` | `Gender`. Community convention (not in APDU.md): `"1"` → `male`, `"2"` → `female`. Unrecognized codes keep `code` and set `value` to `null`. |
| `issuer` | Trimmed TIS-620 / Windows-874 string |
| `issueDate` | `CardDate` (same BE→CE rules as `birthDate`) |
| `expireDate` | `CardDate` (same BE→CE rules as `birthDate`) |
| `address` | `ThaiAddress`. Community hypothesis: `#` between house / village / alley / street / subdistrict / district / province. Card prefixes such as `จังหวัด` are preserved. |
| `portrait` | Optional JPEG. `{ mimeType: "image/jpeg", encoding: "base64", data: "<std base64>" }`. Twenty 255-byte READ BINARY parts from offset `0x017B` (cited Photo_Part1/20 … 20/20). Apps typically use `portraitDataUrl()` (`data:image/jpeg;base64,…`) or `portraitBlob()`. Not in the default grant. Photo bytes must not appear in logs or acceptance reports. |

Laser-id and NHSO are not protocol fields (see `docs/provenance.md`).
## Synthetic examples

See `testdata/synthetic/` for parser fixtures and `docs/acceptance.md` for
protocol acceptance cases.
