import { afterEach, describe, expect, it } from "vitest";
import {
  applyAutofillValues,
  autofillThaiID,
  collectAutofillTargets,
  discoverAutofillTargets,
  filterTargetsByRequest,
  formatAutofillValue,
  formatCardDateIso,
  normalizeAutofillRequest,
  parseSlot,
  slotsToFields,
  ThaiID,
  ThaiIDError,
  type FieldValues,
} from "./index";

/** Seeded fictional card from `internal/pcsc/synthetic.go` / `openthaiid -synthetic`. */
const SYNTHETIC_FIELDS: FieldValues = {
  citizenId: "1234567890121",
  thaiName: {
    title: "นาย",
    givenName: "สมชาย",
    middleName: null,
    familyName: "ใจดี",
    fullName: "นาย สมชาย ใจดี",
  },
  englishName: {
    title: "Mr.",
    givenName: "Somchai",
    middleName: null,
    familyName: "Jaidee",
    fullName: "Mr. Somchai Jaidee",
  },
  birthDate: { year: 1990, month: 4, day: 12 },
  gender: { code: "1", value: "male" },
  issuer: "เทศบาลนครนิทาน",
  issueDate: { year: 2017, month: 1, day: 15 },
  expireDate: { year: 2027, month: 1, day: 14 },
  address: {
    houseNo: "99",
    villageNo: "หมู่ที่ 1",
    alley: null,
    street: "ถนนนิทาน",
    subdistrict: "ตำบลบ้านใหม่",
    district: "อำเภอเมือง",
    province: "จังหวัดนิทาน",
    fullAddress: "99 หมู่ที่ 1 ถนนนิทาน ตำบลบ้านใหม่ อำเภอเมือง จังหวัดนิทาน",
  },
  portrait: {
    mimeType: "image/jpeg",
    encoding: "base64",
    data: "QQ==",
  },
};

type FakeWS = {
  readyState: number;
  sent: string[];
  onopen: (() => void) | null;
  onerror: (() => void) | null;
  onmessage: ((ev: { data: string }) => void) | null;
  onclose: (() => void) | null;
  send: (s: string) => void;
  close: () => void;
  respondTo: (s: string) => void;
};

function installFakeWebSocket(opts: { fields?: FieldValues; failFirst?: number; probeReaders?: Array<Record<string, unknown>> } = {}) {
  const sockets: FakeWS[] = [];
  const cardFields = opts.fields ?? SYNTHETIC_FIELDS;
  let remainingFails = opts.failFirst ?? 0;
  const probeReaders = opts.probeReaders ?? [
    { readerId: "r1", name: "ACR39U", presence: "present", selected: true },
  ];
  class Fake {
    static OPEN = 1;
    static CONNECTING = 0;
    static CLOSING = 2;
    static CLOSED = 3;
    readyState = 0;
    sent: string[] = [];
    onopen: (() => void) | null = null;
    onerror: (() => void) | null = null;
    onmessage: ((ev: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    constructor(public url: string, public protocols?: string[]) {
      sockets.push(this as unknown as FakeWS);
      if (remainingFails > 0) {
        remainingFails--;
        queueMicrotask(() => {
          this.onerror?.();
        });
        return;
      }
      queueMicrotask(() => {
        this.readyState = 1;
        this.onopen?.();
      });
    }
    send(s: string) {
      this.sent.push(s);
      queueMicrotask(() => this.respondTo(s));
    }
    close() {
      this.readyState = 3;
      this.onclose?.();
    }
    respondTo(s: string) {
      if (!this.onmessage) return;
      const msg = JSON.parse(s);
      if (msg.type === "bridge.probe") {
        this.onmessage({
          data: JSON.stringify({
            v: 1,
            id: msg.id,
            result: {
              version: "0.0.1-test",
              instanceId: "bridge-x",
              synthetic: true,
              backend: { name: "synthetic", status: "available" },
              readers: probeReaders,
              managerUrl: "http://127.0.0.1:17891/",
            },
          }),
        });
        return;
      }
      if (msg.type === "session.open") {
        this.onmessage({
          data: JSON.stringify({
            v: 1,
            type: "approval.required",
            instanceId: "bridge-x",
            sequence: 1,
            payload: { approvalId: 1, comparisonCode: "654321", fields: msg.payload?.fields },
          }),
        });
        this.onmessage({
          data: JSON.stringify({
            v: 1,
            id: msg.id,
            result: {
              token: "tok-1",
              fields: msg.payload?.fields ?? [],
              expiresAt: new Date(Date.now() + 60_000).toISOString(),
              instanceId: "bridge-x",
            },
          }),
        });
        setTimeout(() => {
          this.onmessage?.({
            data: JSON.stringify({
              v: 1,
              type: "status.snapshot",
              instanceId: "bridge-x",
              sequence: 2,
              payload: {
                connection: "connected",
                backend: "available",
                readers: [{ readerId: "r1", name: "ACR39U", presence: "present", cardSessionId: "cs_1" }],
              },
            }),
          });
        }, 0);
      }
      if (msg.type === "card.read") {
        const requested: string[] = msg.payload?.fields ?? [];
        const fields: Record<string, unknown> = {};
        for (const f of requested) {
          if (f in cardFields) fields[f] = cardFields[f as keyof FieldValues];
        }
        this.onmessage({
          data: JSON.stringify({
            v: 1,
            id: msg.id,
            result: {
              readerId: msg.payload?.readerId ?? "r1",
              cardSessionId: "cs_1",
              fields,
              warnings: [],
            },
          }),
        });
      }
    }
  }
  const g = globalThis as { WebSocket?: unknown };
  const prev = g.WebSocket;
  g.WebSocket = Fake as unknown as WebSocket;
  return {
    sockets,
    restore: () => {
      g.WebSocket = prev;
    },
  };
}

function allSent(fake: { sockets: FakeWS[] }): Array<{ type?: string; payload?: { fields?: string[] } }> {
  return fake.sockets.flatMap((s) => s.sent.map((x) => JSON.parse(x)));
}

function sessionOpen(fake: { sockets: FakeWS[] }) {
  return allSent(fake).find((m) => m.type === "session.open");
}

afterEach(() => {
  document.body.innerHTML = "";
});

describe("autofill mapping", () => {
  it("maps slots to the least-privilege field set", () => {
    expect(slotsToFields(["citizenId", "thaiName.givenName", "thaiName.familyName", "birthDate"])).toEqual([
      "citizenId",
      "thaiName",
      "birthDate",
    ]);
  });

  it("rejects portrait and unknown slots", () => {
    expect(() => parseSlot("laserId")).toThrow(ThaiIDError);
    expect(() => parseSlot("thaiName.nickname")).toThrow(ThaiIDError);
    expect(() => parseSlot("address.country")).toThrow(ThaiIDError);
    expect(parseSlot("portrait")).toEqual({ field: "portrait", path: null });
    expect(parseSlot("portrait.dataUrl")).toEqual({ field: "portrait", path: "dataUrl" });
  });

  it("formats synthetic demo values for inputs", () => {
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "citizenId")).toBe("1234567890121");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "thaiName")).toBe("นาย สมชาย ใจดี");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "thaiName.givenName")).toBe("สมชาย");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "thaiName.familyName")).toBe("ใจดี");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "englishName")).toBe("Mr. Somchai Jaidee");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "birthDate")).toBe("1990-04-12");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "gender")).toBe("male");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "gender.code")).toBe("1");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "issuer")).toBe("เทศบาลนครนิทาน");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "address")).toBe(
      "99 หมู่ที่ 1 ถนนนิทาน ตำบลบ้านใหม่ อำเภอเมือง จังหวัดนิทาน",
    );
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "address.province")).toBe("จังหวัดนิทาน");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "portrait")).toBe("data:image/jpeg;base64,QQ==");
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "portrait.data")).toBe("QQ==");
  });

  it("strips citizenId to digits", () => {
    expect(formatAutofillValue({ citizenId: "1-2345-67890-12-1" }, "citizenId")).toBe("1234567890121");
  });

  it("pads ISO dates and keeps partial dates for text inputs", () => {
    expect(formatCardDateIso({ year: 1990, month: 4, day: 12 })).toBe("1990-04-12");
    expect(formatAutofillValue({ birthDate: { year: 1990, month: 4, day: null } }, "birthDate")).toBe("1990-04");
    expect(formatAutofillValue({ birthDate: { year: 1990, month: null, day: null } }, "birthDate")).toBe("1990");
    expect(formatAutofillValue({ birthDate: { year: null, month: null, day: null } }, "birthDate")).toBe("");
  });

  it("uses complete ISO on input[type=date] and skips incomplete dates", () => {
    const date = document.createElement("input");
    date.type = "date";
    expect(formatAutofillValue(SYNTHETIC_FIELDS, "birthDate", {}, date)).toBe("1990-04-12");
    expect(formatAutofillValue({ birthDate: { year: 1990, month: 4, day: null } }, "birthDate", {}, date)).toBe("");
  });

  it("applies gender labels", () => {
    expect(
      formatAutofillValue(SYNTHETIC_FIELDS, "gender", { genderLabels: { male: "ชาย", female: "หญิง" } }),
    ).toBe("ชาย");
    expect(formatAutofillValue({ gender: { code: "2", value: "female" } }, "gender")).toBe("female");
    expect(formatAutofillValue({ gender: { code: "9", value: null } }, "gender")).toBe("");
    expect(
      formatAutofillValue({ gender: { code: "9", value: null } }, "gender", {
        genderLabels: { male: "male", female: "female", unknown: "unspecified" },
      }),
    ).toBe("unspecified");
  });

  it("skips null card fields", () => {
    expect(formatAutofillValue({ thaiName: null, address: null }, "thaiName")).toBe("");
    expect(formatAutofillValue({ address: { ...SYNTHETIC_FIELDS.address!, alley: null } }, "address.alley")).toBe("");
  });
});

describe("autofill DOM helpers", () => {
  it("discovers data-openthaiid attributes", () => {
    document.body.innerHTML = `
      <form id="kyc">
        <input data-openthaiid="citizenId" />
        <input data-openthaiid="thaiName.givenName" />
        <input data-openthaiid="birthDate" type="date" />
      </form>
    `;
    const found = discoverAutofillTargets(document.getElementById("kyc")!);
    expect(found.map((t) => t.slot)).toEqual(["citizenId", "thaiName.givenName", "birthDate"]);
    expect(slotsToFields(found.map((t) => t.slot))).toEqual(["citizenId", "thaiName", "birthDate"]);
  });

  it("fills inputs and dispatches input/change", () => {
    document.body.innerHTML = `
      <input id="cid" />
      <input id="name" />
      <input id="dob" type="date" />
      <select id="gender">
        <option value=""></option>
        <option value="male">Male</option>
        <option value="female">Female</option>
      </select>
      <textarea id="addr"></textarea>
    `;
    const events: string[] = [];
    for (const id of ["cid", "name", "dob", "gender", "addr"]) {
      const el = document.getElementById(id)!;
      el.addEventListener("input", () => events.push(`${id}:input`));
      el.addEventListener("change", () => events.push(`${id}:change`));
    }
    const applied = applyAutofillValues(
      [
        { slot: "citizenId", elements: [document.getElementById("cid")!] },
        { slot: "thaiName", elements: [document.getElementById("name")!] },
        { slot: "birthDate", elements: [document.getElementById("dob")!] },
        { slot: "gender", elements: [document.getElementById("gender")!] },
        { slot: "address", elements: [document.getElementById("addr")!] },
      ],
      SYNTHETIC_FIELDS,
    );
    expect((document.getElementById("cid") as HTMLInputElement).value).toBe("1234567890121");
    expect((document.getElementById("name") as HTMLInputElement).value).toBe("นาย สมชาย ใจดี");
    expect((document.getElementById("dob") as HTMLInputElement).value).toBe("1990-04-12");
    expect((document.getElementById("gender") as HTMLSelectElement).value).toBe("male");
    expect((document.getElementById("addr") as HTMLTextAreaElement).value).toBe(
      "99 หมู่ที่ 1 ถนนนิทาน ตำบลบ้านใหม่ อำเภอเมือง จังหวัดนิทาน",
    );
    expect(applied.values.citizenId).toBe("1234567890121");
    expect(events).toContain("cid:input");
    expect(events).toContain("cid:change");
    expect(events).toContain("dob:change");
  });

  it("sets img src from a portrait data URL", () => {
    document.body.innerHTML = `<img id="photo" alt="" />`;
    applyAutofillValues(
      [{ slot: "portrait", elements: [document.getElementById("photo")!] }],
      SYNTHETIC_FIELDS,
    );
    expect((document.getElementById("photo") as HTMLImageElement).getAttribute("src")).toBe(
      "data:image/jpeg;base64,QQ==",
    );
  });

  it("does not wipe existing input when the card field is null", () => {
    document.body.innerHTML = `<input id="name" value="keep me" />`;
    applyAutofillValues(
      [{ slot: "thaiName", elements: [document.getElementById("name")!] }],
      { thaiName: null },
    );
    expect((document.getElementById("name") as HTMLInputElement).value).toBe("keep me");
  });

  it("records missing selectors", () => {
    const applied = applyAutofillValues([{ slot: "citizenId", elements: [] }], SYNTHETIC_FIELDS);
    expect(applied.missing).toEqual(["citizenId"]);
    expect(applied.filled).toEqual([]);
  });

  it("collects top-level slot keys and CSS selectors", () => {
    document.body.innerHTML = `<input id="citizen-id" /><input id="full-name" />`;
    const targets = collectAutofillTargets({
      citizenId: "#citizen-id",
      thaiName: "#full-name",
    });
    expect(targets.map((t) => t.slot).sort()).toEqual(["citizenId", "thaiName"]);
    expect(targets.every((t) => t.elements.length === 1)).toBe(true);
  });

  it("checks matching radio buttons", () => {
    document.body.innerHTML = `
      <input type="radio" name="g" value="male" data-openthaiid="gender" />
      <input type="radio" name="g" value="female" data-openthaiid="gender" />
    `;
    const targets = discoverAutofillTargets();
    applyAutofillValues(targets, SYNTHETIC_FIELDS);
    const radios = Array.from(document.querySelectorAll<HTMLInputElement>('input[name="g"]'));
    expect(radios[0]!.checked).toBe(true);
    expect(radios[1]!.checked).toBe(false);
  });
});

describe("autofillThaiID", () => {
  it("connects, fills, and disconnects from a user-gesture-style call", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `
      <form id="kyc">
        <input data-openthaiid="citizenId" />
        <input data-openthaiid="thaiName" />
        <input data-openthaiid="birthDate" type="date" />
      </form>
    `;
    const approvals: string[] = [];
    try {
      const out = await autofillThaiID({
        root: document.getElementById("kyc")!,
        onApprovalRequired: ({ comparisonCode }) => approvals.push(comparisonCode),
      });
      expect(approvals).toEqual(["654321"]);
      expect((document.querySelector("[data-openthaiid=citizenId]") as HTMLInputElement).value).toBe("1234567890121");
      expect((document.querySelector("[data-openthaiid=thaiName]") as HTMLInputElement).value).toBe("นาย สมชาย ใจดี");
      expect((document.querySelector("[data-openthaiid=birthDate]") as HTMLInputElement).value).toBe("1990-04-12");
      expect(out.filled.map((f) => f.slot).sort()).toEqual(["birthDate", "citizenId", "thaiName"]);
      const open = sessionOpen(fake);
      expect(open.payload.fields).toEqual(["citizenId", "thaiName", "birthDate"]);
      const closed = allSent(fake).some((m) => m.type === "session.close");
      expect(closed).toBe(true);
      expect(allSent(fake).some((m) => m.type === "bridge.probe")).toBe(true);
      expect(allSent(fake).filter((m) => m.type === "card.read")).toHaveLength(1);
    } finally {
      fake.restore();
    }
  });

  it("uses the selected reader from probe", async () => {
    const fake = installFakeWebSocket({
      probeReaders: [
        { readerId: "r1", name: "Reader One", presence: "present", selected: false },
        { readerId: "r2", name: "Reader Two", presence: "present", selected: true },
      ],
    });
    document.body.innerHTML = `<input data-openthaiid="citizenId" />`;
    try {
      await autofillThaiID({ root: document.body });
      const reads = allSent(fake).filter((m) => m.type === "card.read");
      expect(reads).toHaveLength(1);
      expect(reads[0].payload.readerId).toBe("r2");
    } finally {
      fake.restore();
    }
  });

  it("fails Fill when two readers are present and none is selected", async () => {
    const fake = installFakeWebSocket({
      probeReaders: [
        { readerId: "r1", name: "Reader One", presence: "present" },
        { readerId: "r2", name: "Reader Two", presence: "present" },
      ],
    });
    document.body.innerHTML = `<input data-openthaiid="citizenId" />`;
    try {
      await expect(autofillThaiID({ root: document.body })).rejects.toMatchObject({
        code: "READER_SELECTION_REQUIRED",
      });
      expect(allSent(fake).some((m) => m.type === "session.open")).toBe(false);
      expect(allSent(fake).some((m) => m.type === "card.read")).toBe(false);
    } finally {
      fake.restore();
    }
  });

  it("requests only mapped fields (selector map)", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `
      <input id="citizen-id" />
      <input id="full-name" />
      <textarea id="addr"></textarea>
    `;
    try {
      await autofillThaiID({
        fields: {
          citizenId: "#citizen-id",
          thaiName: "#full-name",
          address: "#addr",
        },
      });
      const open = sessionOpen(fake);
      expect(open.payload.fields).toEqual(["citizenId", "thaiName", "address"]);
      expect((document.getElementById("addr") as HTMLTextAreaElement).value).toContain("จังหวัดนิทาน");
    } finally {
      fake.restore();
    }
  });

  it("does not disconnect a caller-supplied client by default", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `<input data-openthaiid="citizenId" />`;
    const client = new ThaiID({ fields: ["citizenId"], autoReconnect: false });
    try {
      await autofillThaiID({ client, root: document.body });
      expect(client.state()).toBe("connected");
      await client.disconnect();
    } finally {
      fake.restore();
    }
  });

  it("throws when there are no targets", async () => {
    document.body.innerHTML = `<input id="unrelated" />`;
    const err = await autofillThaiID({ discover: false }).catch((e) => e);
    expect(err).toMatchObject({ code: "INVALID_REQUEST" });
    expect(err.message).toContain("คำขอกรอกข้อมูลไม่ถูกต้อง");
    expect(err.message).not.toContain("no autofill targets");
  });

  it("rejects a supplied client that is missing a required field", async () => {
    document.body.innerHTML = `<input data-openthaiid="address" />`;
    const client = new ThaiID({ fields: ["citizenId"] });
    const err = await autofillThaiID({ client }).catch((e) => e);
    expect(err).toMatchObject({ code: "FIELD_NOT_ALLOWED" });
    expect(err.message).toContain("ช่องที่ยังไม่อนุญาต");
    expect(err.message).not.toContain("approved set");
  });

  it("English Fill errors keep the same codes", async () => {
    document.body.innerHTML = `<input data-openthaiid="address" />`;
    const client = new ThaiID({ fields: ["citizenId"] });
    const err = await autofillThaiID({ client, lang: "en" }).catch((e) => e);
    expect(err).toMatchObject({ code: "FIELD_NOT_ALLOWED" });
    expect(err.message).toMatch(/field that is not allowed/i);
  });

  it("uses map + request allowlist (ERP config)", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `
      <input id="patient-cid" />
      <input id="patient-given" />
      <input id="patient-family" />
      <input id="patient-dob" type="date" />
      <select id="patient-gender">
        <option value=""></option>
        <option value="male">male</option>
        <option value="female">female</option>
      </select>
      <textarea id="patient-address"></textarea>
      <input id="patient-phone" />
    `;
    const errors: unknown[] = [];
    try {
      const out = await autofillThaiID({
        request: ["citizenId", "thaiName", "birthDate", "gender", "address"],
        map: {
          citizenId: "#patient-cid",
          "thaiName.givenName": "#patient-given",
          "thaiName.familyName": "#patient-family",
          birthDate: "#patient-dob",
          gender: "#patient-gender",
          address: "#patient-address",
        },
        onError: (err) => errors.push(err),
      });
      const open = sessionOpen(fake);
      expect(open.payload.fields).toEqual(["citizenId", "thaiName", "birthDate", "gender", "address"]);
      expect((document.getElementById("patient-cid") as HTMLInputElement).value).toBe("1234567890121");
      expect((document.getElementById("patient-given") as HTMLInputElement).value).toBe("สมชาย");
      expect((document.getElementById("patient-family") as HTMLInputElement).value).toBe("ใจดี");
      expect((document.getElementById("patient-phone") as HTMLInputElement).value).toBe("");
      expect(out.filled.map((f) => f.slot).sort()).toEqual([
        "address",
        "birthDate",
        "citizenId",
        "gender",
        "thaiName.familyName",
        "thaiName.givenName",
      ]);
      expect(errors).toEqual([]);
    } finally {
      fake.restore();
    }
  });

  it("does not request portrait unless it is on the allowlist", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `
      <input id="patient-cid" />
      <img id="patient-photo" alt="" />
    `;
    try {
      await autofillThaiID({
        request: ["citizenId"],
        map: {
          citizenId: "#patient-cid",
          portrait: "#patient-photo",
        },
        discover: false,
      });
      const open = sessionOpen(fake);
      expect(open.payload.fields).toEqual(["citizenId"]);
      expect((document.getElementById("patient-photo") as HTMLImageElement).getAttribute("src")).toBeNull();
    } finally {
      fake.restore();
    }
  });

  it("opts into portrait preview when request includes portrait", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `
      <input id="patient-cid" />
      <img id="patient-photo" alt="" />
    `;
    try {
      await autofillThaiID({
        request: ["citizenId", "portrait"],
        map: {
          citizenId: "#patient-cid",
          portrait: "#patient-photo",
        },
        discover: false,
      });
      const open = sessionOpen(fake);
      expect(open.payload.fields).toEqual(["citizenId", "portrait"]);
      expect((document.getElementById("patient-photo") as HTMLImageElement).getAttribute("src")).toBe(
        "data:image/jpeg;base64,QQ==",
      );
    } finally {
      fake.restore();
    }
  });

  it("does not request fields omitted from the allowlist", async () => {
    const fake = installFakeWebSocket();
    document.body.innerHTML = `
      <input id="patient-cid" />
      <textarea id="patient-address"></textarea>
    `;
    try {
      await autofillThaiID({
        request: ["citizenId"],
        map: {
          citizenId: "#patient-cid",
          address: "#patient-address",
        },
      });
      const open = sessionOpen(fake);
      expect(open.payload.fields).toEqual(["citizenId"]);
      expect((document.getElementById("patient-cid") as HTMLInputElement).value).toBe("1234567890121");
      expect((document.getElementById("patient-address") as HTMLTextAreaElement).value).toBe("");
    } finally {
      fake.restore();
    }
  });

  it("throws when the allowlist matches no mapped slots", async () => {
    document.body.innerHTML = `<input id="cid" />`;
    await expect(
      autofillThaiID({
        request: ["englishName"],
        map: { citizenId: "#cid" },
      }),
    ).rejects.toMatchObject({ code: "INVALID_REQUEST" });
  });

  it("calls onError then still rejects", async () => {
    document.body.innerHTML = `<input id="unrelated" />`;
    const errors: unknown[] = [];
    await expect(
      autofillThaiID({
        discover: false,
        onError: (err) => errors.push(err),
      }),
    ).rejects.toMatchObject({ code: "INVALID_REQUEST" });
    expect(errors).toHaveLength(1);
    expect(errors[0]).toMatchObject({ code: "INVALID_REQUEST" });
  });

  it("waits for the helper to return before opening a session or reading", async () => {
    const fake = installFakeWebSocket({ failFirst: 2 });
    document.body.innerHTML = `<input data-openthaiid="citizenId" />`;
    try {
      await autofillThaiID({ root: document.body, connectTimeoutMs: 5_000 });
      expect((document.querySelector("[data-openthaiid=citizenId]") as HTMLInputElement).value).toBe("1234567890121");
      expect(allSent(fake).some((m) => m.type === "bridge.probe")).toBe(true);
      expect(allSent(fake).filter((m) => m.type === "session.open")).toHaveLength(1);
      expect(allSent(fake).filter((m) => m.type === "card.read")).toHaveLength(1);
      expect(JSON.stringify(allSent(fake).filter((m) => m.type === "bridge.probe"))).not.toContain("1234567890121");
    } finally {
      fake.restore();
    }
  });
});

describe("autofill request allowlist", () => {
  it("normalizes protocol fields and rejects slots", () => {
    expect(normalizeAutofillRequest(["citizenId", "thaiName", "citizenId"])).toEqual(["citizenId", "thaiName"]);
    expect(normalizeAutofillRequest(undefined)).toBeNull();
    expect(() => normalizeAutofillRequest([])).toThrow(ThaiIDError);
    expect(() => normalizeAutofillRequest(["thaiName.givenName"])).toThrow(ThaiIDError);
    expect(normalizeAutofillRequest(["portrait"])).toEqual(["portrait"]);
    expect(() => normalizeAutofillRequest(["laserId"])).toThrow(ThaiIDError);
  });

  it("filters targets to requested protocol fields", () => {
    const filtered = filterTargetsByRequest(
      [
        { slot: "citizenId", elements: [] },
        { slot: "thaiName.givenName", elements: [] },
        { slot: "address", elements: [] },
      ],
      ["citizenId", "thaiName"],
    );
    expect(filtered.map((t) => t.slot)).toEqual(["citizenId", "thaiName.givenName"]);
  });

  it("collects map alias and applies request", () => {
    document.body.innerHTML = `<input id="cid" /><input id="name" /><textarea id="addr"></textarea>`;
    const targets = collectAutofillTargets({
      request: ["citizenId", "thaiName"],
      map: {
        citizenId: "#cid",
        thaiName: "#name",
        address: "#addr",
      },
    });
    expect(targets.map((t) => t.slot).sort()).toEqual(["citizenId", "thaiName"]);
  });

  it("allowlists discovered data-openthaiid slots", () => {
    document.body.innerHTML = `
      <form id="patient">
        <input data-openthaiid="citizenId" />
        <input data-openthaiid="thaiName" />
        <textarea data-openthaiid="address"></textarea>
        <input id="phone" />
      </form>
    `;
    const targets = collectAutofillTargets({
      root: document.getElementById("patient")!,
      request: ["citizenId", "thaiName"],
    });
    expect(targets.map((t) => t.slot).sort()).toEqual(["citizenId", "thaiName"]);
  });
});
