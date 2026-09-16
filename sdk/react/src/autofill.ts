/**
 * One-shot Thai ID autofill for React ERP screens.
 *
 * Call `autofill()` from a click. Pass the same config object as
 * `autofillThaiID` (`request`, `map`, format, `onApprovalRequired`).
 */

import { useCallback, useRef, useState } from "react";
import {
  ThaiIDError,
  autofillThaiID,
  type AutofillResult,
  type AutofillThaiIDOptions,
  type Field,
} from "@openthaiid/web";

export type AutofillApproval = {
  approvalId: number;
  comparisonCode: string;
  fields: Field[];
};

export type UseThaiIDAutofillResult = {
  autofill: (overrides?: AutofillThaiIDOptions) => Promise<AutofillResult>;
  filling: boolean;
  approvalRequired: AutofillApproval | null;
  error: ThaiIDError | null;
  result: AutofillResult | null;
};

function isThaiIDError(err: unknown): err is ThaiIDError {
  return err instanceof ThaiIDError;
}

/**
 * Wrap `autofillThaiID` with filling / approval / error state.
 * Does not connect on mount.
 */
export function useThaiIDAutofill(options: AutofillThaiIDOptions = {}): UseThaiIDAutofillResult {
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const [filling, setFilling] = useState(false);
  const [approvalRequired, setApprovalRequired] = useState<AutofillApproval | null>(null);
  const [error, setError] = useState<ThaiIDError | null>(null);
  const [result, setResult] = useState<AutofillResult | null>(null);

  const autofill = useCallback(async (overrides: AutofillThaiIDOptions = {}) => {
    setFilling(true);
    setError(null);
    setApprovalRequired(null);
    const merged: AutofillThaiIDOptions = { ...optionsRef.current, ...overrides };
    try {
      const out = await autofillThaiID({
        ...merged,
        onApprovalRequired: (info) => {
          setApprovalRequired(info);
          merged.onApprovalRequired?.(info);
        },
        onError: (err) => {
          if (isThaiIDError(err)) setError(err);
          merged.onError?.(err);
        },
      });
      setApprovalRequired(null);
      setResult(out);
      return out;
    } catch (err) {
      if (isThaiIDError(err)) setError(err);
      throw err;
    } finally {
      setFilling(false);
    }
  }, []);

  return { autofill, filling, approvalRequired, error, result };
}
