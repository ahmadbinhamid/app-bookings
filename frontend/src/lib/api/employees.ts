import { apiClient } from "./client";
import { type Employee } from "@/types";

export async function listEmployees(locationId: string): Promise<Employee[]> {
  const res = await apiClient.get<{ employees: Employee[] }>(`/locations/${locationId}/employees`);
  return res.data.employees;
}
