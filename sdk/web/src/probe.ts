/**
 * Library helpers: check the local reader from an ERP page without a
 * separate OpenThaiID install. Does not read personal card fields.
 */

import { pageBlocksInsecureLoopbackWs, replaceElementChildren } from "./browser.js";
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

export type WaitForThaiIDOptions = ProbeThaiIDOptions & {
  /** Pause between unsuccessful probes. Default 400ms. */
  intervalMs?: number;
  /** Per-probe WebSocket timeout. Default 2000ms. */
  probeTimeoutMs?: number;
};

export type HelperWatchPhase =
  | "checking"
  | "offline"
  | "online"
  | "waiting"
  | "reconnected"
  | "service-reconnected";

export type HelperWatchUpdate = {
  probe: ProbeResult;
  summary: StatusSummary;
  phase: HelperWatchPhase;
  /** True when this tick recovered after a previous reachable helper. */
  recovered: boolean;
};

export type WatchThaiIDOptions = ProbeThaiIDOptions & {
  /** Poll interval while the helper is reachable but not Ready. Default 2000ms. */
  intervalMs?: number;
  /**
   * Poll interval while the helper is Ready (card present). Default 400ms
   * so a pulled card or unplug does not leave a stale พร้อม / Ready.
   */
  readyIntervalMs?: number;
  /** Poll interval while waiting for the helper to return. Default 500ms. */
  offlineIntervalMs?: number;
  /**
   * Poll interval while the helper is up but the card service is
   * unavailable/stalled. Default matches `offlineIntervalMs`.
   */
  serviceIntervalMs?: number;
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

function withBackendError(result: ProbeResult, lang?: StaffLang | string | null): ProbeResult {
  if (!result.reachable || result.error) return result;
  const status = result.backend?.status;
  if (status === "unavailable") {
    return { ...result, error: probeError("PCSC_UNAVAILABLE", lang, true) };
  }
  if (status === "stalled") {
    return { ...result, error: probeError("BACKEND_STALLED", lang, true) };
  }
  return result;
}

function backendNeedsRecovery(probe: ProbeResult): boolean {
  if (!probe.reachable) return false;
  const status = probe.backend?.status;
  return status === "unavailable" || status === "stalled";
}

function probeUnreachable(
  endpoint: string,
  lang: StaffLang,
  code: ThaiIDError["code"] = "BRIDGE_UNAVAILABLE",
  retryable = true,
): ProbeResult {
  if (code !== "CANCELLED" && pageBlocksInsecureLoopbackWs(endpoint)) {
    return { reachable: false, error: probeError("HTTPS_WS_BLOCKED", lang, false) };
  }
  return { reachable: false, error: probeError(code, lang, retryable) };
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
      finish(probeUnreachable(endpoint, lang, "BRIDGE_UNAVAILABLE", true));
    }, timeoutMs);

    try {
      ws = new WebSocket(endpoint, [SUBPROTOCOL]);
    } catch {
      finish(probeUnreachable(endpoint, lang, "CONNECTION_FAILED", true));
      return;
    }

    ws.onerror = () => {
      finish(probeUnreachable(endpoint, lang, "BRIDGE_UNAVAILABLE", true));
    };
    ws.onopen = () => {
      try {
        ws?.send(JSON.stringify({ v: 1, id: 1, type: "bridge.probe" }));
      } catch {
        finish(probeUnreachable(endpoint, lang, "CONNECTION_FAILED", true));
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
      finish(
        withBackendError(
          {
            reachable: true,
            version: typeof r.version === "string" ? r.version : undefined,
            instanceId: typeof r.instanceId === "string" ? r.instanceId : undefined,
            synthetic: Boolean(r.synthetic),
            backend,
            readers,
            managerUrl: typeof r.managerUrl === "string" ? r.managerUrl : managerUrlFromEndpoint(endpoint),
            wsUrl: typeof r.wsUrl === "string" ? r.wsUrl : endpoint,
          },
          lang,
        ),
      );
    };
  });
}

export interface StatusSummary {
  ready: boolean;
  headline: string;
  helper: string;
  reader: string;
  card: string;
  next: string;
}

/** Plain-language helper / reader / card state. Default Thai; pass `"en"` for English. */
export function summarizeProbe(probe: ProbeResult, lang?: StaffLang | string | null): StatusSummary {
  const c = staffCopy(lang);
  if (!probe.reachable) {
    const cancelled = probe.error?.code === "CANCELLED";
    const httpsBlocked = probe.error?.code === "HTTPS_WS_BLOCKED";
    return {
      ready: false,
      headline: cancelled ? c.cancelled : c.notReady,
      helper: cancelled ? c.cancelled + (c.lang === "th" ? "" : ".") : httpsBlocked ? c.httpsWsBlocked : c.helperOff,
      reader: c.readerUnknown,
      card: c.cardUnknown,
      next: cancelled ? "" : httpsBlocked ? c.httpsWsBlockedNext : c.startHelper,
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
  const selected = readers.find((r) => r.selected) ?? (readers.length === 1 ? readers[0] : undefined);
  const needChoice = readers.length > 1 && !readers.some((r) => r.selected);
  const present = selected?.presence === "present";
  const unsupported = selected?.presence === "unsupported";
  const defaultName = c.lang === "th" ? "เครื่องอ่านบัตร" : "card reader";
  const readerLine = selected
    ? c.readerConnected(selected.name ?? defaultName)
    : needChoice
      ? c.chooseReader
      : c.readerConnected(defaultName);
  if (needChoice) {
    return {
      ready: false,
      headline: c.notReady,
      helper: c.helperOn,
      reader: readerLine,
      card: c.insertAfterReader,
      next: c.chooseReader,
    };
  }
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

/** Override Ready copy while the helper is missing or just returned. */
export function summarizeHelperWatch(
  probe: ProbeResult,
  phase: HelperWatchPhase,
  lang?: StaffLang | string | null,
): StatusSummary {
  const c = staffCopy(lang);
  const base = summarizeProbe(probe, lang);
  if (phase === "checking") {
    return {
      ready: false,
      headline: c.checking,
      helper: c.checking,
      reader: c.readerUnknown,
      card: c.cardUnknown,
      next: c.oneMoment,
    };
  }
  if (phase === "waiting") {
    return {
      ...base,
      ready: false,
      headline: c.helperLost,
      helper: c.helperOff,
      next: c.helperWaiting,
    };
  }
  if (phase === "reconnected") {
    return {
      ...base,
      headline: c.helperBack,
      next: base.ready ? c.helperBackNext : base.next,
    };
  }
  if (phase === "service-reconnected") {
    return {
      ...base,
      headline: base.ready ? base.headline : c.serviceBack,
      next: base.ready ? c.serviceBackNext : base.next,
    };
  }
  return base;
}

/** One-line staff result for an explicit connection test. Never includes card values. */
export function formatTestConnection(probe: ProbeResult, lang?: StaffLang | string | null): string {
  const c = staffCopy(lang);
  if (!probe.reachable) {
    if (probe.error?.code === "HTTPS_WS_BLOCKED") {
      return `${c.httpsWsBlocked} ${c.httpsWsBlockedNext}`.trim();
    }
    return c.testFail;
  }
  const status = probe.backend?.status;
  if (status === "unavailable") return c.testServiceOff;
  if (status === "stalled") return c.testServiceStalled;
  const summary = summarizeProbe(probe, lang);
  return summary.ready ? c.testOk : `${c.testOk} ${summary.next}`.trim();
}

function delay(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal?.aborted) {
      resolve();
      return;
    }
    const timer = setTimeout(resolve, ms);
    const onAbort = () => {
      clearTimeout(timer);
      resolve();
    };
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

/**
 * Poll `bridge.probe` until the helper accepts a WebSocket or the budget expires.
 * Does not open a session and never reads personal fields.
 */
export async function waitForThaiID(options: WaitForThaiIDOptions = {}): Promise<ProbeResult> {
  const lang = staffLang(options.lang);
  const overall = options.timeoutMs ?? 30_000;
  const intervalMs = options.intervalMs ?? 400;
  const probeTimeoutMs = options.probeTimeoutMs ?? 2_000;
  const started = Date.now();
  let last: ProbeResult | null = null;

  while (true) {
    if (options.signal?.aborted) {
      return { reachable: false, error: probeError("CANCELLED", lang) };
    }
    const remaining = overall - (Date.now() - started);
    if (remaining <= 0) {
      return (
        last ?? {
          reachable: false,
          error: probeError("BRIDGE_UNAVAILABLE", lang, true),
        }
      );
    }
    const probe = await probeThaiID({
      endpoint: options.endpoint,
      signal: options.signal,
      lang: options.lang,
      timeoutMs: Math.max(50, Math.min(probeTimeoutMs, remaining)),
    });
    last = probe;
    if (
      probe.reachable ||
      probe.error?.code === "CANCELLED" ||
      probe.error?.code === "HTTPS_WS_BLOCKED"
    ) {
      return probe;
    }
    const wait = Math.min(intervalMs, overall - (Date.now() - started));
    if (wait <= 0) return probe;
    await delay(wait, options.signal);
  }
}

/**
 * Keep probing the helper and report Thai (default) staff copy as it drops
 * and comes back — including after sleep/wake or a card-service hiccup.
 * Faster while waiting for a restart or while the card service is down.
 * Never reads card fields.
 */
export function watchThaiID(
  onUpdate: (update: HelperWatchUpdate) => void,
  options: WatchThaiIDOptions = {},
): () => void {
  const lang = staffLang(options.lang);
  const onlineInterval = options.intervalMs ?? 2000;
  const readyInterval = options.readyIntervalMs ?? 400;
  const offlineInterval = options.offlineIntervalMs ?? 500;
  const serviceInterval = options.serviceIntervalMs ?? offlineInterval;
  let stopped = false;
  let inFlight = false;
  let timer: ReturnType<typeof setTimeout> | null = null;
  let sawOnline = false;
  let lastReachable = false;
  let sawServiceDown = false;
  let abort: AbortController | null = null;

  const emit = (probe: ProbeResult, phase: HelperWatchPhase, recovered: boolean) => {
    if (stopped) return;
    onUpdate({
      probe,
      summary: summarizeHelperWatch(probe, phase, lang),
      phase,
      recovered,
    });
  };

  const schedule = (ms: number) => {
    if (stopped) return;
    timer = setTimeout(() => {
      timer = null;
      void tick();
    }, ms);
  };

  const tick = async () => {
    if (stopped || inFlight) return;
    inFlight = true;
    abort = new AbortController();
    const probeTimeout = options.timeoutMs ?? (lastReachable || !sawOnline ? 8_000 : 2_000);
    const probe = await probeThaiID({
      endpoint: options.endpoint,
      lang: options.lang,
      timeoutMs: probeTimeout,
      signal: abort.signal,
    });
    inFlight = false;
    if (stopped) return;

    if (probe.error?.code === "CANCELLED") {
      return;
    }

    let phase: HelperWatchPhase;
    let recovered = false;
    const serviceDown = backendNeedsRecovery(probe);
    if (probe.reachable) {
      const helperRecovered = sawOnline && !lastReachable;
      const serviceRecovered = sawServiceDown && probe.backend?.status === "available";
      recovered = helperRecovered || serviceRecovered;
      sawOnline = true;
      lastReachable = true;
      if (serviceDown) sawServiceDown = true;
      else if (probe.backend?.status === "available") sawServiceDown = false;
      if (helperRecovered) phase = "reconnected";
      else if (serviceRecovered) phase = "service-reconnected";
      else phase = "online";
    } else if (sawOnline) {
      lastReachable = false;
      phase = "waiting";
    } else {
      lastReachable = false;
      phase = "offline";
    }

    emit(probe, phase, recovered);
    const nextMs = !probe.reachable
      ? offlineInterval
      : serviceDown
        ? serviceInterval
        : summarizeProbe(probe, lang).ready
          ? readyInterval
          : onlineInterval;
    schedule(nextMs);
  };

  const onWake = () => {
    if (stopped) return;
    if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    void tick();
  };

  emit({ reachable: false }, "checking", false);
  void tick();

  if (typeof document !== "undefined") {
    document.addEventListener("visibilitychange", onWake);
  }
  if (typeof window !== "undefined") {
    window.addEventListener("pageshow", onWake);
    window.addEventListener("online", onWake);
  }

  return () => {
    stopped = true;
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    abort?.abort();
    if (typeof document !== "undefined") {
      document.removeEventListener("visibilitychange", onWake);
    }
    if (typeof window !== "undefined") {
      window.removeEventListener("pageshow", onWake);
      window.removeEventListener("online", onWake);
    }
  };
}

export type MountThaiIDStatusOptions = WatchThaiIDOptions & {
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
  /**
   * Show an explicit connection test (probe only, no personal fields).
   * Default true in `user` mode.
   */
  showTestConnection?: boolean;
};

/**
 * Drop-in reader status into an existing ERP page. No personal field values.
 * Auto-detects the helper coming back after a restart. Default copy is for
 * clinic staff. Returns a stop function.
 */
export function mountThaiIDStatus(root: Element, options: MountThaiIDStatusOptions = {}): () => void {
  const mode = options.mode ?? "user";
  const lang = staffLang(options.lang);
  const copy = staffCopy(lang);
  const showTest = options.showTestConnection ?? mode === "user";
  let lastUpdate: HelperWatchUpdate | null = null;
  let testLine = "";
  let testing = false;

  const render = (update: HelperWatchUpdate) => {
    lastUpdate = update;
    const { probe, summary, phase } = update;
    replaceElementChildren(root);
    const box = document.createElement("div");
    box.setAttribute("data-openthaiid-status", "1");
    box.setAttribute("data-openthaiid-status-mode", mode);
    box.setAttribute("data-openthaiid-lang", lang);
    box.setAttribute("data-openthaiid-phase", phase);
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

    const readers = probe.readers ?? [];
    if (readers.length > 1) {
      const list = document.createElement("ul");
      list.setAttribute("data-openthaiid-readers", "1");
      list.style.margin = "0.45rem 0 0";
      list.style.paddingLeft = "1.15rem";
      list.style.fontSize = "0.9rem";
      for (const r of readers) {
        const li = document.createElement("li");
        const using = r.selected ? (lang === "th" ? " · กำลังใช้" : " · selected") : "";
        const card = r.presence === "present" ? (lang === "th" ? " · มีบัตร" : " · card") : "";
        li.textContent = `${r.name || r.readerId}${using}${card}`;
        if (r.selected) li.style.fontWeight = "650";
        list.append(li);
      }
      box.append(list);
    }

    if (showTest) {
      const actions = document.createElement("div");
      actions.style.marginTop = "0.7rem";
      const btn = document.createElement("button");
      btn.type = "button";
      btn.setAttribute("data-openthaiid-test-connection", "1");
      btn.textContent = testing ? copy.testingConnection : copy.testConnection;
      btn.disabled = testing;
      btn.style.font = "inherit";
      btn.style.fontSize = "0.9rem";
      btn.style.padding = "0.35rem 0.7rem";
      btn.style.cursor = testing ? "not-allowed" : "pointer";
      btn.addEventListener("click", () => {
        void runTest();
      });
      actions.append(btn);
      if (testLine) {
        const result = document.createElement("p");
        result.setAttribute("data-openthaiid-test-result", "1");
        result.style.margin = "0.4rem 0 0";
        result.style.fontSize = "0.9rem";
        result.textContent = testLine;
        actions.append(result);
      }
      box.append(actions);
    }

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

  const runTest = async () => {
    if (testing) return;
    testing = true;
    testLine = "";
    if (lastUpdate) render(lastUpdate);
    const probe = await probeThaiID({
      endpoint: options.endpoint,
      lang: options.lang,
      timeoutMs: options.timeoutMs ?? 8_000,
    });
    testing = false;
    testLine = formatTestConnection(probe, lang);
    const phase: HelperWatchPhase = probe.reachable ? "online" : lastUpdate?.phase === "waiting" ? "waiting" : "offline";
    render({
      probe,
      summary: summarizeHelperWatch(probe, phase, lang),
      phase,
      recovered: false,
    });
  };

  const stopWatch = watchThaiID((update) => {
    render(update);
  }, options);

  return () => {
    stopWatch();
  };
}
