/**
 * @openthaiid/react — React hooks wrapping `@openthaiid/web`.
 *
 * Call connect() from a click so the browser can prompt for local network
 * access. Importing this module is SSR-safe; connect() still requires a browser.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import {
  ThaiID,
  ThaiIDError,
  type BridgeStatus,
  type ConnectionState,
  type Field,
  type FieldValues,
  type ReadResult,
  type ThaiIDOptions,
} from "@openthaiid/web";

export type {
  BridgeStatus,
  CardDate,
  ConnectionState,
  Field,
  FieldValues,
  Gender,
  PersonName,
  Portrait,
  ReadResult,
  ThaiAddress,
  ThaiIDErrorCode,
  ThaiIDOptions,
  ThaiName,
} from "@openthaiid/web";

export {
  DEFAULT_FIELDS,
  SUPPORTED_FIELDS,
  TEXTUAL_FIELDS,
  ThaiID,
  ThaiIDError,
  AUTOFILL_CLINIC_CONFIG_KIND,
  CODE_ONLY_AUTOFILL_OPTION_KEYS,
  autofillThaiID,
  autofillClinicConfigDocument,
  clinicConfigToAutofillOptions,
  parseAutofillClinicConfig,
  serializeAutofillClinicConfig,
  probeThaiID,
  mountThaiIDStatus,
  summarizeProbe,
  summarizeHelperWatch,
  formatTestConnection,
  waitForThaiID,
  watchThaiID,
  staffLang,
  staffCopy,
  staffErrorGuidance,
  staffErrorCode,
  formatStaffError,
  staffApprovalGuidance,
  localizeThaiIDError,
  portraitDataUrl,
  portraitBlob,
} from "@openthaiid/web";
export type {
  AutofillClinicConfig,
  AutofillFieldMap,
  AutofillResult,
  AutofillThaiIDOptions,
  HelperWatchPhase,
  HelperWatchUpdate,
  ProbeResult,
  ProbeThaiIDOptions,
  StatusSummary,
  WaitForThaiIDOptions,
  WatchThaiIDOptions,
  StaffApprovalGuidance,
  StaffCopy,
  StaffErrorGuidance,
  StaffLang,
} from "@openthaiid/web";

export { useThaiIDAutofill } from "./autofill.js";
export type { AutofillApproval, UseThaiIDAutofillResult } from "./autofill.js";

export type ApprovalRequired = {
  approvalId: number;
  comparisonCode: string;
  fields: Field[];
};

/** Subset of ThaiID used by the hook (real client or a test double). */
export type ThaiIDClient = Pick<ThaiID, "on" | "connect" | "read" | "disconnect" | "status" | "state">;

export type UseThaiIDOptions = ThaiIDOptions & {
  /** Reuse an existing client instead of constructing one. */
  client?: ThaiIDClient;
};

export type UseThaiIDResult = {
  client: ThaiIDClient;
  connection: ConnectionState;
  status: BridgeStatus;
  /** Last successful `read()` values; cleared on disconnect, removal, or revoke. */
  fields: FieldValues | null;
  approvalRequired: ApprovalRequired | null;
  error: ThaiIDError | null;
  connect: (options?: { signal?: AbortSignal }) => Promise<void>;
  read: (options?: {
    fields?: Field[];
    readerId?: string;
    cardSessionId?: string;
    signal?: AbortSignal;
  }) => Promise<ReadResult>;
  disconnect: () => Promise<void>;
};

const INITIAL_STATUS: BridgeStatus = {
  connection: "disconnected",
  backend: "unavailable",
  readers: [],
};

function isThaiIDError(err: unknown): err is ThaiIDError {
  return err instanceof ThaiIDError;
}

/**
 * Subscribe to a local OpenThaiID bridge from React.
 *
 * Options (endpoint, field set, timeouts) are captured on first render unless
 * `client` is passed in. Pass AbortSignal through `connect` / `read`.
 */
export function useThaiID(options: UseThaiIDOptions = {}): UseThaiIDResult {
  const optionsRef = useRef(options);
  const clientRef = useRef<ThaiIDClient | null>(null);
  if (options.client) {
    clientRef.current = options.client;
  } else if (!clientRef.current) {
    clientRef.current = new ThaiID(optionsRef.current);
  }
  const client = options.client ?? clientRef.current;

  const [connection, setConnection] = useState<ConnectionState>(() => client.state());
  const [status, setStatus] = useState<BridgeStatus>(() => ({
    ...INITIAL_STATUS,
    connection: client.state(),
  }));
  const [fields, setFields] = useState<FieldValues | null>(null);
  const [approvalRequired, setApprovalRequired] = useState<ApprovalRequired | null>(null);
  const [error, setError] = useState<ThaiIDError | null>(null);

  useEffect(() => {
    const offConnection = client.on("connectionChanged", (state) => {
      setConnection(state);
      if (state === "connected" || state === "disconnected") {
        setApprovalRequired(null);
      }
      if (state === "disconnected") {
        setFields(null);
      }
    });
    const offStatus = client.on("statusChanged", (next) => {
      setStatus(next);
      setConnection(next.connection);
    });
    const offApproval = client.on("approvalRequired", (info) => {
      setApprovalRequired(info);
    });
    const offCard = client.on("cardChanged", ({ present }) => {
      if (!present) setFields(null);
    });
    const offError = client.on("error", (err) => {
      setError(err);
      if (err.code === "AUTH_EXPIRED") {
        setFields(null);
        setApprovalRequired(null);
      }
    });
    return () => {
      offConnection();
      offStatus();
      offApproval();
      offCard();
      offError();
      void client.disconnect();
    };
  }, [client]);

  const connect = useCallback(
    async (connectOptions?: { signal?: AbortSignal }) => {
      setError(null);
      try {
        await client.connect(connectOptions);
      } catch (err) {
        if (isThaiIDError(err)) setError(err);
        throw err;
      }
    },
    [client],
  );

  const read = useCallback(
    async (readOptions?: {
      fields?: Field[];
      readerId?: string;
      cardSessionId?: string;
      signal?: AbortSignal;
    }) => {
      setError(null);
      try {
        const result = await client.read(readOptions);
        setFields(result.fields);
        return result;
      } catch (err) {
        if (isThaiIDError(err)) setError(err);
        throw err;
      }
    },
    [client],
  );

  const disconnect = useCallback(async () => {
    setError(null);
    await client.disconnect();
    setFields(null);
    setApprovalRequired(null);
  }, [client]);

  return {
    client,
    connection,
    status,
    fields,
    approvalRequired,
    error,
    connect,
    read,
    disconnect,
  };
}

export default useThaiID;
