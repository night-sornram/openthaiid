const csrf = document.querySelector('meta[name="openthaiid-csrf"]')?.content || "";
const headers = {
  "X-OpenThaiID-Manager": csrf,
  Accept: "application/json",
};

const I18N = {
  th: {
    title: "ตัวช่วย OpenThaiID",
    subtitle: "บนเครื่องนี้ · เสียบเครื่องอ่าน แล้วกด กรอกข้อมูล ในหน้าคลินิก",
    techToggle: "ช่างเทคนิค / Technician",
    techToggleHide: "ซ่อนเครื่องมือช่างเทคนิค",
    allowHeading: "เว็บไซต์ต้องการอ่านบัตร",
    allowHelp:
      "อนุญาตเฉพาะหน้าคลินิกที่กำลังใช้ ถ้าหน้าคลินิกแสดงรหัส ให้รหัสตรงกัน เว็บไซต์อนุมัติตัวเองไม่ได้",
    footer:
      "เจ้าหน้าที่ใช้ไฟสถานะด้านบนก็พอ · Independent open-source project — not affiliated with or endorsed by the Department of Provincial Administration.",
    checking: "กำลังตรวจ…",
    oneMoment: "สักครู่",
    ready: "พร้อม",
    readyPractice: "พร้อม (บัตรฝึกซ้อม)",
    notReady: "ยังไม่พร้อม",
    helperOn: "OpenThaiID กำลังทำงานบนเครื่องนี้",
    helperOff: "OpenThaiID ยังไม่ทำงาน",
    plugReader: "เสียบเครื่องอ่านบัตรเข้าเครื่องนี้",
    serviceStalled: "บริการอ่านบัตรไม่ตอบสนอง เปิด Technician tools",
    serviceOff: "บริการอ่านบัตรปิดอยู่ ให้ช่างเทคนิคเปิดบริการ",
    readerConnected: "ต่อเครื่องอ่านแล้ว",
    insertAfterReader: "เสียบบัตรหลังจากต่อเครื่องอ่านแล้ว",
    unsupportedCard: "บัตรนี้ไม่ใช่บัตรประชาชนไทย",
    insertCard: "เสียบบัตรประชาชนไทย",
    cardInserted: "เสียบบัตรแล้ว",
    nextFillClinic: "เปิดหน้าคลินิก แล้วกด กรอกข้อมูล",
    startHelper: "เปิด OpenThaiID บนเครื่องนี้",
    fixService: "ให้ช่างเทคนิคแก้ไขบริการอ่านบัตร",
    insertThenFill: "เสียบบัตรประชาชนของผู้ป่วย แล้วกด กรอกข้อมูล ในหน้าคลินิก",
    pendingAllow: "หน้าคลินิกกำลังรอ — กดอนุญาตด้านล่าง แล้วกด กรอกข้อมูล",
    multiReader: "เสียบเครื่องอ่านมากกว่าหนึ่งเครื่อง เปิด Technician tools เพื่อเลือกเครื่อง",
    practiceNext: "โหมดฝึกซ้อม: กรอกข้อมูลครั้งแรกยังต้องกดอนุญาต",
    bannerPractice:
      "โหมดฝึกซ้อม: เครื่องอ่านและบัตรจำลอง ไม่มีข้อมูลบุคคล กรอกข้อมูลครั้งแรกยังต้องกดอนุญาต",
    allow: "อนุญาต",
    deny: "ไม่อนุญาต",
    wouldFill: "จะกรอก:",
    requestedFields: "ช่องที่ขอจากบัตร",
    fields: {
      citizenId: "เลขประจำตัวประชาชน",
      thaiName: "ชื่อภาษาไทย",
      englishName: "ชื่อภาษาอังกฤษ",
      birthDate: "วันเกิด",
      gender: "เพศ",
      issuer: "ผู้ออกบัตร",
      issueDate: "วันออกบัตร",
      expireDate: "วันหมดอายุ",
      address: "ที่อยู่",
      portrait: "รูปถ่าย (ถ้าขอ)",
    },
  },
  en: {
    title: "OpenThaiID helper",
    subtitle: "On this computer · plug in the reader, then Fill in your clinic page",
    techToggle: "Technician / developer",
    techToggleHide: "Hide technician tools",
    allowHeading: "A website wants to read the card",
    allowHelp:
      "Allow only the clinic page you are using. Match the code if the clinic page shows one. This does not skip security — the website still cannot approve itself.",
    footer:
      "Clinic staff only need the lights above. Independent open-source project — not affiliated with or endorsed by the Department of Provincial Administration.",
    checking: "Checking…",
    oneMoment: "One moment.",
    ready: "Ready",
    readyPractice: "Ready (practice card)",
    notReady: "Not ready yet",
    helperOn: "OpenThaiID is running on this computer.",
    helperOff: "OpenThaiID is not running.",
    plugReader: "Plug the card reader into this computer.",
    serviceStalled: "The card service is not responding. Open Technician tools.",
    serviceOff: "The card service is off. Ask a technician to start it.",
    readerConnected: "Reader connected",
    insertAfterReader: "Insert the card after the reader is connected.",
    unsupportedCard: "This card is not a Thai National ID.",
    insertCard: "Insert the Thai ID card.",
    cardInserted: "Card is inserted.",
    nextFillClinic: "Open the clinic page and click Fill.",
    startHelper: "Start OpenThaiID on this computer.",
    fixService: "Ask a technician to fix the card service.",
    insertThenFill: "Insert the patient’s Thai ID card, then click Fill in the clinic page.",
    pendingAllow: "A clinic page is waiting — Allow it below, then click Fill.",
    multiReader: "More than one reader is plugged in. Open Technician tools and choose which one to use.",
    practiceNext: "Practice mode: Fill in the clinic page still needs Allow the first time.",
    bannerPractice:
      "Practice mode: fictional reader and card. No personal data. Fill in the clinic page still needs Allow the first time.",
    allow: "Allow",
    deny: "Don't allow",
    wouldFill: "This would fill:",
    requestedFields: "the requested card fields",
    fields: {
      citizenId: "Citizen ID",
      thaiName: "Thai name",
      englishName: "English name",
      birthDate: "Date of birth",
      gender: "Gender",
      issuer: "Issuer",
      issueDate: "Issue date",
      expireDate: "Expiry date",
      address: "Address",
      portrait: "Portrait (if requested)",
    },
  },
};

function readWantedLang() {
  try {
    const q = new URLSearchParams(location.search).get("lang");
    if (q && /^en/i.test(q)) return "en";
    if (q && /^th/i.test(q)) return "th";
  } catch {
    // ignore
  }
  try {
    const stored = sessionStorage.getItem("openthaiid-lang");
    if (stored === "en" || stored === "th") return stored;
  } catch {
    // ignore
  }
  return "th";
}

let lang = readWantedLang();

function t(key) {
  const pack = I18N[lang] || I18N.th;
  return pack[key] ?? I18N.en[key] ?? key;
}

function fieldLabels(fields) {
  const pack = I18N[lang] || I18N.th;
  if (!Array.isArray(fields) || !fields.length) return t("requestedFields");
  return fields.map((f) => pack.fields[f] || f).join(", ");
}

function applyStaticCopy() {
  document.documentElement.lang = lang;
  document.querySelectorAll("[data-i18n]").forEach((el) => {
    const key = el.getAttribute("data-i18n");
    if (key) el.textContent = t(key);
  });
  const thBtn = document.getElementById("lang-th");
  const enBtn = document.getElementById("lang-en");
  if (thBtn) thBtn.setAttribute("aria-pressed", lang === "th" ? "true" : "false");
  if (enBtn) enBtn.setAttribute("aria-pressed", lang === "en" ? "true" : "false");
  if (els && els.techToggle) {
    const open = !els.technician.hidden;
    els.techToggle.textContent = open ? t("techToggleHide") : t("techToggle");
  }
}

function setLang(next) {
  lang = next === "en" ? "en" : "th";
  try {
    sessionStorage.setItem("openthaiid-lang", lang);
  } catch {
    // ignore
  }
  applyStaticCopy();
  if (window.__lastStatus) render(window.__lastStatus);
}

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
  els.techToggle.textContent = open ? t("techToggleHide") : t("techToggle");
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

  const helperText = helperOk ? t("helperOn") : t("helperOff");
  let readerText = t("plugReader");
  let readerKind = "bad";
  if (!serviceOk) {
    readerText = backend === "stalled" ? t("serviceStalled") : t("serviceOff");
  } else if (readerOk) {
    readerText = t("readerConnected") + (readerName ? `: ${readerName}` : ".");
    readerKind = "ok";
  }

  let cardText = t("insertCard");
  let cardKind = "";
  if (!readerOk || !serviceOk) {
    cardText = t("insertAfterReader");
  } else if (unsupported && !present) {
    cardText = t("unsupportedCard");
    cardKind = "warn";
  } else if (present) {
    cardText = t("cardInserted");
    cardKind = "ok";
  }

  const ready = helperOk && serviceOk && readerOk && present;
  let headline = ready ? t("ready") : t("notReady");
  if (ready && data.synthetic) headline = t("readyPractice");

  let next = t("nextFillClinic");
  if (!helperOk) next = t("startHelper");
  else if (!serviceOk) next = t("fixService");
  else if (!readerOk) next = t("plugReader");
  else if (!present) next = t("insertThenFill");
  else if (pending) next = t("pendingAllow");
  else if (readers.length > 1 && !readers.some((r) => r.selected)) {
    next = t("multiReader");
  }
  if (ready && data.synthetic) {
    next = t("practiceNext");
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
    fields.textContent = `${t("wouldFill")} ${fieldLabels(p.fields)}`;
  }
  const actions = document.createElement("div");
  actions.className = "actions";
  actions.style.marginTop = "0.55rem";
  const allow = document.createElement("button");
  allow.type = "button";
  allow.className = "btn btn-ok";
  allow.textContent = t("allow");
  allow.addEventListener("click", () => decide(p.id, "allow"));
  const deny = document.createElement("button");
  deny.type = "button";
  deny.className = "btn btn-danger";
  deny.textContent = t("deny");
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
    els.banner.textContent = t("bannerPractice");
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

applyStaticCopy();
setTechnician(technicianWanted());
document.getElementById("lang-th")?.addEventListener("click", () => setLang("th"));
document.getElementById("lang-en")?.addEventListener("click", () => setLang("en"));
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
