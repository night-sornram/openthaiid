import { DEFAULT_FIELDS, ThaiID } from "/sdk/web/dist/index.js";

const statusEl = document.getElementById("status");
const approvalEl = document.getElementById("approval");
const outEl = document.getElementById("out");
const photoEl = document.getElementById("photo");
const connectBtn = document.getElementById("connect");
const readBtn = document.getElementById("read");
const disconnectBtn = document.getElementById("disconnect");

function clearPhoto() {
  if (!photoEl) return;
  photoEl.removeAttribute("src");
  photoEl.hidden = true;
}

function showPortrait(fields) {
  const p = fields && fields.portrait;
  if (!photoEl || !p || !p.data) {
    clearPhoto();
    return;
  }
  const mime = p.mimeType || "image/jpeg";
  photoEl.src = `data:${mime};base64,${p.data}`;
  photoEl.hidden = false;
}

const qs = new URLSearchParams(location.search);
const endpoint = qs.get("ws") || undefined;

function requestedFields() {
  const raw = qs.get("fields");
  if (!raw) return [...DEFAULT_FIELDS];
  const list = raw.split(",").map((s) => s.trim()).filter(Boolean);
  return list.length ? list : [...DEFAULT_FIELDS];
}

const thaiID = new ThaiID({
  fields: requestedFields(),
  ...(endpoint ? { endpoint } : {}),
});

thaiID.on("connectionChanged", (s) => {
  statusEl.textContent = s;
  const connected = s === "connected";
  readBtn.disabled = !connected;
  disconnectBtn.disabled = s === "disconnected";
  connectBtn.disabled = s !== "disconnected";
  if (s !== "awaiting-approval") approvalEl.textContent = "";
  if (s === "disconnected") {
    outEl.textContent = "No card data yet.";
    clearPhoto();
  }
});
thaiID.on("approvalRequired", ({ comparisonCode, approvalId }) => {
  approvalEl.textContent = ` — waiting for terminal approval (request ${approvalId}, code ${comparisonCode})`;
});
thaiID.on("cardChanged", ({ present }) => {
  if (!present) {
    outEl.textContent = "No card data yet.";
    clearPhoto();
  }
});
thaiID.on("error", (err) => {
  if (err && (err.code === "AUTH_EXPIRED" || err.code === "CONNECTION_FAILED" || err.code === "APPROVAL_DENIED")) {
    outEl.textContent = err.code === "APPROVAL_DENIED" ? "APPROVAL_DENIED" : "No card data yet.";
    approvalEl.textContent = "";
    clearPhoto();
  }
});

connectBtn.addEventListener("click", async () => {
  outEl.textContent = "Connecting…";
  clearPhoto();
  try {
    await thaiID.connect();
    const status = await thaiID.status();
    outEl.textContent = JSON.stringify(status, null, 2);
  } catch (err) {
    outEl.textContent = `Connect failed: ${err.code ?? err.message}\n${err.message ?? ""}`;
  }
});

readBtn.addEventListener("click", async () => {
  outEl.textContent = "Reading…";
  try {
    const result = await thaiID.read();
    outEl.textContent = JSON.stringify({ warnings: result.warnings, fields: result.fields }, null, 2);
    showPortrait(result.fields);
  } catch (err) {
    outEl.textContent = `Read failed: ${err.code ?? err.message}`;
    clearPhoto();
  }
});

disconnectBtn.addEventListener("click", async () => {
  await thaiID.disconnect();
  outEl.textContent = "No card data yet.";
  clearPhoto();
});
