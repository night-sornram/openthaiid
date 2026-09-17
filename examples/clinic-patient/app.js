import {
  SUPPORTED_FIELDS,
  autofillClinicConfigDocument,
  autofillThaiID,
  clinicConfigToAutofillOptions,
  formatStaffError,
  mountThaiIDStatus,
  parseAutofillClinicConfig,
  serializeAutofillClinicConfig,
  staffApprovalGuidance,
  staffLang,
} from "../../sdk/web/dist/index.js";
import {
  CLINIC_GENDER_LABELS,
  CLINIC_PATIENT_MAP,
  CLINIC_PATIENT_REQUEST,
  clinicPatientAutofill,
} from "./settings.js";

const form = document.getElementById("patient-form");
const fillBtn = document.getElementById("fill");
const resetBtn = document.getElementById("reset");
const banner = document.getElementById("banner");
const requestHost = document.getElementById("request-fields");
const mapRows = document.getElementById("map-rows");
const preview = document.getElementById("config-preview");
const genderLabelsSelect = document.getElementById("gender-labels");
const downloadBtn = document.getElementById("download-config");
const uploadBtn = document.getElementById("upload-config");
const uploadFile = document.getElementById("upload-config-file");
const loadKycBtn = document.getElementById("load-kyc-minimal");
const loadClinicBtn = document.getElementById("load-clinic-full");
const loadHttpsBtn = document.getElementById("load-clinic-https");

const qs = new URLSearchParams(location.search);
const endpoint = qs.get("ws") || undefined;
const pageLang = staffLang(qs.get("lang"));

const TH_GENDER_LABELS = { male: "ชาย", female: "หญิง" };
const requestable = [...SUPPORTED_FIELDS];
const selected = new Set(CLINIC_PATIENT_REQUEST);

let currentMap = { ...CLINIC_PATIENT_MAP };
let currentDateFormat = "iso";
let currentDateLocale = undefined;
let currentDiscover = false;
let currentConfigLang = pageLang;
let currentPreferLoopbackTls = undefined;
let currentJsonEndpoint = undefined;
let customGenderLabels = null;

function showBanner(kind, text) {
  banner.className = `banner show ${kind}`;
  banner.textContent = text;
}

function hideBanner() {
  banner.className = "banner";
  banner.textContent = "";
}

function labelsMatch(a, b) {
  return a.male === b.male && a.female === b.female && (a.unknown ?? "") === (b.unknown ?? "");
}

function genderLabels() {
  if (genderLabelsSelect.value === "custom" && customGenderLabels) {
    return { ...customGenderLabels };
  }
  if (genderLabelsSelect.value === "th") {
    return { ...TH_GENDER_LABELS };
  }
  return { ...CLINIC_GENDER_LABELS };
}

function currentConfig() {
  return clinicPatientAutofill({
    root: form,
    request: requestable.filter((f) => selected.has(f)),
    map: currentMap,
    dateFormat: currentDateFormat,
    ...(currentDateLocale ? { dateLocale: currentDateLocale } : {}),
    genderLabels: genderLabels(),
    discover: currentDiscover,
    lang: currentConfigLang,
    ...(typeof currentPreferLoopbackTls === "boolean" ? { preferLoopbackTls: currentPreferLoopbackTls } : {}),
    ...(endpoint || currentJsonEndpoint ? { endpoint: endpoint || currentJsonEndpoint } : {}),
  });
}

function previewConfig() {
  return serializeAutofillClinicConfig(currentConfig());
}

function renderSettings() {
  requestHost.innerHTML = "";
  for (const field of requestable) {
    const label = document.createElement("label");
    const input = document.createElement("input");
    input.type = "checkbox";
    input.checked = selected.has(field);
    input.addEventListener("change", () => {
      if (input.checked) selected.add(field);
      else selected.delete(field);
      renderPreview();
    });
    label.append(input, document.createTextNode(field));
    if (field === "portrait") {
      const hint = document.createElement("span");
      hint.className = "hint";
      hint.textContent = " (opt-in; example <img> preview only)";
      label.append(hint);
    }
    requestHost.append(label);
  }

  mapRows.innerHTML = "";
  for (const [slot, selector] of Object.entries(currentMap)) {
    const tr = document.createElement("tr");
    const tdSlot = document.createElement("td");
    const tdSel = document.createElement("td");
    tdSlot.innerHTML = `<code>${slot}</code>`;
    tdSel.innerHTML = `<code>${selector}</code>`;
    tr.append(tdSlot, tdSel);
    mapRows.append(tr);
  }
  renderPreview();
}

function renderPreview() {
  preview.textContent = JSON.stringify(autofillClinicConfigDocument(previewConfig()), null, 2);
}

function applyClinicConfig(cfg, sourceLabel) {
  selected.clear();
  for (const field of cfg.request ?? []) selected.add(field);
  if (cfg.map) currentMap = { ...cfg.map };
  if (cfg.dateFormat) currentDateFormat = cfg.dateFormat;
  currentDateLocale = cfg.dateLocale;
  if (typeof cfg.discover === "boolean") currentDiscover = cfg.discover;
  if (cfg.lang) currentConfigLang = cfg.lang;
  currentPreferLoopbackTls = typeof cfg.preferLoopbackTls === "boolean" ? cfg.preferLoopbackTls : undefined;
  currentJsonEndpoint = cfg.endpoint;
  const customOpt = genderLabelsSelect.querySelector('option[value="custom"]');
  if (cfg.genderLabels) {
    customGenderLabels = cfg.genderLabels;
    if (labelsMatch(cfg.genderLabels, TH_GENDER_LABELS)) {
      genderLabelsSelect.value = "th";
      customOpt.hidden = true;
    } else if (labelsMatch(cfg.genderLabels, CLINIC_GENDER_LABELS)) {
      genderLabelsSelect.value = "en";
      customOpt.hidden = true;
    } else {
      customOpt.hidden = false;
      genderLabelsSelect.value = "custom";
    }
  }
  renderSettings();
  syncGenderOptions();
  showBanner("ok", sourceLabel);
}

async function loadClinicJson(url, label) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Could not fetch ${url} (${res.status})`);
  const parsed = parseAutofillClinicConfig(await res.json());
  applyClinicConfig(parsed, label);
}

fillBtn.addEventListener("click", async () => {
  fillBtn.disabled = true;
  hideBanner();
  const config = currentConfig();
  if (!config.request.length) {
    showBanner("error", pageLang === "en" ? "Enable at least one field in request." : "เลือกอย่างน้อยหนึ่งช่องใน request");
    fillBtn.disabled = false;
    return;
  }
  try {
    const { filled } = await autofillThaiID({
      ...clinicConfigToAutofillOptions(serializeAutofillClinicConfig(config)),
      root: form,
      lang: config.lang ?? pageLang,
      ...(endpoint ? { endpoint } : {}),
      onApprovalRequired: (info) => {
        const g = staffApprovalGuidance(info, pageLang);
        showBanner("approval", `${g.message} ${g.next}`);
      },
      onError: (err) => {
        const code = err && err.code ? err.code : "ERROR";
        showBanner("error", `${formatStaffError(err, pageLang)} (${code})`);
      },
    });
    const photo = document.getElementById("patient-photo");
    const filledPortrait = filled.some((f) => f.slot === "portrait" || f.slot.startsWith("portrait."));
    if (photo) {
      photo.hidden = !filledPortrait || !photo.getAttribute("src");
    }
    const okTh = filled.length
      ? `กรอกแล้ว ${filled.length} ช่องจากบัตรประชาชน เบอร์โทรและ HN ไม่ได้อยู่บนบัตรจึงว่างไว้${
          filledPortrait ? " รูปถ่ายเป็นตัวอย่างในหน้านี้เท่านั้น — ห้ามถ่ายภาพหน้าจอ" : ""
        }`
      : "เชื่อมต่อแล้ว แต่ไม่มีค่าให้กรอก";
    const okEn = filled.length
      ? `Filled ${filled.length} field(s) from Thai ID. Phone and HN were left blank.${
          filledPortrait ? " Portrait preview is example-only — do not screenshot it." : ""
        }`
      : "Connected, but no values to fill.";
    showBanner("ok", pageLang === "en" ? okEn : okTh);
  } catch {
    // onError already updated ERP chrome; keep the rejection from bubbling as an unhandled promise.
  } finally {
    fillBtn.disabled = false;
  }
});

resetBtn.addEventListener("click", () => {
  form.reset();
  hideBanner();
  const photo = document.getElementById("patient-photo");
  if (photo) {
    photo.removeAttribute("src");
    photo.hidden = true;
  }
});

function syncGenderOptions() {
  const labels = genderLabels();
  const select = document.getElementById("patient-gender");
  const previous = select.value;
  select.innerHTML = "";
  const blank = document.createElement("option");
  blank.value = "";
  blank.textContent = "—";
  select.append(blank);
  for (const value of [labels.male, labels.female]) {
    const opt = document.createElement("option");
    opt.value = value;
    opt.textContent = value;
    select.append(opt);
  }
  select.value = [...select.options].some((o) => o.value === previous) ? previous : "";
}

genderLabelsSelect.addEventListener("change", () => {
  if (genderLabelsSelect.value !== "custom") {
    customGenderLabels = null;
    const customOpt = genderLabelsSelect.querySelector('option[value="custom"]');
    if (customOpt) customOpt.hidden = true;
  }
  syncGenderOptions();
  renderPreview();
});

downloadBtn.addEventListener("click", () => {
  const doc = autofillClinicConfigDocument(previewConfig());
  const blob = new Blob([`${JSON.stringify(doc, null, 2)}\n`], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "openthaiid-clinic-config.json";
  a.click();
  URL.revokeObjectURL(url);
});

uploadBtn.addEventListener("click", () => {
  uploadFile.click();
});

uploadFile.addEventListener("change", async () => {
  const file = uploadFile.files && uploadFile.files[0];
  uploadFile.value = "";
  if (!file) return;
  try {
    const parsed = parseAutofillClinicConfig(await file.text());
    applyClinicConfig(parsed, `Loaded ${file.name}`);
  } catch (err) {
    const message = err && err.message ? err.message : String(err);
    showBanner("error", `Invalid clinic config JSON: ${message}`);
  }
});

loadKycBtn.addEventListener("click", async () => {
  try {
    await loadClinicJson("./kyc-minimal.json", "Loaded kyc-minimal.json");
  } catch (err) {
    const message = err && err.message ? err.message : String(err);
    showBanner("error", `Could not load KYC sample: ${message}`);
  }
});

loadClinicBtn.addEventListener("click", async () => {
  try {
    await loadClinicJson("./clinic-full.json", "Loaded clinic-full.json");
  } catch (err) {
    const message = err && err.message ? err.message : String(err);
    showBanner("error", `Could not load clinic sample: ${message}`);
  }
});

loadHttpsBtn.addEventListener("click", async () => {
  try {
    await loadClinicJson("./clinic-https.json", "Loaded clinic-https.json (preferLoopbackTls)");
  } catch (err) {
    const message = err && err.message ? err.message : String(err);
    showBanner("error", `Could not load HTTPS sample: ${message}`);
  }
});

renderSettings();
syncGenderOptions();
const statusHost = document.getElementById("thaiid-status");
if (statusHost) mountThaiIDStatus(statusHost, { lang: pageLang, ...(endpoint ? { endpoint } : {}) });
