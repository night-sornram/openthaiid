# Protocol v1

## Transport

- Default endpoint: `ws://127.0.0.1:17891/v1/ws`
- Subprotocol: `openthaiid.v1`
- Health: `GET /healthz` — liveness only. No readers, grants, or card state.
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
- Unknown operations and unsupported protocol majors are rejected.

## Operations

| Type | Direction | Purpose |
|---|---|---|
| `session.open` | client | Request authentication with intended fields |
| `session.renew` | client | Refresh an active session token |
| `session.close` | client | End the session |
| `bridge.status` | client | Request current status snapshot |
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
  "protocol": { "major": 1, "minor": 0 },
  "fieldsSupported": ["citizenId", "thaiName", "birthDate"]
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
    "birthDate": { "year": 2533, "month": 4, "day": 12 }
  },
  "warnings": []
}
```

- Requested fields must be a subset of the approved grant.
- Invalid or unavailable individual fields become `null` plus a
  field-specific warning.
- Transport loss, unsupported card, or generation change fails the whole read.

## Events

| Type | Payload sketch |
|---|---|
| `approval.required` | `approvalId`, requested fields, comparison code (metadata only) |
| `status.snapshot` | connection/backend/reader/card view |
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

Messages never contain raw field contents, command responses, or native error
dumps.

## Normalized MVP fields

```ts
type CardDate = {
  year: number | null;
  month: number | null;
  day: number | null;
};

type ThaiName = {
  title: string | null;
  givenName: string | null;
  middleName: string | null;
  familyName: string | null;
  fullName: string;
};
```

| Field | Representation |
|---|---|
| `citizenId` | String preserving all 13 digits |
| `thaiName` | `ThaiName` |
| `birthDate` | `CardDate` (Buddhist Era year when the profile uses BE) |

## Synthetic examples

See `testdata/synthetic/` for parser fixtures and `docs/acceptance.md` for
protocol acceptance cases.
