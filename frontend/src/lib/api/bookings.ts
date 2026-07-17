import { apiClient } from "./client";
import type { Booking, Proposal, ProposeInput, ConfirmInput, RescheduleInput } from "@/types";

export async function proposeBooking(locationId: string, input: ProposeInput): Promise<Proposal> {
  const res = await apiClient.post<Proposal>(`/locations/${locationId}/bookings/propose`, input);
  return res.data;
}

export async function confirmBooking(locationId: string, input: ConfirmInput): Promise<Booking> {
  const res = await apiClient.post<Booking>(`/locations/${locationId}/bookings/confirm`, input);
  return res.data;
}

export async function listBookings(
  locationId: string,
  params?: { employeeId?: string; from?: string; to?: string }
): Promise<Booking[]> {
  const res = await apiClient.get<{ bookings: Booking[] }>(`/locations/${locationId}/bookings`, {
    params: { employee_id: params?.employeeId, from: params?.from, to: params?.to },
  });
  return res.data.bookings ?? [];
}

export async function getBooking(locationId: string, bookingId: string): Promise<Booking> {
  const res = await apiClient.get<Booking>(`/locations/${locationId}/bookings/${bookingId}`);
  return res.data;
}

export async function cancelBooking(locationId: string, bookingId: string): Promise<Booking> {
  const res = await apiClient.post<Booking>(`/locations/${locationId}/bookings/${bookingId}/cancel`);
  return res.data;
}

export async function cancelBookingSegment(locationId: string, bookingId: string, segmentId: string): Promise<Booking> {
  const res = await apiClient.post<Booking>(`/locations/${locationId}/bookings/${bookingId}/segments/${segmentId}/cancel`);
  return res.data;
}

export async function rescheduleBooking(locationId: string, bookingId: string, input: RescheduleInput): Promise<Booking> {
  const res = await apiClient.post<Booking>(`/locations/${locationId}/bookings/${bookingId}/reschedule`, input);
  return res.data;
}
