import { apiClient } from "./client";
import { type Location } from "@/types";

export async function listLocations(): Promise<Location[]> {
  const res = await apiClient.get<{ locations: Location[] }>("/locations");
  return res.data.locations ?? [];
}

export async function setLocationTimezone(locationId: string, timezone: string): Promise<Location> {
  const res = await apiClient.patch<{ location: Location }>(`/locations/${locationId}/timezone`, { timezone });
  return res.data.location;
}
