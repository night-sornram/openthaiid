const csrf = document.querySelector('meta[name="openthaiid-csrf"]')?.content || "";
const headers = {
  "X-OpenThaiID-Manager": csrf,
  Accept: "application/json",
};

const FIELD_LABELS = {
  citizenId: "Citizen ID",
  thaiName: "Thai name",
  englishName: "English name",
  birthDate: "Date of birth",
  gender: "Gender",
  issuer: "Issuer",
  issueDate: "Issue date",
  expireDate: "Expiry date",
  address: "Address",
};

const els = {
  banner: document.getElementById("banner"),
  headline: document.getElementById("ready-heading"),
  next: document.getElementById("next-step"),
  lightHelper: document.getElementById("light-helper"),
  lightReader: document.getElementById("light-reader"),
  lightCard: document.getElementById("light-card"),
  userPendingCard: document.getElementById("user-pending-card"),
  userPending: document.getElementById("user-pending"),
  technician: document.getElementById("technician"),
  techToggle: document.getElementById("tech-toggle"),
  version: document.getElementById("version"),
  instance: document.getElementById("instance"),
  os: document.getElementById("os"),
  connections: document.getElementById("connections"),
  bridge: document.getElementById("bridge-state"),
  pcsc: document.getElementById("pcsc-state"),
  listen: document.getElementById("listen"),
  ws: document.getElementById("ws-url"),
  readers: document.getElementById("readers"),
  readersEmpty: document.getElementById("readers-empty"),
  pending: document.getElementById("pending"),
  pendingCount: document.getElementById("pending-count"),
  trusted: document.getElementById("trusted"),
  revokeAll: document.getElementById("revoke-all"),
  refresh: document.getElementById("refresh"),
  reconnect: document.getElementById("reconnect"),
  copyDiag: document.getElementById("copy-diag"),
  verbose: document.getElementById("verbose"),
  lastErrors: document.getElementById("last-errors"),
};

function pill(text, kind) {
  const span = document.createElement("span");
  span.className = kind ? `pill ${kind}` : "pill";
  span.textContent = text;
  return span;
}

function setText(el, text) {
  el.textContent = text;
}

function fieldLabels(fields) {
  if (!Array.isArray(fields) || !fields.length) return "the requested card fields";
  return fields.map((f) => FIELD_LABELS[f] || f).join(", ");
}

async function api(path, opts) {
  const res = await fetch(path, {
    credentials: "same-origin",
    ...opts,
    headers: {
      ...headers,
      ...(opts && opts.headers),
    },
  });
  const text = await res.text();
  let body = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = { error: text };
    }
  }
  if (!res.ok) {
    const msg = (body && (body.error || body.message)) || res.statusText;
    throw new Error(msg);
  }
  return body;
}

function technicianWanted() {
  if (location.hash === "#technician") return true;
  try {
    return sessionStorage.getItem("openthaiid-technician") === "1";
  } catch {
    return false;
  }
}

function setTechnician(open) {
  els.technician.hidden = !open;
  els.techToggle.setAttribute("aria-pressed", open ? "true" : "false");
  els.techToggle.textContent = open ? "Hide technician tools" : "Technician / developer";
  try {
    sessionStorage.setItem("openthaiid-technician", open ? "1" : "0");
  } catch {
    // ignore
  }
  if (open) {
    if (location.hash !== "#technician") location.hash = "technician";
  } else if (location.hash === "#technician") {
    history.replaceState(null, "", location.pathname + location.search);
  }
}

function setLight(el, kind, text) {
  el.className = kind ? `light-${kind}` : "";
  const label = el.querySelector(".light-text");
  if (label) label.textContent = text;
}

function pcscLabel(status, synthetic) {
  if (synthetic) {
    if (status === "available") return "available · practice reader";
    return `${status} · practice`;
  }
  return status;
}

function pcscKind(status) {
  if (status === "available") return "ok";
  if (status === "stalled") return "warn";
  return "bad";
}

function presenceLabel(p) {
  if (p === "present") return "Card present";
  if (p === "unsupported") return "Unsupported card";
  return "No card";
}

function presenceKind(p) {
  if (p === "present") return "ok";
  if (p === "unsupported") return "warn";
  return "";
}

function summarize(data) {
  const readers = data.readers || [];
  const backend = (data.backend && data.backend.status) || "unavailable";
  const helperOk = Boolean(data.running);
  const serviceOk = backend === "available";
  const readerOk = readers.length > 0;
  const present = readers.some((r) => r.presence === "present");
  const unsupported = readers.some((r) => r.presence === "unsupported");
  const selected = readers.find((r) => r.selected) || readers[0];
  const readerName = selected ? selected.name : "";
  const pending = (data.pending || []).length;

  const helperText = helperOk
    ? "OpenThaiID is running on this computer."
    : "OpenThaiID is not running.";
  let readerText = "Plug the card reader into this computer.";
  let readerKind = "bad";
  if (!serviceOk) {
    readerText =
      backend === "stalled"
        ? "The card service is not responding. Open Technician tools."
        : "The card service is off. Ask a technician to start it.";
  } else if (readerOk) {
    readerText = `Reader connected${readerName ? `: ${readerName}` : "."}`;
    readerKind = "ok";
  }

  let cardText = "Insert the Thai ID card.";
  let cardKind = "";
  if (!readerOk || !serviceOk) {
    cardText = "Insert the card after the reader is connected.";
  } else if (unsupported && !present) {
    cardText = "This card is not a Thai National ID.";
    cardKind = "warn";
  } else if (present) {
    cardText = "Card is inserted.";
    cardKind = "ok";
  }

  const ready = helperOk && serviceOk && readerOk && present;
  let headline = ready ? "Ready" : "Not ready yet";
  if (ready && data.synthetic) headline = "Ready (practice card)";

  let next = "Open the clinic page and click Fill.";
  if (!helperOk) next = "Start OpenThaiID on this computer.";
  else if (!serviceOk) next = "Ask a technician to fix the card service.";
  else if (!readerOk) next = "Plug the card reader into this computer.";
  else if (!present) next = "Insert the patient’s Thai ID card, then click Fill in the clinic page.";
  else if (pending) next = "A clinic page is waiting — Allow it below, then click Fill.";
  else if (readers.length > 1 && !readers.some((r) => r.selected)) {
    next = "More than one reader is plugged in. Open Technician tools and choose which one to use.";
  }
  if (ready && data.synthetic) {
    next = "Practice mode: Fill in the clinic page still needs Allow the first time.";
  }

  return {
    ready,
    headline,
    next,
    helperText,
    helperKind: helperOk ? "ok" : "bad",
    readerText,
    readerKind,
    cardText,
    cardKind,
  };
}

function renderHero(data) {
  const s = summarize(data);
  setText(els.headline, s.headline);
  els.headline.className = `headline ${s.ready ? "ok" : "wait"}`;
  setText(els.next, s.next);
  setLight(els.lightHelper, s.helperKind, s.helperText);
  setLight(els.lightReader, s.readerKind, s.readerText);
  setLight(els.lightCard, s.cardKind, s.cardText);
}

function allowRow(p, { detailed }) {
  const li = document.createElement("li");
  const origin = document.createElement("div");
  origin.className = "origin";
  origin.textContent = p.origin;
  const code = document.createElement("div");
  code.className = "code";
  code.textContent = p.comparisonCode || "";
  const fields = document.createElement("p");
  fields.className = "fields";
  if (detailed) {
    const fieldList = Array.isArray(p.fields) ? p.fields.join(", ") : "";
    fields.textContent = `Request ${p.id} · fields: ${fieldList} · expires ${p.expiresAt || ""}`;
  } else {
    fields.textContent = `This would fill: ${fieldLabels(p.fields)}`;
  }
  const actions = document.createElement("div");
  actions.className = "actions";
  actions.style.marginTop = "0.55rem";
  const allow = document.createElement("button");
  allow.type = "button";
  allow.className = "btn btn-ok";
  allow.textContent = "Allow";
  allow.addEventListener("click", () => decide(p.id, "allow"));
  const deny = document.createElement("button");
  deny.type = "button";
  deny.className = "btn btn-danger";
  deny.textContent = "Don't allow";
  deny.addEventListener("click", () => decide(p.id, "deny"));
  actions.append(allow, deny);
  li.append(origin, code, fields, actions);
  return li;
}

function renderUserPending(data) {
  const pending = data.pending || [];
  els.userPending.replaceChildren();
  if (!pending.length) {
    els.userPendingCard.hidden = true;
    return;
  }
  els.userPendingCard.hidden = false;
  for (const p of pending) {
    els.userPending.append(allowRow(p, { detailed: false }));
  }
}

function renderReaders(data) {
  const readers = data.readers || [];
  els.readers.replaceChildren();
  if (!readers.length) {
    els.readersEmpty.hidden = false;
    if (data.backend && data.backend.status === "unavailable") {
      els.readersEmpty.textContent =
        "Card service is down. On Linux start pcscd; on Windows start the Smart Card service; then Reconnect.";
    } else {
      els.readersEmpty.textContent = "No card readers detected. Plug in a reader and click Refresh readers.";
    }
    return;
  }
  els.readersEmpty.hidden = true;
  for (const r of readers) {
    const li = document.createElement("li");
    const row = document.createElement("div");
    row.className = "row";
    const left = document.createElement("div");
    const name = document.createElement("div");
    name.className = "name";
    name.textContent = r.name || r.readerId;
    left.append(name);
    const bits = document.createElement("div");
    bits.className = "actions";
    bits.style.marginTop = "0.35rem";
    bits.append(pill(presenceLabel(r.presence), presenceKind(r.presence)));
    if (r.selected) bits.append(pill("Selected", "ok"));
    left.append(bits);
    const actions = document.createElement("div");
    actions.className = "actions";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = r.selected ? "btn" : "btn btn-ok";
    btn.textContent = r.selected ? "Selected" : "Use this reader";
    btn.disabled = r.selected;
    btn.addEventListener("click", () => selectReader(r.readerId));
    actions.append(btn);
    row.append(left, actions);
    li.append(row);
    els.readers.append(li);
  }
}

function renderPending(data) {
  const pending = data.pending || [];
  els.pendingCount.textContent = String(pending.length);
  els.pending.replaceChildren();
  if (!pending.length) {
    const li = document.createElement("li");
    li.className = "muted";
    li.textContent = "No pending requests.";
    els.pending.append(li);
    return;
  }
  for (const p of pending) {
    els.pending.append(allowRow(p, { detailed: true }));
  }
}

function renderTrusted(data) {
  const trusted = data.trusted || [];
  els.trusted.replaceChildren();
  els.revokeAll.hidden = trusted.length === 0;
  if (!trusted.length) {
    const li = document.createElement("li");
    li.className = "muted";
    li.textContent = "No approved websites yet.";
    els.trusted.append(li);
    return;
  }
  for (const g of trusted) {
    const li = document.createElement("li");
    const row = document.createElement("div");
    row.className = "row";
    const left = document.createElement("div");
    const origin = document.createElement("div");
    origin.className = "origin";
    origin.textContent = g.origin;
    const fields = document.createElement("p");
    fields.className = "fields";
    const fieldList = Array.isArray(g.fields) ? g.fields.join(", ") : "";
    fields.textContent = `fields: ${fieldList} · until ${g.expiresAt || ""}`;
    left.append(origin, fields);
    const actions = document.createElement("div");
    actions.className = "actions";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-danger";
    btn.textContent = "Revoke";
    btn.addEventListener("click", () => revoke(g.origin));
    actions.append(btn);
    row.append(left, actions);
    li.append(row);
    els.trusted.append(li);
  }
}

function renderLastErrors(data) {
  const errs = data.lastErrors || [];
  if (!errs.length) {
    els.lastErrors.textContent = "No recent errors.";
    return;
  }
  els.lastErrors.textContent = errs
    .map((e) => `${e.at || ""}  ${e.source || ""}  ${e.code || ""}  ${e.message || ""}`.trim())
    .join("\n");
}

function diagnosticsText(data) {
  const readers = data.readers || [];
  const pending = data.pending || [];
  const trusted = data.trusted || [];
  const errs = data.lastErrors || [];
  const lines = [
    "OpenThaiID diagnostics (no personal card fields)",
    `version: ${data.version || ""}`,
    `os: ${data.os || ""}`,
    `helper: ${data.running ? "running" : "stopped"}`,
    `practiceReader: ${data.synthetic ? "yes" : "no"}`,
    `cardService: ${(data.backend && data.backend.name) || ""} ${(data.backend && data.backend.status) || ""}`,
    `listen: ${data.listenAddr || ""}`,
    `ws: ${data.wsUrl || ""}`,
    `instance: ${data.instanceId || ""}`,
    `logLevel: ${data.logLevel || ""}`,
    `erpConnections: ${data.connections ?? 0}`,
    `readers: ${readers.length}`,
  ];
  for (const r of readers) {
    lines.push(`  - ${r.name || r.readerId} presence=${r.presence}${r.selected ? " selected" : ""}`);
  }
  lines.push(`pending: ${pending.length}`);
  for (const p of pending) {
    lines.push(`  - id=${p.id} origin=${p.origin} fields=${(p.fields || []).join(",")}`);
  }
  lines.push(`approved: ${trusted.length}`);
  for (const g of trusted) {
    lines.push(`  - origin=${g.origin} fields=${(g.fields || []).join(",")}`);
  }
  lines.push("lastErrors:");
  if (!errs.length) lines.push("  (none)");
  for (const e of errs) {
    lines.push(`  - ${e.at} ${e.source} ${e.code} ${e.message}`);
  }
  return lines.join("\n");
}

function render(data) {
  window.__lastStatus = data;
  renderHero(data);
  renderUserPending(data);

  setText(els.version, data.version ? `v${data.version}` : "");
  setText(els.instance, data.instanceId || "");
  setText(els.os, data.os || "");
  setText(els.connections, String(data.connections ?? 0));
  els.bridge.replaceChildren();
  els.bridge.append(pill(data.running ? "running" : "stopped", data.running ? "ok" : "bad"));
  if (data.synthetic) {
    els.bridge.append(document.createTextNode(" "));
    els.bridge.append(pill("practice reader", "demo"));
  }
  els.pcsc.replaceChildren();
  const st = (data.backend && data.backend.status) || "unavailable";
  els.pcsc.append(pill(pcscLabel(st, data.synthetic), pcscKind(st)));
  setText(els.listen, data.listenAddr || "");
  setText(els.ws, data.wsUrl || "");
  els.verbose.checked = data.logLevel === "debug";

  if (data.synthetic) {
    els.banner.hidden = false;
    els.banner.textContent =
      "Practice mode: fictional reader and card. No personal data. Fill in the clinic page still needs Allow the first time.";
  } else {
    els.banner.hidden = true;
  }

  renderReaders(data);
  renderPending(data);
  renderTrusted(data);
  renderLastErrors(data);
}

async function load() {
  const data = await api("/v1/manager/status");
  render(data);
}

async function refresh() {
  els.refresh.disabled = true;
  try {
    await api("/v1/manager/refresh", { method: "POST" });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  } finally {
    els.refresh.disabled = false;
  }
}

async function reconnect() {
  els.reconnect.disabled = true;
  try {
    await api("/v1/manager/reconnect", { method: "POST" });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  } finally {
    els.reconnect.disabled = false;
  }
}

async function selectReader(id) {
  try {
    await api("/v1/manager/reader/select", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ readerId: id }),
    });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

async function decide(id, action) {
  try {
    await api(`/v1/manager/approvals/${id}/${action}`, { method: "POST" });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

async function revoke(origin) {
  try {
    await api("/v1/manager/trust/revoke", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ origin }),
    });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

async function revokeAll() {
  if (!window.confirm("Revoke every approved website on this PC?")) return;
  try {
    await api("/v1/manager/trust/revoke-all", { method: "POST" });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

async function copyDiagnostics() {
  const data = window.__lastStatus;
  if (!data) return;
  const text = diagnosticsText(data);
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(text);
    } else {
      throw new Error("no clipboard");
    }
    els.copyDiag.textContent = "Copied";
  } catch {
    window.prompt("Copy diagnostics", text);
    els.copyDiag.textContent = "Copy diagnostics";
    return;
  }
  setTimeout(() => {
    els.copyDiag.textContent = "Copy diagnostics";
  }, 1500);
}

async function setVerbose(on) {
  try {
    await api("/v1/manager/log-level", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ verbose: on }),
    });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
    els.verbose.checked = !on;
  }
}

setTechnician(technicianWanted());
els.techToggle.addEventListener("click", () => {
  setTechnician(els.technician.hidden);
});
window.addEventListener("hashchange", () => {
  setTechnician(technicianWanted());
});
els.refresh.addEventListener("click", () => void refresh());
els.reconnect.addEventListener("click", () => void reconnect());
els.revokeAll.addEventListener("click", () => void revokeAll());
els.copyDiag.addEventListener("click", () => void copyDiagnostics());
els.verbose.addEventListener("change", () => void setVerbose(els.verbose.checked));
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") void load();
});

void load();
setInterval(() => {
  if (document.visibilityState === "visible") void load();
}, 1500);
