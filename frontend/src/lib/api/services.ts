import { apiClient } from "./client";
import { type Service, type ServiceInput, type Employee, type Page } from "@/types";

export async function listServices(
  locationId: string,
  page: number,
  limit: number,
  search?: string
): Promise<Page<Service>> {
  const res = await apiClient.get<Page<Service>>(`/locations/${locationId}/services`, {
    params: { page, limit, search: search || undefined },
  });
  return res.data;
}

export async function getService(locationId: string, serviceId: string): Promise<Service> {
  const res = await apiClient.get<Service>(`/locations/${locationId}/services/${serviceId}`);
  return res.data;
}

export async function createService(locationId: string, input: ServiceInput): Promise<Service> {
  const res = await apiClient.post<Service>(`/locations/${locationId}/services`, input);
  return res.data;
}

export async function updateService(locationId: string, serviceId: string, input: ServiceInput): Promise<Service> {
  const res = await apiClient.put<Service>(`/locations/${locationId}/services/${serviceId}`, input);
  return res.data;
}

// Rejected with a 409 (code SERVICE_HAS_BOOKINGS) if the service has ever
// actually been booked — the backend won't delete it out from under real
// booking history.
export async function deleteService(locationId: string, serviceId: string): Promise<void> {
  await apiClient.delete(`/locations/${locationId}/services/${serviceId}`);
}

// --- Employee assignments (which employees can perform this service) ---

export async function listAssignedEmployees(locationId: string, serviceId: string): Promise<Employee[]> {
  const res = await apiClient.get<{ employees: Employee[] }>(`/locations/${locationId}/services/${serviceId}/employees`);
  return res.data.employees ?? [];
}

export async function assignEmployee(locationId: string, serviceId: string, employeeId: string): Promise<void> {
  await apiClient.post(`/locations/${locationId}/services/${serviceId}/employees`, { employee_id: employeeId });
}

export async function unassignEmployee(locationId: string, serviceId: string, employeeId: string): Promise<void> {
  await apiClient.delete(`/locations/${locationId}/services/${serviceId}/employees/${employeeId}`);
}
