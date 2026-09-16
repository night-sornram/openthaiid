/**
 * Cheap, correct guards for clinic ERP pages on Firefox / Safari.
 * Chromium still has a localhost mixed-content exception; Firefox and
 * Safari do not. Do not tell clinics to disable mixed-content checks.
 */

export type PageWsContext = {
  pageProtocol?: string;
  userAgent?: string;
  endpoint?: string;
};

export function firefoxLike(ua: string): boolean {
  return /Firefox\//i.test(ua) || /FxiOS\//i.test(ua);
}

export function safariLike(ua: string): boolean {
  if (!/Safari\//i.test(ua)) return false;
  if (/Chrome|Chromium|CriOS|Edg|OPR|Android/i.test(ua)) return false;
  return true;
}

/**
 * True when this HTTPS page will not be allowed to open `ws://` to the
 * helper (Firefox mixed-content, Safari mixed-content). Chromium allows
 * loopback mixed WebSockets; this returns false there.
 */
export function insecureLoopbackWsBlocked(ctx: PageWsContext = {}): boolean {
  const protocol = ctx.pageProtocol ?? "";
  const endpoint = ctx.endpoint ?? "";
  const ua = ctx.userAgent ?? "";
  if (protocol !== "https:") return false;
  if (!/^ws:\/\//i.test(endpoint)) return false;
  return firefoxLike(ua) || safariLike(ua);
}

let pageContextOverride: PageWsContext | null = null;

/** Test-only. Not re-exported from the package entry. */
export function setPageWsContextForTests(ctx: PageWsContext | null): void {
  pageContextOverride = ctx;
}

export function pageBlocksInsecureLoopbackWs(endpoint: string): boolean {
  if (pageContextOverride) {
    return insecureLoopbackWsBlocked({
      pageProtocol: pageContextOverride.pageProtocol,
      userAgent: pageContextOverride.userAgent,
      endpoint: pageContextOverride.endpoint ?? endpoint,
    });
  }
  const protocol = typeof location !== "undefined" ? location.protocol : "";
  const ua = typeof navigator !== "undefined" ? navigator.userAgent : "";
  return insecureLoopbackWsBlocked({ pageProtocol: protocol, userAgent: ua, endpoint });
}

export function currentPageProtocol(): string {
  if (pageContextOverride?.pageProtocol) return pageContextOverride.pageProtocol;
  return typeof location !== "undefined" ? location.protocol : "";
}

export function currentUserAgent(): string {
  if (pageContextOverride?.userAgent) return pageContextOverride.userAgent;
  return typeof navigator !== "undefined" ? navigator.userAgent : "";
}

/** `Element.replaceChildren` is missing on some older Safari engines. */
export function replaceElementChildren(el: Element, ...nodes: Node[]): void {
  const anyEl = el as Element & { replaceChildren?: (...args: Node[]) => void };
  if (typeof anyEl.replaceChildren === "function") {
    anyEl.replaceChildren(...nodes);
    return;
  }
  while (el.firstChild) {
    el.removeChild(el.firstChild);
  }
  for (const node of nodes) {
    el.appendChild(node);
  }
}
