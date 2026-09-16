import { autofillThaiID, formatStaffError, mountThaiIDStatus, staffApprovalGuidance } from "../../sdk/web/dist/index.js";

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
      onApprovalRequired: (info) => {
        const g = staffApprovalGuidance(info);
        statusEl.textContent = `${g.message} ${g.next}`;
      },
    });
    const photo = document.getElementById("photo");
    const gotPhoto = filled.some((f) => f.slot === "portrait" || f.slot.startsWith("portrait."));
    if (photo) photo.hidden = !gotPhoto;
    statusEl.textContent = filled.length
      ? `กรอกแล้ว ${filled.length} ช่อง`
      : "เชื่อมต่อแล้ว แต่ไม่มีค่าให้กรอก";
  } catch (err) {
    const code = err && err.code ? err.code : "";
    statusEl.textContent = `${formatStaffError(err)}${code ? ` (${code})` : ""}`;
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
