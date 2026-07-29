import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
  Button, Badge,
} from "@flowposltd/ui";
import { X, Check, RefreshCw, CalendarOff } from "lucide-react";
import { getBooking, cancelBooking, cancelBookingSegment, completeBookingSegment, listEmployees, listServices } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type ApiError } from "@/lib/api/client";
import { DialogBody } from "@/components/ui/dialog-body";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { BookingTimeline } from "@/components/ui/booking-timeline";
import { BOOKING_STATUS } from "@/constants";
import { serviceVisual } from "@/constants/service-visuals";
import { cn, formatMoney, formatTimeRange, buildTimeline } from "@/utils";
import { useLocationTimezone } from "@/hooks/use-location-timezone";

interface BookingDetailDialogProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  bookingId: string | undefined;
  onReschedule: (bookingId: string, serviceIds: string[]) => void;
}

export function BookingDetailDialog({ open, onClose, locationId, bookingId, onReschedule }: BookingDetailDialogProps) {
  const qc = useQueryClient();
  const timeZone = useLocationTimezone();

  const { data: booking } = useQuery({
    queryKey: bookingId ? queryKeys.booking(locationId, bookingId) : ["booking", "none"],
    queryFn: () => getBooking(locationId, bookingId!),
    enabled: open && Boolean(bookingId),
  });

  const { data: employees = [] } = useQuery({
    queryKey: queryKeys.employees(locationId),
    queryFn: () => listEmployees(locationId),
    enabled: open,
  });
  const { data: servicesPage } = useQuery({
    queryKey: queryKeys.services(locationId, 1, 100),
    queryFn: () => listServices(locationId, 1, 100),
    enabled: open,
  });
  const employeeByID = new Map(employees.map((e) => [e.id, e]));
  const serviceByID = new Map((servicesPage?.data ?? []).map((s) => [s.id, s]));

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["bookings"] });
    if (bookingId) qc.invalidateQueries({ queryKey: queryKeys.booking(locationId, bookingId) });
  };

  const cancelBookingMut = useMutation({
    mutationFn: () => cancelBooking(locationId, bookingId!),
    onSuccess: (b) => {
      toast.success(b.status === "cancelled" ? "Booking cancelled" : "Remaining bookable segments cancelled");
      invalidate();
    },
    onError: (err: ApiError) => toast.error(err.message),
  });

  const cancelSegmentMut = useMutation({
    mutationFn: (segmentId: string) => cancelBookingSegment(locationId, bookingId!, segmentId),
    onSuccess: () => {
      toast.success("Segment cancelled");
      invalidate();
    },
    onError: (err: ApiError) => toast.error(err.message),
  });

  const completeSegmentMut = useMutation({
    mutationFn: (segmentId: string) => completeBookingSegment(locationId, bookingId!, segmentId),
    onSuccess: (b) => {
      toast.success(b.status === "completed" ? "Booking completed" : "Marked as completed");
      invalidate();
    },
    onError: (err: ApiError) => toast.error(err.message),
  });

  if (!booking) return null;

  const segments = booking.segments ?? [];
  const activeSegments = segments.filter((s) => s.status !== "cancelled");
  const activeServiceIds = activeSegments.map((s) => s.service_id);
  const isEmpty = activeSegments.length === 0;
  const status = BOOKING_STATUS[booking.status];

  const timeline = !isEmpty
    ? buildTimeline(
        activeSegments.map((s) => ({
          key: s.id,
          name: serviceByID.get(s.service_id)?.name ?? s.service_id,
          start: s.start_time,
          end: s.end_time,
        })),
        timeZone
      )
    : null;

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="w-[calc(100%-2rem)] max-w-xl">
        <DialogHeader>
          <div className="flex items-start gap-3.5">
            <PersonAvatar name={booking.customer_name} size="lg" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2.5">
                <DialogTitle>{booking.customer_name}</DialogTitle>
                <Badge variant={status.badgeVariant} dotColor={status.dotClass}>{status.label}</Badge>
              </div>
              <DialogDescription>
                {booking.customer_phone}
                {booking.customer_phone && booking.customer_email && " · "}
                {booking.customer_email}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <DialogBody>
          <div className="flex flex-col gap-1">
            {/* Cancelling never deletes anything on the backend (only flips
                segment status) — a fully-cancelled booking still has its
                original service/employee/schedule data below, this is just
                an additional heads-up banner, not a replacement for it. */}
            {isEmpty && segments.length > 0 && (
              <div className="flex items-center gap-3 rounded-md border border-dashed border-border bg-muted/30 p-4 mb-3.5">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-destructive/10 text-destructive">
                  <CalendarOff className="size-4.5" />
                </div>
                <div>
                  <p className="font-medium text-body-2 text-foreground">This booking was cancelled</p>
                  <p className="text-body-3 text-content-secondary">The original slots were released back to the calendar.</p>
                </div>
              </div>
            )}

            {segments.length === 0 ? (
              <div className="flex flex-col items-center gap-3 rounded-md border border-dashed border-border bg-muted/30 p-8 text-center">
                <div className="flex size-12 items-center justify-center rounded-md bg-destructive/10 text-destructive">
                  <CalendarOff className="size-6" />
                </div>
                <div>
                  <p className="font-medium text-body-2 text-foreground">No appointment details available</p>
                  <p className="text-body-3 text-content-secondary mt-1">
                    This booking has no service or schedule data on record.
                  </p>
                </div>
              </div>
            ) : (
              <>
              <div className="text-body-3 font-medium tracking-[0.06em] text-content-tertiary mb-2.5">
                APPOINTMENT TIMELINE
              </div>
              {timeline && (
                <BookingTimeline
                  timeline={timeline}
                  segments={activeSegments.map((s) => ({
                    key: s.id,
                    name: serviceByID.get(s.service_id)?.name ?? s.service_id,
                    start: s.start_time,
                    end: s.end_time,
                  }))}
                  variant="large"
                  timeZone={timeZone}
                />
              )}

              <div className="flex flex-col gap-2 mt-5">
                {segments.map((seg) => {
                  const name = serviceByID.get(seg.service_id)?.name ?? seg.service_id;
                  const employee = employeeByID.get(seg.employee_id)?.name ?? seg.employee_id;
                  const v = serviceVisual(name);
                  const cancelled = seg.status === "cancelled";
                  const completed = seg.status === "completed";

                  return (
                    <div
                      key={seg.id}
                      className={cn(
                        "flex flex-wrap items-center gap-3 rounded-md border border-border p-3",
                        cancelled && "opacity-55"
                      )}
                    >
                      <span className="self-stretch w-1 rounded-full shrink-0" style={{ backgroundColor: v.solid }} />
                      <PersonAvatar name={employee} size="md" />
                      <div className="flex-1 min-w-40">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-body-2 text-foreground capitalize">{name}</span>
                          {completed && <Badge variant="status-info">Completed</Badge>}
                          {cancelled && <Badge variant="destructive">Cancelled</Badge>}
                        </div>
                        <div className="text-body-3 text-content-secondary mt-0.5">
                          {employee} · {formatTimeRange(seg.start_time, seg.end_time, timeZone)}
                        </div>
                      </div>
                      <div className="flex flex-wrap items-center gap-3 w-full sm:w-auto sm:ml-0">
                        <div className="text-body-2 font-medium tabular-nums shrink-0">{formatMoney(seg.price)}</div>
                        {seg.status === "booked" && (
                          <div className="flex gap-1.5 shrink-0">
                            <Button
                              variant="secondary"
                              size="sm"
                              title="Mark this service completed"
                              onClick={() => completeSegmentMut.mutate(seg.id)}
                              disabled={completeSegmentMut.isPending || cancelSegmentMut.isPending}
                              className="border-transparent bg-tag-success text-tag-success-text! hover:opacity-80"
                            >
                              <Check className="size-3.5" />
                              Complete
                            </Button>
                            <Button
                              variant="secondary"
                              size="sm"
                              title="Cancel this service"
                              onClick={() => cancelSegmentMut.mutate(seg.id)}
                              disabled={cancelSegmentMut.isPending || completeSegmentMut.isPending}
                              className="border-transparent bg-tag-destructive text-tag-destructive-text! hover:opacity-80"
                            >
                              <X className="size-3.5" />
                              Cancel
                            </Button>
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
              </>
            )}

            <div className="flex items-center justify-between border-t border-border pt-3.5 mt-4">
              <span className="text-body-2 text-content-secondary">Total</span>
              <span className="text-heading-3 font-semibold tabular-nums">{formatMoney(booking.total_price)}</span>
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          {booking.status === "confirmed" && (
            <>
              <Button
                variant="secondary"
                onClick={() => onReschedule(booking.id, activeServiceIds)}
              >
                <RefreshCw className="size-3.5" />
                Reschedule
              </Button>
              <Button
                variant="destructive"
                onClick={() => cancelBookingMut.mutate()}
                loading={cancelBookingMut.isPending}
              >
                Cancel booking
              </Button>
            </>
          )}
          {booking.status !== "confirmed" && (
            <Button variant="secondary" onClick={onClose}>Close</Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
