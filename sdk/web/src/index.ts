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
};

const DEFAULT_ENDPOINT = "ws://127.0.0.1:17891/v1/ws";
const SUBPROTOCOL = "openthaiid.v1";

type WireError = { code: string; message: string; retryable?: boolean; requestId?: number };

/**
 * ThaiID talks to a locally running OpenThaiID bridge.
 * Importing this module is SSR-safe; connect() requires a browser.
 */
export class ThaiID {
  readonly fields: Field[];
  readonly endpoint: string;
  readonly #connectTimeoutMs: number;

  #state: ConnectionState = "disconnected";
  #status: BridgeStatus = {
    connection: "disconnected",
    backend: "unavailable",
    readers: [],
  };
  #listeners: { [K in keyof ThaiIDEvents]?: Set<Handler<never>> } = {};
  #ws: WebSocket | null = null;
  #token: string | null = null;
  #nextId = 1;
  #pending = new Map<
    number,
    { resolve: (v: unknown) => void; reject: (e: Error) => void }
  >();

  constructor(options: ThaiIDOptions = {}) {
    this.endpoint = options.endpoint ?? DEFAULT_ENDPOINT;
    this.#connectTimeoutMs = options.connectTimeoutMs ?? 30_000;
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

  #onMessage = (ev: MessageEvent) => {
    let msg: any;
    try {
      msg = JSON.parse(typeof ev.data === "string" ? ev.data : String(ev.data));
    } catch {
      return;
    }

    // Event envelope (no request id).
    if (msg.type && msg.id === undefined) {
      this.#handleEvent(msg);
      return;
    }
    // Response envelope.
    if (typeof msg.id === "number") {
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
          this.#status = {
            connection: this.#state,
            backend: msg.payload.backend ?? "available",
            readers: msg.payload.readers ?? [],
          };
          this.#emit("statusChanged", this.#status);
        }
        break;
      case "card.inserted":
        this.#emit("cardChanged", {
          present: true,
          readerId: msg.payload?.readerId,
          cardSessionId: msg.payload?.cardSessionId,
        });
        break;
      case "card.removed":
        this.#emit("cardChanged", {
          present: false,
          readerId: msg.payload?.readerId,
          cardSessionId: msg.payload?.cardSessionId,
        });
        break;
      case "authorization.revoked":
        this.#token = null;
        this.#emit("error", new ThaiIDError("AUTH_EXPIRED", "authorization revoked"));
        break;
      default:
        break;
    }
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
    if (!ws || ws.readyState !== WebSocket.OPEN) {
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
      } catch (err) {
        this.#pending.delete(id);
        reject(new ThaiIDError("CONNECTION_FAILED", "send failed", true));
      }
    });
  }

  /**
   * Open a connection and authenticate.
   * May block while the operator approves this origin in the bridge terminal.
   */
  async connect(options: { signal?: AbortSignal } = {}): Promise<void> {
    if (typeof WebSocket === "undefined") {
      throw new ThaiIDError("CONNECTION_FAILED", "connect() requires a browser environment");
    }
    if (this.#state === "connected") return;

    this.#setState("connecting");
    try {
      const ws = new WebSocket(this.endpoint, [SUBPROTOCOL]);
      this.#ws = ws;

      await new Promise<void>((resolve, reject) => {
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
          options.signal?.removeEventListener("abort", onAbort);
        };
        options.signal?.addEventListener("abort", onAbort, { once: true });
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

      ws.onmessage = this.#onMessage;
      ws.onclose = () => {
        this.#ws = null;
        this.#token = null;
        this.#setState("disconnected");
        for (const [, w] of this.#pending) {
          w.reject(new ThaiIDError("CONNECTION_FAILED", "connection closed", true));
        }
        this.#pending.clear();
      };

      const openPromise = this.#send("session.open", { fields: this.fields }) as Promise<{
        token: string;
        fields: Field[];
        expiresAt: string;
      }>;
      const result = await withTimeout(openPromise, this.#connectTimeoutMs);
      this.#token = result.token;
      this.#setState("connected");
    } catch (err) {
      this.#setState("disconnected");
      this.#ws = null;
      this.#token = null;
      if (err instanceof ThaiIDError) throw err;
      throw new ThaiIDError("CONNECTION_FAILED", "connection failed", true);
    }
  }

  async disconnect(): Promise<void> {
    const ws = this.#ws;
    this.#ws = null;
    this.#token = null;
    this.#setState("disconnected");
    this.#status = { connection: "disconnected", backend: this.#status.backend, readers: [] };
    this.#emit("cardChanged", { present: false });
    for (const [, w] of this.#pending) {
      w.reject(new ThaiIDError("CANCELLED", "disconnected"));
    }
    this.#pending.clear();
    if (ws) {
      try {
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
    const fields = options.fields ?? this.fields;
    for (const f of fields) {
      if (!this.fields.includes(f)) {
        throw new ThaiIDError("FIELD_NOT_ALLOWED", `field ${f} is not in the approved set`);
      }
    }

    let readerId = options.readerId;
    if (!readerId) {
      const present = this.#status.readers.filter((r) => r.presence === "present");
      if (present.length === 0) {
        throw new ThaiIDError("NO_CARD", "no card present");
      }
      if (present.length > 1) {
        throw new ThaiIDError("READER_SELECTION_REQUIRED", "multiple readers; pass readerId");
      }
      readerId = present[0]!.readerId;
      options.cardSessionId = options.cardSessionId ?? present[0]!.cardSessionId;
    }

    const onAbort = () => {
      void this.#send("request.cancel", { requestId: 0 });
    };
    options.signal?.addEventListener("abort", onAbort, { once: true });
    try {
      const result = (await this.#send("card.read", {
        readerId,
        cardSessionId: options.cardSessionId,
        fields,
      })) as ReadResult;
      return result;
    } finally {
      options.signal?.removeEventListener("abort", onAbort);
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
