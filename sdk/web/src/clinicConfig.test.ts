import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  AUTOFILL_CLINIC_CONFIG_KIND,
  AUTOFILL_CLINIC_CONFIG_SCHEMA_ID,
  AUTOFILL_SLOTS,
  CODE_ONLY_AUTOFILL_OPTION_KEYS,
  DEFAULT_FIELDS,
  SUPPORTED_FIELDS,
  ThaiIDError,
  autofillClinicConfigDocument,
  clinicConfigToAutofillOptions,
  parseAutofillClinicConfig,
  parseSlot,
  serializeAutofillClinicConfig,
} from "./index";
import {
  CLINIC_PATIENT_MAP,
  CLINIC_PATIENT_REQUEST,
} from "../../../examples/clinic-patient/settings.js";

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(here, "../../..");

function readJson(relative: string): unknown {
  return JSON.parse(readFileSync(join(repoRoot, relative), "utf8"));
}

const schema = readJson("sdk/web/schema/autofill-clinic-config.schema.json") as {
  $id: string;
  properties: {
    request: { items: { enum: string[] } };
    map: { propertyNames: { enum: string[] } };
  };
  additionalProperties: boolean;
};

const kycMinimal = readJson("examples/clinic-patient/kyc-minimal.json");
const clinicFull = readJson("examples/clinic-patient/clinic-full.json");
const clinicHttps = readJson("examples/clinic-patient/clinic-https.json") as {
  $comment?: string;
  preferLoopbackTls?: boolean;
};

describe("AUTOFILL_SLOTS", () => {
  it("parses every advertised slot", () => {
    for (const slot of AUTOFILL_SLOTS) {
      expect(() => parseSlot(slot)).not.toThrow();
    }
    expect(AUTOFILL_SLOTS).toHaveLength(42);
  });

  it("matches the JSON Schema map propertyNames enum", () => {
    expect(schema.properties.map.propertyNames.enum).toEqual([...AUTOFILL_SLOTS]);
  });

  it("matches the JSON Schema request enum to SUPPORTED_FIELDS", () => {
    expect(schema.properties.request.items.enum).toEqual([...SUPPORTED_FIELDS]);
  });
});

describe("parseAutofillClinicConfig", () => {
  it("parses the KYC-minimal sample", () => {
    const cfg = parseAutofillClinicConfig(kycMinimal);
    expect(cfg.request).toEqual([...DEFAULT_FIELDS]);
    expect(cfg.map).toEqual({
      citizenId: "#patient-cid",
      "thaiName.givenName": "#patient-given",
      "thaiName.familyName": "#patient-family",
      birthDate: "#patient-dob",
    });
    expect(cfg.discover).toBe(false);
    expect(cfg.lang).toBe("th");
    expect(cfg.dateFormat).toBe("iso");
  });

  it("parses the fuller clinic sample and matches settings.js map/request", () => {
    const cfg = parseAutofillClinicConfig(clinicFull);
    expect(cfg.request).toEqual([...CLINIC_PATIENT_REQUEST]);
    expect(cfg.map).toEqual({ ...CLINIC_PATIENT_MAP });
    expect(cfg.genderLabels).toEqual({ male: "ชาย", female: "หญิง" });
  });

  it("parses preferLoopbackTls for https ERP pages", () => {
    const cfg = parseAutofillClinicConfig({
      request: ["citizenId"],
      map: { citizenId: "#cid" },
      preferLoopbackTls: true,
    });
    expect(cfg.preferLoopbackTls).toBe(true);
    expect(clinicConfigToAutofillOptions(cfg).preferLoopbackTls).toBe(true);
    expect(serializeAutofillClinicConfig(cfg).preferLoopbackTls).toBe(true);
  });

  it("parses the https ERP sample (preferLoopbackTls, no secrets)", () => {
    const cfg = parseAutofillClinicConfig(clinicHttps);
    expect(cfg.preferLoopbackTls).toBe(true);
    expect(cfg.endpoint).toBeUndefined();
    expect(cfg.request).toEqual([...DEFAULT_FIELDS]);
    expect(cfg.map).toEqual({
      citizenId: "#patient-cid",
      "thaiName.givenName": "#patient-given",
      "thaiName.familyName": "#patient-family",
      birthDate: "#patient-dob",
    });
    expect(clinicConfigToAutofillOptions(cfg).preferLoopbackTls).toBe(true);
    expect(clinicHttps.$comment).toContain("docs/loopback-tls.md");
    expect(clinicHttps.$comment).toContain("docs/compatibility.md");
    expect(clinicHttps.$comment).toContain("wss://127.0.0.1:17892/v1/ws");
  });

  it("parses a JSON string", () => {
    const cfg = parseAutofillClinicConfig(JSON.stringify(kycMinimal));
    expect(cfg.request).toEqual(["citizenId", "thaiName", "birthDate"]);
  });

  it("rejects Laser-id / unknown protocol fields", () => {
    const err = (() => {
      try {
        parseAutofillClinicConfig({
          request: ["citizenId", "laserId"],
          map: { citizenId: "#cid" },
        });
      } catch (e) {
        return e;
      }
    })();
    expect(err).toBeInstanceOf(ThaiIDError);
    expect(err).toMatchObject({ code: "UNSUPPORTED_FIELD" });
    expect((err as ThaiIDError).message).toContain("laserId");
    expect((err as ThaiIDError).message).not.toMatch(/laser[- ]?id layout/i);
  });

  it("rejects nested slots in request", () => {
    expect(() =>
      parseAutofillClinicConfig({
        request: ["thaiName.givenName"],
        map: { "thaiName.givenName": "#given" },
      }),
    ).toThrow(/not slots/);
  });

  it("rejects callbacks as code-only keys", () => {
    for (const key of CODE_ONLY_AUTOFILL_OPTION_KEYS) {
      expect(() =>
        parseAutofillClinicConfig({
          request: ["citizenId"],
          map: { citizenId: "#cid" },
          [key]: "nope",
        }),
      ).toThrow(/code-only/);
    }
  });

  it("rejects top-level selector keys (use map)", () => {
    expect(() =>
      parseAutofillClinicConfig({
        request: ["citizenId"],
        citizenId: "#cid",
      }),
    ).toThrow(/put selectors in map/);
  });

  it("rejects empty request and empty map", () => {
    expect(() => parseAutofillClinicConfig({ request: [] })).toThrow(/at least one field/);
    expect(() => parseAutofillClinicConfig({ map: {} })).toThrow(/at least one slot/);
    expect(() => parseAutofillClinicConfig({ lang: "th" })).toThrow(/request and\/or map/);
  });

  it("rejects non-string map values", () => {
    expect(() =>
      parseAutofillClinicConfig({
        map: { citizenId: { selector: "#cid" } },
      }),
    ).toThrow(/CSS selector string/);
  });

  it("round-trips through serialize and document helper", () => {
    const parsed = parseAutofillClinicConfig(clinicFull);
    const serialized = serializeAutofillClinicConfig(parsed);
    expect(serialized).toEqual(parsed);
    const doc = autofillClinicConfigDocument(serialized);
    expect(doc.kind).toBe(AUTOFILL_CLINIC_CONFIG_KIND);
    expect(doc.$schema).toContain("autofill-clinic-config.schema.json");
    expect(parseAutofillClinicConfig(doc)).toEqual(parsed);
  });

  it("clinicConfigToAutofillOptions drops nothing needed for autofillThaiID", () => {
    const cfg = parseAutofillClinicConfig(kycMinimal);
    const options = clinicConfigToAutofillOptions(cfg);
    expect(options.request).toEqual(cfg.request);
    expect(options.map).toEqual(cfg.map);
    expect(options.lang).toBe("th");
    expect(options.onApprovalRequired).toBeUndefined();
  });

  it("serialize drops callbacks and keeps string selectors", () => {
    const out = serializeAutofillClinicConfig({
      request: ["citizenId", "thaiName"],
      map: { citizenId: "#cid", thaiName: "#name" },
      onError: () => undefined,
      onApprovalRequired: () => undefined,
      dateFormat: "iso",
      lang: "en",
    });
    expect(out).toEqual({
      request: ["citizenId", "thaiName"],
      map: { citizenId: "#cid", thaiName: "#name" },
      dateFormat: "iso",
      lang: "en",
    });
  });
});

describe("clinic config samples", () => {
  it("schema forbids additional properties", () => {
    expect(schema.additionalProperties).toBe(false);
    expect(schema.$id).toBe(AUTOFILL_CLINIC_CONFIG_SCHEMA_ID);
  });

  it("do not embed personal field values", () => {
    for (const sample of [kycMinimal, clinicFull, clinicHttps]) {
      const text = JSON.stringify(sample);
      expect(text).not.toMatch(/\d{13}/);
      expect(text).not.toContain("สมชาย");
      expect(text).not.toContain("1234567890121");
      expect(text).not.toContain("data:image");
    }
  });
});
