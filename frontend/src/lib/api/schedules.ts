import { apiClient } from "./client";
import { type Schedule, type ScheduleInput } from "@/types";

export async function listSchedules(locationId: string, employeeId: string): Promise<Schedule[]> {
  const res = await apiClient.get<{ schedules: Schedule[] }>(`/locations/${locationId}/employees/${employeeId}/schedules`);
  return res.data.schedules;
}

export async function createSchedule(locationId: string, employeeId: string, input: ScheduleInput): Promise<Schedule> {
  const res = await apiClient.post<Schedule>(`/locations/${locationId}/employees/${employeeId}/schedules`, input);
  return res.data;
}

export async function deleteSchedule(locationId: string, employeeId: string, scheduleId: string): Promise<void> {
  await apiClient.delete(`/locations/${locationId}/employees/${employeeId}/schedules/${scheduleId}`);
}
