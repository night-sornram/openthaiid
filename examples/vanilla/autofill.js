import { autofillThaiID } from "../../sdk/web/dist/index.js";

const form = document.getElementById("kyc");
const statusEl = document.getElementById("status");
const fillBtn = document.getElementById("fill");
const resetBtn = document.getElementById("reset");

const qs = new URLSearchParams(location.search);
const endpoint = qs.get("ws") || undefined;

fillBtn.addEventListener("click", async () => {
  fillBtn.disabled = true;
  statusEl.textContent = "Connecting…";
  try {
    const { filled } = await autofillThaiID({
      root: form,
      ...(endpoint ? { endpoint } : {}),
      onConnectionChanged: (state) => {
        if (state === "awaiting-approval") return;
        statusEl.textContent = state === "connecting" ? "Connecting…" : `Status: ${state}`;
      },
      onApprovalRequired: ({ comparisonCode, approvalId }) => {
        statusEl.textContent = `Approve in the terminal — request ${approvalId}, code ${comparisonCode}`;
      },
    });
    statusEl.textContent = filled.length ? `Filled ${filled.length} field(s).` : "Connected, but no values to fill.";
  } catch (err) {
    statusEl.textContent = `Autofill failed: ${err.code ?? err.message}${err.message ? ` — ${err.message}` : ""}`;
  } finally {
    fillBtn.disabled = false;
  }
});

resetBtn.addEventListener("click", () => {
  form.reset();
  statusEl.textContent = "Idle.";
});
