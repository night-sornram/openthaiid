# Optional loopback TLS (technician)

Friday-morning index: [getting-started.md](getting-started.md).
Fill-fails checklist: [technician.md](technician.md).

Use this when the clinic ERP is **https://** and staff use **Firefox or Safari**.
Those browsers block `ws://127.0.0.1` as mixed content (`HTTPS_WS_BLOCKED`).
Chrome/Edge still allow insecure loopback WebSockets. The Ready page stays
`http://127.0.0.1:17891/` and does not need TLS.

This is **optional** and **per clinic PC**. Plain HTTP/WS remains the default
for http ERP pages and Ready.

## What you must not do

- Do **not** disable mixed-content checks or other browser security.
- Do **not** ship or commit a private key. OpenThaiID never bundles one.
- Do **not** bind TLS off `127.0.0.1`.
- Do **not** tell clinic staff to flip Firefox `about:config` prefs.

## Once per clinic PC

1. Create a certificate whose SAN includes **`127.0.0.1`** (and `localhost` if
   you want). `mkcert` is the usual tool:

```bash
# Linux user config dir; Windows: %APPDATA%\openthaiid
# macOS: ~/Library/Application Support/openthaiid
mkdir -p "${OPENTHAIID_HOME:-$HOME/.config/openthaiid}"
cd "${OPENTHAIID_HOME:-$HOME/.config/openthaiid}"
mkcert -install
mkcert -cert-file loopback.pem -key-file loopback-key.pem 127.0.0.1 localhost
chmod 600 loopback.pem loopback-key.pem
```

`mkcert -install` trusts a **local** CA on this PC. Firefox on Linux usually
picks up the NSS store; on Windows, import the mkcert root into Firefox if
Fill still reports `TLS_UNTRUSTED`.

Without mkcert, `scripts/loopback-tls-cert.sh` writes a self-signed pair.
Then **trust that cert** in the clinic browser (Firefox: Settings →
Certificates → Authorities / Servers). Staff still must not turn off security.

2. Point the helper at those files (flags, env, or `config.json`). Relative
   paths resolve against the OpenThaiID user directory:

```bash
openthaiid tray -tls-cert loopback.pem -tls-key loopback-key.pem
# optional: -tls-port 17892 (default)

# or
export OPENTHAIID_TLS_CERT=/absolute/path/loopback.pem
export OPENTHAIID_TLS_KEY=/absolute/path/loopback-key.pem
```

`config.json`:

```json
{
  "tlsCertFile": "loopback.pem",
  "tlsKeyFile": "loopback-key.pem",
  "tlsPort": 17892
}
```

3. Confirm:
   - Ready is still `http://127.0.0.1:17891/`
   - `GET http://127.0.0.1:17891/healthz` includes `"tls": true` and `wssUrl`
   - `GET https://127.0.0.1:17892/healthz` succeeds **without** a certificate
     warning in that browser
   - Technician diagnostics show both `ws://127.0.0.1:17891/v1/ws` and
     `wss://127.0.0.1:17892/v1/ws`

4. HTTPS ERP Fill: Firefox/Safari will use `wss://127.0.0.1:17892/v1/ws`
   automatically. Chrome can keep `ws://` or set `preferLoopbackTls: true` in
   the ERP JSON. Origin Allow is unchanged. CSP `connect-src` must allow the
   WSS URL.

If the cert is not trusted, the library returns **`TLS_UNTRUSTED`** (when the
ERP asserted TLS) or **`HTTPS_WS_BLOCKED`** (Firefox/Safari auto-try). Neither
means “start OpenThaiID”. Do not disable browser security; trust the local
cert or use Chrome/Edge / an http ERP.

## ERP config flag

```json
{
  "request": ["citizenId", "thaiName", "birthDate"],
  "map": { "citizenId": "#cid" },
  "preferLoopbackTls": true
}
```

Use this when clinic IT enabled helper TLS and the page is https (including
Chrome). An explicit `"endpoint": "wss://127.0.0.1:17892/v1/ws"` also works.

## Restart

Certificate files are read at helper start. Replace files, then restart
OpenThaiID. There is no shared project key and no GitHub Actions job for this.
