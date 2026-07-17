import { apiClient } from "./client";
import type { Installation } from "@/types";

// GET /api/v1/me — confirms the JWT round-trip works and shows whether the
// calling tenant has this app installed. A working example of the
// api client + react-query pairing; replace/extend once real features exist.
export async function getMe(): Promise<{ installation: Installation | null }> {
  const res = await apiClient.get<{ installation: Installation | null }>("/me");
  return res.data;
}
