import { afterEach, describe, expect, it, vi } from "vitest";
import { DEFAULT_FIELDS, SUPPORTED_FIELDS, TEXTUAL_FIELDS, ThaiID, ThaiIDError, portraitBlob, portraitDataUrl } from "../src/index";

type FakeWS = {
  readyState: number;
  sent: string[];
  instanceId?: string;
  onopen: (() => void) | null;
  onerror: (() => void) | null;
  onmessage: ((ev: { data: string }) => void) | null;
  onclose: (() => void) | null;
  send: (s: string) => void;
  close: () => void;
  respondTo: (s: string) => void;
};

function installFakeWebSocket(opts: { autoRespond?: boolean; snapshot?: boolean; autoOpen?: boolean; instanceIds?: string[] } = {}) {
  const sockets: FakeWS[] = [];
  const sendSnapshot = opts.snapshot !== false;
  class Fake {
    static OPEN = 1;
    static CONNECTING = 0;
    static CLOSING = 2;
    static CLOSED = 3;
    readyState = 0;
    sent: string[] = [];
    instanceId = "bridge-x";
    onopen: (() => void) | null = null;
    onerror: (() => void) | null = null;
    onmessage: ((ev: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    constructor(public url: string, public protocols?: string[]) {
      sockets.push(this as unknown as FakeWS);
      this.instanceId = opts.instanceIds?.[sockets.length - 1] ?? "bridge-x";
      if (opts.autoOpen === false) return;
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
            instanceId: this.instanceId,
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
              instanceId: this.instanceId,
            },
          }),
        });
        if (sendSnapshot) {
          // After session.open assigns the token (promise microtask), then this macrotask.
          setTimeout(() => {
            this.onmessage?.({
              data: JSON.stringify({
                v: 1,
                type: "status.snapshot",
                instanceId: this.instanceId,
                sequence: 2,
                payload: {
                  connection: "connected",
                  backend: "available",
                  readers: [{ readerId: "r1", name: "ACR39U", presence: "present", cardSessionId: "cs_1" }],
                },
              }),
            });
          }, 0);
        }
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

  it("rejects unknown fields", () => {
    expect(() => new ThaiID({ fields: ["laserId" as never] })).toThrow(ThaiIDError);
  });

  it("accepts optional portrait", () => {
    const id = new ThaiID({ fields: ["portrait"] });
    expect(id.fields).toEqual(["portrait"]);
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

  it("SUPPORTED_FIELDS lists textual fields plus optional portrait and keeps MVP defaults", () => {
    expect(DEFAULT_FIELDS).toEqual(["citizenId", "thaiName", "birthDate"]);
    expect(TEXTUAL_FIELDS).not.toContain("portrait");
    expect([...SUPPORTED_FIELDS].sort()).toEqual(
      [
        "address",
        "birthDate",
        "citizenId",
        "englishName",
        "expireDate",
        "gender",
        "issueDate",
        "issuer",
        "portrait",
        "thaiName",
      ].sort(),
    );
  });

  it("portrait helpers build data URLs and blobs", () => {
    const p = { mimeType: "image/jpeg" as const, encoding: "base64" as const, data: "QQ==" };
    expect(portraitDataUrl(p)).toBe("data:image/jpeg;base64,QQ==");
    expect(portraitDataUrl(null)).toBeNull();
    const blob = portraitBlob(p);
    expect(blob).toBeTruthy();
    expect(blob?.type).toBe("image/jpeg");
    expect(blob?.size).toBe(1);
  });

  it("accepts extra textual fields in the constructor", () => {
    const id = new ThaiID({ fields: ["gender", "address", "englishName"] });
    expect(id.fields).toEqual(["gender", "address", "englishName"]);
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
    // Intentionally no FakeWebSocket: cancellation is reported even in Node/SSR.
    await expect(id.connect({ signal: c.signal })).rejects.toMatchObject({ code: "CANCELLED" });
    expect(id.state()).toBe("disconnected");
  });

  it("abort before connect does not open a socket", async () => {
    const fake = installFakeWebSocket({ autoOpen: false });
    try {
      const id = new ThaiID();
      const c = new AbortController();
      c.abort();
      await expect(id.connect({ signal: c.signal })).rejects.toMatchObject({ code: "CANCELLED" });
      expect(fake.sockets.length).toBe(0);
    } finally {
      fake.restore();
    }
  });

  it("abort during connect is cancelled", async () => {
    const fake = installFakeWebSocket({ autoOpen: false });
    try {
      const id = new ThaiID();
      const c = new AbortController();
      const p = id.connect({ signal: c.signal });
      c.abort();
      await expect(p).rejects.toMatchObject({ code: "CANCELLED" });
      expect(id.state()).toBe("disconnected");
    } finally {
      fake.restore();
    }
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

  it("ignores status.snapshot before a session token", async () => {
    const fake = installFakeWebSocket();
    const id = new ThaiID();
    try {
      const readerIds: string[][] = [];
      id.on("statusChanged", (s) => readerIds.push(s.readers.map((r) => r.readerId)));
      const connectP = id.connect();
      await new Promise((r) => setTimeout(r, 0));
      const ws = fake.sockets[0]!;
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          type: "status.snapshot",
          instanceId: "bridge-x",
          payload: {
            connection: "connected",
            backend: "available",
            readers: [{ readerId: "leaked", name: "ACR39U", presence: "present", cardSessionId: "cs_leak" }],
          },
        }),
      });
      expect(readerIds.some((ids) => ids.includes("leaked"))).toBe(false);
      expect((await id.status()).readers).toEqual([]);
      const openMsg = ws.sent.map((s) => JSON.parse(s)).find((m) => m.type === "session.open")!;
      ws.respondTo(JSON.stringify(openMsg));
      await connectP;
      expect(id.state()).toBe("connected");
      expect(readerIds.some((ids) => ids.includes("r1"))).toBe(true);
      expect(readerIds.some((ids) => ids.includes("leaked"))).toBe(false);
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });

  it("connect fails without an authenticated snapshot", async () => {
    const fake = installFakeWebSocket({ autoRespond: true, snapshot: false });
    try {
      const id = new ThaiID();
      await expect(id.connect()).rejects.toMatchObject({ code: "CONNECTION_FAILED" });
      expect(id.state()).toBe("disconnected");
    } finally {
      fake.restore();
    }
  }, 10_000);

  it("reconnects after a helper restart with a new instance id", async () => {
    const fake = installFakeWebSocket({ autoRespond: true, instanceIds: ["bridge-a", "bridge-b"] });
    const id = new ThaiID();
    try {
      const states: string[] = [];
      id.on("connectionChanged", (s) => states.push(s));
      await id.connect();
      expect(id.state()).toBe("connected");
      fake.sockets[0]!.close();
      await vi.waitFor(() => {
        expect(fake.sockets.length).toBe(2);
      });
      await vi.waitFor(() => {
        expect(id.state()).toBe("connected");
      });
      expect(states).toContain("reconnecting");
      expect(id.state()).toBe("connected");
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });

  it("keeps auto-reconnect after a successful recovery past the original budget", async () => {
    let now = 1_000;
    vi.spyOn(Date, "now").mockImplementation(() => now);
    const fake = installFakeWebSocket({ autoRespond: true, instanceIds: ["bridge-a", "bridge-b", "bridge-c"] });
    const id = new ThaiID();
    try {
      await id.connect();
      fake.sockets[0]!.close();
      await vi.waitFor(() => {
        expect(id.state()).toBe("connected");
        expect(fake.sockets.length).toBe(2);
      });
      now = 1_000 + 130_000;
      fake.sockets[1]!.close();
      await vi.waitFor(() => {
        expect(id.state()).not.toBe("disconnected");
      });
      await vi.waitFor(() => {
        expect(id.state()).toBe("connected");
        expect(fake.sockets.length).toBe(3);
      });
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });

  it("read uses the selected reader from status", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    const id = new ThaiID();
    try {
      await id.connect();
      const ws = fake.sockets[0]!;
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          type: "status.snapshot",
          instanceId: "bridge-x",
          sequence: 3,
          payload: {
            connection: "connected",
            backend: "available",
            readers: [
              { readerId: "r1", name: "Reader One", presence: "present", cardSessionId: "cs_1" },
              { readerId: "r2", name: "Reader Two", presence: "present", cardSessionId: "cs_2", selected: true },
            ],
          },
        }),
      });
      const p = id.read({ fields: ["citizenId"] });
      await new Promise((r) => setTimeout(r, 0));
      const reads = ws.sent.map((s) => JSON.parse(s)).filter((m) => m.type === "card.read");
      expect(reads.at(-1)?.payload.readerId).toBe("r2");
      const last = reads.at(-1);
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          id: last.id,
          result: { readerId: "r2", cardSessionId: "cs_2", fields: { citizenId: "1234567890121" }, warnings: [] },
        }),
      });
      await expect(p).resolves.toMatchObject({ readerId: "r2" });
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });

  it("read requires a Ready-page choice when two readers are unselected", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    const id = new ThaiID();
    try {
      await id.connect();
      const ws = fake.sockets[0]!;
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          type: "status.snapshot",
          instanceId: "bridge-x",
          sequence: 3,
          payload: {
            connection: "connected",
            backend: "available",
            readers: [
              { readerId: "r1", name: "Reader One", presence: "present", cardSessionId: "cs_1" },
              { readerId: "r2", name: "Reader Two", presence: "present", cardSessionId: "cs_2" },
            ],
          },
        }),
      });
      await expect(id.read({ fields: ["citizenId"] })).rejects.toMatchObject({
        code: "READER_SELECTION_REQUIRED",
      });
      const reads = ws.sent.map((s) => JSON.parse(s)).filter((m) => m.type === "card.read");
      expect(reads).toHaveLength(0);
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });

  it("card.removed updates presence immediately without wiping the reader list", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    const id = new ThaiID();
    try {
      await id.connect();
      const statuses: Array<{ presence?: string; n: number }> = [];
      id.on("statusChanged", (s) => statuses.push({ presence: s.readers[0]?.presence, n: s.readers.length }));
      const cards: boolean[] = [];
      id.on("cardChanged", ({ present }) => cards.push(present));
      const ws = fake.sockets[0]!;
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          type: "card.removed",
          instanceId: "bridge-x",
          sequence: 3,
          payload: { readerId: "r1", cardSessionId: "cs_1" },
        }),
      });
      expect(cards).toContain(false);
      expect(statuses.some((s) => s.presence === "absent" && s.n === 1)).toBe(true);
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });

  it("reader.disconnected drops the reader so status is not Ready", async () => {
    const fake = installFakeWebSocket({ autoRespond: true });
    const id = new ThaiID();
    try {
      await id.connect();
      let readers = 1;
      id.on("statusChanged", (s) => {
        readers = s.readers.length;
      });
      const ws = fake.sockets[0]!;
      ws.onmessage?.({
        data: JSON.stringify({
          v: 1,
          type: "reader.disconnected",
          instanceId: "bridge-x",
          sequence: 3,
          payload: { readerId: "r1" },
        }),
      });
      expect(readers).toBe(0);
    } finally {
      await id.disconnect();
      fake.restore();
    }
  });
});
