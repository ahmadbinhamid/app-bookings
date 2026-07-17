import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
  Button, Badge,
} from "@flowposltd/ui";
import { X, Check, RefreshCw } from "lucide-react";
import { getBooking, cancelBooking, cancelBookingSegment, completeBookingSegment, listEmployees, listServices } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type ApiError } from "@/lib/api/client";

interface BookingDetailDialogProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  bookingId: string | undefined;
  onReschedule: (bookingId: string, serviceIds: string[]) => void;
}

const STATUS_VARIANT: Record<string, "status-success" | "secondary" | "destructive"> = {
  confirmed: "status-success",
  completed: "secondary",
  cancelled: "destructive",
  pending: "secondary",
};

export function BookingDetailDialog({ open, onClose, locationId, bookingId, onReschedule }: BookingDetailDialogProps) {
  const qc = useQueryClient();

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

  const activeServiceIds = (booking.segments ?? [])
    .filter((s) => s.status !== "cancelled")
    .map((s) => s.service_id);

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {booking.customer_name}
            <Badge variant={STATUS_VARIANT[booking.status] ?? "secondary"}>{booking.status}</Badge>
          </DialogTitle>
          <DialogDescription>
            {booking.customer_phone}
            {booking.customer_phone && booking.customer_email && " · "}
            {booking.customer_email}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-2 py-2">
          {(booking.segments ?? []).map((seg) => (
            <div
              key={seg.id}
              className={`flex items-center justify-between rounded-sm border border-border p-3 ${seg.status === "cancelled" ? "opacity-50" : ""}`}
            >
              <div>
                <p className="font-medium text-foreground">
                  {serviceByID.get(seg.service_id)?.name ?? seg.service_id}
                  {seg.status === "cancelled" && <span className="text-content-secondary font-normal"> (cancelled)</span>}
                  {seg.status === "completed" && <span className="text-content-secondary font-normal"> (completed)</span>}
                </p>
                <p className="text-xs text-content-secondary">{employeeByID.get(seg.employee_id)?.name ?? seg.employee_id}</p>
              </div>
              <div className="flex items-center gap-3">
                <div className="text-right">
                  <p className="tabular-nums text-sm">
                    {new Date(seg.start_time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                    {" – "}
                    {new Date(seg.end_time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                  </p>
                  <p className="text-xs text-content-secondary">${seg.price.toFixed(2)}</p>
                </div>
                {seg.status === "booked" && (
                  <>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-7"
                      title="Mark completed"
                      onClick={() => completeSegmentMut.mutate(seg.id)}
                      disabled={completeSegmentMut.isPending || cancelSegmentMut.isPending}
                    >
                      <Check className="size-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-7"
                      title="Cancel this service"
                      onClick={() => cancelSegmentMut.mutate(seg.id)}
                      disabled={cancelSegmentMut.isPending || completeSegmentMut.isPending}
                    >
                      <X className="size-3.5" />
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))}

          <div className="flex items-center justify-between border-t border-border pt-3 mt-1">
            <span className="text-sm text-content-secondary">Total</span>
            <span className="font-bold">${booking.total_price.toFixed(2)}</span>
          </div>
        </div>

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
