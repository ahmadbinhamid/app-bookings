// Mirrors backend/internal/modules/installation/model.go
export interface Installation {
  id: number;
  tenant_id: number;
  installed: boolean;
  created_at: string;
  updated_at: string;
}

// Mirrors backend/internal/modules/location/model.go. timezone_confirmed is
// false until an admin explicitly sets it (design resolution: "Location
// timezone is a manually-managed field") — treat timezone as "UTC"
// placeholder, not truth, until then.
export interface Location {
  id: string;
  tenant_id: number;
  name: string;
  timezone: string;
  timezone_confirmed: boolean;
  flowpos_location_id: string;
  created_at: string;
  updated_at: string;
}

// Mirrors backend/internal/modules/employee/model.go
export interface Employee {
  id: string;
  location_id: string;
  flowpos_employee_id: string;
  name: string;
  email?: string;
  phone?: string;
  active: boolean;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

// Mirrors backend/internal/modules/services/model.go
export interface Service {
  id: string;
  location_id: string;
  name: string;
  description?: string | null;
  duration_minutes: number;
  buffer_minutes: number;
  price: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ServiceInput {
  name: string;
  description?: string | null;
  duration_minutes: number;
  buffer_minutes: number;
  price: number;
  active?: boolean;
}

// Mirrors backend/internal/modules/schedules/model.go. start_time/end_time
// are "HH:MM" or "HH:MM:SS" wall-clock strings, not full datetimes.
export interface Schedule {
  id: string;
  employee_id: string;
  day_of_week: number; // 0 = Sunday .. 6 = Saturday
  start_time: string;
  end_time: string;
  created_at: string;
  updated_at: string;
}

export interface ScheduleInput {
  day_of_week: number;
  start_time: string;
  end_time: string;
}

// Mirrors backend/internal/modules/timeoff/model.go
export interface TimeOff {
  id: string;
  employee_id: string;
  start_datetime: string;
  end_datetime: string;
  reason?: string | null;
  created_at: string;
}

export interface TimeOffInput {
  start_datetime: string;
  end_datetime: string;
  reason?: string | null;
}

// Mirrors backend/internal/modules/booking/model.go

// ProposedSegment is what /bookings/propose returns AND what /bookings/confirm
// and /bookings/{id}/reschedule accept back verbatim — the round trip is
// fully stateless (no server-side proposal cache), so the frontend is what
// carries a proposal from "solve" to "confirm."
export interface ProposedSegment {
  service_id: string;
  employee_id: string;
  start: string;
  end: string;
  blocked_until: string;
  price: number;
}

export interface Proposal {
  segments: ProposedSegment[];
  window_start: string;
  window_end: string;
  total_price: number;
}

export interface ProposeInput {
  services: { service_id: string }[];
  earliest_start: string;
}

export interface BookingSegment {
  id: string;
  booking_id: string;
  employee_id: string;
  service_id: string;
  sequence_order: number;
  start_time: string;
  end_time: string;
  blocked_until: string;
  status: "booked" | "completed" | "cancelled";
  original_price: number;
  price: number;
  created_at: string;
  updated_at: string;
}

export interface Booking {
  id: string;
  location_id: string;
  created_by_admin_id?: number;
  customer_name: string;
  customer_phone?: string;
  customer_email?: string;
  status: "pending" | "confirmed" | "completed" | "cancelled";
  window_start?: string;
  window_end?: string;
  total_price: number;
  created_at: string;
  updated_at: string;
  segments?: BookingSegment[];
}

export interface ConfirmInput {
  customer_name: string;
  customer_phone?: string;
  customer_email?: string;
  segments: ProposedSegment[];
}

export interface RescheduleInput {
  segments: ProposedSegment[];
}

// A structured NO_AVAILABILITY error body from the backend (see
// internal/server/handlers/errors.go) — lets the UI say "no one offers
// this service" vs. "no one's free" instead of a flat error string.
export interface NoAvailabilityErrorBody {
  error: string;
  code: "NO_AVAILABILITY";
  service_id: string;
  reason: "NO_EMPLOYEE_ASSIGNED" | "NO_EMPLOYEE_FREE_THAT_DAY";
}

export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface Page<T> {
  data: T[];
  meta: PaginationMeta;
}
