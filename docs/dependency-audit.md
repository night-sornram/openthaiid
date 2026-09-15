# Dependency audit checklist

Pinned dependencies for Phase 1. Complete the audit items before calling the
PC/SC path production-ready (Phase 7).

## Go

| Module | Version / revision | License | Purpose |
|---|---|---|---|
| `github.com/ebfe/scard` | `v0.0.0-20241214075232-7af069cabc25` (git `7af069cabc25c2ba8051ea6216c0eb0d376f95fb`) | BSD-2-Clause | PC/SC bindings |
| `github.com/coder/websocket` | `v1.8.12` | MIT | Loopback WebSocket server/client |

### ebfe/scard audit

- [x] Pin to immutable git revision
- [x] Record license
- [ ] Native structure sizes and pointer lifetimes on darwin/arm64
- [ ] Native structure sizes and pointer lifetimes on darwin/amd64
- [ ] Native structure sizes and pointer lifetimes on linux/amd64
- [ ] Native structure sizes and pointer lifetimes on linux/arm64
- [ ] Windows system-library loading (WinSCard)
- [ ] Concurrent context and reader-state handling
- [ ] Returned buffer lengths from `Transmit`
- [ ] Shutdown and blocked-call behavior (`Cancel` unblocks `GetStatusChange`)
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

## Rejected

| Candidate | Reason |
|---|---|
| `sf1/go-card` | Archived, unmaintained, no macOS |
| Project-owned native bindings | Highest maintenance burden; deferred |
