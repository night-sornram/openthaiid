import { autofillThaiID, formatStaffError, mountThaiIDStatus, staffApprovalGuidance, staffLang } from "../../sdk/web/dist/index.js";
import {
  CLINIC_GENDER_LABELS,
  CLINIC_PATIENT_MAP,
  CLINIC_PATIENT_REQUEST,
  OPTIONAL_PATIENT_REQUEST,
  OPTIONAL_PORTRAIT_REQUEST,
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

const qs = new URLSearchParams(location.search);
const endpoint = qs.get("ws") || undefined;
const lang = staffLang(qs.get("lang"));

const requestable = [...CLINIC_PATIENT_REQUEST, ...OPTIONAL_PATIENT_REQUEST, ...OPTIONAL_PORTRAIT_REQUEST];
const selected = new Set(CLINIC_PATIENT_REQUEST);

function showBanner(kind, text) {
  banner.className = `banner show ${kind}`;
  banner.textContent = text;
}

function hideBanner() {
  banner.className = "banner";
  banner.textContent = "";
}

function genderLabels() {
  if (genderLabelsSelect.value === "th") {
    return { male: "ชาย", female: "หญิง" };
  }
  return { ...CLINIC_GENDER_LABELS };
}

function currentConfig() {
  return clinicPatientAutofill({
    root: form,
    request: requestable.filter((f) => selected.has(f)),
    genderLabels: genderLabels(),
    ...(endpoint ? { endpoint } : {}),
  });
}

function previewConfig() {
  const cfg = currentConfig();
  return {
    request: cfg.request,
    map: cfg.map,
    dateFormat: cfg.dateFormat,
    genderLabels: cfg.genderLabels,
    discover: cfg.discover,
  };
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
  for (const [slot, selector] of Object.entries(CLINIC_PATIENT_MAP)) {
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
  preview.textContent = JSON.stringify(previewConfig(), null, 2);
}

fillBtn.addEventListener("click", async () => {
  fillBtn.disabled = true;
  hideBanner();
  const config = currentConfig();
  if (!config.request.length) {
    showBanner("error", lang === "en" ? "Enable at least one field in request." : "เลือกอย่างน้อยหนึ่งช่องใน request");
    fillBtn.disabled = false;
    return;
  }
  try {
    const { filled } = await autofillThaiID({
      ...config,
      lang,
      onApprovalRequired: (info) => {
        const g = staffApprovalGuidance(info, lang);
        showBanner("approval", `${g.message} ${g.next}`);
      },
      onError: (err) => {
        const code = err && err.code ? err.code : "ERROR";
        showBanner("error", `${formatStaffError(err, lang)} (${code})`);
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
    showBanner("ok", lang === "en" ? okEn : okTh);
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
  syncGenderOptions();
  renderPreview();
});

renderSettings();
syncGenderOptions();
const statusHost = document.getElementById("thaiid-status");
if (statusHost) mountThaiIDStatus(statusHost, { lang, ...(endpoint ? { endpoint } : {}) });
