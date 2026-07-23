import type { Booking } from "@/types";

export interface StatusVisual {
  label: string;
  badgeVariant: "status-success" | "status-warning" | "status-info" | "destructive";
  dotClass: string;
}

export const BOOKING_STATUS: Record<Booking["status"], StatusVisual> = {
  confirmed: { label: "Confirmed", badgeVariant: "status-success", dotClass: "bg-emerald-500" },
  pending: { label: "Pending", badgeVariant: "status-warning", dotClass: "bg-amber-500" },
  completed: { label: "Completed", badgeVariant: "status-info", dotClass: "bg-blue-500" },
  cancelled: { label: "Cancelled", badgeVariant: "destructive", dotClass: "bg-red-500" },
};
