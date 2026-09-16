/**
 * JSON-serializable clinic / ERP config for `autofillThaiID`.
 *
 * Callbacks (`onApprovalRequired`, `onError`, `onConnectionChanged`) and
 * runtime objects (`root`, `client`, `signal`, Element map values, a
 * `dateFormat` function) are code-only: attach them in the click handler.
 */

import { SUPPORTED_FIELDS, ThaiIDError, type Field } from "./index.js";
import {
  AUTOFILL_SLOTS,
  parseSlot,
  type AutofillFieldMap,
  type AutofillSlot,
  type AutofillThaiIDOptions,
  type GenderLabels,
} from "./autofill.js";
import { staffLang, type StaffLang } from "./staffCopy.js";

export const AUTOFILL_CLINIC_CONFIG_KIND = "openthaiid.autofill-clinic" as const;

export const AUTOFILL_CLINIC_CONFIG_SCHEMA_ID =
  "https://raw.githubusercontent.com/night-sornram/openthaiid/dev/sdk/web/schema/autofill-clinic-config.schema.json";

/** Keys that belong in application code, never in a JSON config file. */
export const CODE_ONLY_AUTOFILL_OPTION_KEYS = [
  "onApprovalRequired",
  "onError",
  "onConnectionChanged",
  "root",
  "client",
  "signal",
] as const;

const META_KEYS = new Set(["$schema", "$id", "$comment", "kind"]);

const SERIALIZABLE_KEYS = new Set([
  "request",
  "map",
  "lang",
  "dateFormat",
  "dateLocale",
  "genderLabels",
  "discover",
  "endpoint",
  "connectTimeoutMs",
  "renewLeadMs",
  "disconnect",
  "attribute",
  "readerId",
  "cardSessionId",
]);

export type AutofillClinicConfigMap = Partial<Record<AutofillSlot, string>> & {
  [slot: string]: string | undefined;
};

/**
 * Drop-in JSON object for a clinic ERP Create Patient (or KYC) screen.
 * Pass the result of `parseAutofillClinicConfig` into `autofillThaiID`, then
 * attach callbacks in code.
 */
export type AutofillClinicConfig = {
  request?: Field[];
  map?: AutofillClinicConfigMap;
  /** Staff language for Fill-error messages. Default `"th"`. */
  lang?: StaffLang;
  /** `"iso"` or `"locale"` only. A formatter function is code-only. */
  dateFormat?: "iso" | "locale";
  dateLocale?: string;
  genderLabels?: GenderLabels;
  discover?: boolean;
  endpoint?: string;
  connectTimeoutMs?: number;
  renewLeadMs?: number;
  disconnect?: boolean;
  attribute?: string;
  readerId?: string;
  cardSessionId?: string;
};

function fail(message: string, code: "INVALID_REQUEST" | "UNSUPPORTED_FIELD" = "INVALID_REQUEST"): never {
  throw new ThaiIDError(code, message);
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseRequest(raw: unknown): Field[] {
  if (!Array.isArray(raw)) {
    fail("request must be an array of protocol field names");
  }
  if (raw.length === 0) {
    fail("request must list at least one field");
  }
  const seen = new Set<Field>();
  const out: Field[] = [];
  for (const item of raw) {
    if (typeof item !== "string" || !item.trim()) {
      fail(`request must be protocol fields (not slots); unsupported: ${String(item)}`, "UNSUPPORTED_FIELD");
    }
    const name = item.trim();
    if (name.includes(".")) {
      fail(`request must be protocol fields (not slots); unsupported: ${name}`, "UNSUPPORTED_FIELD");
    }
    if (!(SUPPORTED_FIELDS as readonly string[]).includes(name)) {
      fail(`request must be protocol fields (not slots); unsupported: ${name}`, "UNSUPPORTED_FIELD");
    }
    const field = name as Field;
    if (seen.has(field)) continue;
    seen.add(field);
    out.push(field);
  }
  return out;
}

function parseMap(raw: unknown): AutofillClinicConfigMap {
  if (!isPlainObject(raw)) {
    fail("map must be an object of slot → CSS selector strings");
  }
  const keys = Object.keys(raw);
  if (keys.length === 0) {
    fail("map must include at least one slot");
  }
  const map: AutofillClinicConfigMap = {};
  for (const slot of keys) {
    parseSlot(slot);
    const selector = raw[slot];
    if (typeof selector !== "string" || !selector.trim()) {
      fail(`map.${slot} must be a CSS selector string (Element targets are code-only)`);
    }
    map[slot] = selector.trim();
  }
  return map;
}

function parseGenderLabels(raw: unknown): GenderLabels {
  if (!isPlainObject(raw)) {
    fail("genderLabels must be an object with male and female strings");
  }
  const extra = Object.keys(raw).filter((k) => k !== "male" && k !== "female" && k !== "unknown");
  if (extra.length) {
    fail(`genderLabels has unknown keys: ${extra.join(", ")}`);
  }
  if (typeof raw.male !== "string" || !raw.male.trim() || typeof raw.female !== "string" || !raw.female.trim()) {
    fail("genderLabels.male and genderLabels.female must be non-empty strings");
  }
  const labels: GenderLabels = { male: raw.male.trim(), female: raw.female.trim() };
  if (raw.unknown != null) {
    if (typeof raw.unknown !== "string") {
      fail("genderLabels.unknown must be a string when present");
    }
    labels.unknown = raw.unknown;
  }
  return labels;
}

function parseTimeout(raw: unknown, key: string): number {
  if (typeof raw !== "number" || !Number.isFinite(raw) || raw < 0 || !Number.isInteger(raw)) {
    fail(`${key} must be a non-negative integer (milliseconds)`);
  }
  return raw;
}

function parseOptionalString(raw: unknown, key: string): string {
  if (typeof raw !== "string" || !raw.trim()) {
    fail(`${key} must be a non-empty string`);
  }
  return raw.trim();
}

/**
 * Parse clinic JSON (string or object) into a typed config.
 * Rejects Laser-id / unknown protocol fields, nested `request` slots,
 * callbacks, and other code-only keys.
 */
export function parseAutofillClinicConfig(input: unknown): AutofillClinicConfig {
  let value = input;
  if (typeof value === "string") {
    const text = value.replace(/^\uFEFF/, "").trim();
    try {
      value = JSON.parse(text) as unknown;
    } catch {
      fail("clinic config is not valid JSON");
    }
  }
  if (!isPlainObject(value)) {
    fail("clinic config must be a JSON object");
  }

  if (value.kind != null && value.kind !== AUTOFILL_CLINIC_CONFIG_KIND) {
    fail(`kind must be "${AUTOFILL_CLINIC_CONFIG_KIND}" when present`);
  }

  for (const key of Object.keys(value)) {
    if (META_KEYS.has(key) || SERIALIZABLE_KEYS.has(key)) continue;
    if ((CODE_ONLY_AUTOFILL_OPTION_KEYS as readonly string[]).includes(key)) {
      fail(`${key} is code-only; attach it in the click handler, not in JSON`);
    }
    if (key === "fields") {
      fail("JSON configs use map (fields is a runtime alias)");
    }
    let slotKey = false;
    try {
      parseSlot(key);
      slotKey = true;
    } catch {
      slotKey = false;
    }
    if (slotKey) {
      fail(`put selectors in map, not as top-level keys (${key})`);
    }
    fail(`unknown clinic config key: ${key}`);
  }

  const config: AutofillClinicConfig = {};

  if (value.request !== undefined) {
    config.request = parseRequest(value.request);
  }
  if (value.map !== undefined) {
    config.map = parseMap(value.map);
  }
  if (value.lang !== undefined) {
    if (typeof value.lang !== "string" || !value.lang.trim()) {
      fail("lang must be \"th\" or \"en\"");
    }
    const normalized = staffLang(value.lang);
    const raw = value.lang.trim().toLowerCase();
    if (raw !== "th" && raw !== "en" && !raw.startsWith("en-") && raw !== "th-th") {
      fail("lang must be \"th\" or \"en\"");
    }
    config.lang = normalized;
  }
  if (value.dateFormat !== undefined) {
    if (value.dateFormat !== "iso" && value.dateFormat !== "locale") {
      fail("dateFormat must be \"iso\" or \"locale\" (functions are code-only)");
    }
    config.dateFormat = value.dateFormat;
  }
  if (value.dateLocale !== undefined) {
    config.dateLocale = parseOptionalString(value.dateLocale, "dateLocale");
  }
  if (value.genderLabels !== undefined) {
    config.genderLabels = parseGenderLabels(value.genderLabels);
  }
  if (value.discover !== undefined) {
    if (typeof value.discover !== "boolean") fail("discover must be a boolean");
    config.discover = value.discover;
  }
  if (value.endpoint !== undefined) {
    config.endpoint = parseOptionalString(value.endpoint, "endpoint");
  }
  if (value.connectTimeoutMs !== undefined) {
    config.connectTimeoutMs = parseTimeout(value.connectTimeoutMs, "connectTimeoutMs");
  }
  if (value.renewLeadMs !== undefined) {
    config.renewLeadMs = parseTimeout(value.renewLeadMs, "renewLeadMs");
  }
  if (value.disconnect !== undefined) {
    if (typeof value.disconnect !== "boolean") fail("disconnect must be a boolean");
    config.disconnect = value.disconnect;
  }
  if (value.attribute !== undefined) {
    config.attribute = parseOptionalString(value.attribute, "attribute");
  }
  if (value.readerId !== undefined) {
    config.readerId = parseOptionalString(value.readerId, "readerId");
  }
  if (value.cardSessionId !== undefined) {
    config.cardSessionId = parseOptionalString(value.cardSessionId, "cardSessionId");
  }

  if (!config.request && !config.map) {
    fail("clinic config needs request and/or map");
  }

  return config;
}

function stringMap(map: AutofillFieldMap | AutofillClinicConfigMap | undefined, into: AutofillClinicConfigMap) {
  if (!map) return;
  for (const [slot, target] of Object.entries(map)) {
    if (typeof target === "string" && target.trim()) {
      parseSlot(slot);
      into[slot] = target.trim();
    }
  }
}

/**
 * Extract the JSON-safe subset of an `autofillThaiID` options object.
 * Drops callbacks, DOM nodes, and formatter functions.
 */
export function serializeAutofillClinicConfig(
  options: AutofillThaiIDOptions | AutofillClinicConfig,
): AutofillClinicConfig {
  const map: AutofillClinicConfigMap = {};
  stringMap("fields" in options ? options.fields : undefined, map);
  stringMap(options.map, map);

  const config: AutofillClinicConfig = {};
  if (options.request?.length) config.request = [...options.request];
  if (Object.keys(map).length) config.map = map;
  if (options.lang != null && String(options.lang).trim()) {
    config.lang = staffLang(String(options.lang));
  }
  if (options.dateFormat === "iso" || options.dateFormat === "locale") {
    config.dateFormat = options.dateFormat;
  }
  if (typeof options.dateLocale === "string" && options.dateLocale.trim()) {
    config.dateLocale = options.dateLocale.trim();
  }
  if (options.genderLabels) {
    config.genderLabels = parseGenderLabels(options.genderLabels);
  }
  if (typeof options.discover === "boolean") config.discover = options.discover;
  if (typeof options.endpoint === "string" && options.endpoint.trim()) {
    config.endpoint = options.endpoint.trim();
  }
  if (typeof options.connectTimeoutMs === "number") {
    config.connectTimeoutMs = parseTimeout(options.connectTimeoutMs, "connectTimeoutMs");
  }
  if (typeof options.renewLeadMs === "number") {
    config.renewLeadMs = parseTimeout(options.renewLeadMs, "renewLeadMs");
  }
  if (typeof options.disconnect === "boolean") config.disconnect = options.disconnect;
  if (typeof options.attribute === "string" && options.attribute.trim()) {
    config.attribute = options.attribute.trim();
  }
  if (typeof options.readerId === "string" && options.readerId.trim()) {
    config.readerId = options.readerId.trim();
  }
  if (typeof options.cardSessionId === "string" && options.cardSessionId.trim()) {
    config.cardSessionId = options.cardSessionId.trim();
  }
  if (!config.request && !config.map) {
    fail("nothing JSON-serializable (need request and/or map with string selectors)");
  }
  return config;
}

/** Spread a parsed clinic config into `autofillThaiID` options (no callbacks). */
export function clinicConfigToAutofillOptions(config: AutofillClinicConfig): AutofillThaiIDOptions {
  const options: AutofillThaiIDOptions = {};
  if (config.request) options.request = config.request;
  if (config.map) options.map = config.map;
  if (config.lang) options.lang = config.lang;
  if (config.dateFormat) options.dateFormat = config.dateFormat;
  if (config.dateLocale) options.dateLocale = config.dateLocale;
  if (config.genderLabels) options.genderLabels = config.genderLabels;
  if (typeof config.discover === "boolean") options.discover = config.discover;
  if (config.endpoint) options.endpoint = config.endpoint;
  if (config.connectTimeoutMs != null) options.connectTimeoutMs = config.connectTimeoutMs;
  if (config.renewLeadMs != null) options.renewLeadMs = config.renewLeadMs;
  if (typeof config.disconnect === "boolean") options.disconnect = config.disconnect;
  if (config.attribute) options.attribute = config.attribute;
  if (config.readerId) options.readerId = config.readerId;
  if (config.cardSessionId) options.cardSessionId = config.cardSessionId;
  return options;
}

export function autofillClinicConfigDocument(
  config: AutofillClinicConfig,
  schemaHref = "../../sdk/web/schema/autofill-clinic-config.schema.json",
): Record<string, unknown> {
  return {
    $schema: schemaHref,
    kind: AUTOFILL_CLINIC_CONFIG_KIND,
    ...config,
  };
}

/** Slot names advertised by the JSON Schema `map` propertyNames enum. */
export function autofillSlotNames(): readonly string[] {
  return AUTOFILL_SLOTS;
}
