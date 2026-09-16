/** Clinic-staff status copy. Default Thai; English is an explicit fallback. */

export type StaffLang = "th" | "en";

export function staffLang(tag?: string | null): StaffLang {
  const t = (tag ?? "").trim().toLowerCase().replace(/_/g, "-");
  if (t === "en" || t.startsWith("en-")) return "en";
  return "th";
}

export type StaffCopy = {
  lang: StaffLang;
  ready: string;
  readyPractice: string;
  notReady: string;
  cancelled: string;
  helperOn: string;
  helperOff: string;
  readerUnknown: string;
  cardUnknown: string;
  startHelper: string;
  serviceOff: string;
  serviceStalled: string;
  cardUntilService: string;
  fixService: string;
  noReader: string;
  insertAfterReader: string;
  plugReader: string;
  readerConnected: (name: string) => string;
  chooseReader: string;
  insertCard: string;
  insertThenFill: string;
  cardInserted: string;
  clickFill: string;
  unsupportedCard: string;
  insertThaiID: string;
  technicianDetails: string;
  technicianLine: (service: string, ver: string, practice: boolean) => string;
  openTechnician: string;
  helperUnreachable: string;
  checking: string;
  oneMoment: string;
  helperLost: string;
  helperWaiting: string;
  helperBack: string;
  helperBackNext: string;
  testConnection: string;
  testingConnection: string;
  testOk: string;
  testFail: string;
  testServiceOff: string;
  testServiceStalled: string;
  serviceBack: string;
  serviceBackNext: string;
  httpsWsBlocked: string;
  httpsWsBlockedNext: string;
};

const TH: Omit<StaffCopy, "lang" | "readerConnected" | "technicianLine"> & {
  readerConnectedPrefix: string;
} = {
  ready: "พร้อม",
  readyPractice: "พร้อม (บัตรฝึกซ้อม)",
  notReady: "ยังไม่พร้อม",
  cancelled: "ยกเลิกแล้ว",
  helperOn: "OpenThaiID กำลังทำงานบนเครื่องนี้",
  helperOff: "OpenThaiID ยังไม่ทำงานบนเครื่องนี้ กรุณาเปิดโปรแกรม แล้วลองอีกครั้ง",
  readerUnknown: "เครื่องอ่านบัตร: ยังไม่ทราบ จนกว่าจะเปิด OpenThaiID",
  cardUnknown: "บัตร: ยังไม่ทราบ จนกว่าจะเปิด OpenThaiID",
  startHelper: "เปิด OpenThaiID บนเครื่องนี้",
  serviceOff: "บริการอ่านบัตรยังไม่พร้อม (หลังเครื่องหลับก็เจอได้) ไม่ต้องรีสตาร์ทเครื่อง",
  serviceStalled: "บริการอ่านบัตรไม่ตอบสนอง ให้ช่างเทคนิคช่วยตรวจ — ไม่ต้องรีสตาร์ทเครื่อง",
  cardUntilService: "บัตร: ยังไม่ทราบ จนกว่าบริการอ่านบัตรจะทำงาน",
  fixService: "กด ทดสอบการเชื่อมต่อ หรือให้ช่างเทคนิคกด Reconnect card service แล้วรอ พร้อม",
  noReader: "ไม่พบเครื่องอ่านบัตร เสียบเครื่องอ่านเข้าเครื่องนี้",
  insertAfterReader: "เสียบบัตรหลังจากต่อเครื่องอ่านแล้ว",
  plugReader: "เสียบเครื่องอ่านบัตรเข้าเครื่องนี้",
  readerConnectedPrefix: "ต่อเครื่องอ่านแล้ว: ",
  chooseReader: "มีเครื่องอ่านมากกว่าหนึ่งเครื่อง เลือกเครื่องบนหน้า พร้อม",
  insertCard: "เสียบบัตรประชาชนไทย",
  insertThenFill: "เสียบบัตรประชาชนของผู้ป่วย แล้วกด กรอกข้อมูล",
  cardInserted: "เสียบบัตรแล้ว",
  clickFill: "กด กรอกข้อมูล",
  unsupportedCard: "บัตรนี้ไม่ใช่บัตรประชาชนไทย",
  insertThaiID: "เสียบบัตรประชาชนไทย",
  technicianDetails: "รายละเอียดช่างเทคนิค",
  openTechnician: "เปิดเครื่องมือช่างเทคนิคบนเครื่องนี้",
  helperUnreachable: "หน้านี้ติดต่อตัวช่วยไม่ได้",
  checking: "กำลังตรวจ…",
  oneMoment: "สักครู่",
  helperLost: "ตัวช่วยหลุดการเชื่อมต่อ",
  helperWaiting: "กำลังรอให้ OpenThaiID กลับมาบนเครื่องนี้ ไม่ต้องปิดหน้านี้",
  helperBack: "ตัวช่วยกลับมาแล้ว",
  helperBackNext: "กด กรอกข้อมูล ได้ตามปกติ",
  testConnection: "ทดสอบการเชื่อมต่อ",
  testingConnection: "กำลังทดสอบการเชื่อมต่อ…",
  testOk: "การเชื่อมต่อปกติ — ไม่ได้อ่านข้อมูลบนบัตร",
  testFail: "ทดสอบการเชื่อมต่อไม่สำเร็จ เปิด OpenThaiID แล้วลองอีกครั้ง",
  testServiceOff: "ตัวช่วยทำงาน แต่บริการอ่านบัตรยังไม่พร้อม — ไม่ได้อ่านข้อมูลบนบัตร",
  testServiceStalled: "ตัวช่วยทำงาน แต่บริการอ่านบัตรไม่ตอบสนอง — ไม่ได้อ่านข้อมูลบนบัตร",
  serviceBack: "บริการอ่านบัตรกลับมาแล้ว",
  serviceBackNext: "กด กรอกข้อมูล ได้ตามปกติ",
  httpsWsBlocked: "หน้าคลินิกเป็น HTTPS — Firefox และ Safari บล็อกการติดต่อตัวช่วยบนเครื่องนี้",
  httpsWsBlockedNext:
    "ใช้ Chrome หรือ Edge หรือเปิดหน้าคลินิกเป็น http บนเครื่องนี้ หน้า พร้อม ที่ 127.0.0.1 ใช้ได้ตามปกติ ไม่ต้องปิดตัวช่วย",
};

const EN: typeof TH = {
  ready: "Ready",
  readyPractice: "Ready (practice card)",
  notReady: "Not ready yet",
  cancelled: "Cancelled",
  helperOn: "OpenThaiID is running on this computer.",
  helperOff: "OpenThaiID is not running on this computer. Start it, then try again.",
  readerUnknown: "Card reader: unknown until OpenThaiID is running.",
  cardUnknown: "Card: unknown until OpenThaiID is running.",
  startHelper: "Start OpenThaiID on this computer.",
  serviceOff: "The card service is not ready (this can happen after sleep). Do not reboot.",
  serviceStalled: "The card service is not responding. Ask a technician — do not reboot.",
  cardUntilService: "Card: unknown until the card service is working.",
  fixService: "Click Test connection, or ask a technician to click Reconnect card service, then wait until Ready.",
  noReader: "No card reader. Plug the reader into this computer.",
  insertAfterReader: "Insert the card after the reader is connected.",
  plugReader: "Plug the card reader into this computer.",
  readerConnectedPrefix: "Reader connected: ",
  chooseReader: "More than one reader is connected. Choose one on the Ready page.",
  insertCard: "Insert the Thai ID card.",
  insertThenFill: "Insert the patient’s Thai ID card, then click Fill.",
  cardInserted: "Card is inserted.",
  clickFill: "Click Fill.",
  unsupportedCard: "This card is not a Thai National ID.",
  insertThaiID: "Insert a Thai National ID card.",
  technicianDetails: "Technician details",
  openTechnician: "Open technician tools on this PC",
  helperUnreachable: "Helper is not reachable from this page.",
  checking: "Checking…",
  oneMoment: "One moment.",
  helperLost: "The helper disconnected",
  helperWaiting: "Waiting for OpenThaiID to come back on this computer. Leave this page open.",
  helperBack: "The helper is back",
  helperBackNext: "Click Fill as usual.",
  testConnection: "Test connection",
  testingConnection: "Testing connection…",
  testOk: "Connection is OK — no personal card fields were read.",
  testFail: "Connection test failed. Start OpenThaiID, then try again.",
  testServiceOff: "The helper is running, but the card service is not ready — no personal card fields were read.",
  testServiceStalled: "The helper is running, but the card service is not responding — no personal card fields were read.",
  serviceBack: "The card service is back",
  serviceBackNext: "Click Fill as usual.",
  httpsWsBlocked:
    "This clinic page is HTTPS — Firefox and Safari block reaching the helper on this computer.",
  httpsWsBlockedNext:
    "Use Chrome or Edge, or open the clinic page over http on this computer. The Ready page at 127.0.0.1 still works. Leave the helper running.",
};

export function staffCopy(lang?: string | null): StaffCopy {
  const resolved = staffLang(lang);
  const src = resolved === "en" ? EN : TH;
  return {
    lang: resolved,
    ...src,
    readerConnected: (name: string) => `${src.readerConnectedPrefix}${name}`,
    technicianLine: (service: string, ver: string, practice: boolean) => {
      const extra = `${ver ? ` · ${ver}` : ""}${practice ? (resolved === "th" ? " · เครื่องอ่านฝึกซ้อม" : " · practice reader") : ""}`;
      if (resolved === "th") {
        return `ช่างเทคนิค: บริการอ่านบัตร ${service}${extra}`;
      }
      return `Technician: card service ${service}${extra}`;
    },
  };
}

/** Clinic-facing Fill / probe error. `code` is the stable protocol token. */
export type StaffErrorGuidance = {
  code: string;
  message: string;
  next: string;
};

type ErrorPack = { message: string; next: string };

const TH_ERRORS: Record<string, ErrorPack> = {
  BRIDGE_UNAVAILABLE: {
    message: "OpenThaiID ยังไม่ทำงานบนเครื่องนี้",
    next: "เปิดโปรแกรม OpenThaiID แล้วกด กรอกข้อมูล อีกครั้ง — หลังเครื่องหลับหรือตัวช่วยหลุด ไม่ต้องปิดหน้านี้ กด ทดสอบการเชื่อมต่อ ได้",
  },
  CONNECTION_FAILED: {
    message: "หน้าคลินิกติดต่อตัวช่วยไม่ได้",
    next: "เปิด OpenThaiID บนเครื่องนี้ หลังเครื่องหลับหรือเพิ่งรีสตาร์ทตัวช่วย ให้รอไฟสถานะ กด ทดสอบการเชื่อมต่อ แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  HTTPS_WS_BLOCKED: {
    message: "หน้าคลินิกเป็น HTTPS — Firefox และ Safari บล็อกการติดต่อตัวช่วยบนเครื่องนี้",
    next: "ใช้ Chrome หรือ Edge หรือเปิดหน้าคลินิกเป็น http บนเครื่องนี้ หน้า พร้อม ที่ 127.0.0.1 ใช้ได้ตามปกติ ไม่ต้องปิดตัวช่วย",
  },
  PROTOCOL_MISMATCH: {
    message: "หน้าคลินิกกับตัวช่วยใช้โปรโตคอลคนละรุ่น",
    next: "ให้ช่างเทคนิคอัปเดต OpenThaiID หรือไลบรารีในหน้าคลินิก",
  },
  APPROVAL_REQUIRED: {
    message: "หน้าคลินิกกำลังรออนุญาตให้อ่านบัตร",
    next: "ที่หน้า พร้อม ของ OpenThaiID กด อนุญาต สำหรับหน้านี้",
  },
  APPROVAL_DENIED: {
    message: "การอ่านบัตรถูกปฏิเสธ",
    next: "ถ้าเป็นหน้าคลินิกที่ใช้จริง ให้กด กรอกข้อมูล อีกครั้ง แล้วกด อนุญาต",
  },
  APPROVAL_TIMEOUT: {
    message: "หมดเวลารออนุญาต",
    next: "กด กรอกข้อมูล อีกครั้ง แล้วกด อนุญาต บนหน้า พร้อม",
  },
  AUTH_EXPIRED: {
    message: "สิทธิ์อ่านบัตรหมดอายุหรือถูกเพิกถอน",
    next: "กด กรอกข้อมูล อีกครั้ง แล้วกด อนุญาต",
  },
  FIELD_NOT_ALLOWED: {
    message: "หน้าคลินิกขอช่องที่ยังไม่อนุญาต",
    next: "ให้ช่างเทคนิคตรวจสิทธิ์บนหน้า พร้อม แล้วกด อนุญาต ชุดช่องใหม่",
  },
  PCSC_UNAVAILABLE: {
    message: "บริการอ่านบัตรยังไม่พร้อม",
    next: "ไม่ต้องรีสตาร์ทเครื่อง กด ทดสอบการเชื่อมต่อ บนหน้า พร้อม — ถ้ายังไม่ขึ้น พร้อม ให้ช่างเทคนิคเปิด Technician tools แล้วกด Reconnect card service รอไฟ พร้อม",
  },
  NO_READER: {
    message: "ไม่พบเครื่องอ่านบัตร",
    next: "เสียบเครื่องอ่านเข้าเครื่องนี้ แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  READER_SELECTION_REQUIRED: {
    message: "มีเครื่องอ่านมากกว่าหนึ่งเครื่อง",
    next: "เลือกเครื่องอ่านบนหน้า พร้อม แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  NO_CARD: {
    message: "ยังไม่ได้เสียบบัตรประชาชน",
    next: "เสียบบัตรประชาชนของผู้ป่วยให้สุด แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  CARD_REMOVED: {
    message: "บัตรถูกถอดระหว่างอ่าน",
    next: "เสียบบัตรให้สุด รอให้สถานะขึ้น พร้อม แล้วกด กรอกข้อมูล อีกครั้ง — อย่าใช้ค่าในช่องถ้ากรอกไม่สำเร็จ",
  },
  READER_REMOVED: {
    message: "เครื่องอ่านถูกถอดระหว่างอ่าน",
    next: "เสียบเครื่องอ่าน เสียบบัตรให้สุด รอ พร้อม แล้วกด กรอกข้อมูล อีกครั้ง — อย่าใช้ค่าในช่องถ้ากรอกไม่สำเร็จ",
  },
  BUSY: {
    message: "กำลังอ่านบัตรอยู่",
    next: "รอสักครู่ แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  UNSUPPORTED_CARD: {
    message: "บัตรนี้ไม่ใช่บัตรประชาชนไทย",
    next: "เสียบบัตรประชาชนไทย แล้วลองอีกครั้ง",
  },
  READ_FAILED: {
    message: "อ่านบัตรไม่สำเร็จ",
    next: "เสียบบัตรให้สุด รอ พร้อม แล้วกด กรอกข้อมูล อีกครั้ง หรือเรียกช่างเทคนิค — อย่าใช้ค่าในช่องถ้ากรอกไม่สำเร็จ",
  },
  READ_TIMEOUT: {
    message: "อ่านบัตรหมดเวลา",
    next: "เสียบบัตรให้สุด รอ พร้อม แล้วกด กรอกข้อมูล อีกครั้ง — อย่าใช้ค่าในช่องถ้ากรอกไม่สำเร็จ",
  },
  CANCELLED: {
    message: "ยกเลิกแล้ว",
    next: "",
  },
  BACKEND_STALLED: {
    message: "บริการอ่านบัตรไม่ตอบสนอง",
    next: "ไม่ต้องรีสตาร์ทเครื่อง ให้ช่างเทคนิคกด Reconnect card service บนหน้า Technician แล้วรอ พร้อม",
  },
  INVALID_REQUEST: {
    message: "คำขอกรอกข้อมูลไม่ถูกต้อง",
    next: "ให้ช่างเทคนิคตรวจการตั้งค่าช่องในหน้าคลินิก",
  },
  UNSUPPORTED_FIELD: {
    message: "หน้าคลินิกขอช่องที่ไม่รองรับ",
    next: "ให้ช่างเทคนิคตรวจรายการช่องที่ขอ",
  },
  RATE_LIMITED: {
    message: "มีคำขออ่านบัตรถี่เกินไป",
    next: "รอสักครู่ แล้วกด กรอกข้อมูล อีกครั้ง",
  },
};

const EN_ERRORS: Record<string, ErrorPack> = {
  BRIDGE_UNAVAILABLE: {
    message: "OpenThaiID is not running on this computer.",
    next: "Start OpenThaiID, then click Fill again. After sleep or a helper restart, leave this page open and click Test connection.",
  },
  CONNECTION_FAILED: {
    message: "This page could not reach the local helper.",
    next: "Start OpenThaiID on this computer. After sleep or a helper restart, wait for the status lights, click Test connection, then click Fill again.",
  },
  HTTPS_WS_BLOCKED: {
    message: "This clinic page is HTTPS — Firefox and Safari block reaching the helper on this computer.",
    next: "Use Chrome or Edge, or open the clinic page over http on this computer. The Ready page at 127.0.0.1 still works. Leave the helper running.",
  },
  PROTOCOL_MISMATCH: {
    message: "This page and the helper speak different protocol versions.",
    next: "Ask a technician to update OpenThaiID or the clinic library.",
  },
  APPROVAL_REQUIRED: {
    message: "This clinic page is waiting for permission to read the card.",
    next: "On the OpenThaiID Ready page, click Allow for this origin.",
  },
  APPROVAL_DENIED: {
    message: "Reading the card was denied.",
    next: "If this is the clinic page you use, click Fill again and then Allow.",
  },
  APPROVAL_TIMEOUT: {
    message: "The Allow prompt timed out.",
    next: "Click Fill again, then Allow on the Ready page.",
  },
  AUTH_EXPIRED: {
    message: "Permission to read the card expired or was revoked.",
    next: "Click Fill again, then Allow.",
  },
  FIELD_NOT_ALLOWED: {
    message: "This clinic page asked for a field that is not allowed.",
    next: "Ask a technician to Allow the new field set on the Ready page.",
  },
  PCSC_UNAVAILABLE: {
    message: "The card service is not ready.",
    next: "Do not reboot. Click Test connection on the Ready page. If it is still not Ready, ask a technician to open Technician tools and click Reconnect card service, then wait until Ready.",
  },
  NO_READER: {
    message: "No card reader is connected.",
    next: "Plug the reader into this computer, then click Fill again.",
  },
  READER_SELECTION_REQUIRED: {
    message: "More than one card reader is connected.",
    next: "Choose the reader on the Ready page, then click Fill again.",
  },
  NO_CARD: {
    message: "No Thai ID card is inserted.",
    next: "Insert the patient’s Thai ID card fully, then click Fill again.",
  },
  CARD_REMOVED: {
    message: "The card was removed while reading.",
    next: "Insert the card fully, wait until status says Ready, then click Fill again. Do not use values in the form if this Fill did not succeed.",
  },
  READER_REMOVED: {
    message: "The card reader was unplugged while reading.",
    next: "Plug the reader in, insert the card fully, wait until Ready, then click Fill again. Do not use values in the form if this Fill did not succeed.",
  },
  BUSY: {
    message: "A card read is already in progress.",
    next: "Wait a moment, then click Fill again.",
  },
  UNSUPPORTED_CARD: {
    message: "This card is not a Thai National ID.",
    next: "Insert a Thai National ID card, then try again.",
  },
  READ_FAILED: {
    message: "The card could not be read.",
    next: "Insert the card fully, wait until Ready, then click Fill again, or ask a technician. Do not use values in the form if this Fill did not succeed.",
  },
  READ_TIMEOUT: {
    message: "Reading the card timed out.",
    next: "Insert the card fully, wait until Ready, then click Fill again. Do not use values in the form if this Fill did not succeed.",
  },
  CANCELLED: {
    message: "Cancelled.",
    next: "",
  },
  BACKEND_STALLED: {
    message: "The card service is not responding.",
    next: "Do not reboot. Ask a technician to click Reconnect card service on Technician tools, then wait until Ready.",
  },
  INVALID_REQUEST: {
    message: "This Fill request is not set up correctly.",
    next: "Ask a technician to check the field map on this clinic page.",
  },
  UNSUPPORTED_FIELD: {
    message: "This clinic page asked for an unsupported field.",
    next: "Ask a technician to check the requested fields.",
  },
  RATE_LIMITED: {
    message: "Too many card-read requests.",
    next: "Wait a moment, then click Fill again.",
  },
};

const TH_ERROR_FALLBACK: ErrorPack = {
  message: "กรอกข้อมูลไม่สำเร็จ",
  next: "ดูไฟสถานะ แล้วลองอีกครั้ง หรือเรียกช่างเทคนิค",
};

const EN_ERROR_FALLBACK: ErrorPack = {
  message: "Fill did not succeed.",
  next: "Check the status lights, wait until Ready, try Fill again, or ask a technician.",
};

/** Protocol codes for a Fill/read that stopped mid-operation. Codes stay stable. */
export const MID_READ_ABORT_CODES = ["CARD_REMOVED", "READER_REMOVED", "READ_FAILED", "READ_TIMEOUT"] as const;

export function isMidReadAbort(err: unknown): boolean {
  return (MID_READ_ABORT_CODES as readonly string[]).includes(staffErrorCode(err));
}

/** Pull a stable protocol code off a thrown error, wire object, or raw code. */
export function staffErrorCode(err: unknown): string {
  if (typeof err === "string" && err.trim()) return err.trim();
  if (err && typeof err === "object" && "code" in err) {
    const code = (err as { code?: unknown }).code;
    if (typeof code === "string" && code.trim()) return code.trim();
  }
  return "READ_FAILED";
}

/**
 * Clinic-staff guidance for a Fill / probe failure.
 * Default Thai; pass `"en"` for English. Codes stay unchanged.
 * Messages never include citizen ID, names, or other card values.
 */
export function staffErrorGuidance(err: unknown, lang?: string | null): StaffErrorGuidance {
  const resolved = staffLang(lang);
  const code = staffErrorCode(err);
  const pack = resolved === "en" ? EN_ERRORS : TH_ERRORS;
  const fallback = resolved === "en" ? EN_ERROR_FALLBACK : TH_ERROR_FALLBACK;
  const row = pack[code] ?? fallback;
  return { code, message: row.message, next: row.next };
}

/** One-line staff banner: what happened, then what to do. Does not include the code. */
export function formatStaffError(err: unknown, lang?: string | null): string {
  const g = staffErrorGuidance(err, lang);
  return g.next ? `${g.message} ${g.next}` : g.message;
}

export type StaffApprovalGuidance = {
  message: string;
  next: string;
};

/**
 * Banner copy while the helper waits for Allow.
 * Comparison codes are pairing digits, not personal card fields.
 */
export function staffApprovalGuidance(
  info?: { approvalId?: number; comparisonCode?: string } | null,
  lang?: string | null,
): StaffApprovalGuidance {
  const resolved = staffLang(lang);
  const code = typeof info?.comparisonCode === "string" ? info.comparisonCode.trim() : "";
  if (resolved === "en") {
    return {
      message: code
        ? `Allow this clinic page on the OpenThaiID Ready page (code ${code}).`
        : "Allow this clinic page on the OpenThaiID Ready page.",
      next: "The website cannot approve itself. After Allow, click Fill again if it did not continue.",
    };
  }
  return {
    message: code
      ? `อนุญาตหน้าคลินิกนี้บนหน้า พร้อม ของ OpenThaiID (รหัส ${code})`
      : "อนุญาตหน้าคลินิกนี้บนหน้า พร้อม ของ OpenThaiID",
    next: "เว็บไซต์อนุมัติตัวเองไม่ได้ หลังกดอนุญาต หากยังไม่กรอก ให้กด กรอกข้อมูล อีกครั้ง",
  };
}
