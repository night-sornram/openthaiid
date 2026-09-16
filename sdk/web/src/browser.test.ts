import { afterEach, describe, expect, it } from "vitest";
import {
  firefoxLike,
  insecureLoopbackWsBlocked,
  replaceElementChildren,
  safariLike,
  setPageWsContextForTests,
} from "./browser";

const FIREFOX = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0";
const SAFARI =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15";
const CHROME =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36";
const EDGE =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0";

afterEach(() => {
  setPageWsContextForTests(null);
});

describe("insecureLoopbackWsBlocked", () => {
  it("blocks Firefox and Safari HTTPS pages talking to ws:// loopback", () => {
    const endpoint = "ws://127.0.0.1:17891/v1/ws";
    expect(
      insecureLoopbackWsBlocked({ pageProtocol: "https:", userAgent: FIREFOX, endpoint }),
    ).toBe(true);
    expect(
      insecureLoopbackWsBlocked({ pageProtocol: "https:", userAgent: SAFARI, endpoint }),
    ).toBe(true);
  });

  it("allows Chromium HTTPS (localhost mixed-content exception)", () => {
    const endpoint = "ws://127.0.0.1:17891/v1/ws";
    expect(
      insecureLoopbackWsBlocked({ pageProtocol: "https:", userAgent: CHROME, endpoint }),
    ).toBe(false);
    expect(
      insecureLoopbackWsBlocked({ pageProtocol: "https:", userAgent: EDGE, endpoint }),
    ).toBe(false);
  });

  it("allows HTTP clinic pages and wss://", () => {
    expect(
      insecureLoopbackWsBlocked({
        pageProtocol: "http:",
        userAgent: FIREFOX,
        endpoint: "ws://127.0.0.1:17891/v1/ws",
      }),
    ).toBe(false);
    expect(
      insecureLoopbackWsBlocked({
        pageProtocol: "https:",
        userAgent: FIREFOX,
        endpoint: "wss://127.0.0.1:17891/v1/ws",
      }),
    ).toBe(false);
  });

  it("classifies UA strings without treating Chrome as Safari", () => {
    expect(firefoxLike(FIREFOX)).toBe(true);
    expect(safariLike(SAFARI)).toBe(true);
    expect(safariLike(CHROME)).toBe(false);
    expect(safariLike(EDGE)).toBe(false);
    expect(firefoxLike(CHROME)).toBe(false);
  });
});

describe("replaceElementChildren", () => {
  it("clears and appends when replaceChildren is missing", () => {
    const root = document.createElement("div");
    root.append(document.createElement("span"));
    const fallback = root as Element & { replaceChildren?: unknown };
    const prev = fallback.replaceChildren;
    fallback.replaceChildren = undefined;
    try {
      const p = document.createElement("p");
      p.textContent = "ok";
      replaceElementChildren(root, p);
      expect(root.childElementCount).toBe(1);
      expect(root.textContent).toBe("ok");
    } finally {
      fallback.replaceChildren = prev;
    }
  });

  it("uses native replaceChildren when present", () => {
    const root = document.createElement("div");
    root.innerHTML = "<span>old</span>";
    replaceElementChildren(root);
    expect(root.childNodes.length).toBe(0);
  });
});
