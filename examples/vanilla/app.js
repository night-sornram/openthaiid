import { ThaiID } from "../../sdk/web/dist/index.js";

const statusEl = document.getElementById("status");
const approvalEl = document.getElementById("approval");
const outEl = document.getElementById("out");
const connectBtn = document.getElementById("connect");
const readBtn = document.getElementById("read");
const disconnectBtn = document.getElementById("disconnect");

const qs = new URLSearchParams(location.search);
const endpoint = qs.get("ws") || undefined;

const thaiID = new ThaiID({
  fields: ["citizenId", "thaiName", "birthDate"],
  ...(endpoint ? { endpoint } : {}),
});

thaiID.on("connectionChanged", (s) => {
  statusEl.textContent = s;
  const connected = s === "connected";
  readBtn.disabled = !connected;
  disconnectBtn.disabled = s === "disconnected";
  connectBtn.disabled = s !== "disconnected";
  if (s !== "awaiting-approval") approvalEl.textContent = "";
});
thaiID.on("approvalRequired", ({ comparisonCode, approvalId }) => {
  approvalEl.textContent = ` — waiting for terminal approval (request ${approvalId}, code ${comparisonCode})`;
});
thaiID.on("cardChanged", ({ present }) => {
  if (!present) outEl.textContent = "No card data yet.";
});
thaiID.on("error", (err) => {
  if (err.code === "AUTH_EXPIRED" || err.code === "CONNECTION_FAILED") {
    outEl.textContent = "No card data yet.";
    approvalEl.textContent = "";
  }
});

connectBtn.addEventListener("click", async () => {
  outEl.textContent = "Connecting…";
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
    outEl.textContent = JSON.stringify(
      { warnings: result.warnings, fields: result.fields },
      null,
      2,
    );
  } catch (err) {
    outEl.textContent = `Read failed: ${err.code ?? err.message}`;
  }
});

disconnectBtn.addEventListener("click", async () => {
  await thaiID.disconnect();
  outEl.textContent = "No card data yet.";
});
