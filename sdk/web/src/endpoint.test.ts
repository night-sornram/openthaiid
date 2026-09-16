import { afterEach, describe, expect, it } from "vitest";
import { setPageWsContextForTests } from "./browser";
import {
  DEFAULT_ENDPOINT,
  DEFAULT_TLS_ENDPOINT,
  isLoopbackWss,
  loopbackTlsFailureCode,
  resolveHelperEndpoint,
} from "./endpoint";

const FIREFOX = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0";
const SAFARI =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15";
const CHROME =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36";

afterEach(() => {
  setPageWsContextForTests(null);
});

describe("resolveHelperEndpoint", () => {
  it("keeps plain ws for http pages and Chrome https", () => {
    expect(resolveHelperEndpoint({ pageProtocol: "http:", userAgent: FIREFOX })).toBe(DEFAULT_ENDPOINT);
    expect(resolveHelperEndpoint({ pageProtocol: "https:", userAgent: CHROME })).toBe(DEFAULT_ENDPOINT);
  });

  it("prefers well-known wss on Firefox/Safari https pages", () => {
    expect(resolveHelperEndpoint({ pageProtocol: "https:", userAgent: FIREFOX })).toBe(DEFAULT_TLS_ENDPOINT);
    expect(resolveHelperEndpoint({ pageProtocol: "https:", userAgent: SAFARI })).toBe(DEFAULT_TLS_ENDPOINT);
  });

  it("uses advertised wssUrl when helper TLS is on", () => {
    expect(
      resolveHelperEndpoint({
        pageProtocol: "https:",
        userAgent: CHROME,
        helperTls: true,
        wssUrl: "wss://127.0.0.1:19000/v1/ws",
      }),
    ).toBe("wss://127.0.0.1:19000/v1/ws");
  });

  it("honors preferLoopbackTls on Chrome https", () => {
    expect(
      resolveHelperEndpoint({
        pageProtocol: "https:",
        userAgent: CHROME,
        preferLoopbackTls: true,
      }),
    ).toBe(DEFAULT_TLS_ENDPOINT);
  });

  it("lets an explicit endpoint win", () => {
    expect(
      resolveHelperEndpoint({
        endpoint: "ws://127.0.0.1:19000/v1/ws",
        pageProtocol: "https:",
        userAgent: FIREFOX,
        preferLoopbackTls: true,
      }),
    ).toBe("ws://127.0.0.1:19000/v1/ws");
  });
});

describe("loopbackTlsFailureCode", () => {
  it("keeps HTTPS_WS_BLOCKED for mixed-content ws://", () => {
    setPageWsContextForTests({ pageProtocol: "https:", userAgent: FIREFOX });
    expect(loopbackTlsFailureCode("ws://127.0.0.1:17891/v1/ws")).toBe("HTTPS_WS_BLOCKED");
  });

  it("maps auto Firefox wss failure to HTTPS_WS_BLOCKED", () => {
    setPageWsContextForTests({ pageProtocol: "https:", userAgent: FIREFOX });
    expect(loopbackTlsFailureCode(DEFAULT_TLS_ENDPOINT)).toBe("HTTPS_WS_BLOCKED");
  });

  it("maps asserted TLS failures to TLS_UNTRUSTED", () => {
    setPageWsContextForTests({ pageProtocol: "https:", userAgent: FIREFOX });
    expect(loopbackTlsFailureCode(DEFAULT_TLS_ENDPOINT, { preferLoopbackTls: true })).toBe("TLS_UNTRUSTED");
    expect(loopbackTlsFailureCode(DEFAULT_TLS_ENDPOINT, { explicitEndpoint: true })).toBe("TLS_UNTRUSTED");
  });

  it("recognizes loopback wss URLs", () => {
    expect(isLoopbackWss(DEFAULT_TLS_ENDPOINT)).toBe(true);
    expect(isLoopbackWss("wss://localhost:17892/v1/ws")).toBe(true);
    expect(isLoopbackWss(DEFAULT_ENDPOINT)).toBe(false);
  });
});
