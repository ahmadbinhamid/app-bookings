// Central registry of react-query keys — add one entry per resource as
// features are built, the same way appointments/src/lib/query-keys.ts does.
export const queryKeys = {
  me: () => ["me"] as const,

  locations: () => ["locations"] as const,

  employees: (locationId: string) => ["employees", locationId] as const,

  services: (locationId: string, page: number, limit: number, search?: string) =>
    ["services", locationId, page, limit, search] as const,
  service: (locationId: string, serviceId: string) => ["services", locationId, serviceId] as const,
  serviceEmployees: (locationId: string, serviceId: string) =>
    ["services", locationId, serviceId, "employees"] as const,

  schedules: (locationId: string, employeeId: string) => ["schedules", locationId, employeeId] as const,
  timeOff: (locationId: string, employeeId: string) => ["time-off", locationId, employeeId] as const,

  bookings: (locationId: string, employeeId?: string) => ["bookings", locationId, employeeId] as const,
  booking: (locationId: string, bookingId: string) => ["bookings", locationId, bookingId] as const,
};
