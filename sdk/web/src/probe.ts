/**
 * Library helpers: check the local reader from an ERP page without a
 * separate OpenThaiID install. Does not read personal card fields.
 */

import { ThaiIDError, type ReaderView } from "./index.js";
import { formatStaffError, staffCopy, staffLang, type StaffLang } from "./staffCopy.js";
import { localizeThaiIDError } from "./staffError.js";

export const DEFAULT_ENDPOINT = "ws://127.0.0.1:17891/v1/ws";
const SUBPROTOCOL = "openthaiid.v1";

export type ProbeBackend = {
  name: string;
  status: "available" | "unavailable" | "stalled" | string;
};

export type ProbeReader = {
  readerId: string;
  name: string;
  presence: ReaderView["presence"];
  selected?: boolean;
};

export type ProbeResult = {
  /** True when the local helper accepted the WebSocket. */
  reachable: boolean;
  version?: string;
  instanceId?: string;
  synthetic?: boolean;
  backend?: ProbeBackend;
  readers?: ProbeReader[];
  managerUrl?: string;
  wsUrl?: string;
  error?: ThaiIDError;
};

export type ProbeThaiIDOptions = {
  endpoint?: string;
  signal?: AbortSignal;
  timeoutMs?: number;
  /** Staff language for `error.message`. Default `"th"`. Codes stay unchanged. */
  lang?: StaffLang | string;
};

/** Derive the loopback manager URL from a WebSocket endpoint. */
export function managerUrlFromEndpoint(endpoint = DEFAULT_ENDPOINT): string {
  try {
    const u = new URL(endpoint);
    const http = u.protocol === "wss:" ? "https:" : "http:";
    return `${http}//${u.host}/`;
  } catch {
    return "http://127.0.0.1:17891/";
  }
}

function probeError(
  code: ThaiIDError["code"],
  lang?: StaffLang | string | null,
  retryable = false,
): ThaiIDError {
  return new ThaiIDError(code, formatStaffError({ code }, lang), retryable);
}

/**
 * Ask the local helper whether it is running and what the reader looks like.
 *
 * Call from the ERP page (a user gesture helps with local-network permission).
 * This does not request card fields and does not skip origin approval for reads.
 * SSR-safe: returns `{ reachable: false }` when `WebSocket` is missing.
 */
export async function probeThaiID(options: ProbeThaiIDOptions = {}): Promise<ProbeResult> {
  const lang = staffLang(options.lang);
  if (options.signal?.aborted) {
    return {
      reachable: false,
      error: probeError("CANCELLED", lang),
    };
  }
  if (typeof WebSocket === "undefined") {
    return {
      reachable: false,
      error: probeError("CONNECTION_FAILED", lang),
    };
  }

  const endpoint = options.endpoint ?? DEFAULT_ENDPOINT;
  const timeoutMs = options.timeoutMs ?? 8000;

  return new Promise<ProbeResult>((resolve) => {
    let settled = false;
    let ws: WebSocket | null = null;
    const finish = (result: ProbeResult) => {
      if (settled) return;
      settled = true;
      options.signal?.removeEventListener("abort", onAbort);
      clearTimeout(timer);
      if (ws) {
        try {
          ws.onopen = null;
          ws.onerror = null;
          ws.onmessage = null;
          ws.onclose = null;
          ws.close();
        } catch {
          // ignore
        }
      }
      resolve(result);
    };
    const onAbort = () => {
      finish({ reachable: false, error: probeError("CANCELLED", lang) });
    };
    options.signal?.addEventListener("abort", onAbort, { once: true });
    if (options.signal?.aborted) {
      onAbort();
      return;
    }

    const timer = setTimeout(() => {
      finish({
        reachable: false,
        error: probeError("BRIDGE_UNAVAILABLE", lang, true),
      });
    }, timeoutMs);

    try {
      ws = new WebSocket(endpoint, [SUBPROTOCOL]);
    } catch {
      finish({
        reachable: false,
        error: probeError("CONNECTION_FAILED", lang, true),
      });
      return;
    }

    ws.onerror = () => {
      finish({
        reachable: false,
        error: probeError("BRIDGE_UNAVAILABLE", lang, true),
      });
    };
    ws.onopen = () => {
      try {
        ws?.send(JSON.stringify({ v: 1, id: 1, type: "bridge.probe" }));
      } catch {
        finish({
          reachable: false,
          error: probeError("CONNECTION_FAILED", lang, true),
        });
      }
    };
    ws.onmessage = (ev) => {
      let msg: { id?: number; error?: { code?: string; message?: string }; result?: Record<string, unknown> };
      try {
        msg = JSON.parse(typeof ev.data === "string" ? ev.data : String(ev.data));
      } catch {
        return;
      }
      if (msg.id !== 1) return;
      if (msg.error) {
        finish({
          reachable: true,
          error: localizeThaiIDError(
            new ThaiIDError(
              (msg.error.code as ThaiIDError["code"]) ?? "CONNECTION_FAILED",
              msg.error.message ?? "probe failed",
            ),
            lang,
          ) as ThaiIDError,
        });
        return;
      }
      const r = msg.result ?? {};
      const backend = r.backend as ProbeBackend | undefined;
      const readers = Array.isArray(r.readers) ? (r.readers as ProbeReader[]) : [];
      finish({
        reachable: true,
        version: typeof r.version === "string" ? r.version : undefined,
        instanceId: typeof r.instanceId === "string" ? r.instanceId : undefined,
        synthetic: Boolean(r.synthetic),
        backend,
        readers,
        managerUrl: typeof r.managerUrl === "string" ? r.managerUrl : managerUrlFromEndpoint(endpoint),
        wsUrl: typeof r.wsUrl === "string" ? r.wsUrl : endpoint,
      });
    };
  });
}

export type StatusSummary = {
  ready: boolean;
  headline: string;
  helper: string;
  reader: string;
  card: string;
  next: string;
};

/** Plain-language helper / reader / card state. Default Thai; pass `"en"` for English. */
export function summarizeProbe(probe: ProbeResult, lang?: StaffLang | string | null): StatusSummary {
  const c = staffCopy(lang);
  if (!probe.reachable) {
    const cancelled = probe.error?.code === "CANCELLED";
    return {
      ready: false,
      headline: cancelled ? c.cancelled : c.notReady,
      helper: cancelled ? c.cancelled + (c.lang === "th" ? "" : ".") : c.helperOff,
      reader: c.readerUnknown,
      card: c.cardUnknown,
      next: cancelled ? "" : c.startHelper,
    };
  }
  const readers = probe.readers ?? [];
  const backend = probe.backend?.status ?? "unknown";
  if (backend === "unavailable" || backend === "stalled") {
    return {
      ready: false,
      headline: c.notReady,
      helper: c.helperOn,
      reader: backend === "stalled" ? c.serviceStalled : c.serviceOff,
      card: c.cardUntilService,
      next: c.fixService,
    };
  }
  if (readers.length === 0) {
    return {
      ready: false,
      headline: c.notReady,
      helper: c.helperOn,
      reader: c.noReader,
      card: c.insertAfterReader,
      next: c.plugReader,
    };
  }
  const selected = readers.find((r) => r.selected) ?? readers[0];
  const present = readers.some((r) => r.presence === "present");
  const unsupported = readers.some((r) => r.presence === "unsupported");
  const defaultName = c.lang === "th" ? "เครื่องอ่านบัตร" : "card reader";
  const readerLine = c.readerConnected(selected?.name ?? defaultName);
  if (unsupported && !present) {
    return {
      ready: false,
      headline: c.notReady,
      helper: c.helperOn,
      reader: readerLine,
      card: c.unsupportedCard,
      next: c.insertThaiID,
    };
  }
  if (!present) {
    return {
      ready: false,
      headline: c.notReady,
      helper: c.helperOn,
      reader: readerLine,
      card: c.insertCard,
      next: c.insertThenFill,
    };
  }
  return {
    ready: true,
    headline: probe.synthetic ? c.readyPractice : c.ready,
    helper: c.helperOn,
    reader: readerLine,
    card: c.cardInserted,
    next: c.clickFill,
  };
}

export type MountThaiIDStatusOptions = ProbeThaiIDOptions & {
  /** Poll interval. Default 2000ms. */
  intervalMs?: number;
  /**
   * Default `user`: helper / reader / card in plain language.
   * `advanced` also shows backend, version, and the local helper link.
   */
  mode?: "user" | "advanced";
  /**
   * Staff language. Default `"th"`. Pass `"en"` for English.
   * Technician chrome stays English when `mode` is `advanced`.
   */
  lang?: StaffLang | string;
};

/**
 * Drop-in reader status into an existing ERP page. No personal field values.
 * Default copy is for clinic staff. Returns a stop function.
 */
export function mountThaiIDStatus(root: Element, options: MountThaiIDStatusOptions = {}): () => void {
  const intervalMs = options.intervalMs ?? 2000;
  const mode = options.mode ?? "user";
  const lang = staffLang(options.lang);
  const copy = staffCopy(lang);
  let stopped = false;
  let timer: ReturnType<typeof setInterval> | null = null;

  const render = (probe: ProbeResult) => {
    const summary = summarizeProbe(probe, lang);
    root.replaceChildren();
    const box = document.createElement("div");
    box.setAttribute("data-openthaiid-status", "1");
    box.setAttribute("data-openthaiid-status-mode", mode);
    box.setAttribute("data-openthaiid-lang", lang);
    box.style.fontFamily = "system-ui, sans-serif";
    box.style.fontSize = "0.95rem";
    box.style.lineHeight = "1.4";

    const headline = document.createElement("p");
    headline.style.margin = "0 0 0.35rem";
    headline.style.fontWeight = "700";
    headline.style.fontSize = "1.05rem";
    headline.textContent = summary.headline;
    box.append(headline);

    const next = document.createElement("p");
    next.style.margin = "0 0 0.55rem";
    next.textContent = summary.next;
    box.append(next);

    const ul = document.createElement("ul");
    ul.style.margin = "0";
    ul.style.paddingLeft = "1.15rem";
    for (const text of [summary.helper, summary.reader, summary.card]) {
      const li = document.createElement("li");
      li.textContent = text;
      ul.append(li);
    }
    box.append(ul);

    const pcsc = probe.backend?.status ?? "unknown";
    const ver = probe.version ? `v${probe.version}` : "";
    if (mode === "advanced" && probe.reachable) {
      const details = document.createElement("p");
      details.style.margin = "0.55rem 0 0";
      details.style.opacity = "0.78";
      details.style.fontSize = "0.85rem";
      details.textContent = `Technician: card service ${pcsc}${ver ? ` · ${ver}` : ""}${
        probe.synthetic ? " · practice reader" : ""
      }`;
      box.append(details);
      if (probe.managerUrl) {
        const link = document.createElement("a");
        link.href = `${probe.managerUrl}#technician`;
        link.textContent = "Open technician tools on this PC";
        link.style.display = "inline-block";
        link.style.marginTop = "0.35rem";
        link.style.fontSize = "0.85rem";
        box.append(link);
      }
    } else if (mode === "user") {
      const details = document.createElement("details");
      details.style.marginTop = "0.55rem";
      const summaryEl = document.createElement("summary");
      summaryEl.textContent = copy.technicianDetails;
      summaryEl.style.cursor = "pointer";
      summaryEl.style.color = "inherit";
      summaryEl.style.opacity = "0.72";
      summaryEl.style.fontSize = "0.85rem";
      details.append(summaryEl);
      const extra = document.createElement("p");
      extra.style.margin = "0.35rem 0 0";
      extra.style.fontSize = "0.85rem";
      extra.style.opacity = "0.78";
      if (!probe.reachable) {
        extra.textContent = copy.helperUnreachable;
      } else {
        extra.textContent = copy.technicianLine(pcsc, ver, Boolean(probe.synthetic));
      }
      details.append(extra);
      if (probe.managerUrl) {
        const link = document.createElement("a");
        link.href = `${probe.managerUrl}#technician`;
        link.textContent = copy.openTechnician;
        link.style.display = "inline-block";
        link.style.marginTop = "0.3rem";
        link.style.fontSize = "0.85rem";
        details.append(link);
      }
      box.append(details);
    }
    root.append(box);
  };

  const tick = () => {
    if (stopped) return;
    void probeThaiID(options).then(render);
  };
  tick();
  timer = setInterval(tick, intervalMs);
  return () => {
    stopped = true;
    if (timer) clearInterval(timer);
  };
}
