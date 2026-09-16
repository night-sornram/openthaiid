/** Clinic-staff status copy. Default Thai; English is an explicit fallback. */

export type StaffLang = "th" | "en";

export function staffLang(tag?: string | null): StaffLang {
  const t = (tag ?? "").trim().toLowerCase().replaceAll("_", "-");
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
  serviceOff: "บริการอ่านบัตรปิดอยู่ ให้ช่างเทคนิคเปิดบริการ",
  serviceStalled: "บริการอ่านบัตรไม่ตอบสนอง ให้ช่างเทคนิคช่วยตรวจ",
  cardUntilService: "บัตร: ยังไม่ทราบ จนกว่าบริการอ่านบัตรจะทำงาน",
  fixService: "ให้ช่างเทคนิคแก้ไขบริการอ่านบัตร",
  noReader: "ไม่พบเครื่องอ่านบัตร เสียบเครื่องอ่านเข้าเครื่องนี้",
  insertAfterReader: "เสียบบัตรหลังจากต่อเครื่องอ่านแล้ว",
  plugReader: "เสียบเครื่องอ่านบัตรเข้าเครื่องนี้",
  readerConnectedPrefix: "ต่อเครื่องอ่านแล้ว: ",
  insertCard: "เสียบบัตรประชาชนไทย",
  insertThenFill: "เสียบบัตรประชาชนของผู้ป่วย แล้วกด กรอกข้อมูล",
  cardInserted: "เสียบบัตรแล้ว",
  clickFill: "กด กรอกข้อมูล",
  unsupportedCard: "บัตรนี้ไม่ใช่บัตรประชาชนไทย",
  insertThaiID: "เสียบบัตรประชาชนไทย",
  technicianDetails: "รายละเอียดช่างเทคนิค",
  openTechnician: "เปิดเครื่องมือช่างเทคนิคบนเครื่องนี้",
  helperUnreachable: "หน้านี้ติดต่อตัวช่วยไม่ได้",
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
  serviceOff: "The card service is off. Ask a technician to start it.",
  serviceStalled: "The card service is not responding. Ask a technician.",
  cardUntilService: "Card: unknown until the card service is working.",
  fixService: "Ask a technician to fix the card service.",
  noReader: "No card reader. Plug the reader into this computer.",
  insertAfterReader: "Insert the card after the reader is connected.",
  plugReader: "Plug the card reader into this computer.",
  readerConnectedPrefix: "Reader connected: ",
  insertCard: "Insert the Thai ID card.",
  insertThenFill: "Insert the patient’s Thai ID card, then click Fill.",
  cardInserted: "Card is inserted.",
  clickFill: "Click Fill.",
  unsupportedCard: "This card is not a Thai National ID.",
  insertThaiID: "Insert a Thai National ID card.",
  technicianDetails: "Technician details",
  openTechnician: "Open technician tools on this PC",
  helperUnreachable: "Helper is not reachable from this page.",
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
    next: "เปิดโปรแกรม OpenThaiID แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  CONNECTION_FAILED: {
    message: "หน้าคลินิกติดต่อตัวช่วยไม่ได้",
    next: "เปิด OpenThaiID บนเครื่องนี้ ตรวจว่าเบราว์เซอร์อนุญาตเครือข่ายภายใน แล้วลองอีกครั้ง",
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
    message: "บริการอ่านบัตรปิดอยู่",
    next: "ให้ช่างเทคนิคเปิดบริการอ่านบัตร",
  },
  NO_READER: {
    message: "ไม่พบเครื่องอ่านบัตร",
    next: "เสียบเครื่องอ่านเข้าเครื่องนี้ แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  READER_SELECTION_REQUIRED: {
    message: "มีเครื่องอ่านมากกว่าหนึ่งเครื่อง",
    next: "ให้ช่างเทคนิคเลือกเครื่องอ่านใน Technician tools",
  },
  NO_CARD: {
    message: "ยังไม่ได้เสียบบัตรประชาชน",
    next: "เสียบบัตรประชาชนของผู้ป่วยให้สุด แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  CARD_REMOVED: {
    message: "บัตรถูกถอดระหว่างอ่าน",
    next: "เสียบบัตรให้สุด แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  READER_REMOVED: {
    message: "เครื่องอ่านถูกถอด",
    next: "เสียบเครื่องอ่าน แล้วลองอีกครั้ง",
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
    next: "เสียบบัตรให้สุด แล้วลองอีกครั้ง หรือเรียกช่างเทคนิค",
  },
  READ_TIMEOUT: {
    message: "อ่านบัตรหมดเวลา",
    next: "เสียบบัตรให้สุด แล้วกด กรอกข้อมูล อีกครั้ง",
  },
  CANCELLED: {
    message: "ยกเลิกแล้ว",
    next: "",
  },
  BACKEND_STALLED: {
    message: "บริการอ่านบัตรไม่ตอบสนอง",
    next: "ให้ช่างเทคนิคช่วยตรวจบริการอ่านบัตร",
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
    next: "Start OpenThaiID, then click Fill again.",
  },
  CONNECTION_FAILED: {
    message: "This page could not reach the local helper.",
    next: "Start OpenThaiID on this computer, allow local-network access if asked, then try again.",
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
    message: "The card service is off.",
    next: "Ask a technician to start the card service.",
  },
  NO_READER: {
    message: "No card reader is connected.",
    next: "Plug the reader into this computer, then click Fill again.",
  },
  READER_SELECTION_REQUIRED: {
    message: "More than one card reader is connected.",
    next: "Ask a technician to choose a reader in Technician tools.",
  },
  NO_CARD: {
    message: "No Thai ID card is inserted.",
    next: "Insert the patient’s Thai ID card fully, then click Fill again.",
  },
  CARD_REMOVED: {
    message: "The card was removed while reading.",
    next: "Insert the card fully, then click Fill again.",
  },
  READER_REMOVED: {
    message: "The card reader was unplugged.",
    next: "Plug the reader in, then try again.",
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
    next: "Insert the card fully and try again, or ask a technician.",
  },
  READ_TIMEOUT: {
    message: "Reading the card timed out.",
    next: "Insert the card fully, then click Fill again.",
  },
  CANCELLED: {
    message: "Cancelled.",
    next: "",
  },
  BACKEND_STALLED: {
    message: "The card service is not responding.",
    next: "Ask a technician to check the card service.",
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
  next: "Check the status lights, try again, or ask a technician.",
};

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
