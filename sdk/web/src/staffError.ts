/**
 * Localize ThaiIDError.message for clinic-staff chrome.
 * Protocol `code` / retryable / requestId stay unchanged.
 */

import { ThaiIDError } from "./index.js";
import { formatStaffError, staffLang } from "./staffCopy.js";

export function localizeThaiIDError(err: unknown, lang?: string | null): unknown {
  if (!(err instanceof ThaiIDError)) return err;
  const message = formatStaffError(err, staffLang(lang));
  if (err.message === message) return err;
  return new ThaiIDError(err.code, message, err.retryable, err.requestId);
}
