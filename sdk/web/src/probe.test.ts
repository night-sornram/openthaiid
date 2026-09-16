import { afterEach, describe, expect, it, vi } from "vitest";
import { ThaiIDError } from "../src/index";
import { managerUrlFromEndpoint, mountThaiIDStatus, probeThaiID, summarizeProbe } from "../src/probe";

type FakeWS = {
  readyState: number;
  sent: string[];
  onopen: (() => void) | null;
  onerror: (() => void) | null;
  onmessage: ((ev: { data: string }) => void) | null;
  onclose: (() => void) | null;
  send: (s: string) => void;
  close: () => void;
};

function installFakeWebSocket(mode: "probe" | "fail-open" | "hang" = "probe") {
  const sockets: FakeWS[] = [];
  class Fake {
    static OPEN = 1;
    readyState = 0;
    sent: string[] = [];
    onopen: (() => void) | null = null;
    onerror: (() => void) | null = null;
    onmessage: ((ev: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    constructor(public url: string) {
      sockets.push(this as unknown as FakeWS);
      if (mode === "hang") return;
      queueMicrotask(() => {
        if (mode === "fail-open") {
          this.onerror?.();
          return;
        }
        this.readyState = 1;
        this.onopen?.();
      });
    }
    send(s: string) {
      this.sent.push(s);
      const msg = JSON.parse(s);
      queueMicrotask(() => {
        this.onmessage?.({
          data: JSON.stringify({
            v: 1,
            id: msg.id,
            result: {
              version: "0.0.1-test",
              synthetic: true,
              backend: { name: "synthetic", status: "available" },
              readers: [{ readerId: "r1", name: "SYNTHETIC ACR39U", presence: "present", selected: true }],
              managerUrl: "http://127.0.0.1:17891/",
            },
          }),
        });
      });
    }
    close() {
      this.readyState = 3;
      this.onclose?.();
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

describe("probeThaiID", () => {
  it("is SSR-safe without WebSocket", async () => {
    const g = globalThis as { WebSocket?: unknown };
    const prev = g.WebSocket;
    try {
      delete g.WebSocket;
      const r = await probeThaiID();
      expect(r.reachable).toBe(false);
      expect(r.error).toBeInstanceOf(ThaiIDError);
    } finally {
      g.WebSocket = prev;
    }
  });

  it("returns reader metadata and no personal fields", async () => {
    const fake = installFakeWebSocket();
    try {
      const r = await probeThaiID();
      expect(r.reachable).toBe(true);
      expect(r.synthetic).toBe(true);
      expect(r.readers?.[0]?.presence).toBe("present");
      expect(JSON.stringify(r)).not.toContain("1234567890121");
      expect(fake.sockets[0]!.sent.some((s) => s.includes("bridge.probe"))).toBe(true);
      expect(fake.sockets[0]!.sent.some((s) => s.includes("session.open"))).toBe(false);
    } finally {
      fake.restore();
    }
  });

  it("reports unreachable when the helper is down", async () => {
    const fake = installFakeWebSocket("fail-open");
    try {
      const r = await probeThaiID({ timeoutMs: 200 });
      expect(r.reachable).toBe(false);
      expect(r.error?.code).toBe("BRIDGE_UNAVAILABLE");
    } finally {
      fake.restore();
    }
  });

  it("honors abort", async () => {
    const fake = installFakeWebSocket("hang");
    try {
      const c = new AbortController();
      c.abort();
      const r = await probeThaiID({ signal: c.signal });
      expect(r.reachable).toBe(false);
      expect(r.error?.code).toBe("CANCELLED");
      expect(fake.sockets.length).toBe(0);
    } finally {
      fake.restore();
    }
  });
});

describe("managerUrlFromEndpoint", () => {
  it("maps the default ws endpoint to loopback http", () => {
    expect(managerUrlFromEndpoint()).toBe("http://127.0.0.1:17891/");
  });
});

describe("summarizeProbe", () => {
  it("uses plain language and no protocol jargon", () => {
    const down = summarizeProbe({ reachable: false });
    expect(down.headline).toBe("Not ready yet");
    expect(down.helper).toContain("not running");
    expect(JSON.stringify(down)).not.toMatch(/PC\/SC|WebSocket|bridge/i);

    const ready = summarizeProbe({
      reachable: true,
      synthetic: true,
      backend: { name: "synthetic", status: "available" },
      readers: [{ readerId: "r1", name: "SYNTHETIC ACR39U", presence: "present", selected: true }],
    });
    expect(ready.ready).toBe(true);
    expect(ready.headline).toContain("Ready");
    expect(ready.next).toContain("Fill");
  });
});

describe("mountThaiIDStatus", () => {
  it("renders helper + reader state in user language", async () => {
    const fake = installFakeWebSocket();
    const root = document.createElement("div");
    try {
      const stop = mountThaiIDStatus(root, { intervalMs: 60_000 });
      await vi.waitFor(() => {
        expect(root.textContent).toContain("Ready");
        expect(root.textContent).toContain("SYNTHETIC ACR39U");
        expect(root.textContent).toContain("Card is inserted");
      });
      expect(root.textContent).not.toContain("1234567890121");
      expect(root.textContent).not.toContain("PC/SC");
      expect(root.querySelector("[data-openthaiid-status-mode='user']")).not.toBeNull();
      stop();
    } finally {
      fake.restore();
    }
  });

  it("advanced mode shows technician backend details", async () => {
    const fake = installFakeWebSocket();
    const root = document.createElement("div");
    try {
      const stop = mountThaiIDStatus(root, { intervalMs: 60_000, mode: "advanced" });
      await vi.waitFor(() => {
        expect(root.textContent).toContain("Technician:");
        expect(root.textContent).toContain("card service");
      });
      stop();
    } finally {
      fake.restore();
    }
  });
});
