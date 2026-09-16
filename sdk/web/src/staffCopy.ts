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
