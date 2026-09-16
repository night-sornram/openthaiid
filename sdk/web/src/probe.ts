/**
 * Library helpers: check the local reader from an ERP page without a
 * separate OpenThaiID install. Does not read personal card fields.
 */

import { ThaiIDError, type ReaderView } from "./index.js";

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

/**
 * Ask the local helper whether it is running and what the reader looks like.
 *
 * Call from the ERP page (a user gesture helps with local-network permission).
 * This does not request card fields and does not skip origin approval for reads.
 * SSR-safe: returns `{ reachable: false }` when `WebSocket` is missing.
 */
export async function probeThaiID(options: ProbeThaiIDOptions = {}): Promise<ProbeResult> {
  if (options.signal?.aborted) {
    return {
      reachable: false,
      error: new ThaiIDError("CANCELLED", "probe aborted"),
    };
  }
  if (typeof WebSocket === "undefined") {
    return {
      reachable: false,
      error: new ThaiIDError("CONNECTION_FAILED", "probeThaiID() requires a browser"),
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
      finish({ reachable: false, error: new ThaiIDError("CANCELLED", "probe aborted") });
    };
    options.signal?.addEventListener("abort", onAbort, { once: true });
    if (options.signal?.aborted) {
      onAbort();
      return;
    }

    const timer = setTimeout(() => {
      finish({
        reachable: false,
        error: new ThaiIDError("BRIDGE_UNAVAILABLE", "OpenThaiID helper is not running on this PC", true),
      });
    }, timeoutMs);

    try {
      ws = new WebSocket(endpoint, [SUBPROTOCOL]);
    } catch {
      finish({
        reachable: false,
        error: new ThaiIDError("CONNECTION_FAILED", "could not open local WebSocket", true),
      });
      return;
    }

    ws.onerror = () => {
      finish({
        reachable: false,
        error: new ThaiIDError(
          "BRIDGE_UNAVAILABLE",
          "OpenThaiID helper is not running on this PC",
          true,
        ),
      });
    };
    ws.onopen = () => {
      try {
        ws?.send(JSON.stringify({ v: 1, id: 1, type: "bridge.probe" }));
      } catch {
        finish({
          reachable: false,
          error: new ThaiIDError("CONNECTION_FAILED", "probe send failed", true),
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
          error: new ThaiIDError(
            (msg.error.code as ThaiIDError["code"]) ?? "CONNECTION_FAILED",
            msg.error.message ?? "probe failed",
          ),
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

export type MountThaiIDStatusOptions = ProbeThaiIDOptions & {
  /** Poll interval. Default 2000ms. */
  intervalMs?: number;
};

/**
 * Drop-in reader status into an existing ERP page. No personal field values.
 * Returns a stop function.
 */
export function mountThaiIDStatus(root: Element, options: MountThaiIDStatusOptions = {}): () => void {
  const intervalMs = options.intervalMs ?? 2000;
  let stopped = false;
  let timer: ReturnType<typeof setInterval> | null = null;

  const render = (probe: ProbeResult) => {
    root.replaceChildren();
    const box = document.createElement("div");
    box.setAttribute("data-openthaiid-status", "1");
    box.style.fontFamily = "system-ui, sans-serif";
    box.style.fontSize = "0.92rem";
    box.style.lineHeight = "1.4";

    const line = document.createElement("p");
    line.style.margin = "0 0 0.4rem";
    if (!probe.reachable) {
      line.textContent =
        probe.error?.code === "CANCELLED"
          ? "Cancelled."
          : "OpenThaiID helper is not running on this PC. Start it locally — there is nothing to deploy.";
    } else {
      const pcsc = probe.backend?.status ?? "unknown";
      const demo = probe.synthetic ? " (demo)" : "";
      line.textContent = `Helper connected${demo} · PC/SC ${pcsc}`;
    }
    box.append(line);

    if (probe.reachable && probe.readers && probe.readers.length === 0) {
      const empty = document.createElement("p");
      empty.style.margin = "0";
      empty.textContent = "No card reader detected. Plug the reader into this PC.";
      box.append(empty);
    }
    if (probe.readers) {
      const ul = document.createElement("ul");
      ul.style.margin = "0";
      ul.style.paddingLeft = "1.1rem";
      for (const r of probe.readers) {
        const li = document.createElement("li");
        const presence =
          r.presence === "present" ? "card present" : r.presence === "unsupported" ? "unsupported card" : "no card";
        li.textContent = `${r.name} — ${presence}${r.selected ? " (selected)" : ""}`;
        ul.append(li);
      }
      box.append(ul);
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
