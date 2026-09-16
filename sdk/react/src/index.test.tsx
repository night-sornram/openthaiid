import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ThaiID,
  ThaiIDError,
  useThaiID,
  type ApprovalRequired,
  type BridgeStatus,
  type ConnectionState,
  type Field,
  type FieldValues,
  type ReadResult,
  type ThaiIDClient,
} from "./index";

type Handler = (...args: never[]) => void;

class FakeThaiID implements ThaiIDClient {
  readonly fields: Field[];
  readonly endpoint: string;
  lastConnectSignal?: AbortSignal;
  lastReadSignal?: AbortSignal;
  lastReadFields?: Field[];
  connectImpl?: (options?: { signal?: AbortSignal }) => Promise<void>;
  readResult: ReadResult = {
    readerId: "r1",
    cardSessionId: "cs_1",
    fields: { citizenId: "1234567890121" },
    warnings: [],
  };

  #state: ConnectionState = "disconnected";
  #status: BridgeStatus = {
    connection: "disconnected",
    backend: "unavailable",
    readers: [],
  };
  #listeners: Record<string, Set<Handler>> = {};

  constructor(options: { fields?: Field[]; endpoint?: string } = {}) {
    this.fields = options.fields ?? ["citizenId", "thaiName", "birthDate"];
    this.endpoint = options.endpoint ?? "ws://127.0.0.1:17891/v1/ws";
  }

  state(): ConnectionState {
    return this.#state;
  }

  on(event: string, handler: Handler): () => void {
    const set = (this.#listeners[event] ??= new Set());
    set.add(handler);
    return () => set.delete(handler);
  }

  emit(event: string, payload: unknown) {
    const set = this.#listeners[event];
    if (!set) return;
    for (const h of set) h(payload as never);
  }

  setState(next: ConnectionState) {
    this.#state = next;
    this.#status = { ...this.#status, connection: next };
    this.emit("connectionChanged", next);
    this.emit("statusChanged", this.#status);
  }

  async connect(options?: { signal?: AbortSignal }): Promise<void> {
    this.lastConnectSignal = options?.signal;
    if (options?.signal?.aborted) {
      throw new ThaiIDError("CANCELLED", "connect aborted");
    }
    if (this.connectImpl) {
      await this.connectImpl(options);
      return;
    }
    this.setState("connecting");
    const info: ApprovalRequired = {
      approvalId: 1,
      comparisonCode: "123456",
      fields: this.fields,
    };
    this.setState("awaiting-approval");
    this.emit("approvalRequired", info);
    this.#status = {
      connection: "connected",
      backend: "available",
      readers: [{ readerId: "r1", name: "ACR39U", presence: "present", cardSessionId: "cs_1" }],
    };
    this.setState("connected");
  }

  async read(options?: {
    fields?: Field[];
    readerId?: string;
    cardSessionId?: string;
    signal?: AbortSignal;
  }): Promise<ReadResult> {
    this.lastReadSignal = options?.signal;
    this.lastReadFields = options?.fields;
    if (options?.signal?.aborted) {
      throw new ThaiIDError("CANCELLED", "read aborted");
    }
    if (this.#state !== "connected") {
      throw new ThaiIDError("CONNECTION_FAILED", "not connected", true);
    }
    return this.readResult;
  }

  async disconnect(): Promise<void> {
    this.setState("disconnected");
  }

  async status(): Promise<BridgeStatus> {
    return { ...this.#status };
  }
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("useThaiID", () => {
  it("starts disconnected without touching WebSocket (SSR-safe construct)", () => {
    const { result } = renderHook(() => useThaiID({ client: new FakeThaiID() }));
    expect(result.current.connection).toBe("disconnected");
    expect(result.current.fields).toBeNull();
    expect(result.current.approvalRequired).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it("constructs a real ThaiID when no client is injected", () => {
    const { result } = renderHook(() => useThaiID());
    expect(result.current.client).toBeInstanceOf(ThaiID);
    expect(result.current.connection).toBe("disconnected");
  });

  it("connects, surfaces approvalRequired, then status", async () => {
    const client = new FakeThaiID({ fields: ["citizenId", "englishName"] });
    const { result } = renderHook(() => useThaiID({ client }));

    await act(async () => {
      await result.current.connect();
    });

    expect(result.current.connection).toBe("connected");
    expect(result.current.status.backend).toBe("available");
    expect(result.current.status.readers[0]?.readerId).toBe("r1");
    // approvalRequired is cleared once the session is connected.
    expect(result.current.approvalRequired).toBeNull();
  });

  it("keeps approvalRequired while awaiting terminal allow", async () => {
    const client = new FakeThaiID();
    client.connectImpl = async () => {
      client.setState("connecting");
      client.setState("awaiting-approval");
      client.emit("approvalRequired", {
        approvalId: 7,
        comparisonCode: "654321",
        fields: client.fields,
      });
    };
    const { result } = renderHook(() => useThaiID({ client }));

    await act(async () => {
      await result.current.connect();
    });

    expect(result.current.connection).toBe("awaiting-approval");
    expect(result.current.approvalRequired).toEqual({
      approvalId: 7,
      comparisonCode: "654321",
      fields: client.fields,
    });
  });

  it("read stores fields and passes AbortSignal through", async () => {
    const client = new FakeThaiID();
    const extra: FieldValues = {
      citizenId: "1234567890121",
      englishName: {
        title: "Mr.",
        givenName: "Somchai",
        middleName: null,
        familyName: "Jaidee",
        fullName: "Mr. Somchai Jaidee",
      },
    };
    client.readResult = {
      readerId: "r1",
      cardSessionId: "cs_1",
      fields: extra,
      warnings: [],
    };
    const { result } = renderHook(() => useThaiID({ client }));
    await act(async () => {
      await result.current.connect();
    });

    const ac = new AbortController();
    let readOut: ReadResult | undefined;
    await act(async () => {
      readOut = await result.current.read({ fields: ["citizenId", "englishName"], signal: ac.signal });
    });

    expect(client.lastReadSignal).toBe(ac.signal);
    expect(client.lastReadFields).toEqual(["citizenId", "englishName"]);
    expect(result.current.fields?.englishName?.givenName).toBe("Somchai");
    expect(readOut?.fields.citizenId).toBe("1234567890121");
  });

  it("connect abort is CANCELLED and recorded on error", async () => {
    const client = new FakeThaiID();
    const { result } = renderHook(() => useThaiID({ client }));
    const ac = new AbortController();
    ac.abort();
    await act(async () => {
      await expect(result.current.connect({ signal: ac.signal })).rejects.toMatchObject({
        code: "CANCELLED",
      });
    });
    expect(client.lastConnectSignal).toBe(ac.signal);
    expect(result.current.error?.code).toBe("CANCELLED");
    expect(result.current.connection).toBe("disconnected");
  });

  it("read abort is CANCELLED without retaining fields", async () => {
    const client = new FakeThaiID();
    const { result } = renderHook(() => useThaiID({ client }));
    await act(async () => {
      await result.current.connect();
    });
    const ac = new AbortController();
    ac.abort();
    await act(async () => {
      await expect(result.current.read({ signal: ac.signal })).rejects.toMatchObject({
        code: "CANCELLED",
      });
    });
    expect(result.current.fields).toBeNull();
    expect(result.current.error?.code).toBe("CANCELLED");
  });

  it("disconnect clears fields and approval", async () => {
    const client = new FakeThaiID();
    const { result } = renderHook(() => useThaiID({ client }));
    await act(async () => {
      await result.current.connect();
      await result.current.read();
    });
    expect(result.current.fields).not.toBeNull();
    await act(async () => {
      await result.current.disconnect();
    });
    expect(result.current.connection).toBe("disconnected");
    expect(result.current.fields).toBeNull();
    expect(result.current.approvalRequired).toBeNull();
  });

  it("card removal clears displayed fields", async () => {
    const client = new FakeThaiID();
    const { result } = renderHook(() => useThaiID({ client }));
    await act(async () => {
      await result.current.connect();
      await result.current.read();
    });
    act(() => {
      client.emit("cardChanged", { present: false, readerId: "r1" });
    });
    await waitFor(() => {
      expect(result.current.fields).toBeNull();
    });
  });

  it("revocation surfaces AUTH_EXPIRED and clears fields", async () => {
    const client = new FakeThaiID();
    const { result } = renderHook(() => useThaiID({ client }));
    await act(async () => {
      await result.current.connect();
      await result.current.read();
    });
    act(() => {
      client.emit("error", new ThaiIDError("AUTH_EXPIRED", "authorization revoked"));
      client.setState("disconnected");
    });
    await waitFor(() => {
      expect(result.current.error?.code).toBe("AUTH_EXPIRED");
      expect(result.current.fields).toBeNull();
    });
  });

  it("unmount disconnects the client", async () => {
    const client = new FakeThaiID();
    const spy = vi.spyOn(client, "disconnect");
    const { unmount } = renderHook(() => useThaiID({ client }));
    unmount();
    expect(spy).toHaveBeenCalled();
  });
});
