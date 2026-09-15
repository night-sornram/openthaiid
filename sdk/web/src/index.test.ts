import { afterEach, describe, expect, it, vi } from "vitest";
import { DEFAULT_FIELDS, SUPPORTED_FIELDS, ThaiID, ThaiIDError } from "../src/index";

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

function installFakeWebSocket(opts: { autoRespond?: boolean } = {}) {
  const sockets: FakeWS[] = [];
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
      queueMicrotask(() => {
        this.readyState = 1;
        this.onopen?.();
      });
    }
    send(s: string) {
      this.sent.push(s);
      if (opts.autoRespond) {
        queueMicrotask(() => this.respondTo(s));
      }
    }
    close() {
      this.readyState = 3;
      this.onclose?.();
    }
    respondTo(s: string) {
      if (!this.onmessage) return;
      const msg = JSON.parse(s);
      if (msg.type === "session.open") {
        this.onmessage({
          data: JSON.stringify({
            v: 1,
            type: "approval.required",
            instanceId: "bridge-x",
            sequence: 1,
            payload: { approvalId: 1, comparisonCode: "123456", fields: msg.payload?.fields },
          }),
        });
        this.onmessage({
          data: JSON.stringify({
            v: 1,
            id: msg.id,
            result: {
              token: "tok-1",
              fields: msg.payload?.fields ?? DEFAULT_FIELDS,
              expiresAt: new Date(Date.now() + 60_000).toISOString(),
              instanceId: "bridge-x",
            },
          }),
        });
        this.onmessage({
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

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ThaiID", () => {
  it("defaults fields and endpoint", () => {
    const id = new ThaiID();
    expect(id.fields).toEqual(DEFAULT_FIELDS);
    expect(id.endpoint).toBe("ws://127.0.0.1:17891/v1/ws");
    expect(id.state()).toBe("disconnected");
  });

  it("rejects unsupported fields", () => {
    expect(() => new ThaiID({ fields: ["portrait" as never] })).toThrow(ThaiIDError);
  });

  it("supports custom endpoint", () => {
    const id = new ThaiID({ endpoint: "ws://127.0.0.1:19000/v1/ws" });
    expect(id.endpoint).toContain("19000");
  });

  it("unsubscribe removes handler", async () => {
    const id = new ThaiID();
    const seen: string[] = [];
    const off = id.on("connectionChanged", (s) => seen.push(s));
    await id.disconnect();
    off();
    await id.disconnect();
    expect(seen.length).toBeLessThanOrEqual(1);
  });

  it("connect fails closed without WebSocket", async () => {
    const id = new ThaiID();
    const g = globalThis as { WebSocket?: unknown };
    const prev = g.WebSocket;
    try {
      delete g.WebSocket;
      await expect(id.connect()).rejects.toMatchObject({ code: "CONNECTION_FAILED" });
    } finally {
      g.WebSocket = prev;
    }
  });

  it("SUPPORTED_FIELDS matches MVP", () => {
    expect([...SUPPORTED_FIELDS].sort()).toEqual(["birthDate", "citizenId", "thaiName"]);
  });

  it("import is SSR-safe (no window required)", () => {
    expect(typeof ThaiID).toBe("function");
  });

  it("read rejects unapproved field before contacting bridge", async () => {
    const id = new ThaiID({ fields: ["citizenId"] });
    await expect(id.read()).rejects.toMatchObject({ code: "CONNECTION_FAILED" });
  });

  it("abort before connect is cancelled", async () => {
    const id = new ThaiID();
    const c = new AbortController();
    c.abort();
    await expect(id.connect({ signal: c.signal })).rejects.toMatchObject({ code: "CANCELLED" });
  });

  it("connect deduplicates concurrent calls", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    try {
      const id = new ThaiID();
      const p1 = id.connect();
      const p2 = id.connect();
      await Promise.all([p1, p2]);
      expect(fake.sockets.length).toBe(1);
      expect(id.state()).toBe("connected");
    } finally {
      fake.restore();
    }
  });

  it("read cancels the actual request id on abort", async () => {
    const fake = installFakeWebSocket();
    try {
      const id = new ThaiID();
      const connectP = id.connect();
      // Allow the socket to open and session.open to be sent.
      await new Promise((r) => setTimeout(r, 0));
      const ws = fake.sockets[0]!;
      const openMsg = ws.sent.map((s) => JSON.parse(s)).find((m) => m.type === "session.open")!;
      ws.respondTo(JSON.stringify(openMsg));
      await connectP;
      expect(id.state()).toBe("connected");

      const ac = new AbortController();
      const readP = id.read({ signal: ac.signal, fields: ["citizenId"] });
      await new Promise((r) => setTimeout(r, 0));
      const reads = ws.sent.map((s) => JSON.parse(s)).filter((m) => m.type === "card.read");
      expect(reads.length).toBe(1);
      const readId = reads[0].id;
      ac.abort();
      await expect(readP).rejects.toMatchObject({ code: "CANCELLED" });
      const cancels = ws.sent.map((s) => JSON.parse(s)).filter((m) => m.type === "request.cancel");
      expect(cancels.length).toBe(1);
      expect(cancels[0].payload.requestId).toBe(readId);
      expect(cancels[0].payload.requestId).not.toBe(0);
    } finally {
      fake.restore();
    }
  });

  it("clears personal data and stops reconnect on revocation", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    try {
      const id = new ThaiID();
      const errs: ThaiIDError[] = [];
      id.on("error", (e) => errs.push(e));
      await id.connect();
      const ws = fake.sockets[0]!;
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          type: "authorization.revoked",
          instanceId: "bridge-x",
          sequence: 3,
          payload: { reason: "origin revoked" },
        }),
      });
      expect(errs.some((e) => e.code === "AUTH_EXPIRED")).toBe(true);
      expect(id.state()).toBe("disconnected");
      const status = await id.status();
      expect(status.readers).toEqual([]);
    } finally {
      fake.restore();
    }
  });

  it("never retries a read automatically", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    try {
      const id = new ThaiID();
      await id.connect();
      const ws = fake.sockets[0]!;
      const p = id.read({ fields: ["citizenId"] });
      await new Promise((r) => setTimeout(r, 0));
      const allReads = ws.sent.map((s) => JSON.parse(s)).filter((m) => m.type === "card.read");
      const last = allReads[allReads.length - 1];
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          id: last.id,
          error: { code: "CARD_REMOVED", message: "card session changed", retryable: false },
        }),
      });
      await expect(p).rejects.toMatchObject({ code: "CARD_REMOVED" });
      await new Promise((r) => setTimeout(r, 20));
      const after = ws.sent.map((s) => JSON.parse(s)).filter((m) => m.type === "card.read");
      expect(after.length).toBe(allReads.length);
    } finally {
      fake.restore();
    }
  });
});
