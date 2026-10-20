const csrfMeta = document.querySelector('meta[name="openthaiid-csrf"]');
let csrf = csrfMeta?.content || "";
const headers = {
  "X-OpenThaiID-Manager": csrf,
  Accept: "application/json",
};

function setCsrf(token) {
  csrf = token || "";
  headers["X-OpenThaiID-Manager"] = csrf;
  if (csrfMeta) csrfMeta.content = csrf;
}

function clearChildren(el) {
  if (!el) return;
  if (typeof el.replaceChildren === "function") {
    el.replaceChildren();
    return;
  }
  while (el.firstChild) el.removeChild(el.firstChild);
}

async function copyText(text, promptLabel) {
  if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Firefox/Safari may deny clipboard.writeText; fall through.
    }
  }
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.top = "0";
    ta.style.left = "0";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    const ok = typeof document.execCommand === "function" && document.execCommand("copy");
    ta.remove();
    if (ok) return true;
  } catch {
    // ignore
  }
  window.prompt(promptLabel || "Copy", text);
  return false;
}

const I18N = {
  th: {
    title: "ตัวช่วย OpenThaiID",
    subtitle: "บนเครื่องนี้ · เสียบเครื่องอ่าน แล้วกด กรอกข้อมูล ในหน้าคลินิก",
    techToggle: "ช่างเทคนิค / Technician",
    techToggleHide: "ซ่อนเครื่องมือช่างเทคนิค",
    allowHeading: "เว็บคลินิกขอใช้บัตรประชาชน",
    allowHelp: "อนุญาตเฉพาะหน้าคลินิกที่กำลังใช้ ถ้าหน้าคลินิกมีรหัส ให้รหัสตรงกัน",
    footer:
      "เจ้าหน้าที่ใช้ไฟสถานะด้านบนก็พอ · Independent open-source project — not affiliated with or endorsed by the Department of Provincial Administration.",
    checking: "กำลังตรวจ…",
    oneMoment: "สักครู่",
    ready: "พร้อม",
    readyPractice: "พร้อม (บัตรฝึกซ้อม)",
    notReady: "ยังไม่พร้อม",
    lightOk: "ปกติ",
    lightWait: "รอ",
    lightWarn: "ต้องตรวจ",
    lightBad: "ยังไม่พร้อม",
    helperOn: "OpenThaiID กำลังทำงานบนเครื่องนี้",
    helperOff: "OpenThaiID ยังไม่ทำงาน",
    plugReader: "เสียบเครื่องอ่านบัตรเข้าเครื่องนี้",
    serviceStalled: "บริการอ่านบัตรไม่ตอบสนอง เปิด Technician tools — ไม่ต้องรีสตาร์ทเครื่อง",
    serviceOff: "บริการอ่านบัตรยังไม่พร้อม (หลังเครื่องหลับก็เจอได้) ไม่ต้องรีสตาร์ทเครื่อง",
    readerConnected: "ต่อเครื่องอ่านแล้ว",
    insertAfterReader: "เสียบบัตรหลังจากต่อเครื่องอ่านแล้ว",
    unsupportedCard: "บัตรนี้ไม่ใช่บัตรประชาชนไทย",
    insertCard: "เสียบบัตรประชาชนไทย",
    cardInserted: "เสียบบัตรแล้ว",
    nextFillClinic: "เปิดหน้าคลินิก แล้วกด กรอกข้อมูล",
    startHelper: "เปิด OpenThaiID บนเครื่องนี้",
    fixService: "กด ทดสอบการเชื่อมต่อ หรือให้ช่างเทคนิคกด Reconnect card service แล้วรอ พร้อม",
    insertThenFill: "เสียบบัตรประชาชนของผู้ป่วย แล้วกด กรอกข้อมูล ในหน้าคลินิก",
    pendingAllow: "หน้าคลินิกกำลังรอ — กด อนุญาต ด้านล่าง แล้วกลับไปกด กรอกข้อมูล",
    multiReader: "มีเครื่องอ่านมากกว่าหนึ่งเครื่อง เลือกเครื่องด้านล่าง",
    usingThisReader: "กำลังใช้",
    useThisReader: "ใช้เครื่องนี้",
    readersHeading: "เลือกเครื่องอ่าน",
    readersHelp: "เสียบเครื่องอ่านมากกว่าหนึ่งเครื่อง กดใช้เครื่องที่เสียบบัตรผู้ป่วย",
    staffCardPresent: "มีบัตร",
    staffNoCard: "ยังไม่มีบัตร",
    practiceNext: "โหมดฝึกซ้อม: กรอกข้อมูลครั้งแรกยังต้องกดอนุญาต",
    bannerPractice:
      "โหมดฝึกซ้อม: เครื่องอ่านและบัตรจำลอง ไม่มีข้อมูลบุคคล กรอกข้อมูลครั้งแรกยังต้องกดอนุญาต",
    helperLostHeadline: "ตัวช่วยหลุดการเชื่อมต่อ",
    helperWaiting: "กำลังรอให้ OpenThaiID กลับมาบนเครื่องนี้ ไม่ต้องปิดหน้านี้",
    helperBack: "ตัวช่วยกลับมาแล้ว กด กรอกข้อมูล ได้ตามปกติ",
    serviceBack: "บริการอ่านบัตรกลับมาแล้ว",
    serviceBackReady: "บริการอ่านบัตรกลับมาแล้ว — พร้อม",
    readerUnknown: "เครื่องอ่านบัตร: ยังไม่ทราบ จนกว่าตัวช่วยจะกลับมา",
    cardUnknown: "บัตร: ยังไม่ทราบ จนกว่าตัวช่วยจะกลับมา",
    testConnection: "ทดสอบการเชื่อมต่อ",
    testingConnection: "กำลังทดสอบการเชื่อมต่อ…",
    testOk: "การเชื่อมต่อปกติ — ไม่ได้อ่านข้อมูลบนบัตร",
    testFail: "ทดสอบการเชื่อมต่อไม่สำเร็จ กำลังรอให้ OpenThaiID กลับมา",
    testServiceOff: "ตัวช่วยทำงาน แต่บริการอ่านบัตรยังไม่พร้อม — ไม่ได้อ่านข้อมูลบนบัตร",
    testServiceStalled: "ตัวช่วยทำงาน แต่บริการอ่านบัตรไม่ตอบสนอง — ไม่ได้อ่านข้อมูลบนบัตร",
    testWsFail: "หน้านี้ติดต่อช่องทางเดียวกับหน้าคลินิกไม่ได้",
    checkUpdate: "ตรวจเวอร์ชัน",
    checkingUpdate: "กำลังตรวจเวอร์ชัน…",
    updateFail: "ตรวจเวอร์ชันจาก GitHub ไม่ได้ — กรอกข้อมูลได้ตามปกติ คำสั่งนี้ไม่ดาวน์โหลดและไม่ติดตั้ง",
    updateOpenRelease: "เปิดหน้า GitHub Release",
    trainingTitle: "ฝึกใช้ก่อนได้เครื่องจริง",
    trainingHelp:
      "ใช้ตัวช่วยโหมดฝึกซ้อม ไม่ต้องมีเครื่องอ่านจริง ครั้งแรกยังต้องกดอนุญาต ห้ามส่งภาพหน้าจอหรือค่าในช่องไปในทิคเก็ต",
    training1: "เปิดตัวช่วยด้วยคำสั่ง openthaiid -synthetic (หน้านี้ถ้าเห็นบัตรฝึกซ้อม แปลว่าเปิดแล้ว)",
    training2: "เปิดตัวอย่างคลินิก examples/clinic-patient/ จากโฟลเดอร์โปรเจกต์",
    training3: "กด กรอกข้อมูล ในหน้าคลินิก",
    training4: "กลับมาที่นี่ แล้วกด อนุญาต สำหรับหน้านั้น (ให้รหัสตรงกันถ้ามี)",
    training5: "ช่องในหน้าคลินิกมีข้อมูลบัตรจำลอง — นั่นไม่ใช่ผู้ป่วยจริง",
    allow: "อนุญาต",
    deny: "ไม่อนุญาต",
    allowConfirmed: "อนุญาตแล้ว — กลับไปหน้าคลินิก กด กรอกข้อมูล ได้ต่อ",
    denyNext: "คุณกดไม่อนุญาต ถ้าเป็นหน้าคลินิกที่ใช้จริง ให้กด กรอกข้อมูล อีกครั้ง แล้วกด อนุญาต",
    siteOnThisComputer: "หน้าคลินิกบนเครื่องนี้",
    unknownSite: "หน้าคลินิก",
    matchCode: "รหัสที่ต้องตรงกับหน้าคลินิก",
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
    allowHeading: "The clinic website wants to use the ID card",
    allowHelp: "Allow only the clinic page you are using. Match the code if the clinic page shows one.",
    footer:
      "Clinic staff only need the lights above. Independent open-source project — not affiliated with or endorsed by the Department of Provincial Administration.",
    checking: "Checking…",
    oneMoment: "One moment.",
    ready: "Ready",
    readyPractice: "Ready (practice card)",
    notReady: "Not ready yet",
    lightOk: "OK",
    lightWait: "Waiting",
    lightWarn: "Check",
    lightBad: "Not ready",
    helperOn: "OpenThaiID is running on this computer.",
    helperOff: "OpenThaiID is not running.",
    plugReader: "Plug the card reader into this computer.",
    serviceStalled: "The card service is not responding. Open Technician tools — do not reboot.",
    serviceOff: "The card service is not ready (this can happen after sleep). Do not reboot.",
    readerConnected: "Reader connected",
    insertAfterReader: "Insert the card after the reader is connected.",
    unsupportedCard: "This card is not a Thai National ID.",
    insertCard: "Insert the Thai ID card.",
    cardInserted: "Card is inserted.",
    nextFillClinic: "Open the clinic page and click Fill.",
    startHelper: "Start OpenThaiID on this computer.",
    fixService: "Click Test connection, or ask a technician to click Reconnect card service, then wait until Ready.",
    insertThenFill: "Insert the patient’s Thai ID card, then click Fill in the clinic page.",
    pendingAllow: "A clinic page is waiting — click Allow below, then click Fill.",
    multiReader: "More than one reader is plugged in. Choose one below.",
    usingThisReader: "Using this reader",
    useThisReader: "Use this reader",
    readersHeading: "Choose a reader",
    readersHelp: "More than one reader is plugged in. Use the reader that has the patient’s card.",
    staffCardPresent: "Card inserted",
    staffNoCard: "No card",
    practiceNext: "Practice mode: Fill in the clinic page still needs Allow the first time.",
    bannerPractice:
      "Practice mode: fictional reader and card. No personal data. Fill in the clinic page still needs Allow the first time.",
    helperLostHeadline: "The helper disconnected",
    helperWaiting: "Waiting for OpenThaiID to come back on this computer. Leave this page open.",
    helperBack: "The helper is back. Click Fill as usual.",
    serviceBack: "The card service is back",
    serviceBackReady: "The card service is back — Ready",
    readerUnknown: "Card reader: unknown until the helper is back.",
    cardUnknown: "Card: unknown until the helper is back.",
    testConnection: "Test connection",
    testingConnection: "Testing connection…",
    testOk: "Connection is OK — no personal card fields were read.",
    testFail: "Connection test failed. Waiting for OpenThaiID to come back.",
    testServiceOff: "The helper is running, but the card service is not ready — no personal card fields were read.",
    testServiceStalled: "The helper is running, but the card service is not responding — no personal card fields were read.",
    testWsFail: "This page could not reach the same channel the clinic page uses.",
    checkUpdate: "Check version",
    checkingUpdate: "Checking GitHub Releases…",
    updateFail: "Could not check GitHub Releases — Fill still works. This check does not download or install.",
    updateOpenRelease: "Open the GitHub Release",
    trainingTitle: "Practice before the real reader arrives",
    trainingHelp:
      "Use the practice helper. No physical reader required. The first Fill still needs Allow. Do not send screenshots or field values in tickets.",
    training1: "Start the helper with openthaiid -synthetic (this page already shows a practice card if that is running).",
    training2: "Open the clinic example at examples/clinic-patient/ from the repo.",
    training3: "Click Fill on the clinic page.",
    training4: "Come back here and click Allow for that clinic page (match the code if one is shown).",
    training5: "The clinic form shows fictional card data — not a real patient.",
    allow: "Allow",
    deny: "Don't allow",
    allowConfirmed: "Allowed. Return to the clinic page and click Fill.",
    denyNext: "You chose Don't allow. If this is the clinic page you use, click Fill again, then Allow.",
    siteOnThisComputer: "Clinic page on this computer",
    unknownSite: "Clinic page",
    matchCode: "Match this code on the clinic page",
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
  if (els.testConnection) els.testConnection.textContent = t("testConnection");
  if (els.checkUpdate) els.checkUpdate.textContent = t("checkUpdate");
  if (window.__helperOffline) {
    renderOffline();
    refreshTestResultCopy();
    refreshUpdateResultCopy();
    return;
  }
  if (window.__lastStatus) render(window.__lastStatus);
  refreshTestResultCopy();
  refreshUpdateResultCopy();
  refreshDecisionNote();
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
  decisionNote: document.getElementById("decision-note"),
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
  preapproved: document.getElementById("preapproved"),
  preapproveForm: document.getElementById("preapprove-form"),
  preapproveOrigin: document.getElementById("preapprove-origin"),
  revokeAll: document.getElementById("revoke-all"),
  refresh: document.getElementById("refresh"),
  reconnect: document.getElementById("reconnect"),
  copyDiag: document.getElementById("copy-diag"),
  deskSnapshot: document.getElementById("desk-snapshot"),
  verbose: document.getElementById("verbose"),
  lastErrors: document.getElementById("last-errors"),
  trainingCard: document.getElementById("training-card"),
  copyAudit: document.getElementById("copy-audit"),
  exportAudit: document.getElementById("export-audit"),
  auditEmpty: document.getElementById("audit-empty"),
  auditEvents: document.getElementById("audit-events"),
  testConnection: document.getElementById("test-connection"),
  testResult: document.getElementById("test-result"),
  checkUpdate: document.getElementById("check-update"),
  updateResult: document.getElementById("update-result"),
  updateLinkWrap: document.getElementById("update-link-wrap"),
  updateLink: document.getElementById("update-link"),
  techCheckUpdate: document.getElementById("tech-check-update"),
  techUpdateResult: document.getElementById("tech-update-result"),
  liveNote: document.getElementById("live-note"),
  staffReadersCard: document.getElementById("staff-readers-card"),
  staffReaders: document.getElementById("staff-readers"),
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
  return (await apiRaw(path, opts)).body;
}

async function apiRaw(path, opts) {
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
    const err = new Error(msg);
    err.status = res.status;
    throw err;
  }
  return { body, text };
}

function parseCsrfFromHtml(html) {
  const m = String(html).match(/name="openthaiid-csrf"\s+content="([a-f0-9]+)"/i);
  return m ? m[1] : "";
}

async function refreshCsrfFromPage() {
  const res = await fetch("/", { credentials: "same-origin", cache: "no-store" });
  if (!res.ok) {
    throw new Error("helper down");
  }
  const token = parseCsrfFromHtml(await res.text());
  if (!token) {
    throw new Error("csrf missing");
  }
  setCsrf(token);
}

function setLiveNote(text) {
  if (!els.liveNote) return;
  if (!text) {
    els.liveNote.hidden = true;
    els.liveNote.textContent = "";
    return;
  }
  els.liveNote.hidden = false;
  els.liveNote.textContent = text;
}

function setTestResult(text) {
  if (!els.testResult) return;
  if (!text) {
    els.testResult.hidden = true;
    els.testResult.textContent = "";
    return;
  }
  els.testResult.hidden = false;
  els.testResult.textContent = text;
}

function refreshTestResultCopy() {
  if (!els.testResult || els.testResult.hidden) return;
  if (window.__helperOffline) {
    setTestResult(t("testFail"));
    return;
  }
  if (window.__lastTestProbe) {
    setTestResult(formatTestResult(window.__lastStatus || {}, window.__lastTestProbe));
  }
}

function setUpdateResult(text, url) {
  if (els.updateResult) {
    if (!text) {
      els.updateResult.hidden = true;
      els.updateResult.textContent = "";
    } else {
      els.updateResult.hidden = false;
      els.updateResult.textContent = text;
    }
  }
  if (els.techUpdateResult) {
    if (!text) {
      els.techUpdateResult.hidden = true;
      els.techUpdateResult.textContent = "";
    } else {
      els.techUpdateResult.hidden = false;
      els.techUpdateResult.textContent = text;
    }
  }
  if (els.updateLinkWrap && els.updateLink) {
    if (!url) {
      els.updateLinkWrap.hidden = true;
      els.updateLink.removeAttribute("href");
      els.updateLink.textContent = "";
    } else {
      els.updateLinkWrap.hidden = false;
      els.updateLink.href = url;
      els.updateLink.textContent = t("updateOpenRelease");
    }
  }
}

function refreshUpdateResultCopy() {
  const body = window.__lastUpdateBody;
  if (!body) return;
  setUpdateResult(formatUpdateMessage(body), body.htmlUrl || body.releasesPage || "");
  if (els.checkUpdate) els.checkUpdate.textContent = t("checkUpdate");
}

function formatUpdateMessage(body) {
  if (!body) return t("updateFail");
  const msg = lang === "en" ? body.message : body.messageTh || body.message;
  return msg || t("updateFail");
}

function renderOffline() {
  lastStatusText = null;
  const wasOnline = Boolean(window.__lastStatus) || window.__helperOffline;
  window.__helperOffline = true;
  setText(els.headline, t("helperLostHeadline"));
  els.headline.className = "headline wait";
  setText(els.next, t("helperWaiting"));
  setLight(els.lightHelper, "bad", t("helperOff"));
  setLight(els.lightReader, "wait", t("readerUnknown"));
  setLight(els.lightCard, "wait", t("cardUnknown"));
  if (els.userPendingCard) els.userPendingCard.hidden = true;
  if (els.staffReadersCard) els.staffReadersCard.hidden = true;
  setDecisionNote("");
  if (!wasOnline) {
    setLiveNote("");
  }
}

function noteHelperBack() {
  if (!window.__helperOffline) return;
  window.__helperOffline = false;
  setLiveNote(t("helperBack"));
  if (window.__liveNoteTimer) clearTimeout(window.__liveNoteTimer);
  window.__liveNoteTimer = setTimeout(() => {
    setLiveNote("");
  }, 4000);
}

function noteServiceBack(data) {
  const prev = window.__lastBackendStatus;
  const next = data && data.backend && data.backend.status;
  window.__lastBackendStatus = next;
  if (window.__helperOffline) return;
  if ((prev === "unavailable" || prev === "stalled") && next === "available") {
    const ready = summarize(data).ready;
    setLiveNote(ready ? t("serviceBackReady") : t("serviceBack"));
    if (window.__liveNoteTimer) clearTimeout(window.__liveNoteTimer);
    window.__liveNoteTimer = setTimeout(() => {
      setLiveNote("");
    }, 4000);
  }
}

function helperWsUrl() {
  const fromStatus = window.__lastStatus && window.__lastStatus.wsUrl;
  if (typeof fromStatus === "string" && fromStatus) return fromStatus;
  const loc = window.location;
  const ws = loc.protocol === "https:" ? "wss:" : "ws:";
  return `${ws}//${loc.host}/v1/ws`;
}

function probeHelper() {
  return new Promise((resolve, reject) => {
    if (typeof WebSocket === "undefined") {
      reject(new Error("websocket"));
      return;
    }
    let settled = false;
    let ws = null;
    const timer = setTimeout(() => finish(null, new Error("timeout")), 5000);
    const finish = (result, err) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      if (ws) {
        try {
          ws.onopen = null;
          ws.onerror = null;
          ws.onmessage = null;
          ws.close();
        } catch {
          // ignore
        }
      }
      if (err) reject(err);
      else resolve(result);
    };
    try {
      ws = new WebSocket(helperWsUrl(), ["openthaiid.v1"]);
    } catch (err) {
      finish(null, err);
      return;
    }
    ws.onerror = () => finish(null, new Error("ws"));
    ws.onopen = () => {
      try {
        ws.send(JSON.stringify({ v: 1, id: 1, type: "bridge.probe" }));
      } catch (err) {
        finish(null, err);
      }
    };
    ws.onmessage = (ev) => {
      let msg;
      try {
        msg = JSON.parse(typeof ev.data === "string" ? ev.data : String(ev.data));
      } catch {
        return;
      }
      if (msg.id !== 1) return;
      if (msg.error) {
        finish(null, new Error(msg.error.message || "probe"));
        return;
      }
      finish(msg.result || {}, null);
    };
  });
}

function formatTestResult(data, probe) {
  const backend = (probe && probe.backend && probe.backend.status) || (data.backend && data.backend.status) || "";
  const readers = (probe && probe.readers) || data.readers || [];
  const selected = activeReader(readers);
  const present = selected ? selected.presence === "present" : readers.some((r) => r.presence === "present");
  if (backend === "stalled") return t("testServiceStalled");
  if (backend === "unavailable") return t("testServiceOff");
  if (!readers.length) return `${t("testOk")} · ${t("plugReader")}`;
  if (needsReaderChoice(readers)) return `${t("testOk")} · ${t("multiReader")}`;
  if (present) return `${t("testOk")} · ${t("cardInserted")}`;
  return `${t("testOk")} · ${t("insertCard")}`;
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

function lightStateWord(kind) {
  if (kind === "ok") return t("lightOk");
  if (kind === "warn") return t("lightWarn");
  if (kind === "bad") return t("lightBad");
  return t("lightWait");
}

function setLight(el, kind, text) {
  const resolved = kind || "wait";
  el.className = `light-${resolved}`;
  const state = el.querySelector(".light-state");
  const label = el.querySelector(".light-text");
  const word = lightStateWord(resolved);
  if (state) state.textContent = word;
  if (label) label.textContent = text;
  el.setAttribute("aria-label", `${word}. ${text}`);
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

function activeReader(readers) {
  if (!Array.isArray(readers) || !readers.length) return null;
  const selected = readers.find((r) => r.selected);
  if (selected) return selected;
  if (readers.length === 1) return readers[0];
  return null;
}

function needsReaderChoice(readers) {
  return Array.isArray(readers) && readers.length > 1 && !readers.some((r) => r.selected);
}

function summarize(data) {
  const readers = data.readers || [];
  const backend = (data.backend && data.backend.status) || "unavailable";
  const helperOk = Boolean(data.running);
  const serviceOk = backend === "available";
  const readerOk = readers.length > 0;
  const selected = activeReader(readers);
  const needChoice = needsReaderChoice(readers);
  const present = selected ? selected.presence === "present" : false;
  const unsupported = selected ? selected.presence === "unsupported" : readers.some((r) => r.presence === "unsupported");
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
  let cardKind = "wait";
  if (!readerOk || !serviceOk) {
    cardText = t("insertAfterReader");
  } else if (needChoice) {
    cardText = t("insertAfterReader");
  } else if (unsupported && !present) {
    cardText = t("unsupportedCard");
    cardKind = "warn";
  } else if (present) {
    cardText = t("cardInserted");
    cardKind = "ok";
  }

  const ready = helperOk && serviceOk && readerOk && present && !needChoice;
  let headline = ready ? t("ready") : t("notReady");
  if (ready && data.synthetic) headline = t("readyPractice");

  let next = t("nextFillClinic");
  if (!helperOk) next = t("startHelper");
  else if (!serviceOk) next = t("fixService");
  else if (!readerOk) next = t("plugReader");
  else if (needChoice) next = t("multiReader");
  else if (!present) next = t("insertThenFill");
  else if (pending) next = t("pendingAllow");
  else if (ready && data.synthetic) {
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

function isLoopbackHost(host) {
  const h = String(host || "")
    .toLowerCase()
    .replace(/^\[|\]$/g, "");
  return h === "localhost" || h === "127.0.0.1" || h === "::1" || h === "0.0.0.0";
}

function staffSiteLabel(origin) {
  const raw = String(origin || "").trim();
  if (!raw) return t("unknownSite");
  try {
    const u = new URL(raw);
    const host = (u.hostname || "").toLowerCase();
    if (isLoopbackHost(host)) return t("siteOnThisComputer");
    let label = host;
    if (u.port) label += ":" + u.port;
    return label || t("unknownSite");
  } catch {
    const stripped = raw.replace(/^[a-z][a-z0-9+.-]*:\/\//i, "");
    const hostpart = stripped.split("/")[0] || "";
    const hostOnly = hostpart.split(":")[0];
    if (isLoopbackHost(hostOnly)) return t("siteOnThisComputer");
    return hostpart || t("unknownSite");
  }
}

function setDecisionNote(kind) {
  const el = els.decisionNote;
  if (!el) return;
  if (!kind) {
    window.__lastStaffDecision = "";
    el.hidden = true;
    el.textContent = "";
    el.classList.remove("warn");
    return;
  }
  window.__lastStaffDecision = kind;
  el.hidden = false;
  el.textContent = kind === "allow" ? t("allowConfirmed") : t("denyNext");
  el.classList.toggle("warn", kind !== "allow");
}

function refreshDecisionNote() {
  if (!window.__lastStaffDecision) return;
  setDecisionNote(window.__lastStaffDecision);
}

function allowRow(p, { detailed }) {
  const li = document.createElement("li");
  if (!detailed) li.className = "staff-pending";
  const site = document.createElement("div");
  if (detailed) {
    site.className = "origin";
    site.textContent = p.origin;
  } else {
    site.className = "site-name";
    site.textContent = staffSiteLabel(p.origin);
  }
  const codeWrap = document.createElement("div");
  const codeValue = p.comparisonCode || "";
  if (codeValue) {
    if (detailed) {
      codeWrap.className = "code";
      codeWrap.textContent = codeValue;
    } else {
      codeWrap.className = "code-wrap";
      const label = document.createElement("span");
      label.className = "code-label";
      label.textContent = t("matchCode");
      const digits = document.createElement("span");
      digits.className = "code";
      digits.textContent = codeValue;
      codeWrap.append(label, digits);
    }
  }
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
  const allow = document.createElement("button");
  allow.type = "button";
  allow.className = detailed ? "btn btn-ok btn-staff" : "btn btn-ok btn-staff btn-allow-primary";
  allow.textContent = t("allow");
  allow.addEventListener("click", () => decide(p.id, "allow"));
  const deny = document.createElement("button");
  deny.type = "button";
  deny.className = detailed ? "btn btn-danger btn-staff" : "btn btn-staff btn-deny-quiet";
  deny.textContent = t("deny");
  deny.addEventListener("click", () => decide(p.id, "deny"));
  actions.append(allow, deny);
  li.append(site);
  if (codeValue) li.append(codeWrap);
  li.append(fields, actions);
  return li;
}

function renderUserPending(data) {
  const pending = data.pending || [];
  clearChildren(els.userPending);
  if (!pending.length) {
    els.userPendingCard.hidden = true;
    return;
  }
  setDecisionNote("");
  els.userPendingCard.hidden = false;
  for (const p of pending) {
    els.userPending.append(allowRow(p, { detailed: false }));
  }
}

function renderStaffReaders(data) {
  const readers = data.readers || [];
  if (!els.staffReadersCard || !els.staffReaders) return;
  if (readers.length <= 1) {
    els.staffReadersCard.hidden = true;
    clearChildren(els.staffReaders);
    return;
  }
  els.staffReadersCard.hidden = false;
  clearChildren(els.staffReaders);
  for (const r of readers) {
    const li = document.createElement("li");
    if (r.selected) li.className = "selected";
    const row = document.createElement("div");
    row.className = "row";
    const left = document.createElement("div");
    const name = document.createElement("div");
    name.className = "name";
    name.textContent = r.name || r.readerId;
    left.append(name);
    const bits = document.createElement("div");
    bits.className = "pills";
    bits.append(pill(r.presence === "present" ? t("staffCardPresent") : t("staffNoCard"), presenceKind(r.presence)));
    if (r.selected) bits.append(pill(t("usingThisReader"), "ok"));
    left.append(bits);
    const actions = document.createElement("div");
    actions.className = "actions";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = r.selected ? "btn btn-staff" : "btn btn-ok btn-staff";
    btn.textContent = r.selected ? t("usingThisReader") : t("useThisReader");
    btn.disabled = Boolean(r.selected);
    btn.addEventListener("click", () => selectReader(r.readerId));
    actions.append(btn);
    row.append(left, actions);
    li.append(row);
    els.staffReaders.append(li);
  }
}

function renderReaders(data) {
  const readers = data.readers || [];
  clearChildren(els.readers);
  if (!readers.length) {
    els.readersEmpty.hidden = false;
    if (data.backend && data.backend.status === "unavailable") {
      els.readersEmpty.textContent =
        "Card service is down (sleep/wake or stopped PC/SC). Start pcscd (Linux) or the Smart Card service (Windows), then Reconnect card service. Ready should return to พร้อม / Ready — do not reboot.";
    } else {
      els.readersEmpty.textContent = "No card readers detected. Plug in a reader and click Refresh readers.";
    }
    return;
  }
  els.readersEmpty.hidden = true;
  const showSelect = readers.length > 1;
  for (const r of readers) {
    const li = document.createElement("li");
    if (r.selected) li.className = "selected";
    const row = document.createElement("div");
    row.className = "row";
    const left = document.createElement("div");
    const name = document.createElement("div");
    name.className = "name";
    name.textContent = r.name || r.readerId;
    left.append(name);
    const idLine = document.createElement("p");
    idLine.className = "fields";
    idLine.textContent = `id: ${r.readerId}`;
    left.append(idLine);
    const bits = document.createElement("div");
    bits.className = "pills";
    bits.append(pill(presenceLabel(r.presence), presenceKind(r.presence)));
    if (r.selected) bits.append(pill("Selected", "ok"));
    left.append(bits);
    const actions = document.createElement("div");
    actions.className = "actions";
    if (showSelect) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = r.selected ? "btn btn-staff" : "btn btn-ok btn-staff";
      btn.textContent = r.selected ? "Selected" : "Use this reader";
      btn.disabled = Boolean(r.selected);
      btn.addEventListener("click", () => selectReader(r.readerId));
      actions.append(btn);
    }
    row.append(left, actions);
    li.append(row);
    els.readers.append(li);
  }
}

function renderPending(data) {
  const pending = data.pending || [];
  els.pendingCount.textContent = String(pending.length);
  clearChildren(els.pending);
  if (!pending.length) {
    const li = document.createElement("li");
    li.className = "empty";
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
  clearChildren(els.trusted);
  els.revokeAll.hidden = trusted.length === 0;
  if (!trusted.length) {
    const li = document.createElement("li");
    li.className = "empty";
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
    if (g.persistent) {
      const badge = document.createElement("span");
      badge.className = "pill-it";
      badge.textContent = "IT pre-approve";
      origin.append(" ", badge);
    }
    const fields = document.createElement("p");
    fields.className = "fields";
    const fieldList = Array.isArray(g.fields) ? g.fields.join(", ") : "";
    const until = g.persistent ? "does not expire" : `until ${g.expiresAt || ""}`;
    fields.textContent = `fields: ${fieldList} · ${until}`;
    left.append(origin, fields);
    const actions = document.createElement("div");
    actions.className = "actions";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-danger btn-staff";
    btn.textContent = g.persistent ? "Remove from IT list" : "Revoke";
    btn.addEventListener("click", () => {
      if (g.persistent) unpreapprove(g.origin);
      else revoke(g.origin);
    });
    actions.append(btn);
    row.append(left, actions);
    li.append(row);
    els.trusted.append(li);
  }
}

function renderPreapproved(data) {
  if (!els.preapproved) return;
  const list = Array.isArray(data.preapprovedOrigins) ? data.preapprovedOrigins : [];
  clearChildren(els.preapproved);
  if (!list.length) {
    const li = document.createElement("li");
    li.className = "empty";
    li.textContent = "No IT pre-approved clinic websites yet.";
    els.preapproved.append(li);
    return;
  }
  for (const origin of list) {
    const li = document.createElement("li");
    const row = document.createElement("div");
    row.className = "row";
    const left = document.createElement("div");
    const site = document.createElement("div");
    site.className = "origin";
    site.textContent = origin;
    left.append(site);
    const actions = document.createElement("div");
    actions.className = "actions";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn btn-danger btn-staff";
    btn.textContent = "Remove";
    btn.addEventListener("click", () => unpreapprove(origin));
    actions.append(btn);
    row.append(left, actions);
    li.append(row);
    els.preapproved.append(li);
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

function auditExportPayload(data) {
  const events = Array.isArray(data && data.audit) ? data.audit : [];
  return {
    schema: "openthaiid.audit.v1",
    pii: "omitted — citizen ID, names, address, portrait, raw APDUs are never stored",
    count: events.length,
    events,
  };
}

function renderAudit(data) {
  const events = Array.isArray(data.audit) ? data.audit.slice() : [];
  events.reverse();
  if (els.auditEmpty) els.auditEmpty.hidden = events.length > 0;
  if (!els.auditEvents) return;
  clearChildren(els.auditEvents);
  for (const ev of events) {
    const li = document.createElement("li");
    const bits = [ev.at || "", ev.kind || "", ev.code || "", ev.origin || "", ev.readerId || "", ev.reader || ""]
      .map((s) => String(s).trim())
      .filter(Boolean);
    if (Array.isArray(ev.fields) && ev.fields.length) {
      bits.push("fields=" + ev.fields.join(","));
    }
    li.textContent = bits.join("  ");
    els.auditEvents.append(li);
  }
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
    `tls: ${data.tls ? "on" : "off"}`,
    `ws: ${data.wsUrl || ""}`,
    `wss: ${data.wssUrl || ""}`,
    `instance: ${data.instanceId || ""}`,
    `logLevel: ${data.logLevel || ""}`,
    `erpConnections: ${data.connections ?? 0}`,
    `readers: ${readers.length}`,
  ];
  for (const r of readers) {
    lines.push(`  - ${r.name || r.readerId} id=${r.readerId} presence=${r.presence}${r.selected ? " selected" : ""}`);
  }
  lines.push(`pending: ${pending.length}`);
  for (const p of pending) {
    lines.push(`  - id=${p.id} origin=${p.origin} fields=${(p.fields || []).join(",")}`);
  }
  const preapproved = data.preapprovedOrigins || [];
  lines.push(`approved: ${trusted.length}`);
  for (const g of trusted) {
    const tag = g.persistent ? " persistent" : "";
    lines.push(`  - origin=${g.origin} fields=${(g.fields || []).join(",")}${tag}`);
  }
  lines.push(`preapproved: ${preapproved.length}`);
  for (const o of preapproved) {
    lines.push(`  - ${o}`);
  }
  lines.push("lastErrors:");
  if (!errs.length) lines.push("  (none)");
  for (const e of errs) {
    lines.push(`  - ${e.at} ${e.source} ${e.code} ${e.message}`);
  }
  lines.push("audit (redacted; codes and origins only):");
  const audit = data.audit || [];
  if (!audit.length) lines.push("  (none)");
  for (const ev of audit) {
    const extra = [];
    if (ev.code) extra.push(ev.code);
    if (ev.origin) extra.push(ev.origin);
    if (ev.readerId) extra.push(`id=${ev.readerId}`);
    if (ev.reader) extra.push(ev.reader);
    if (Array.isArray(ev.fields) && ev.fields.length) extra.push("fields=" + ev.fields.join(","));
    lines.push(`  - ${ev.at || ""} ${ev.kind || ""} ${extra.join(" ")}`.trim());
  }
  return lines.join("\n");
}

function render(data) {
  window.__lastStatus = data;
  renderHero(data);
  renderStaffReaders(data);
  renderUserPending(data);

  setText(els.version, data.version ? `v${data.version}` : "");
  setText(els.instance, data.instanceId || "");
  setText(els.os, data.os || "");
  setText(els.connections, String(data.connections ?? 0));
  clearChildren(els.bridge);
  els.bridge.append(pill(data.running ? "running" : "stopped", data.running ? "ok" : "bad"));
  if (data.synthetic) {
    els.bridge.append(document.createTextNode(" "));
    els.bridge.append(pill("practice reader", "demo"));
  }
  clearChildren(els.pcsc);
  const st = (data.backend && data.backend.status) || "unavailable";
  els.pcsc.append(pill(pcscLabel(st, data.synthetic), pcscKind(st)));
  setText(els.listen, data.listenAddr || "");
  const wsLine = data.wssUrl ? `${data.wsUrl || ""} · ${data.wssUrl}` : data.wsUrl || "";
  setText(els.ws, wsLine);
  els.verbose.checked = data.logLevel === "debug";

  if (data.synthetic) {
    els.banner.hidden = false;
    els.banner.textContent = t("bannerPractice");
    if (els.trainingCard) els.trainingCard.hidden = false;
  } else {
    els.banner.hidden = true;
    if (els.trainingCard) els.trainingCard.hidden = true;
  }

  renderReaders(data);
  renderPending(data);
  renderTrusted(data);
  renderPreapproved(data);
  renderLastErrors(data);
  renderAudit(data);
}

let statusEpoch = 0;
// Raw text of the last rendered status. The status JSON carries no clock
// fields, so identical text means nothing changed: skip the DOM rebuild on
// the 400 ms Ready poll (keeps CPU low and keeps technician text selections).
let lastStatusText = null;

async function loadOnce() {
  const epoch = ++statusEpoch;
  const { body: data, text } = await apiRaw("/v1/manager/status");
  if (epoch !== statusEpoch) return;
  noteHelperBack();
  noteServiceBack(data);
  if (text === lastStatusText) return;
  lastStatusText = text;
  render(data);
}

async function load() {
  try {
    await loadOnce();
    return;
  } catch {
    // Helper restart invalidates the CSRF token baked into this page.
  }
  try {
    await refreshCsrfFromPage();
    await loadOnce();
  } catch {
    renderOffline();
  }
}

async function checkGitHubReleases() {
  const buttons = [els.checkUpdate, els.techCheckUpdate].filter(Boolean);
  buttons.forEach((b) => {
    b.disabled = true;
  });
  if (els.checkUpdate) els.checkUpdate.textContent = t("checkingUpdate");
  if (els.techCheckUpdate) els.techCheckUpdate.textContent = t("checkingUpdate");
  try {
    try {
      await refreshCsrfFromPage();
    } catch {
      window.__lastUpdateBody = null;
      setUpdateResult(t("updateFail"), "");
      return;
    }
    const body = await api("/v1/manager/update/check", { method: "POST" });
    window.__lastUpdateBody = body;
    setUpdateResult(formatUpdateMessage(body), (body && (body.htmlUrl || body.releasesPage)) || "");
  } catch {
    window.__lastUpdateBody = null;
    setUpdateResult(t("updateFail"), "");
  } finally {
    buttons.forEach((b) => {
      b.disabled = false;
    });
    if (els.checkUpdate) els.checkUpdate.textContent = t("checkUpdate");
    if (els.techCheckUpdate) els.techCheckUpdate.textContent = "Check GitHub Releases";
  }
}

async function testConnection() {
  if (!els.testConnection) return;
  els.testConnection.disabled = true;
  els.testConnection.textContent = t("testingConnection");
  setTestResult("");
  try {
    try {
      await refreshCsrfFromPage();
    } catch {
      // Page fetch failing is the same as helper down; load() reports it.
    }
    await load();
    if (window.__helperOffline) {
      setTestResult(t("testFail"));
      return;
    }
    let probe = null;
    try {
      probe = await probeHelper();
    } catch {
      setTestResult(`${t("testOk")} · ${t("testWsFail")}`);
      return;
    }
    const data = window.__lastStatus || {};
    window.__lastTestProbe = probe;
    setTestResult(formatTestResult(data, probe));
  } finally {
    els.testConnection.disabled = false;
    els.testConnection.textContent = t("testConnection");
  }
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
    setDecisionNote(action === "allow" ? "allow" : "deny");
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

async function unpreapprove(origin) {
  try {
    await api("/v1/manager/trust/unpreapprove", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ origin }),
    });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

async function addPreapproved(origin) {
  await api("/v1/manager/trust/preapprove", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ origin }),
  });
  await load();
}

async function revokeAll() {
  if (!window.confirm("Revoke staff Allow grants on this PC? IT pre-approved clinic sites stay.")) return;
  try {
    await api("/v1/manager/trust/revoke-all", { method: "POST" });
    await load();
  } catch (err) {
    window.alert(err.message || String(err));
  }
}

async function copyDeskSnapshot() {
  const btn = els.deskSnapshot;
  if (!btn) return;
  let text = "";
  try {
    const resp = await fetch("/v1/helper/doctor", { headers, cache: "no-store" });
    if (!resp.ok) throw new Error("doctor");
    const body = await resp.json();
    text = (body && body.text) || "";
  } catch {
    text = "";
  }
  if (!text) {
    window.alert("Could not load desk snapshot. Is OpenThaiID still running?");
    return;
  }
  const copied = await copyText(text, "Desk snapshot");
  if (!copied) {
    btn.textContent = "Desk snapshot";
    return;
  }
  btn.textContent = "Copied";
  setTimeout(() => {
    btn.textContent = "Desk snapshot";
  }, 1500);
}

async function copyDiagnostics() {
  const data = window.__lastStatus;
  if (!data) return;
  const text = diagnosticsText(data);
  const copied = await copyText(text, "Copy diagnostics");
  if (!copied) {
    els.copyDiag.textContent = "Copy diagnostics";
    return;
  }
  els.copyDiag.textContent = "Copied";
  setTimeout(() => {
    els.copyDiag.textContent = "Copy diagnostics";
  }, 1500);
}

async function copyAuditJSON() {
  const data = window.__lastStatus;
  if (!data) return;
  const text = JSON.stringify(auditExportPayload(data), null, 2);
  const copied = await copyText(text, "Copy audit JSON");
  if (!copied) {
    if (els.copyAudit) els.copyAudit.textContent = "Copy JSON";
    return;
  }
  if (els.copyAudit) els.copyAudit.textContent = "Copied";
  setTimeout(() => {
    if (els.copyAudit) els.copyAudit.textContent = "Copy JSON";
  }, 1500);
}

function downloadAuditJSON() {
  const data = window.__lastStatus;
  if (!data) return;
  const text = JSON.stringify(auditExportPayload(data), null, 2);
  const blob = new Blob([text], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "openthaiid-audit.json";
  a.rel = "noopener";
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
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
els.testConnection?.addEventListener("click", () => void testConnection());
els.checkUpdate?.addEventListener("click", () => void checkGitHubReleases());
els.techCheckUpdate?.addEventListener("click", () => void checkGitHubReleases());
els.revokeAll.addEventListener("click", () => void revokeAll());
els.preapproveForm?.addEventListener("submit", (ev) => {
  ev.preventDefault();
  const origin = String(els.preapproveOrigin?.value || "").trim();
  if (!origin) return;
  void addPreapproved(origin)
    .then(() => {
      if (els.preapproveOrigin) els.preapproveOrigin.value = "";
    })
    .catch((err) => window.alert(err.message || String(err)));
});
els.copyDiag.addEventListener("click", () => void copyDiagnostics());
els.deskSnapshot?.addEventListener("click", () => void copyDeskSnapshot());
els.copyAudit?.addEventListener("click", () => void copyAuditJSON());
els.exportAudit?.addEventListener("click", () => downloadAuditJSON());
els.verbose.addEventListener("change", () => void setVerbose(els.verbose.checked));

let wakeRefreshTimer = null;
function gentleWakeRefresh() {
  if (typeof document !== "undefined" && document.visibilityState && document.visibilityState !== "visible") {
    return;
  }
  if (wakeRefreshTimer) clearTimeout(wakeRefreshTimer);
  wakeRefreshTimer = setTimeout(() => {
    wakeRefreshTimer = null;
    void (async () => {
      await load();
      if (window.__helperOffline) return;
      try {
        await probeHelper();
      } catch {
        // Status lights already updated; never reads personal card fields.
      }
    })();
  }, 80);
}

document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") gentleWakeRefresh();
});
window.addEventListener("pageshow", () => gentleWakeRefresh());
window.addEventListener("online", () => gentleWakeRefresh());

void load();
let statusPollTimer = null;
function scheduleStatusPoll() {
  if (statusPollTimer) clearTimeout(statusPollTimer);
  const ready = Boolean(window.__lastStatus && summarize(window.__lastStatus).ready);
  const hidden = document.visibilityState !== "visible";
  const backend = window.__lastStatus && window.__lastStatus.backend && window.__lastStatus.backend.status;
  const serviceDown = backend === "unavailable" || backend === "stalled";
  const ms = hidden ? 4000 : window.__helperOffline || serviceDown ? 500 : ready ? 400 : 1500;
  statusPollTimer = setTimeout(async () => {
    if (document.visibilityState === "visible") await load();
    scheduleStatusPoll();
  }, ms);
}
scheduleStatusPoll();
