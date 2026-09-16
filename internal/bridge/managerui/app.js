const csrf = document.querySelector('meta[name="openthaiid-csrf"]')?.content || "";
const headers = {
  "X-OpenThaiID-Manager": csrf,
  Accept: "application/json",
};

const els = {
  banner: document.getElementById("banner"),
  version: document.getElementById("version"),
  instance: document.getElementById("instance"),
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

function pcscLabel(status) {
  const data = window.__lastStatus;
  if (!data) return status;
  if (data.synthetic) {
    if (status === "available") return "available · synthetic demo";
    return `${status} · synthetic`;
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

function renderReaders(data) {
  const readers = data.readers || [];
  els.readers.replaceChildren();
  if (!readers.length) {
    els.readersEmpty.hidden = false;
    if (data.backend && data.backend.status === "unavailable") {
      els.readersEmpty.textContent =
        "PC/SC is down. On Linux start pcscd; on Windows start the Smart Card service; then Refresh.";
    } else {
      els.readersEmpty.textContent = "No card readers detected. Plug in a reader and click Refresh.";
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
    li.textContent = "No pending requests. Keep this page open while staff use the ERP.";
    els.pending.append(li);
    return;
  }
  for (const p of pending) {
    const li = document.createElement("li");
    const origin = document.createElement("div");
    origin.className = "origin";
    origin.textContent = p.origin;
    const code = document.createElement("div");
    code.className = "code";
    code.textContent = p.comparisonCode || "";
    const fields = document.createElement("p");
    fields.className = "fields";
    const fieldList = Array.isArray(p.fields) ? p.fields.join(", ") : "";
    fields.textContent = `Request ${p.id} · fields: ${fieldList} · expires ${p.expiresAt || ""}`;
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
    deny.textContent = "Deny";
    deny.addEventListener("click", () => decide(p.id, "deny"));
    actions.append(allow, deny);
    li.append(origin, code, fields, actions);
    els.pending.append(li);
  }
}

function renderTrusted(data) {
  const trusted = data.trusted || [];
  els.trusted.replaceChildren();
  els.revokeAll.hidden = trusted.length === 0;
  if (!trusted.length) {
    const li = document.createElement("li");
    li.className = "muted";
    li.textContent = "No trusted websites yet.";
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

function render(data) {
  window.__lastStatus = data;
  setText(els.version, data.version ? `v${data.version}` : "");
  setText(els.instance, data.instanceId || "");
  els.bridge.replaceChildren();
  els.bridge.append(pill(data.running ? "running" : "stopped", data.running ? "ok" : "bad"));
  if (data.synthetic) {
    els.bridge.append(document.createTextNode(" "));
    els.bridge.append(pill("synthetic demo", "demo"));
  }
  els.pcsc.replaceChildren();
  const st = (data.backend && data.backend.status) || "unavailable";
  els.pcsc.append(pill(pcscLabel(st), pcscKind(st)));
  setText(els.listen, data.listenAddr || "");
  setText(els.ws, data.wsUrl || "");

  if (data.synthetic) {
    els.banner.hidden = false;
    els.banner.textContent =
      "Demo mode: synthetic reader and a fictional card. No personal data. ERP autofill still needs Allow below.";
  } else {
    els.banner.hidden = true;
  }

  renderReaders(data);
  renderPending(data);
  renderTrusted(data);
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
  if (!window.confirm("Revoke every trusted website on this PC?")) return;
  try {
    await api("/v1/manager/trust/revoke-all", { method: "POST" });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

els.refresh.addEventListener("click", () => void refresh());
els.revokeAll.addEventListener("click", () => void revokeAll());
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") void load();
});

void load();
setInterval(() => {
  if (document.visibilityState === "visible") void load();
}, 1500);
