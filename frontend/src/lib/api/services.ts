import { apiClient } from "./client";
import { type Service, type ServiceInput, type Employee, type Page } from "@/types";

// activeOnly excludes deleted (deactivated) services — pass true for the
// Services management page; leave false (default) for anything that needs
// to resolve a possibly-deleted service, like historical booking display,
// since deleting one never removes the row (see the backend's
// services.Repository.Delete doc comment).
export async function listServices(
  locationId: string,
  page: number,
  limit: number,
  search?: string,
  activeOnly?: boolean
): Promise<Page<Service>> {
  const res = await apiClient.get<Page<Service>>(`/locations/${locationId}/services`, {
    params: { page, limit, search: search || undefined, active: activeOnly || undefined },
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

// Soft-deletes the service (see the backend's services.Repository.Delete
// doc comment) — the row and its booking history are kept, it just stops
// showing up wherever activeOnly is requested.
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
