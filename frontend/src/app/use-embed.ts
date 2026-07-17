import { useEffect } from "react";
import { useFlowposApp } from "@flowposltd/apps-sdk/app";
import { setAuthToken } from "@/lib/auth-token";

/**
 * Wires this app up to the tenant-dashboard iframe host: theme sync and
 * auto-resize. The real auth token comes via the `?token=` query string
 * (src/lib/auth-token.ts) — apps-sdk's postMessage handshake isn't actually
 * sent by the dashboard today, so this only *adds* a token if that ever
 * changes; it must never clear an already-adopted token.
 */
const allowedParentOrigins = (
  import.meta.env.VITE_FLOWPOS_DASHBOARD_ORIGINS ??
  "http://localhost:5173,http://localhost:3000"
)
  .split(",")
  .map((origin: string) => origin.trim())
  .filter(Boolean);

export function useEmbed() {
  const flowpos = useFlowposApp({ allowedParentOrigins });

  useEffect(() => {
    if (flowpos.context?.authToken) {
      setAuthToken(flowpos.context.authToken);
    }
  }, [flowpos.context?.authToken]);

  return flowpos;
}
