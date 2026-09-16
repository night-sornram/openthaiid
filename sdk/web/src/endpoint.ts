/**
 * Loopback helper URL selection for clinic ERP pages.
 *
 * Default remains `ws://127.0.0.1:17891/v1/ws` (http ERP / Ready / Chrome).
 * HTTPS pages prefer `wss://127.0.0.1:17892/v1/ws` when the helper advertises
 * TLS or the ERP sets `preferLoopbackTls`. Firefox/Safari https pages also
 * try that well-known WSS port so a technician-managed local cert can unblock
 * Fill without disabling mixed-content checks.
 */

import {
  currentPageProtocol,
  currentUserAgent,
  firefoxLike,
  pageBlocksInsecureLoopbackWs,
  safariLike,
} from "./browser.js";

export const DEFAULT_ENDPOINT = "ws://127.0.0.1:17891/v1/ws";
export const DEFAULT_TLS_ENDPOINT = "wss://127.0.0.1:17892/v1/ws";
export const DEFAULT_TLS_PORT = 17892;

export type ResolveHelperEndpointOptions = {
  /** Explicit WebSocket URL. Always wins when non-empty. */
  endpoint?: string;
  /**
   * ERP/clinic flag: this PC has technician-managed loopback HTTPS.
   * HTTPS pages then use WSS even in Chrome.
   */
  preferLoopbackTls?: boolean;
  /** True when a previous probe/healthz advertised helper TLS. */
  helperTls?: boolean;
  /** Advertised WSS URL from the helper (optional). */
  wssUrl?: string;
  pageProtocol?: string;
  userAgent?: string;
};

export function isLoopbackWss(endpoint: string): boolean {
  try {
    const u = new URL(endpoint);
    if (u.protocol !== "wss:") return false;
    const host = u.hostname.toLowerCase();
    return host === "127.0.0.1" || host === "localhost";
  } catch {
    return /^wss:\/\/(127\.0\.0\.1|localhost)[:/]/i.test(endpoint);
  }
}

export function resolveHelperEndpoint(opts: ResolveHelperEndpointOptions = {}): string {
  const explicit = opts.endpoint?.trim();
  if (explicit) return explicit;

  const protocol = opts.pageProtocol ?? currentPageProtocol();
  const ua = opts.userAgent ?? currentUserAgent();
  const https = protocol === "https:";
  const wantTls =
    https && (Boolean(opts.preferLoopbackTls) || Boolean(opts.helperTls) || firefoxLike(ua) || safariLike(ua));
  if (wantTls) {
    const advertised = opts.wssUrl?.trim();
    if (advertised) return advertised;
    return DEFAULT_TLS_ENDPOINT;
  }
  return DEFAULT_ENDPOINT;
}

export type LoopbackTlsFailureCode = "HTTPS_WS_BLOCKED" | "TLS_UNTRUSTED";

/**
 * Map a failed loopback connect/probe to a staff-facing code.
 * Untrusted local certs and a missing TLS listener look the same in JS;
 * `TLS_UNTRUSTED` is used when the ERP asserted TLS (flag or explicit WSS).
 */
export function loopbackTlsFailureCode(
  endpoint: string,
  opts: { preferLoopbackTls?: boolean; explicitEndpoint?: boolean } = {},
): LoopbackTlsFailureCode | null {
  if (pageBlocksInsecureLoopbackWs(endpoint)) return "HTTPS_WS_BLOCKED";
  if (!isLoopbackWss(endpoint)) return null;
  const protocol = currentPageProtocol();
  if (protocol !== "https:") return "TLS_UNTRUSTED";
  if (opts.preferLoopbackTls || opts.explicitEndpoint) return "TLS_UNTRUSTED";
  const ua = currentUserAgent();
  if (firefoxLike(ua) || safariLike(ua)) return "HTTPS_WS_BLOCKED";
  return "TLS_UNTRUSTED";
}
