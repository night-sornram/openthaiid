import { afterEach, describe, expect, it, vi } from "vitest";
import { ThaiIDError } from "../src/index";
import {
  formatTestConnection,
  managerUrlFromEndpoint,
  mountThaiIDStatus,
  probeThaiID,
  summarizeHelperWatch,
  summarizeProbe,
  waitForThaiID,
  watchThaiID,
} from "../src/probe";

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

type ProbeMode = "probe" | "fail-open" | "hang";

function installFakeWebSocket(mode: ProbeMode | ProbeMode[] | { current: ProbeMode } = "probe") {
  const sockets: FakeWS[] = [];
  const currentMode = (): ProbeMode => {
    if (typeof mode === "string") return mode;
    if (Array.isArray(mode)) return mode[Math.min(sockets.length - 1, mode.length - 1)] ?? "probe";
    return mode.current;
  };
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
      const current = currentMode();
      if (current === "hang") return;
      queueMicrotask(() => {
        if (current === "fail-open") {
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
      expect(r.error?.message).toContain("ยังไม่ทำงาน");
      expect(r.error?.message).not.toContain("helper is not running");
    } finally {
      fake.restore();
    }
  });

  it("English probe errors keep BRIDGE_UNAVAILABLE", async () => {
    const fake = installFakeWebSocket("fail-open");
    try {
      const r = await probeThaiID({ timeoutMs: 200, lang: "en" });
      expect(r.error?.code).toBe("BRIDGE_UNAVAILABLE");
      expect(r.error?.message).toMatch(/not running/i);
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
      expect(r.error?.message).toContain("ยกเลิก");
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
  it("defaults to Thai staff copy with no protocol jargon", () => {
    const down = summarizeProbe({ reachable: false });
    expect(down.headline).toBe("ยังไม่พร้อม");
    expect(down.helper).toContain("ยังไม่ทำงาน");
    expect(JSON.stringify(down)).not.toMatch(/PC\/SC|WebSocket|bridge/i);

    const ready = summarizeProbe({
      reachable: true,
      synthetic: true,
      backend: { name: "synthetic", status: "available" },
      readers: [{ readerId: "r1", name: "SYNTHETIC ACR39U", presence: "present", selected: true }],
    });
    expect(ready.ready).toBe(true);
    expect(ready.headline).toBe("พร้อม (บัตรฝึกซ้อม)");
    expect(ready.next).toContain("กรอกข้อมูล");
  });

  it("English fallback when lang is en", () => {
    const down = summarizeProbe({ reachable: false }, "en");
    expect(down.headline).toBe("Not ready yet");
    expect(down.helper).toContain("not running");
    const ready = summarizeProbe(
      {
        reachable: true,
        synthetic: true,
        backend: { name: "synthetic", status: "available" },
        readers: [{ readerId: "r1", name: "SYNTHETIC ACR39U", presence: "present", selected: true }],
      },
      "en",
    );
    expect(ready.headline).toContain("Ready");
    expect(ready.next).toContain("Fill");
  });

  it("waits for a Ready-page choice when two readers are unselected", () => {
    const two = summarizeProbe({
      reachable: true,
      backend: { name: "synthetic", status: "available" },
      readers: [
        { readerId: "r1", name: "Reader One", presence: "present" },
        { readerId: "r2", name: "Reader Two", presence: "present" },
      ],
    });
    expect(two.ready).toBe(false);
    expect(two.next).toContain("เลือกเครื่อง");
    expect(JSON.stringify(two)).not.toMatch(/PC\/SC|WebSocket|bridge/i);

    const picked = summarizeProbe({
      reachable: true,
      backend: { name: "synthetic", status: "available" },
      readers: [
        { readerId: "r1", name: "Reader One", presence: "present" },
        { readerId: "r2", name: "Reader Two", presence: "present", selected: true },
      ],
    });
    expect(picked.ready).toBe(true);
    expect(picked.reader).toContain("Reader Two");
  });

  it("card absent is not Ready", () => {
    const gone = summarizeProbe({
      reachable: true,
      backend: { name: "synthetic", status: "available" },
      readers: [{ readerId: "r1", name: "ACR39U", presence: "absent", selected: true }],
    });
    expect(gone.ready).toBe(false);
    expect(gone.headline).toBe("ยังไม่พร้อม");
    expect(gone.card).toContain("บัตร");
    expect(gone.next).toContain("กรอกข้อมูล");
  });

  it("no readers after unplug is not Ready", () => {
    const gone = summarizeProbe({
      reachable: true,
      backend: { name: "synthetic", status: "available" },
      readers: [],
    });
    expect(gone.ready).toBe(false);
    expect(gone.headline).toBe("ยังไม่พร้อม");
    expect(gone.next).toContain("เครื่องอ่าน");
  });
});

describe("mountThaiIDStatus", () => {
  it("renders helper + reader state in user language", async () => {
    const fake = installFakeWebSocket();
    const root = document.createElement("div");
    try {
      const stop = mountThaiIDStatus(root, { intervalMs: 60_000 });
      await vi.waitFor(() => {
        expect(root.textContent).toContain("พร้อม");
        expect(root.textContent).toContain("SYNTHETIC ACR39U");
        expect(root.textContent).toContain("เสียบบัตรแล้ว");
      });
      expect(root.textContent).not.toContain("1234567890121");
      expect(root.textContent).not.toContain("PC/SC");
      expect(root.querySelector("[data-openthaiid-status-mode='user']")).not.toBeNull();
      expect(root.querySelector("[data-openthaiid-lang='th']")).not.toBeNull();
      expect(root.querySelector("[data-openthaiid-test-connection]")).not.toBeNull();
      expect(root.textContent).toContain("ทดสอบการเชื่อมต่อ");
      stop();
    } finally {
      fake.restore();
    }
  });

  it("English fallback on the status mount", async () => {
    const fake = installFakeWebSocket();
    const root = document.createElement("div");
    try {
      const stop = mountThaiIDStatus(root, { intervalMs: 60_000, lang: "en" });
      await vi.waitFor(() => {
        expect(root.textContent).toContain("Ready");
        expect(root.querySelector("[data-openthaiid-lang='en']")).not.toBeNull();
      });
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

describe("waitForThaiID", () => {
  it("returns immediately when the helper is already up", async () => {
    const fake = installFakeWebSocket();
    try {
      const r = await waitForThaiID({ timeoutMs: 2_000 });
      expect(r.reachable).toBe(true);
      expect(r.readers?.[0]?.presence).toBe("present");
      expect(JSON.stringify(r)).not.toContain("1234567890121");
    } finally {
      fake.restore();
    }
  });

  it("retries until the helper accepts a probe", async () => {
    const fake = installFakeWebSocket(["fail-open", "fail-open", "probe"]);
    try {
      const r = await waitForThaiID({ timeoutMs: 5_000, intervalMs: 20, probeTimeoutMs: 200 });
      expect(r.reachable).toBe(true);
      expect(fake.sockets.length).toBeGreaterThanOrEqual(3);
    } finally {
      fake.restore();
    }
  });
});

describe("summarizeHelperWatch / formatTestConnection", () => {
  it("uses Thai reconnect copy without protocol jargon or PII", () => {
    const waiting = summarizeHelperWatch({ reachable: false }, "waiting");
    expect(waiting.headline).toBe("ตัวช่วยหลุดการเชื่อมต่อ");
    expect(waiting.next).toContain("รอ");
    expect(JSON.stringify(waiting)).not.toMatch(/PC\/SC|WebSocket|bridge/i);
    expect(JSON.stringify(waiting)).not.toContain("1234567890121");

    const back = summarizeHelperWatch(
      {
        reachable: true,
        synthetic: true,
        backend: { name: "synthetic", status: "available" },
        readers: [{ readerId: "r1", name: "SYNTHETIC ACR39U", presence: "present", selected: true }],
      },
      "reconnected",
    );
    expect(back.headline).toBe("ตัวช่วยกลับมาแล้ว");
    expect(back.next).toContain("กรอกข้อมูล");

    expect(formatTestConnection({ reachable: false })).toContain("ไม่สำเร็จ");
    expect(formatTestConnection({ reachable: true, readers: [] })).toContain("ไม่ได้อ่านข้อมูลบนบัตร");
  });
});

describe("watchThaiID", () => {
  it("announces helper loss and recovery", async () => {
    const box = { current: "probe" as ProbeMode };
    const fake = installFakeWebSocket(box);
    const phases: string[] = [];
    try {
      const stop = watchThaiID(
        (u) => {
          phases.push(u.phase);
        },
        { intervalMs: 40, offlineIntervalMs: 40, timeoutMs: 200 },
      );
      await vi.waitFor(() => {
        expect(phases).toContain("online");
      });
      box.current = "fail-open";
      await vi.waitFor(() => {
        expect(phases).toContain("waiting");
      });
      box.current = "probe";
      await vi.waitFor(() => {
        expect(phases).toContain("reconnected");
      });
      stop();
    } finally {
      fake.restore();
    }
  });
});

describe("mountThaiIDStatus test connection", () => {
  it("runs a probe-only test from the button", async () => {
    const fake = installFakeWebSocket();
    const root = document.createElement("div");
    try {
      const stop = mountThaiIDStatus(root, { intervalMs: 60_000 });
      await vi.waitFor(() => {
        expect(root.querySelector("[data-openthaiid-test-connection]")).not.toBeNull();
      });
      const btn = root.querySelector("[data-openthaiid-test-connection]") as HTMLButtonElement;
      btn.click();
      await vi.waitFor(() => {
        expect(root.querySelector("[data-openthaiid-test-result]")?.textContent).toContain("ไม่ได้อ่านข้อมูลบนบัตร");
      });
      expect(root.textContent).not.toContain("1234567890121");
      expect(fake.sockets.some((s) => s.sent.some((m) => m.includes("bridge.probe")))).toBe(true);
      expect(fake.sockets.every((s) => s.sent.every((m) => !m.includes("session.open")))).toBe(true);
      expect(fake.sockets.every((s) => s.sent.every((m) => !m.includes("card.read")))).toBe(true);
      stop();
    } finally {
      fake.restore();
    }
  });
});
