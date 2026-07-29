import { apiClient } from "./client";
import type { SyncSummary } from "@/types";

// POST /api/v1/sync/trigger — runs the same tenant-wide location+employee
// sync the background scheduler runs every SYNC_INTERVAL_MINUTES (an hour by
// default), on demand. Synchronous on the backend, so this resolves once the
// sync has actually finished.
export async function triggerSync(): Promise<SyncSummary> {
  const res = await apiClient.post<{ summary: SyncSummary }>("/sync/trigger");
  return res.data.summary;
}
