/**
 * @openthaiid/web — browser SDK for the OpenThaiID local bridge.
 *
 * Connect from a visible user action. The bridge may ask the operator to
 * approve this origin in the terminal before card data is available.
 */

export type Field = "citizenId" | "thaiName" | "birthDate";

export const DEFAULT_FIELDS: Field[] = ["citizenId", "thaiName", "birthDate"];
export const SUPPORTED_FIELDS: Field[] = ["citizenId", "thaiName", "birthDate"];

export type CardDate = {
  year: number | null;
  month: number | null;
  day: number | null;
};

export type ThaiName = {
  title: string | null;
  givenName: string | null;
  middleName: string | null;
  familyName: string | null;
  fullName: string;
};

export type FieldValues = {
  citizenId?: string | null;
  thaiName?: ThaiName | null;
  birthDate?: CardDate | null;
};

export type ThaiIDErrorCode =
  | "BRIDGE_UNAVAILABLE"
  | "CONNECTION_FAILED"
  | "PROTOCOL_MISMATCH"
  | "APPROVAL_REQUIRED"
  | "APPROVAL_DENIED"
  | "APPROVAL_TIMEOUT"
  | "AUTH_EXPIRED"
  | "FIELD_NOT_ALLOWED"
  | "PCSC_UNAVAILABLE"
  | "NO_READER"
  | "READER_SELECTION_REQUIRED"
  | "NO_CARD"
  | "CARD_REMOVED"
  | "READER_REMOVED"
  | "BUSY"
  | "UNSUPPORTED_CARD"
  | "READ_FAILED"
  | "READ_TIMEOUT"
  | "CANCELLED"
  | "BACKEND_STALLED"
  | "INVALID_REQUEST"
  | "UNSUPPORTED_FIELD"
  | "RATE_LIMITED";

export class ThaiIDError extends Error {
  readonly code: ThaiIDErrorCode;
  readonly retryable: boolean;
  readonly requestId?: number;

  constructor(code: ThaiIDErrorCode, message: string, retryable = false, requestId?: number) {
    super(message);
    this.name = "ThaiIDError";
    this.code = code;
    this.retryable = retryable;
    this.requestId = requestId;
  }
}

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "awaiting-approval"
  | "connected"
  | "reconnecting";

export type ReaderView = {
  readerId: string;
  name: string;
  presence: "absent" | "present" | "unsupported";
  cardSessionId?: string;
};

export type BridgeStatus = {
  connection: ConnectionState;
  backend: "available" | "unavailable" | "stalled";
  readers: ReaderView[];
};

export type ReadResult = {
  readerId: string;
  cardSessionId: string;
  fields: FieldValues;
  warnings: Array<{ field: Field; code: string; message: string }>;
};

export type ThaiIDEvents = {
  connectionChanged: (state: ConnectionState) => void;
  statusChanged: (status: BridgeStatus) => void;
  approvalRequired: (info: { approvalId: number; comparisonCode: string; fields: Field[] }) => void;
  cardChanged: (info: { present: boolean; readerId?: string; cardSessionId?: string }) => void;
  error: (err: ThaiIDError) => void;
};

type Handler<T> = (payload: T) => void;

export type ThaiIDOptions = {
  /** Default: ws://127.0.0.1:17891/v1/ws */
  endpoint?: string;
  /** Default: citizenId, thaiName, birthDate */
  fields?: Field[];
  /** Connect timeout ms (default 30000; approval wait included). */
  connectTimeoutMs?: number;
  /** Session renewal lead time ms (default 30000). */
  renewLeadMs?: number;
  /** Disable automatic reconnect (default false). */
  autoReconnect?: boolean;
};

const DEFAULT_ENDPOINT = "ws://127.0.0.1:17891/v1/ws";
const SUBPROTOCOL = "openthaiid.v1";

const RECONNECT_BASE_MS = 1000;
const RECONNECT_CAP_MS = 15_000;
const RECONNECT_BUDGET_MS = 120_000;

type WireError = { code: string; message: string; retryable?: boolean; requestId?: number };

type PendingWaiter = {
  resolve: (v: unknown) => void;
  reject: (e: Error) => void;
};

/**
 * ThaiID talks to a locally running OpenThaiID bridge.
 * Importing this module is SSR-safe; connect() requires a browser.
 */
export class ThaiID {
  readonly fields: Field[];
  readonly endpoint: string;
  readonly #connectTimeoutMs: number;
  readonly #renewLeadMs: number;
  readonly #autoReconnect: boolean;

  #state: ConnectionState = "disconnected";
  #status: BridgeStatus = {
    connection: "disconnected",
    backend: "unavailable",
    readers: [],
  };
  #lastFields: FieldValues | null = null;
  #listeners: { [K in keyof ThaiIDEvents]?: Set<Handler<never>> } = {};
  #ws: WebSocket | null = null;
  #token: string | null = null;
  #expiresAt: number | null = null;
  #instanceId: string | null = null;
  #nextId = 1;
  #pending = new Map<number, PendingWaiter>();
  #connectPromise: Promise<void> | null = null;
  #renewTimer: ReturnType<typeof setTimeout> | null = null;
  #reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  #reconnectAttempt = 0;
  #reconnectDeadline = 0;
  #manualDisconnect = false;
  #lastSeq = 0;
  #gotSnapshot = false;

  constructor(options: ThaiIDOptions = {}) {
    this.endpoint = options.endpoint ?? DEFAULT_ENDPOINT;
    this.#connectTimeoutMs = options.connectTimeoutMs ?? 30_000;
    this.#renewLeadMs = options.renewLeadMs ?? 30_000;
    this.#autoReconnect = options.autoReconnect ?? true;
    const fields = options.fields ?? DEFAULT_FIELDS;
    for (const f of fields) {
      if (!SUPPORTED_FIELDS.includes(f)) {
        throw new ThaiIDError("UNSUPPORTED_FIELD", `unsupported field: ${f}`);
      }
    }
    this.fields = [...fields];
  }

  state(): ConnectionState {
    return this.#state;
  }

  on<K extends keyof ThaiIDEvents>(event: K, handler: ThaiIDEvents[K]): () => void {
    const set = (this.#listeners[event] ??= new Set()) as Set<ThaiIDEvents[K]>;
    set.add(handler);
    return () => set.delete(handler);
  }

  #emit<K extends keyof ThaiIDEvents>(event: K, payload: Parameters<ThaiIDEvents[K]>[0]) {
    const set = this.#listeners[event] as Set<Handler<Parameters<ThaiIDEvents[K]>[0]>> | undefined;
    if (!set) return;
    for (const h of set) h(payload);
  }

  #setState(next: ConnectionState) {
    if (this.#state === next) return;
    this.#state = next;
    this.#status = { ...this.#status, connection: next };
    this.#emit("connectionChanged", next);
    this.#emit("statusChanged", this.#status);
  }

  #clearPersonalData() {
    this.#lastFields = null;
    this.#status = { ...this.#status, readers: [] };
  }

  #clearSession() {
    this.#token = null;
    this.#expiresAt = null;
    this.#clearPersonalData();
    if (this.#renewTimer) {
      clearTimeout(this.#renewTimer);
      this.#renewTimer = null;
    }
  }

  #scheduleRenewal() {
    if (this.#renewTimer) {
      clearTimeout(this.#renewTimer);
      this.#renewTimer = null;
    }
    if (!this.#expiresAt) return;
    const lead = Math.max(1000, this.#renewLeadMs);
    const delay = Math.max(1000, this.#expiresAt - Date.now() - lead);
    this.#renewTimer = setTimeout(() => {
      void this.#renewSession();
    }, delay);
  }

  async #renewSession(): Promise<void> {
    if (!this.#ws || !this.#token || this.#state !== "connected") return;
    try {
      const result = (await this.#send("session.renew")) as {
        token: string;
        expiresAt: string;
      };
      this.#token = result.token;
      this.#expiresAt = Date.parse(result.expiresAt) || null;
      this.#scheduleRenewal();
    } catch {
      // Renewal failure is non-fatal; reconnect path will re-authenticate.
    }
  }

  #onMessage = (ev: MessageEvent) => {
    let msg: any;
    try {
      msg = JSON.parse(typeof ev.data === "string" ? ev.data : String(ev.data));
    } catch {
      return;
    }

    if (msg && typeof msg.v === "number" && msg.v !== 1) {
      this.#emit("error", new ThaiIDError("PROTOCOL_MISMATCH", "unsupported protocol version"));
      return;
    }

    // Event envelope (no request id).
    if (msg && msg.type && msg.id === undefined) {
      this.#handleEvent(msg);
      return;
    }
    // Response envelope.
    if (msg && typeof msg.id === "number") {
      const waiter = this.#pending.get(msg.id);
      if (!waiter) return;
      this.#pending.delete(msg.id);
      if (msg.error) {
        waiter.reject(this.#toError(msg.error));
        return;
      }
      waiter.resolve(msg.result);
    }
  };

  #handleEvent(msg: any) {
    if (typeof msg.sequence === "number") {
      // Contiguous per-connection stream; gaps mean the client missed events.
      if (this.#lastSeq !== 0 && msg.sequence !== this.#lastSeq + 1) {
        this.#emit("error", new ThaiIDError("CONNECTION_FAILED", "event sequence gap; resync required", true));
      }
      this.#lastSeq = msg.sequence;
    }
    if (msg.instanceId && this.#instanceId && msg.instanceId !== this.#instanceId) {
      this.#emit("error", new ThaiIDError("CONNECTION_FAILED", "bridge instance changed", true));
      return;
    }

    switch (msg.type) {
      case "approval.required":
        this.#setState("awaiting-approval");
        this.#emit("approvalRequired", {
          approvalId: msg.payload?.approvalId,
          comparisonCode: msg.payload?.comparisonCode ?? "",
          fields: msg.payload?.fields ?? [],
        });
        break;
      case "status.snapshot":
        if (msg.payload) {
          this.#gotSnapshot = true;
          this.#status = {
            connection: this.#state,
            backend: msg.payload.backend ?? "available",
            readers: msg.payload.readers ?? [],
          };
          this.#emit("statusChanged", this.#status);
        }
        break;
      case "reader.connected":
      case "reader.disconnected":
        // Full snapshot will follow or status() can refresh.
        break;
      case "card.inserted":
        this.#updatePresence(msg.payload?.readerId, true, msg.payload?.cardSessionId);
        this.#emit("cardChanged", {
          present: true,
          readerId: msg.payload?.readerId,
          cardSessionId: msg.payload?.cardSessionId,
        });
        break;
      case "card.removed":
        this.#updatePresence(msg.payload?.readerId, false, undefined);
        this.#clearPersonalData();
        this.#emit("cardChanged", {
          present: false,
          readerId: msg.payload?.readerId,
          cardSessionId: msg.payload?.cardSessionId,
        });
        break;
      case "authorization.revoked":
        this.#handleRevocation();
        break;
      default:
        break;
    }
  }

  #updatePresence(readerId: string | undefined, present: boolean, cardSessionId?: string) {
    if (!readerId) return;
    this.#status = {
      ...this.#status,
      readers: this.#status.readers.map((r) =>
        r.readerId === readerId
          ? {
              ...r,
              presence: present ? "present" : "absent",
              cardSessionId: present ? cardSessionId : undefined,
            }
          : r,
      ),
    };
  }

  #handleRevocation() {
    this.#clearSession();
    this.#manualDisconnect = true;
    this.#emit("error", new ThaiIDError("AUTH_EXPIRED", "authorization revoked"));
    this.#emit("cardChanged", { present: false });
    const ws = this.#ws;
    this.#ws = null;
    this.#rejectAllPending(new ThaiIDError("AUTH_EXPIRED", "authorization revoked"));
    this.#setState("disconnected");
    if (ws) {
      try {
        ws.close();
      } catch {
        // ignore
      }
    }
  }

  #rejectAllPending(err: Error) {
    for (const [, w] of this.#pending) {
      w.reject(err);
    }
    this.#pending.clear();
  }

  #toError(e: WireError): ThaiIDError {
    return new ThaiIDError(
      (e.code as ThaiIDErrorCode) ?? "CONNECTION_FAILED",
      e.message ?? "request failed",
      Boolean(e.retryable),
      e.requestId,
    );
  }

  #send(type: string, payload?: unknown): Promise<unknown> {
    const ws = this.#ws;
    if (!ws || ws.readyState !== 1 /* OPEN */) {
      return Promise.reject(new ThaiIDError("CONNECTION_FAILED", "not connected", true));
    }
    const id = this.#nextId++;
    const msg: Record<string, unknown> = { v: 1, id, type };
    if (this.#token) msg.token = this.#token;
    if (payload !== undefined) msg.payload = payload;
    return new Promise((resolve, reject) => {
      this.#pending.set(id, { resolve, reject });
      try {
        ws.send(JSON.stringify(msg));
      } catch {
        this.#pending.delete(id);
        reject(new ThaiIDError("CONNECTION_FAILED", "send failed", true));
      }
    });
  }

  /**
   * Open a connection and authenticate.
   * May block while the operator approves this origin in the bridge terminal.
   * Concurrent connect() calls share one attempt.
   */
  async connect(options: { signal?: AbortSignal } = {}): Promise<void> {
    if (typeof WebSocket === "undefined") {
      throw new ThaiIDError("CONNECTION_FAILED", "connect() requires a browser environment");
    }
    if (options.signal?.aborted) {
      throw new ThaiIDError("CANCELLED", "connect aborted");
    }
    if (this.#state === "connected" && this.#token) return;
    if (this.#connectPromise) return this.#connectPromise;

    this.#manualDisconnect = false;
    this.#gotSnapshot = false;
    this.#lastSeq = 0;
    this.#connectPromise = this.#doConnect(options.signal).finally(() => {
      this.#connectPromise = null;
    });
    return this.#connectPromise;
  }

  async #doConnect(signal?: AbortSignal): Promise<void> {
    this.#setState("connecting");
    let ws: WebSocket | null = null;
    try {
      const socket = new WebSocket(this.endpoint, [SUBPROTOCOL]);
      ws = socket;
      this.#ws = socket;

      await this.#waitOpen(socket, signal);

      socket.onmessage = this.#onMessage;
      socket.onclose = () => {
        if (this.#ws === socket) {
          this.#onSocketClose(socket);
        }
      };

      const openPromise = this.#send("session.open", { fields: this.fields }) as Promise<{
        token: string;
        fields: Field[];
        expiresAt: string;
        instanceId?: string;
      }>;
      const result = await this.#withAbort(
        withTimeout(openPromise, this.#connectTimeoutMs),
        signal,
      );
      this.#token = result.token;
      this.#expiresAt = Date.parse(result.expiresAt) || null;
      if (result.instanceId) this.#instanceId = result.instanceId;
      this.#scheduleRenewal();

      // Await an authenticated snapshot before resolving.
      await this.#awaitSnapshot(2000);
      this.#reconnectAttempt = 0;
      this.#setState("connected");
    } catch (err) {
      this.#cleanupSocket(ws);
      this.#clearSession();
      this.#setState("disconnected");
      if (err instanceof ThaiIDError) throw err;
      throw new ThaiIDError("CONNECTION_FAILED", "connection failed", true);
    }
  }

  #waitOpen(ws: WebSocket, signal?: AbortSignal): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      const timer = setTimeout(() => {
        cleanup();
        reject(new ThaiIDError("BRIDGE_UNAVAILABLE", "bridge did not accept the connection", true));
      }, 10_000);
      const onAbort = () => {
        cleanup();
        reject(new ThaiIDError("CANCELLED", "connect aborted"));
      };
      const cleanup = () => {
        clearTimeout(timer);
        signal?.removeEventListener("abort", onAbort);
        ws.onopen = null;
        ws.onerror = null;
      };
      signal?.addEventListener("abort", onAbort, { once: true });
      ws.onopen = () => {
        cleanup();
        resolve();
      };
      ws.onerror = () => {
        cleanup();
        reject(
          new ThaiIDError(
            "CONNECTION_FAILED",
            "connection failed (bridge stopped, CSP, or permission denied)",
            true,
          ),
        );
      };
    });
  }

  #awaitSnapshot(timeoutMs: number): Promise<void> {
    if (this.#gotSnapshot) return Promise.resolve();
    return new Promise((resolve) => {
      const t = setTimeout(resolve, timeoutMs);
      const off = this.on("statusChanged", () => {
        if (this.#gotSnapshot) {
          clearTimeout(t);
          off();
          resolve();
        }
      });
    });
  }

  #withAbort<T>(p: Promise<T>, signal?: AbortSignal): Promise<T> {
    if (!signal) return p;
    if (signal.aborted) {
      return Promise.reject(new ThaiIDError("CANCELLED", "aborted"));
    }
    return new Promise<T>((resolve, reject) => {
      const onAbort = () => {
        reject(new ThaiIDError("CANCELLED", "aborted"));
      };
      signal.addEventListener("abort", onAbort, { once: true });
      p.then(
        (v) => {
          signal.removeEventListener("abort", onAbort);
          resolve(v);
        },
        (e) => {
          signal.removeEventListener("abort", onAbort);
          reject(e);
        },
      );
    });
  }

  #onSocketClose(ws: WebSocket) {
    this.#rejectAllPending(new ThaiIDError("CONNECTION_FAILED", "connection closed", true));
    this.#clearSession();
    this.#emit("cardChanged", { present: false });

    if (this.#manualDisconnect || !this.#autoReconnect) {
      this.#setState("disconnected");
      return;
    }
    // Do not reconnect after denial / revocation — those set manualDisconnect.
    this.#scheduleReconnect();
  }

  #cleanupSocket(ws: WebSocket | null) {
    if (!ws) return;
    try {
      ws.onmessage = null;
      ws.onclose = null;
      ws.onerror = null;
      ws.onopen = null;
      if (ws.readyState === 1 || ws.readyState === 0) {
        ws.close();
      }
    } catch {
      // ignore
    }
    if (this.#ws === ws) this.#ws = null;
  }

  #scheduleReconnect() {
    if (this.#reconnectTimer) return;
    const now = Date.now();
    if (this.#reconnectDeadline === 0) {
      this.#reconnectDeadline = now + RECONNECT_BUDGET_MS;
    }
    if (now > this.#reconnectDeadline) {
      this.#setState("disconnected");
      this.#reconnectDeadline = 0;
      this.#reconnectAttempt = 0;
      return;
    }
    this.#setState("reconnecting");
    const exp = Math.min(
      RECONNECT_CAP_MS,
      RECONNECT_BASE_MS * Math.pow(2, this.#reconnectAttempt),
    );
    const jitter = Math.floor(Math.random() * (exp / 2));
    const delay = exp / 2 + jitter;
    this.#reconnectAttempt++;
    this.#reconnectTimer = setTimeout(() => {
      this.#reconnectTimer = null;
      void this.connect().catch(() => {
        this.#scheduleReconnect();
      });
    }, delay);
  }

  async disconnect(): Promise<void> {
    this.#manualDisconnect = true;
    this.#reconnectDeadline = 0;
    this.#reconnectAttempt = 0;
    if (this.#reconnectTimer) {
      clearTimeout(this.#reconnectTimer);
      this.#reconnectTimer = null;
    }
    if (this.#renewTimer) {
      clearTimeout(this.#renewTimer);
      this.#renewTimer = null;
    }
    const ws = this.#ws;
    this.#ws = null;
    this.#clearSession();
    this.#setState("disconnected");
    this.#emit("cardChanged", { present: false });
    this.#rejectAllPending(new ThaiIDError("CANCELLED", "disconnected"));
    if (ws) {
      try {
        if (ws.readyState === 1) {
          // Best-effort session.close; ignore errors.
          try {
            ws.send(JSON.stringify({ v: 1, id: this.#nextId++, type: "session.close" }));
          } catch {
            // ignore
          }
        }
        ws.onmessage = null;
        ws.onclose = null;
        ws.close();
      } catch {
        // ignore
      }
    }
  }

  async status(): Promise<BridgeStatus> {
    if (this.#ws && this.#token) {
      try {
        const snap = (await this.#send("bridge.status")) as {
          backend: BridgeStatus["backend"];
          readers: ReaderView[];
        };
        this.#status = {
          connection: this.#state,
          backend: snap.backend,
          readers: snap.readers ?? [],
        };
      } catch {
        // keep last known
      }
    }
    return { ...this.#status };
  }

  /**
   * Read the configured or requested field subset.
   * Never retries automatically after timeout, removal, or disconnect.
   */
  async read(
    options: { fields?: Field[]; readerId?: string; cardSessionId?: string; signal?: AbortSignal } = {},
  ): Promise<ReadResult> {
    if (!this.#ws || !this.#token) {
      throw new ThaiIDError("CONNECTION_FAILED", "not connected", true);
    }
    if (options.signal?.aborted) {
      throw new ThaiIDError("CANCELLED", "read aborted");
    }
    const fields = options.fields ?? this.fields;
    for (const f of fields) {
      if (!this.fields.includes(f)) {
        throw new ThaiIDError("FIELD_NOT_ALLOWED", `field ${f} is not in the approved set`);
      }
    }

    let readerId = options.readerId;
    let cardSessionId = options.cardSessionId;
    if (!readerId) {
      const present = this.#status.readers.filter((r) => r.presence === "present");
      if (present.length === 0) {
        throw new ThaiIDError("NO_CARD", "no card present");
      }
      if (present.length > 1) {
        throw new ThaiIDError("READER_SELECTION_REQUIRED", "multiple readers; pass readerId");
      }
      readerId = present[0]!.readerId;
      cardSessionId = cardSessionId ?? present[0]!.cardSessionId;
    }

    // Allocate the wire id so abort can cancel the real request.
    const requestId = this.#nextId++;
    const ws = this.#ws;
    const msg: Record<string, unknown> = {
      v: 1,
      id: requestId,
      type: "card.read",
      token: this.#token,
      payload: { readerId, cardSessionId, fields },
    };

    const onAbort = () => {
      void this.#send("request.cancel", { requestId }).catch(() => undefined);
      const waiter = this.#pending.get(requestId);
      if (waiter) {
        this.#pending.delete(requestId);
        waiter.reject(new ThaiIDError("CANCELLED", "read aborted"));
      }
    };
    options.signal?.addEventListener("abort", onAbort, { once: true });

    try {
      const result = await new Promise<ReadResult>((resolve, reject) => {
        this.#pending.set(requestId, {
          resolve: (v) => resolve(v as ReadResult),
          reject,
        });
        try {
          ws.send(JSON.stringify(msg));
        } catch {
          this.#pending.delete(requestId);
          reject(new ThaiIDError("CONNECTION_FAILED", "send failed", true));
        }
      });
      this.#lastFields = result.fields;
      return result;
    } finally {
      options.signal?.removeEventListener("abort", onAbort);
      this.#pending.delete(requestId);
    }
  }
}

function withTimeout<T>(p: Promise<T>, ms: number): Promise<T> {
  return new Promise((resolve, reject) => {
    const t = setTimeout(() => reject(new ThaiIDError("APPROVAL_TIMEOUT", "timed out", true)), ms);
    p.then(
      (v) => {
        clearTimeout(t);
        resolve(v);
      },
      (e) => {
        clearTimeout(t);
        reject(e);
      },
    );
  });
}

export default ThaiID;
