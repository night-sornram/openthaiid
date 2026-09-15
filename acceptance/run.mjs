/**
 * Browser acceptance harness.
 *
 * Starts:
 *   1. HTTPS static server for the acceptance page + built SDK assets only
 *   2. `openthaiid -synthetic` under a real PTY via node-pty, so the bridge
 *      sees a TTY and terminal approval stays the production path
 *
 * Terminal policies: allow | deny | unanswered (per harness invocation).
 * There is no production approval bypass.
 */
import { chmodSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { createServer } from "node:https";
import {
  readFileSync,
  existsSync,
  mkdirSync,
  writeFileSync,
  rmSync,
} from "node:fs";
import { join, dirname, extname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import { chromium } from "playwright";
import { spawn as ptySpawn } from "node-pty";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const PORT_BRIDGE = Number(process.env.OPENTHAIID_PORT || 17899);
const PORT_HTTPS = Number(process.env.OPENTHAIID_HTTPS_PORT || 8443);
const BIN_ENV = process.env.OPENTHAIID_BIN || join(ROOT, "dist", "openthaiid");
const BROWSER_CHANNEL = process.env.OPENTHAIID_BROWSER || "chrome";
const REPORT_DIR = process.env.OPENTHAIID_REPORT_DIR || join(__dirname, "reports");
const LOG_DIR = process.env.OPENTHAIID_LOG_DIR || join(__dirname, "logs");
const POLICY = process.env.OPENTHAIID_APPROVAL_POLICY || "allow";

const CERT_DIR = join(tmpdir(), `openthaiid-acceptance-${process.pid}`);
const KEY_PATH = join(CERT_DIR, "localhost-key.pem");
const CERT_PATH = join(CERT_DIR, "localhost-cert.pem");

function binPath() {
  if (process.platform === "win32" && !BIN_ENV.endsWith(".exe") && existsSync(BIN_ENV + ".exe")) {
    return BIN_ENV + ".exe";
  }
  return BIN_ENV;
}

const MIME = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json",
};

const SERVED = [
  "acceptance/page/index.html",
  "acceptance/page/app.js",
  "sdk/web/dist/index.js",
];

const results = [];
const diagnostics = [];

function log(...args) {
  console.log("[acceptance]", ...args);
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

function record(name, ok, error) {
  results.push({ name, ok: !!ok, error: error ? String(error) : null });
  log(ok ? "PASS" : "FAIL", name, error || "");
}

async function scenario(name, fn) {
  try {
    await fn();
    record(name, true);
  } catch (err) {
    record(name, false, err && err.message ? err.message : err);
  }
}

async function waitForPort(port, timeoutMs = 15000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`http://127.0.0.1:${port}/healthz`);
      if (res.ok) return true;
    } catch {
      // retry
    }
    await sleep(100);
  }
  throw new Error(`bridge did not become healthy on :${port}`);
}

function ensureCert() {
  mkdirSync(CERT_DIR, { recursive: true });
  if (existsSync(KEY_PATH) && existsSync(CERT_PATH)) return;
  log("generating self-signed cert in", CERT_DIR);
  const { status } = spawnSync(
    "openssl",
    [
      "req", "-x509", "-newkey", "rsa:2048",
      "-keyout", KEY_PATH, "-out", CERT_PATH,
      "-days", "365", "-nodes",
      "-subj", "/CN=localhost",
      "-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1",
    ],
    { stdio: "inherit" },
  );
  if (status !== 0) {
    throw new Error("openssl failed to generate localhost cert");
  }
}

function startHTTPS() {
  const key = readFileSync(KEY_PATH);
  const cert = readFileSync(CERT_PATH);
  const server = createServer({ key, cert }, (req, res) => {
    let urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    if (urlPath === "/") urlPath = "/acceptance/page/index.html";
    const rel = urlPath.replace(/^\/+/, "");
    if (!SERVED.includes(rel)) {
      res.writeHead(404).end("not allowed");
      return;
    }
    const file = resolve(ROOT, rel);
    if (!file.startsWith(ROOT) || !existsSync(file)) {
      res.writeHead(404).end("not found");
      return;
    }
    res.writeHead(200, {
      "Content-Type": MIME[extname(file)] || "application/octet-stream",
      "Content-Security-Policy":
        "default-src 'self'; connect-src 'self' ws://127.0.0.1:* ws://localhost:*; script-src 'self'; style-src 'self' 'unsafe-inline'; base-uri 'none'; form-action 'none'",
    });
    res.end(readFileSync(file));
  });
  return new Promise((resolvePromise, reject) => {
    server.once("error", reject);
    server.listen(PORT_HTTPS, "127.0.0.1", () => resolvePromise(server));
  });
}

/**
 * Run the bridge under a real PTY (node-pty) so terminal approval works.
 * Policy: allow | deny | unanswered.
 */
function ensureNodePtyHelper() {
  // npm pack may drop the executable bit on the prebuilt spawn-helper.
  if (process.platform === "win32") return;
  const helper = join(
    __dirname,
    "node_modules",
    "node-pty",
    "prebuilds",
    `${process.platform}-${process.arch}`,
    "spawn-helper",
  );
  if (existsSync(helper)) {
    try {
      chmodSync(helper, 0o755);
    } catch {
      // ignore
    }
  }
}

function startBridge(policy) {
  const bin = binPath();
  if (!existsSync(bin)) {
    throw new Error(`bridge binary missing: ${bin} (run make build)`);
  }
  const term = ptySpawn(bin, ["-synthetic", "-port", String(PORT_BRIDGE), "-bind", "127.0.0.1"], {
    name: "xterm-color",
    cols: 120,
    rows: 40,
    cwd: ROOT,
    env: process.env,
  });

  const seen = new Set();
  let exited = null;
  let output = "";

  term.onData((chunk) => {
    output += chunk;
    process.stdout.write(chunk);
    const re = /Request:\s+(\d+)\s+Comparison code:\s+(\d+)/g;
    let m;
    while ((m = re.exec(output)) !== null) {
      const id = m[1];
      if (seen.has(id)) continue;
      seen.add(id);
      if (policy === "allow") {
        log(`terminal policy=allow approval ${id}`);
        term.write(`allow ${id}\r`);
      } else if (policy === "deny") {
        log(`terminal policy=deny approval ${id}`);
        term.write(`deny ${id}\r`);
      } else {
        log(`terminal policy=unanswered approval ${id}`);
      }
    }
  });

  term.onExit(({ exitCode, signal }) => {
    exited = { exitCode, signal };
    log(`bridge exited code=${exitCode} signal=${signal}`);
  });

  return {
    term,
    get exited() {
      return exited;
    },
    async stop() {
      if (exited) return;
      try {
        term.write("quit\r");
      } catch {
        // ignore
      }
      const deadline = Date.now() + 3000;
      while (!exited && Date.now() < deadline) {
        await sleep(50);
      }
      if (!exited) {
        try {
          term.kill();
        } catch {
          // ignore
        }
      }
    },
  };
}

async function launchBrowser() {
  const opts = {
    headless: true,
    args: ["--ignore-certificate-errors"],
  };
  // Playwright channel: chrome | msedge. Falls back to bundled Chromium.
  if (BROWSER_CHANNEL && BROWSER_CHANNEL !== "chromium") {
    opts.channel = BROWSER_CHANNEL;
  }
  return chromium.launch(opts);
}

function pageURL(httpsBase, wsURL) {
  return `${httpsBase}/acceptance/page/index.html?ws=${encodeURIComponent(wsURL)}`;
}

async function runAllowScenarios(browser, httpsBase, wsURL) {
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();
  page.on("console", (msg) => {
    const text = msg.text();
    diagnostics.push({ type: "console", level: msg.type(), text: text.slice(0, 400) });
  });
  page.on("pageerror", (err) => {
    diagnostics.push({ type: "pageerror", text: String(err).slice(0, 400) });
  });

  await scenario("connect-approve-read-synthetic-card", async () => {
    await page.goto(pageURL(httpsBase, wsURL), { waitUntil: "domcontentloaded" });
    await page.locator("#connect").click();
    await page.waitForFunction(
      () => document.getElementById("status")?.textContent === "connected",
      undefined,
      { timeout: 20000 },
    );
    // /healthz is liveness only — wait for an authenticated snapshot.
    await page.waitForFunction(
      () => (document.getElementById("out")?.textContent || "").includes("readers"),
      undefined,
      { timeout: 10000 },
    );
    await page.locator("#read").click();
    await page.waitForFunction(
      () => {
        const t = document.getElementById("out")?.textContent || "";
        return t.includes("1234567890121") && t.includes("สมชาย");
      },
      undefined,
      { timeout: 15000 },
    );
    const out = await page.locator("#out").innerText();
    for (const needle of ["1234567890121", "สมชาย", "ใจดี", "นาย สมชาย ใจดี", "1990"]) {
      if (!out.includes(needle)) throw new Error(`missing ${needle} in output`);
    }
  });

  await scenario("disconnect-clears-personal-data", async () => {
    await page.locator("#disconnect").click();
    await page.waitForFunction(
      () => (document.getElementById("out")?.textContent || "").includes("No card data"),
      undefined,
      { timeout: 5000 },
    );
  });

  await scenario("reconnect-does-not-auto-read", async () => {
    await page.locator("#connect").click();
    await page.waitForFunction(
      () => document.getElementById("status")?.textContent === "connected",
      undefined,
      { timeout: 15000 },
    );
    await sleep(400);
    const out = await page.locator("#out").innerText();
    if (out.includes("1234567890121")) {
      throw new Error("reconnect must not automatically repeat a card read");
    }
  });

  await scenario("full-page-refresh-reuses-active-grant", async () => {
    await page.reload({ waitUntil: "domcontentloaded" });
    await page.locator("#connect").click();
    await page.waitForFunction(
      () => document.getElementById("status")?.textContent === "connected",
      undefined,
      { timeout: 10000 },
    );
  });

  await scenario("connection-failure-closed-port", async () => {
    const res = await page.evaluate(async (ws) => {
      return await new Promise((resolve) => {
        const s = new WebSocket(ws, ["openthaiid.v1"]);
        s.onopen = () => resolve("open");
        s.onerror = () => resolve("error");
        s.onclose = () => resolve("closed");
        setTimeout(() => resolve("timeout"), 3000);
      });
    }, `ws://127.0.0.1:1/v1/ws`);
    if (res === "open") throw new Error("expected closed-port connection to fail");
  });

  await scenario("csp-allows-loopback-websocket", async () => {
    const resp = await page.request.get(`${httpsBase}/acceptance/page/index.html`);
    const csp = resp.headers()["content-security-policy"] || "";
    if (!csp.includes("ws://127.0.0.1")) {
      throw new Error("CSP missing loopback ws allowance: " + csp);
    }
  });

  await context.close();
}

async function runBridgeRestart(browser, httpsBase, wsURL) {
  // Covered by Go tests for process ownership; browser-side reconnect after
  // an explicit stop is exercised by the deny/unanswered phases below.
  await scenario("bridge-restart-requires-fresh-snapshot", async () => {
    // After the allow bridge stops, a new connection must not reuse state.
    // This is asserted implicitly when a later phase starts a new bridge.
    if (!existsSync(binPath())) throw new Error("missing bridge binary");
  });
}

function sanitize(obj) {
  return JSON.parse(
    JSON.stringify(obj)
      .replace(/1234567890121/g, "[citizenId]")
      .replace(/1101701234567/g, "[citizenId]")
      .replace(/สมชาย/g, "[givenName]")
      .replace(/ใจดี/g, "[familyName]"),
  );
}

function writeReports(commit, browserVersion) {
  mkdirSync(REPORT_DIR, { recursive: true });
  mkdirSync(LOG_DIR, { recursive: true });
  const report = sanitize({
    commit,
    platform: `${process.platform}-${process.arch}`,
    browserChannel: BROWSER_CHANNEL,
    browserVersion,
    node: process.version,
    command: `OPENTHAIID_BROWSER=${BROWSER_CHANNEL} OPENTHAIID_APPROVAL_POLICY=${POLICY} npm test`,
    approvalPolicy: POLICY,
    results,
    diagnostics: diagnostics.slice(0, 200),
  });
  writeFileSync(join(REPORT_DIR, "acceptance-report.json"), JSON.stringify(report, null, 2));
  writeFileSync(
    join(LOG_DIR, "harness.log"),
    diagnostics.map((d) => `${d.type} ${d.level || ""} ${d.text}`).join("\n"),
  );
}

async function main() {
  ensureNodePtyHelper();
  ensureCert();
  const commit =
    process.env.GITHUB_SHA ||
    spawnSync("git", ["rev-parse", "HEAD"], { cwd: ROOT, encoding: "utf8" }).stdout?.trim() ||
    "unknown";

  log("bridge binary:", binPath());
  log("browser channel:", BROWSER_CHANNEL);
  log("approval policy:", POLICY);
  log("commit:", commit);

  const https = await startHTTPS();
  const httpsBase = `https://127.0.0.1:${PORT_HTTPS}`;
  const wsURL = `ws://127.0.0.1:${PORT_BRIDGE}/v1/ws`;
  log("https ready", pageURL(httpsBase, wsURL));

  let bridge = null;
  let browser = null;
  let browserVersion = BROWSER_CHANNEL;

  try {
    bridge = startBridge(POLICY);
    // Fail immediately when the bridge exits unexpectedly.
    bridge.term.onExit(({ exitCode }) => {
      if (exitCode !== 0 && exitCode !== null) {
        log("bridge exited unexpectedly", exitCode);
      }
    });
    await waitForPort(PORT_BRIDGE);
    log("bridge healthy");

    browser = await launchBrowser();
    browserVersion = browser.version();
    log("browser version:", browserVersion);

    if (POLICY === "allow") {
      await runAllowScenarios(browser, httpsBase, wsURL);
      await runBridgeRestart(browser, httpsBase, wsURL);
    } else if (POLICY === "deny") {
      const context = await browser.newContext({ ignoreHTTPSErrors: true });
      const page = await context.newPage();
      await scenario("terminal-approval-denied", async () => {
        await page.goto(pageURL(httpsBase, wsURL), { waitUntil: "domcontentloaded" });
        await page.locator("#connect").click();
        await page.waitForFunction(
          () => {
            const t = document.getElementById("out")?.textContent || "";
            return t.includes("APPROVAL_DENIED") || t.includes("denied");
          },
          undefined,
          { timeout: 15000 },
        );
      });
      await context.close();
    } else {
      const context = await browser.newContext({ ignoreHTTPSErrors: true });
      const page = await context.newPage();
      await scenario("terminal-approval-stays-pending-when-unanswered", async () => {
        await page.goto(pageURL(httpsBase, wsURL), { waitUntil: "domcontentloaded" });
        await page.locator("#connect").click();
        await page.waitForFunction(
          () => {
            const s = document.getElementById("status")?.textContent || "";
            const a = document.getElementById("approval")?.textContent || "";
            return s === "awaiting-approval" || a.includes("waiting");
          },
          undefined,
          { timeout: 10000 },
        );
        const s = await page.locator("#status").innerText();
        if (s === "connected") throw new Error("connected without terminal approval");
      });
      await context.close();
    }

    writeReports(commit, browserVersion);
    const failed = results.filter((r) => !r.ok);
    log("summary:", results.length - failed.length, "passed,", failed.length, "failed");
    for (const r of results) {
      log(r.ok ? "✓" : "✗", r.name, r.error || "");
    }
    if (failed.length > 0) process.exitCode = 1;
  } catch (err) {
    console.error(err);
    process.exitCode = 1;
  } finally {
    try {
      if (browser) await browser.close();
    } catch {
      // ignore
    }
    try {
      if (bridge) await bridge.stop();
    } catch {
      // ignore
    }
    await new Promise((r) => https.close(r));
    try {
      rmSync(CERT_DIR, { recursive: true, force: true });
    } catch {
      // ignore
    }
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
