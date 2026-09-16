import { describe, expect, it } from "vitest";
import { ThaiIDError } from "./index";
import { localizeThaiIDError } from "./staffError";
import {
  formatStaffError,
  isMidReadAbort,
  staffApprovalGuidance,
  staffCopy,
  staffErrorCode,
  staffErrorGuidance,
  staffLang,
} from "./staffCopy";

const CLINIC_CODES = [
  "BRIDGE_UNAVAILABLE",
  "NO_READER",
  "NO_CARD",
  "APPROVAL_REQUIRED",
  "APPROVAL_DENIED",
  "FIELD_NOT_ALLOWED",
] as const;

function assertNoPII(text: string) {
  for (const leak of ["1234567890121", "สมชาย", "Somchai", "laserId", "laser-id", "NHSO"]) {
    expect(text).not.toContain(leak);
  }
}

describe("staffLang", () => {
  it("defaults to Thai; English is explicit", () => {
    expect(staffLang()).toBe("th");
    expect(staffLang("th-TH")).toBe("th");
    expect(staffLang("en")).toBe("en");
    expect(staffLang("en-GB")).toBe("en");
    expect(staffLang("en_GB")).toBe("en");
  });
});

describe("staffErrorGuidance", () => {
  it("keeps protocol codes stable and Thai by default", () => {
    for (const code of CLINIC_CODES) {
      const g = staffErrorGuidance({ code });
      expect(g.code).toBe(code);
      expect(g.message.length).toBeGreaterThan(4);
      expect(g.next.length).toBeGreaterThan(4);
      assertNoPII(`${g.code} ${g.message} ${g.next}`);
    }
    expect(staffErrorGuidance({ code: "BRIDGE_UNAVAILABLE" }).message).toContain("ยังไม่ทำงาน");
    expect(staffErrorGuidance({ code: "NO_READER" }).message).toContain("เครื่องอ่าน");
    expect(staffErrorGuidance({ code: "NO_CARD" }).message).toContain("บัตร");
    expect(staffErrorGuidance({ code: "APPROVAL_REQUIRED" }).next).toContain("อนุญาต");
    expect(staffErrorGuidance({ code: "APPROVAL_DENIED" }).message).toContain("ปฏิเสธ");
    expect(staffErrorGuidance({ code: "FIELD_NOT_ALLOWED" }).message).toContain("ช่อง");
  });

  it("English option does not change codes", () => {
    const g = staffErrorGuidance({ code: "BRIDGE_UNAVAILABLE" }, "en");
    expect(g.code).toBe("BRIDGE_UNAVAILABLE");
    expect(g.message).toContain("not running");
    expect(formatStaffError({ code: "NO_CARD" }, "en")).toContain("Insert");
    const pick = staffErrorGuidance({ code: "READER_SELECTION_REQUIRED" });
    expect(pick.code).toBe("READER_SELECTION_REQUIRED");
    expect(pick.next).toContain("หน้า พร้อม");
    expect(staffErrorGuidance({ code: "READER_SELECTION_REQUIRED" }, "en").next).toContain("Ready page");
    expect(staffErrorGuidance({ code: "CARD_REMOVED" }).next).toContain("พร้อม");
    expect(staffErrorGuidance({ code: "CARD_REMOVED" }).next).toContain("กรอกข้อมูล");
    expect(staffErrorGuidance({ code: "READER_REMOVED" }).message).toContain("ถอด");
    expect(formatStaffError({ code: "CARD_REMOVED" }, "en")).toMatch(/Ready/i);
    expect(isMidReadAbort({ code: "CARD_REMOVED" })).toBe(true);
    expect(isMidReadAbort({ code: "NO_CARD" })).toBe(false);
    const pcsc = staffErrorGuidance({ code: "PCSC_UNAVAILABLE" });
    expect(pcsc.code).toBe("PCSC_UNAVAILABLE");
    expect(pcsc.next).toContain("ทดสอบการเชื่อมต่อ");
    expect(pcsc.next).toContain("Reconnect card service");
    expect(pcsc.next).toContain("พร้อม");
    expect(pcsc.next).toContain("ไม่ต้องรีสตาร์ท");
    expect(staffErrorGuidance({ code: "PCSC_UNAVAILABLE" }, "en").next).toMatch(/Test connection/i);
    expect(staffErrorGuidance({ code: "BRIDGE_UNAVAILABLE" }).next).toContain("ทดสอบการเชื่อมต่อ");
    expect(formatStaffError({ code: "PCSC_UNAVAILABLE" })).not.toMatch(/PC\/SC|WebSocket/i);
  });

  it("unknown codes stay unknown and still get guidance", () => {
    const g = staffErrorGuidance({ code: "SOME_NEW_CODE" });
    expect(g.code).toBe("SOME_NEW_CODE");
    expect(g.message).toContain("ไม่สำเร็จ");
  });

  it("reads the code off a ThaiIDError", () => {
    expect(staffErrorCode(new ThaiIDError("NO_READER", "reader not found"))).toBe("NO_READER");
    expect(staffErrorGuidance(new ThaiIDError("NO_CARD", "no card present")).code).toBe("NO_CARD");
  });

  it("catalog messages never include personal or invented-id tokens", () => {
    const codes = [
      "BRIDGE_UNAVAILABLE",
      "CONNECTION_FAILED",
      "APPROVAL_REQUIRED",
      "APPROVAL_DENIED",
      "APPROVAL_TIMEOUT",
      "FIELD_NOT_ALLOWED",
      "NO_READER",
      "NO_CARD",
      "PCSC_UNAVAILABLE",
      "BACKEND_STALLED",
      "AUTH_EXPIRED",
      "UNSUPPORTED_CARD",
      "CARD_REMOVED",
      "READER_REMOVED",
      "INVALID_REQUEST",
      "UNSUPPORTED_FIELD",
      "READ_FAILED",
      "READ_TIMEOUT",
      "HTTPS_WS_BLOCKED",
    ];
    for (const code of codes) {
      assertNoPII(formatStaffError({ code }));
      assertNoPII(formatStaffError({ code }, "en"));
      expect(staffErrorGuidance({ code }).code).toBe(code);
    }
  });

  it("one-line format includes the next step, not the code", () => {
    const line = formatStaffError({ code: "BRIDGE_UNAVAILABLE" });
    expect(line).toContain("ยังไม่ทำงาน");
    expect(line).toContain("เปิด");
    expect(line).not.toContain("BRIDGE_UNAVAILABLE");
  });
});

describe("staffApprovalGuidance", () => {
  it("Thai default includes the comparison code and no card values", () => {
    const g = staffApprovalGuidance({ approvalId: 3, comparisonCode: "654321" });
    expect(g.message).toContain("654321");
    expect(g.message).toContain("อนุญาต");
    expect(g.next).toContain("อนุมัติตัวเองไม่ได้");
    assertNoPII(`${g.message} ${g.next}`);
  });

  it("English fallback", () => {
    const g = staffApprovalGuidance({ comparisonCode: "111111" }, "en");
    expect(g.message).toContain("111111");
    expect(g.message).toContain("Allow");
  });
});

describe("localizeThaiIDError", () => {
  it("replaces the human message and keeps code / retryable", () => {
    const raw = new ThaiIDError("BRIDGE_UNAVAILABLE", "OpenThaiID helper is not running on this PC", true, 9);
    const out = localizeThaiIDError(raw);
    expect(out).toBeInstanceOf(ThaiIDError);
    expect(out).toMatchObject({ code: "BRIDGE_UNAVAILABLE", retryable: true, requestId: 9 });
    expect((out as ThaiIDError).message).toContain("ยังไม่ทำงาน");
    expect((out as ThaiIDError).message).not.toContain("helper is not running");
  });

  it("leaves non-ThaiIDError values alone", () => {
    expect(localizeThaiIDError("nope")).toBe("nope");
  });
});

describe("staffCopy status strings", () => {
  it("Thai Ready copy has no protocol jargon", () => {
    const c = staffCopy();
    expect(c.ready).toBe("พร้อม");
    expect(c.helperLost).toContain("หลุด");
    expect(c.testConnection).toBe("ทดสอบการเชื่อมต่อ");
    expect(c.fixService).toContain("Reconnect card service");
    expect(c.serviceBack).toContain("กลับมา");
    expect(JSON.stringify(c)).not.toMatch(/PC\/SC|WebSocket|bridge/i);
    const blocked = staffErrorGuidance({ code: "HTTPS_WS_BLOCKED" });
    expect(blocked.code).toBe("HTTPS_WS_BLOCKED");
    expect(blocked.message).toContain("HTTPS");
    expect(`${blocked.message} ${blocked.next}`).not.toMatch(/WebSocket|PC\/SC|bridge/i);
    expect(staffErrorGuidance({ code: "HTTPS_WS_BLOCKED" }, "en").next).toMatch(/Chrome or Edge/i);
  });
});
