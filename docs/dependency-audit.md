# Dependency audit checklist

Pinned dependencies for Phase 1. Complete the audit items before calling the
PC/SC path production-ready (Phase 7).

## Go

| Module | Version / revision | License | Purpose |
|---|---|---|---|
| `github.com/ebfe/scard` | `v0.0.0-20241214075232-7af069cabc25` (git `7af069cabc25c2ba8051ea6216c0eb0d376f95fb`) | BSD-2-Clause | PC/SC bindings |
| `github.com/coder/websocket` | `v1.8.12` | MIT | Loopback WebSocket server/client |

### ebfe/scard audit

Compilation and local synthetic tests do **not** close these items.

- [x] Pin to immutable git revision
- [x] Record license
- [ ] Native structure sizes and pointer lifetimes on darwin/arm64
- [ ] Native structure sizes and pointer lifetimes on darwin/amd64
- [ ] Native structure sizes and pointer lifetimes on linux/amd64
- [ ] Native structure sizes and pointer lifetimes on linux/arm64
- [ ] Windows system-library loading (WinSCard)
- [ ] Concurrent context and reader-state handling
      *Partial: monitor and read use separate Backend/context instances; a
      single native worker owns each read. Hardware stress still required.*
- [ ] Returned buffer lengths from `Transmit`
      *Partial: engine rejects bodies larger than the profile `MaxResponse`
      bound. Native length reporting still needs platform verification.*
- [ ] Shutdown and blocked-call behavior (`Cancel` unblocks `GetStatusChange`)
      *Partial: reader manager never releases a PC/SC context while a worker
      holds the read slot; Stop returns at its deadline. Confirm scard.Cancel
      behavior on each OS with a blocked GetStatusChange.*
- [ ] GC interaction with cgo pointers under stress

If a confirmed defect is found and upstream will not fix promptly, maintain a
narrow, attributed fork. Do not silently switch libraries.

## Planned (not yet required)

| Module | Use | When |
|---|---|---|
| `github.com/coder/websocket` | Bridge WebSocket | Phase 5 |
| `golang.org/x/text` | Thai encoding helpers | Phase 3 |

## npm (`sdk/web`)

Dev-only for now (typescript, vitest). Production runtime has zero dependencies.

- [ ] `npm audit` review on release branch
- [ ] License scan of the published tarball

## npm (`acceptance`)

Dev-only harness. Never shipped to end users.

| Module | Version | Purpose |
|---|---|---|
| `playwright` | ^1.63.0 | Browser automation (chrome / msedge channels) |
| `node-pty` | 1.1.0 | Cross-platform PTY for real terminal approval |

- [x] `node-pty` pinned to exact `1.1.0`
- [ ] `npm audit` review on release branch
- [ ] Confirm node-pty native prebuilds on Windows CI runner

## Rejected

| Candidate | Reason |
|---|---|
| `sf1/go-card` | Archived, unmaintained, no macOS |
| Project-owned native bindings | Highest maintenance burden; deferred |
