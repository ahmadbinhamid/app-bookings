import { apiClient } from "./client";
import { type TimeOff, type TimeOffInput } from "@/types";

export async function listTimeOff(locationId: string, employeeId: string): Promise<TimeOff[]> {
  const res = await apiClient.get<{ time_off: TimeOff[] }>(`/locations/${locationId}/employees/${employeeId}/time-off`);
  return res.data.time_off ?? [];
}

export async function createTimeOff(locationId: string, employeeId: string, input: TimeOffInput): Promise<TimeOff> {
  const res = await apiClient.post<TimeOff>(`/locations/${locationId}/employees/${employeeId}/time-off`, input);
  return res.data;
}

export async function deleteTimeOff(locationId: string, employeeId: string, timeOffId: string): Promise<void> {
  await apiClient.delete(`/locations/${locationId}/employees/${employeeId}/time-off/${timeOffId}`);
}
