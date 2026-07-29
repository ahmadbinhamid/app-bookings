import { apiClient } from "./client";
import { type Employee } from "@/types";

export async function listEmployees(locationId: string): Promise<Employee[]> {
  const res = await apiClient.get<{ employees: Employee[] }>(`/locations/${locationId}/employees`);
  return res.data.employees ?? [];
}

// Every employee synced for the tenant, assigned or not — the Employees
// page's roster view.
export async function listAllEmployees(): Promise<Employee[]> {
  const res = await apiClient.get<{ employees: Employee[] }>("/employees");
  return res.data.employees ?? [];
}

// The tenant-wide pool of synced employees not yet assigned to any location
// — this is the only list an "assign employees to this location" picker
// should ever offer from (see the Locations page's assign-employees dialog).
// An already-assigned employee is only ever moved via assignEmployeeLocation
// below, not through this list.
export async function listUnassignedEmployees(): Promise<Employee[]> {
  const res = await apiClient.get<{ employees: Employee[] }>("/employees/unassigned");
  return res.data.employees ?? [];
}

// Assigns (or reassigns) an employee to exactly one location. The backend
// blocks a reassignment (409, code HAS_FUTURE_BOOKINGS) while the employee
// still has upcoming bookings at their current location.
export async function assignEmployeeLocation(employeeId: string, locationId: string): Promise<Employee> {
  const res = await apiClient.patch<{ employee: Employee }>(`/employees/${employeeId}/location`, {
    location_id: locationId,
  });
  return res.data.employee;
}
