import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ThaiIDError, autofillThaiID, type AutofillResult } from "@openthaiid/web";
import { useThaiIDAutofill } from "./index";

vi.mock("@openthaiid/web", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@openthaiid/web")>();
  return {
    ...actual,
    autofillThaiID: vi.fn(),
  };
});

const mockAutofill = vi.mocked(autofillThaiID);

const fakeResult: AutofillResult = {
  values: { citizenId: "1234567890121" },
  filled: [],
  missing: [],
  result: {
    readerId: "r1",
    cardSessionId: "cs_1",
    fields: { citizenId: "1234567890121" },
    warnings: [],
  },
};

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("useThaiIDAutofill", () => {
  it("does not call autofillThaiID on mount", () => {
    renderHook(() =>
      useThaiIDAutofill({
        request: ["citizenId", "thaiName"],
        map: { citizenId: "#patient-cid", thaiName: "#patient-name" },
      }),
    );
    expect(mockAutofill).not.toHaveBeenCalled();
  });

  it("passes the clinic config from a click-style autofill() call", async () => {
    mockAutofill.mockImplementation(async (options) => {
      options?.onApprovalRequired?.({
        approvalId: 3,
        comparisonCode: "654321",
        fields: ["citizenId"],
      });
      return fakeResult;
    });
    const config = {
      request: ["citizenId"],
      map: { citizenId: "#patient-cid" },
    };
    const { result } = renderHook(() => useThaiIDAutofill(config));

    await act(async () => {
      await result.current.autofill();
    });

    expect(mockAutofill).toHaveBeenCalledTimes(1);
    expect(mockAutofill.mock.calls[0]![0]).toMatchObject({
      request: ["citizenId"],
      map: { citizenId: "#patient-cid" },
    });
    expect(result.current.result?.values.citizenId).toBe("1234567890121");
    expect(result.current.approvalRequired).toBeNull();
    expect(result.current.filling).toBe(false);
  });

  it("surfaces approvalRequired while the helper is waiting", async () => {
    let finish!: (value: AutofillResult) => void;
    mockAutofill.mockImplementation(async (options) => {
      options?.onApprovalRequired?.({
        approvalId: 9,
        comparisonCode: "111222",
        fields: ["citizenId", "thaiName"],
      });
      return await new Promise<AutofillResult>((resolve) => {
        finish = resolve;
      });
    });
    const { result } = renderHook(() => useThaiIDAutofill({ request: ["citizenId"] }));

    let pending!: Promise<AutofillResult>;
    await act(async () => {
      pending = result.current.autofill();
    });

    await waitFor(() => {
      expect(result.current.filling).toBe(true);
      expect(result.current.approvalRequired).toEqual({
        approvalId: 9,
        comparisonCode: "111222",
        fields: ["citizenId", "thaiName"],
      });
    });

    await act(async () => {
      finish(fakeResult);
      await pending;
    });
    expect(result.current.filling).toBe(false);
    expect(result.current.approvalRequired).toBeNull();
  });

  it("records onError then still rejects", async () => {
    const err = new ThaiIDError("BRIDGE_UNAVAILABLE", "bridge is not running", true);
    mockAutofill.mockImplementation(async (options) => {
      options?.onError?.(err);
      throw err;
    });
    const { result } = renderHook(() => useThaiIDAutofill());
    await act(async () => {
      await expect(result.current.autofill()).rejects.toMatchObject({ code: "BRIDGE_UNAVAILABLE" });
    });
    expect(result.current.error?.code).toBe("BRIDGE_UNAVAILABLE");
    expect(result.current.filling).toBe(false);
  });
});
