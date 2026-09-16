/**
 * One-shot form autofill on top of ThaiID.
 *
 * Call from a visible user gesture (click). This is a library helper for an
 * existing ERP page — not a product to install. Origin + field Allow still
 * happens on the local helper (Manager page or terminal).
 */

import {
  SUPPORTED_FIELDS,
  ThaiID,
  ThaiIDError,
  type Portrait,
  type CardDate,
  type ConnectionState,
  type Field,
  type FieldValues,
  type Gender,
  type PersonName,
  type ReadResult,
  type ThaiAddress,
  type ThaiIDOptions,
} from "./index.js";
import { staffLang, type StaffLang } from "./staffCopy.js";
import { localizeThaiIDError } from "./staffError.js";
import { waitForThaiID } from "./probe.js";

export const AUTOFILL_ATTRIBUTE = "data-openthaiid";

type NamePrefix = "thaiName" | "englishName";
type DatePrefix = "birthDate" | "issueDate" | "expireDate";
type NamePart = "title" | "givenName" | "middleName" | "familyName" | "fullName";
type DatePart = "year" | "month" | "day";
type AddressPart =
  | "houseNo"
  | "villageNo"
  | "alley"
  | "street"
  | "subdistrict"
  | "district"
  | "province"
  | "fullAddress";

type NameSlot<P extends NamePrefix> = P | `${P}.${NamePart}`;
type DateSlot<P extends DatePrefix> = P | `${P}.${DatePart}`;
type AddressSlot = "address" | `address.${AddressPart}`;
type GenderSlot = "gender" | "gender.code" | "gender.value";
type PortraitSlot = "portrait" | "portrait.dataUrl" | "portrait.data" | "portrait.mimeType";

export type AutofillSlot =
  | "citizenId"
  | "issuer"
  | NameSlot<"thaiName">
  | NameSlot<"englishName">
  | DateSlot<"birthDate">
  | DateSlot<"issueDate">
  | DateSlot<"expireDate">
  | GenderSlot
  | AddressSlot
  | PortraitSlot;

export type AutofillTarget = string | Element | ReadonlyArray<Element> | NodeListOf<Element>;

export type AutofillFieldMap = Partial<Record<AutofillSlot, AutofillTarget>> & {
  [slot: string]: AutofillTarget | undefined;
};

export type GenderLabels = {
  male: string;
  female: string;
  unknown?: string;
};

export type AutofillFormatOptions = {
  /** Default `"iso"` (`YYYY-MM-DD`). `"locale"` uses `dateLocale` / runtime locale. */
  dateFormat?: "iso" | "locale" | ((date: CardDate) => string);
  dateLocale?: string;
  /** Default `{ male: "male", female: "female" }`. */
  genderLabels?: GenderLabels;
};

export type AutofillFilled = {
  slot: string;
  element: Element;
  value: string;
};

export type AutofillResult = {
  /** Formatted strings keyed by slot (only slots that produced a value). */
  values: Record<string, string>;
  filled: AutofillFilled[];
  /** Slots that resolved to no matching element. */
  missing: string[];
  result: ReadResult;
};

export type AutofillThaiIDOptions = AutofillFormatOptions &
  Pick<ThaiIDOptions, "endpoint" | "connectTimeoutMs" | "renewLeadMs"> & {
    /**
     * Slot → CSS selector, element, or list. Combined with any top-level
     * slot keys on this object (`citizenId: "#id"`). Prefer `map` in new
     * ERP config objects; this name stays for compatibility.
     */
    fields?: AutofillFieldMap;
    /**
     * Alias of `fields`. Slot → selector / element for this screen.
     * Clinics typically keep one `map` and vary `request`.
     */
    map?: AutofillFieldMap;
    /**
     * Allowlist of protocol fields to request from the card (subset of
     * `SUPPORTED_FIELDS`). Nested slots belong in `map`, not here.
     * The session asks only for the intersection of this list and mapped
     * or discovered slots (least privilege). Omit to request every mapped
     * slot's field.
     */
    request?: Field[];
    /** Query / attribute scan root. Default: `document`. */
    root?: ParentNode;
    /** Attribute used when discovering targets. Default: `data-openthaiid`. */
    attribute?: string;
    /**
     * Scan `data-openthaiid` (or `attribute`) on `root`.
     * Default: true when no explicit slot targets are given.
     */
    discover?: boolean;
    /** Reuse an existing client (its approved field set must cover the slots). */
    client?: ThaiID;
    /** Called when the bridge asks the operator to approve this origin. */
    onApprovalRequired?: (info: {
      approvalId: number;
      comparisonCode: string;
      fields: Field[];
    }) => void;
    onConnectionChanged?: (state: ConnectionState) => void;
    /**
     * Optional ERP chrome hook. The promise still rejects after this runs.
     */
    onError?: (err: unknown) => void;
    /**
     * Disconnect after fill. Default `true` when this helper constructed the
     * client, `false` when `client` was passed in.
     */
    disconnect?: boolean;
    /**
     * Staff language for thrown / `onError` messages. Default `"th"`.
     * Protocol `code` is unchanged. Pass `"en"` for English guidance.
     */
    lang?: StaffLang | string;
    signal?: AbortSignal;
    readerId?: string;
    cardSessionId?: string;
  } & Partial<Record<AutofillSlot, AutofillTarget>>;

const OPTION_KEYS = new Set<string>([
  "fields",
  "map",
  "request",
  "root",
  "attribute",
  "discover",
  "client",
  "endpoint",
  "connectTimeoutMs",
  "renewLeadMs",
  "onApprovalRequired",
  "onConnectionChanged",
  "onError",
  "disconnect",
  "lang",
  "dateFormat",
  "dateLocale",
  "genderLabels",
  "signal",
  "readerId",
  "cardSessionId",
]);

const NAME_PART_LIST = ["title", "givenName", "middleName", "familyName", "fullName"] as const;
const DATE_PART_LIST = ["year", "month", "day"] as const;
const ADDRESS_PART_LIST = [
  "houseNo",
  "villageNo",
  "alley",
  "street",
  "subdistrict",
  "district",
  "province",
  "fullAddress",
] as const;
const GENDER_PART_LIST = ["code", "value"] as const;
const PORTRAIT_PART_LIST = ["dataUrl", "data", "mimeType"] as const;

const NAME_PARTS = new Set<string>(NAME_PART_LIST);
const DATE_PARTS = new Set<string>(DATE_PART_LIST);
const ADDRESS_PARTS = new Set<string>(ADDRESS_PART_LIST);
const GENDER_PARTS = new Set<string>(GENDER_PART_LIST);

function expandSlots<P extends string, Parts extends readonly string[]>(
  prefix: P,
  parts: Parts,
): Array<P | `${P}.${Parts[number]}`> {
  return [prefix, ...parts.map((part) => `${prefix}.${part}` as `${P}.${Parts[number]}`)];
}

/** Every JSON-safe autofill slot name (protocol field or nested path). */
export const AUTOFILL_SLOTS: readonly AutofillSlot[] = [
  "citizenId",
  "issuer",
  ...expandSlots("thaiName", NAME_PART_LIST),
  ...expandSlots("englishName", NAME_PART_LIST),
  ...expandSlots("birthDate", DATE_PART_LIST),
  ...expandSlots("issueDate", DATE_PART_LIST),
  ...expandSlots("expireDate", DATE_PART_LIST),
  ...expandSlots("gender", GENDER_PART_LIST),
  ...expandSlots("address", ADDRESS_PART_LIST),
  ...expandSlots("portrait", PORTRAIT_PART_LIST),
];

const DEFAULT_GENDER_LABELS: GenderLabels = { male: "male", female: "female" };

type ParsedSlot = { field: Field; path: string | null };

/**
 * Map autofill slots to the protocol field set (least privilege).
 */
export function slotsToFields(slots: Iterable<string>): Field[] {
  const seen = new Set<Field>();
  const out: Field[] = [];
  for (const slot of slots) {
    const { field } = parseSlot(slot);
    if (seen.has(field)) continue;
    seen.add(field);
    out.push(field);
  }
  return out;
}

export function parseSlot(slot: string): ParsedSlot {
  const trimmed = slot.trim();
  if (!trimmed) {
    throw new ThaiIDError("UNSUPPORTED_FIELD", "empty autofill slot");
  }
  const dot = trimmed.indexOf(".");
  const head = (dot === -1 ? trimmed : trimmed.slice(0, dot)) as Field;
  const path = dot === -1 ? null : trimmed.slice(dot + 1);
  if (!SUPPORTED_FIELDS.includes(head)) {
    throw new ThaiIDError("UNSUPPORTED_FIELD", `unsupported autofill slot: ${trimmed}`);
  }
  if (path === null || path === "") {
    if (path === "") {
      throw new ThaiIDError("UNSUPPORTED_FIELD", `unsupported autofill slot: ${trimmed}`);
    }
    return { field: head, path: null };
  }
  if (!isKnownPath(head, path)) {
    throw new ThaiIDError("UNSUPPORTED_FIELD", `unsupported autofill slot: ${trimmed}`);
  }
  return { field: head, path };
}

function isKnownPath(field: Field, path: string): boolean {
  switch (field) {
    case "thaiName":
    case "englishName":
      return NAME_PARTS.has(path);
    case "birthDate":
    case "issueDate":
    case "expireDate":
      return DATE_PARTS.has(path);
    case "address":
      return ADDRESS_PARTS.has(path);
    case "gender":
      return GENDER_PARTS.has(path);
    case "portrait":
      return (PORTRAIT_PART_LIST as readonly string[]).includes(path);
    default:
      return false;
  }
}

export function formatCardDateIso(date: CardDate): string {
  if (date.year == null) return "";
  const y = String(date.year).padStart(4, "0");
  if (date.month == null) return y;
  const m = String(date.month).padStart(2, "0");
  if (date.day == null) return `${y}-${m}`;
  return `${y}-${m}-${String(date.day).padStart(2, "0")}`;
}

function formatCardDateLocale(date: CardDate, locale?: string): string {
  if (date.year == null || date.month == null || date.day == null) {
    return formatCardDateIso(date);
  }
  const d = new Date(Date.UTC(date.year, date.month - 1, date.day));
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "medium" }).format(d);
}

function formatDate(date: CardDate, options: AutofillFormatOptions, el?: Element): string {
  const dateInput = el instanceof HTMLInputElement && el.type === "date";
  if (dateInput) {
    if (date.year == null || date.month == null || date.day == null) return "";
    return `${String(date.year).padStart(4, "0")}-${String(date.month).padStart(2, "0")}-${String(date.day).padStart(2, "0")}`;
  }
  if (typeof options.dateFormat === "function") {
    return options.dateFormat(date) ?? "";
  }
  if (options.dateFormat === "locale") {
    return formatCardDateLocale(date, options.dateLocale);
  }
  return formatCardDateIso(date);
}

function genderLabel(gender: Gender, options: AutofillFormatOptions): string {
  const labels = options.genderLabels ?? DEFAULT_GENDER_LABELS;
  if (gender.value === "male") return labels.male;
  if (gender.value === "female") return labels.female;
  return labels.unknown ?? "";
}

function namePart(name: PersonName, path: string | null): string {
  if (path === null || path === "fullName") return name.fullName ?? "";
  const v = name[path as NamePart];
  return v ?? "";
}

function addressPart(address: ThaiAddress, path: string | null): string {
  if (path === null || path === "fullAddress") return address.fullAddress ?? "";
  const v = address[path as AddressPart];
  return v ?? "";
}

function datePart(date: CardDate, path: string | null, options: AutofillFormatOptions, el?: Element): string {
  if (path === null) return formatDate(date, options, el);
  const v = date[path as DatePart];
  return v == null ? "" : String(v);
}

function digitsOnly(value: string): string {
  return value.replace(/\D/g, "");
}

/**
 * Map a structured card field onto a string suitable for an `<input>`.
 * Returns `""` when the value is absent; callers skip empty fills.
 */
export function formatAutofillValue(
  fields: FieldValues,
  slot: string,
  options: AutofillFormatOptions = {},
  el?: Element,
): string {
  const { field, path } = parseSlot(slot);
  const raw = fields[field];
  if (raw == null) return "";

  switch (field) {
    case "citizenId":
      return digitsOnly(String(raw));
    case "issuer":
      return String(raw);
    case "thaiName":
    case "englishName":
      return namePart(raw as PersonName, path);
    case "birthDate":
    case "issueDate":
    case "expireDate":
      return datePart(raw as CardDate, path, options, el);
    case "gender": {
      const g = raw as Gender;
      if (path === "code") return g.code ?? "";
      if (path === "value") return g.value ?? "";
      return genderLabel(g, options);
    }
    case "address":
      return addressPart(raw as ThaiAddress, path);
    case "portrait": {
      const p = raw as Portrait;
      if (path === "data") return p.data ?? "";
      if (path === "mimeType") return p.mimeType ?? "";
      if (!p.data) return "";
      return `data:${p.mimeType || "image/jpeg"};base64,${p.data}`;
    }
    default:
      return "";
  }
}

export type AutofillTargetList = Array<{ slot: string; elements: Element[] }>;

function isTargetValue(v: unknown): v is AutofillTarget {
  if (typeof v === "string" && v.length > 0) return true;
  if (typeof Element !== "undefined" && v instanceof Element) return true;
  if (Array.isArray(v) && v.every((x) => typeof Element !== "undefined" && x instanceof Element)) return true;
  if (typeof NodeList !== "undefined" && v instanceof NodeList) return true;
  return false;
}

function resolveTarget(target: AutofillTarget, root: ParentNode): Element[] {
  if (typeof target === "string") {
    try {
      return Array.from(root.querySelectorAll(target));
    } catch {
      throw new ThaiIDError("INVALID_REQUEST", `invalid autofill selector: ${target}`);
    }
  }
  if (typeof Element !== "undefined" && target instanceof Element) {
    return [target];
  }
  return Array.from(target as ArrayLike<Element>);
}

function pushTargets(into: Map<string, Element[]>, slot: string, elements: Element[]) {
  parseSlot(slot); // validate
  const existing = into.get(slot);
  if (existing) {
    existing.push(...elements);
    return;
  }
  into.set(slot, [...elements]);
}

/**
 * Find `[data-openthaiid]` (or a custom attribute) under `root`.
 */
export function discoverAutofillTargets(
  root: ParentNode = document,
  attribute = AUTOFILL_ATTRIBUTE,
): AutofillTargetList {
  const attr = attribute.startsWith("data-") ? attribute : `data-${attribute}`;
  const selector = `[${attr}]`;
  const grouped = new Map<string, Element[]>();
  for (const el of Array.from(root.querySelectorAll(selector))) {
    const slot = el.getAttribute(attr)?.trim() ?? "";
    if (!slot) continue;
    pushTargets(grouped, slot, [el]);
  }
  return [...grouped.entries()].map(([slot, elements]) => ({ slot, elements }));
}

/**
 * Validate `request` as protocol field names (not nested slots).
 * Returns `null` when the caller omitted the allowlist.
 */
export function normalizeAutofillRequest(request: readonly string[] | undefined): Field[] | null {
  if (request == null) return null;
  if (request.length === 0) {
    throw new ThaiIDError("INVALID_REQUEST", "request must list at least one field");
  }
  const seen = new Set<Field>();
  const out: Field[] = [];
  for (const raw of request) {
    const name = typeof raw === "string" ? raw.trim() : "";
    if (!name || name.includes(".") || !SUPPORTED_FIELDS.includes(name as Field)) {
      throw new ThaiIDError(
        "UNSUPPORTED_FIELD",
        `request must be protocol fields (not slots); unsupported: ${String(raw)}`,
      );
    }
    const field = name as Field;
    if (seen.has(field)) continue;
    seen.add(field);
    out.push(field);
  }
  return out;
}

export function filterTargetsByRequest(
  targets: AutofillTargetList,
  request: readonly Field[] | null,
): AutofillTargetList {
  if (request == null) return targets;
  const allowed = new Set<Field>(request);
  return targets.filter((t) => allowed.has(parseSlot(t.slot).field));
}

export function collectAutofillTargets(options: AutofillThaiIDOptions = {}): AutofillTargetList {
  if (typeof document === "undefined") {
    throw new ThaiIDError("CONNECTION_FAILED", "autofillThaiID() requires a browser environment");
  }
  const root = options.root ?? document;
  const grouped = new Map<string, Element[]>();
  const request = normalizeAutofillRequest(options.request);

  const addMap = (map: AutofillFieldMap | undefined) => {
    if (!map) return;
    for (const [slot, target] of Object.entries(map)) {
      if (target == null) continue;
      pushTargets(grouped, slot, resolveTarget(target, root));
    }
  };

  addMap(options.fields);
  addMap(options.map);

  for (const [key, value] of Object.entries(options)) {
    if (OPTION_KEYS.has(key)) continue;
    if (!isTargetValue(value)) continue;
    pushTargets(grouped, key, resolveTarget(value, root));
  }

  const hasExplicit = grouped.size > 0;
  const discover = options.discover ?? !hasExplicit;
  if (discover) {
    for (const { slot, elements } of discoverAutofillTargets(root, options.attribute ?? AUTOFILL_ATTRIBUTE)) {
      pushTargets(grouped, slot, elements);
    }
  }

  const collected = [...grouped.entries()].map(([slot, elements]) => ({
    slot,
    elements: uniqueElements(elements),
  }));
  return filterTargetsByRequest(collected, request);
}

function uniqueElements(elements: Element[]): Element[] {
  const seen = new Set<Element>();
  const out: Element[] = [];
  for (const el of elements) {
    if (seen.has(el)) continue;
    seen.add(el);
    out.push(el);
  }
  return out;
}

function setControlValue(el: Element, value: string): boolean {
  if (el instanceof HTMLImageElement) {
    el.src = value;
    return true;
  }
  if (el instanceof HTMLInputElement) {
    if (el.type === "file") return false;
    if (el.type === "checkbox" || el.type === "radio") {
      el.checked = el.value === value;
      dispatchChange(el);
      return true;
    }
    if (el.type === "date" && !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
      return false;
    }
    el.value = value;
    dispatchChange(el);
    return true;
  }
  if (el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) {
    el.value = value;
    dispatchChange(el);
    return true;
  }
  return false;
}

function dispatchChange(el: Element) {
  el.dispatchEvent(new Event("input", { bubbles: true }));
  el.dispatchEvent(new Event("change", { bubbles: true }));
}

/**
 * Write formatted values into resolved targets. Empty values are skipped so
 * missing card fields do not wipe user input.
 */
export function applyAutofillValues(
  targets: AutofillTargetList,
  fields: FieldValues,
  options: AutofillFormatOptions = {},
): { values: Record<string, string>; filled: AutofillFilled[]; missing: string[] } {
  const values: Record<string, string> = {};
  const filled: AutofillFilled[] = [];
  const missing: string[] = [];

  for (const { slot, elements } of targets) {
    if (elements.length === 0) {
      missing.push(slot);
      continue;
    }
    for (const el of elements) {
      const value = formatAutofillValue(fields, slot, options, el);
      if (!value) continue;
      values[slot] = value;
      if (setControlValue(el, value)) {
        filled.push({ slot, element: el, value });
      }
    }
  }

  return { values, filled, missing };
}

/**
 * Connect (from the current user gesture), wait for approval, read, fill
 * matching inputs, then optionally disconnect.
 */
export async function autofillThaiID(options: AutofillThaiIDOptions = {}): Promise<AutofillResult> {
  if (typeof document === "undefined") {
    throw new ThaiIDError("CONNECTION_FAILED", "autofillThaiID() requires a browser environment");
  }

  try {
    const targets = collectAutofillTargets(options);
    if (targets.length === 0) {
      const hint = options.request
        ? "no autofill targets after applying request allowlist (map a slot for those fields, or widen request)"
        : `no autofill targets (pass map/fields or add ${options.attribute ?? AUTOFILL_ATTRIBUTE} attributes)`;
      throw new ThaiIDError("INVALID_REQUEST", hint);
    }

    const requested = slotsToFields(targets.map((t) => t.slot));
    const ownsClient = !options.client;
    const client =
      options.client ??
      new ThaiID({
        fields: requested,
        endpoint: options.endpoint,
        connectTimeoutMs: options.connectTimeoutMs,
        renewLeadMs: options.renewLeadMs,
        autoReconnect: false,
      });

    if (!ownsClient) {
      for (const f of requested) {
        if (!client.fields.includes(f)) {
          throw new ThaiIDError("FIELD_NOT_ALLOWED", `field ${f} is not in the approved set`);
        }
      }
    }

    const offs: Array<() => void> = [];
    if (options.onApprovalRequired) {
      offs.push(client.on("approvalRequired", options.onApprovalRequired));
    }
    if (options.onConnectionChanged) {
      offs.push(client.on("connectionChanged", options.onConnectionChanged));
    }

    const shouldDisconnect = options.disconnect ?? ownsClient;
    try {
      if (ownsClient || client.state() !== "connected") {
        options.onConnectionChanged?.("connecting");
        const helper = await waitForThaiID({
          endpoint: options.endpoint ?? (ownsClient ? undefined : client.endpoint),
          signal: options.signal,
          lang: options.lang,
          timeoutMs: options.connectTimeoutMs ?? 30_000,
        });
        if (!helper.reachable) {
          throw helper.error ?? new ThaiIDError("BRIDGE_UNAVAILABLE", "helper is not reachable", true);
        }
      }
      await client.connect({ signal: options.signal });
      const result = await client.read({
        fields: requested,
        readerId: options.readerId,
        cardSessionId: options.cardSessionId,
        signal: options.signal,
      });
      const applied = applyAutofillValues(targets, result.fields, options);
      return { ...applied, result };
    } finally {
      for (const off of offs) off();
      if (shouldDisconnect) {
        await client.disconnect();
      }
    }
  } catch (err) {
    const localized = localizeThaiIDError(err, staffLang(options.lang));
    options.onError?.(localized);
    throw localized;
  }
}
