import { autofillThaiID, mountThaiIDStatus } from "../../sdk/web/dist/index.js";

const form = document.getElementById("kyc");
const statusEl = document.getElementById("status");
const fillBtn = document.getElementById("fill");
const resetBtn = document.getElementById("reset");

const qs = new URLSearchParams(location.search);
const endpoint = qs.get("ws") || undefined;

const statusHost = document.getElementById("reader-status");
if (statusHost) {
  mountThaiIDStatus(statusHost, endpoint ? { endpoint } : {});
}

fillBtn.addEventListener("click", async () => {
  fillBtn.disabled = true;
  statusEl.textContent = "Connecting…";
  const wantPortrait = document.getElementById("want-portrait")?.checked;
  const request = ["citizenId", "thaiName", "birthDate", "gender", "address"];
  if (wantPortrait) request.push("portrait");
  try {
    const { filled } = await autofillThaiID({
      root: form,
      request,
      ...(endpoint ? { endpoint } : {}),
      onConnectionChanged: (state) => {
        if (state === "awaiting-approval") return;
        statusEl.textContent = state === "connecting" ? "Connecting…" : `Status: ${state}`;
      },
      onApprovalRequired: ({ comparisonCode, approvalId }) => {
        statusEl.textContent = `Approve in the terminal — request ${approvalId}, code ${comparisonCode}`;
      },
    });
    const photo = document.getElementById("photo");
    const gotPhoto = filled.some((f) => f.slot === "portrait" || f.slot.startsWith("portrait."));
    if (photo) photo.hidden = !gotPhoto;
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
  const photo = document.getElementById("photo");
  if (photo) {
    photo.removeAttribute("src");
    photo.hidden = true;
  }
});
