# Vanilla example

Minimal web page that talks to a local OpenThaiID bridge.

> Full connect → approve → read flow requires bridge WebSocket dispatch
> (Phase 5–6). This scaffold documents the intended developer workflow.

## Run

```bash
# Terminal 1: bridge
go run ./cmd/openthaiid

# Terminal 2: example (once implemented with a bundler)
# Prefer serving from HTTPS in real tests; localhost HTTP is development-only.
```

The page must call `connect()` from a visible user gesture so the browser can
show a local-network permission prompt when required.

## Data handling

Clear any displayed card fields on card removal, disconnect, or revocation.
Do not store results in localStorage.
