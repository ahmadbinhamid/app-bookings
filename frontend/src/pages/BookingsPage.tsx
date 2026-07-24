import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge, Button,
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
  Tooltip, TooltipTrigger, TooltipContent,
} from "@flowposltd/ui";
import { Plus, CalendarDays } from "lucide-react";
import { listBookings, listEmployees, listServices } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { EmptyState } from "@/components/ui/empty-state";
import { PageFallback } from "@/components/ui/page-fallback";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { BookingTimeline } from "@/components/ui/booking-timeline";
import { DateRangePopover, type SimpleDateRange } from "@/components/ui/calendar-popover";
import { BookingWizard } from "@/components/bookings/booking-wizard";
import { BookingDetailDialog } from "@/components/bookings/booking-detail-dialog";
import { useLocationContext } from "@/contexts/location-context";
import { useLocationTimezone } from "@/hooks/use-location-timezone";
import { BOOKING_STATUS } from "@/constants";
import { cn, formatMoney, buildTimeline } from "@/utils";
import { serviceVisual } from "@/constants/service-visuals";
import type { Booking } from "@/types";

// "YYYY-MM-DD" (from the date-range popover) → start/end-of-day ISO instants
// in the browser's local time, matching what the backend's from/to query
// params expect (RFC3339).
function startOfDayISO(dateStr: string): string {
  return new Date(`${dateStr}T00:00:00`).toISOString();
}
function endOfDayISO(dateStr: string): string {
  return new Date(`${dateStr}T23:59:59.999`).toISOString();
}

const STATUS_FILTERS = ["all", "confirmed", "pending", "completed", "cancelled"] as const;
type StatusFilter = (typeof STATUS_FILTERS)[number];

// Keep the row scannable — more than this and the cell starts fighting the
// Team/Schedule columns for space, so the rest collapse into a "+N" chip
// with the full list on hover.
const MAX_VISIBLE_CHIPS = 2;

function activeSegments(booking: Booking) {
  return (booking.segments ?? []).filter((s) => s.status !== "cancelled");
}

export default function BookingsPage() {
  const { selectedLocationId, isLoading: locationsLoading } = useLocationContext();
  const timeZone = useLocationTimezone();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [rescheduling, setRescheduling] = useState<{ bookingId: string; serviceIds: string[] } | undefined>();
  const [viewingBookingId, setViewingBookingId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [range, setRange] = useState<SimpleDateRange>({ from: null, to: null });

  const from = range.from ? startOfDayISO(range.from) : undefined;
  const to = range.to ? endOfDayISO(range.to) : undefined;

  const { data: bookings = [], isLoading } = useQuery({
    queryKey: selectedLocationId ? [...queryKeys.bookings(selectedLocationId), from, to] : ["bookings", "none"],
    queryFn: () => listBookings(selectedLocationId!, { from, to }),
    enabled: Boolean(selectedLocationId),
  });

  const { data: employees = [] } = useQuery({
    queryKey: selectedLocationId ? queryKeys.employees(selectedLocationId) : ["employees", "none"],
    queryFn: () => listEmployees(selectedLocationId!),
    enabled: Boolean(selectedLocationId),
  });
  const { data: servicesPage } = useQuery({
    queryKey: selectedLocationId ? queryKeys.services(selectedLocationId, 1, 100) : ["services", "none"],
    queryFn: () => listServices(selectedLocationId!, 1, 100),
    enabled: Boolean(selectedLocationId),
  });
  const employeeByID = new Map(employees.map((e) => [e.id, e]));
  const serviceByID = new Map((servicesPage?.data ?? []).map((s) => [s.id, s]));

  if (locationsLoading) return <PageFallback />;
  if (!selectedLocationId) {
    return (
      <div className="p-6">
        <EmptyState variant="hero" icon={CalendarDays} title="No location yet" description="Bookings are managed per location." />
      </div>
    );
  }

  const shown = statusFilter === "all" ? bookings : bookings.filter((b) => b.status === statusFilter);
  const hasFilters = Boolean(range.from || range.to) || statusFilter !== "all";
  // Nothing to filter when the location has zero bookings — only show the
  // date/status controls once there's something (or a filter) for them to act on.
  const showToolbar = isLoading || hasFilters || bookings.length > 0;
  const isTrueEmpty = !isLoading && !hasFilters && bookings.length === 0;

  function openReschedule(bookingId: string, serviceIds: string[]) {
    setViewingBookingId(undefined);
    setRescheduling({ bookingId, serviceIds });
  }

  function clearFilters() {
    setRange({ from: null, to: null });
    setStatusFilter("all");
  }

  return (
    <div className="p-4 sm:p-6 flex flex-col gap-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1>Bookings</h1>
            {bookings.length > 0 && <Badge>{bookings.length}</Badge>}
          </div>
          <p className="lead mt-0.5">Every booking for this location — upcoming and past.</p>
        </div>
        <Button onClick={() => setWizardOpen(true)} className="self-start">
          <Plus className="size-4" />
          New Booking
        </Button>
      </div>

      {showToolbar && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <DateRangePopover value={range} onChange={setRange} className="w-full sm:w-64" />

          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as StatusFilter)}>
            <SelectTrigger className="w-full sm:w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_FILTERS.map((f) => (
                <SelectItem key={f} value={f}>{f === "all" ? "All statuses" : BOOKING_STATUS[f].label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {isLoading ? (
        <p className="text-body-2 text-content-secondary">Loading…</p>
      ) : shown.length === 0 ? (
        isTrueEmpty ? (
          <EmptyState
            variant="hero"
            icon={CalendarDays}
            title="No bookings yet"
            description="Create your first booking to get started."
            action={<Button onClick={() => setWizardOpen(true)}><Plus className="size-4" />New Booking</Button>}
          />
        ) : (
          <EmptyState
            icon={CalendarDays}
            title="No bookings match these filters"
            description="Try widening the date range or clearing the status filter."
            action={
              <div className="flex items-center gap-2">
                <Button size="sm" variant="secondary" onClick={clearFilters}>Clear filters</Button>
                <Button size="sm" onClick={() => setWizardOpen(true)}><Plus className="size-3.5" />New Booking</Button>
              </div>
            }
          />
        )
      ) : (
        <div className="rounded-md border border-border bg-card shadow-card overflow-x-auto">
          <Table className="min-w-205">
            <TableHeader>
              <TableRow>
                <TableHead>Customer</TableHead>
                <TableHead>Services</TableHead>
                <TableHead>Team</TableHead>
                <TableHead>Schedule</TableHead>
                <TableHead>When</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Total</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {shown.map((b) => {
                const segs = activeSegments(b);
                const chips = [...new Set(segs.map((s) => serviceByID.get(s.service_id)?.name ?? s.service_id))];
                const team = [...new Set(segs.map((s) => s.employee_id))].map((id) => employeeByID.get(id)?.name ?? id);
                const timeline = buildTimeline(
                  segs.map((s) => ({
                    key: s.id,
                    name: serviceByID.get(s.service_id)?.name ?? s.service_id,
                    start: s.start_time,
                    end: s.end_time,
                  })),
                  timeZone
                );
                const status = BOOKING_STATUS[b.status];

                return (
                  <TableRow key={b.id} className="cursor-pointer" onClick={() => setViewingBookingId(b.id)}>
                    <TableCell>
                      <div className="flex items-center gap-2.5 min-w-0">
                        <PersonAvatar name={b.customer_name} size="sm" />
                        <div className="min-w-0">
                          <div className="font-medium text-foreground truncate">{b.customer_name}</div>
                          <div className="text-body-3 text-content-tertiary truncate">{b.customer_phone || "—"}</div>
                        </div>
                      </div>
                    </TableCell>

                    <TableCell>
                      {chips.length === 0 ? (
                        <span className="text-body-3 text-content-tertiary italic">No services</span>
                      ) : (
                        <div className="flex flex-wrap items-center gap-1.5">
                          {chips.slice(0, MAX_VISIBLE_CHIPS).map((name) => {
                            const v = serviceVisual(name);
                            return (
                              <span
                                key={name}
                                className="inline-flex items-center gap-1.5 rounded-md py-0.5 pl-1.5 pr-2.5 text-body-3 font-medium capitalize"
                                style={{ backgroundColor: v.bg, color: v.fg }}
                              >
                                <span className="size-1.5 rounded-full" style={{ backgroundColor: v.solid }} />
                                {name}
                              </span>
                            );
                          })}
                          {chips.length > MAX_VISIBLE_CHIPS && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-body-3 font-medium text-content-secondary">
                                  +{chips.length - MAX_VISIBLE_CHIPS}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent className="capitalize">
                                {chips.slice(MAX_VISIBLE_CHIPS).join(", ")}
                              </TooltipContent>
                            </Tooltip>
                          )}
                        </div>
                      )}
                    </TableCell>

                    <TableCell>
                      {team.length === 0 ? (
                        <span className="text-content-tertiary">·</span>
                      ) : (
                        <div className="flex">
                          {team.map((name, i) => (
                            <PersonAvatar
                              key={name + i}
                              name={name}
                              size="xs"
                              className={cn("ring-2 ring-card", i > 0 && "-ml-2")}
                            />
                          ))}
                        </div>
                      )}
                    </TableCell>

                    <TableCell>
                      {segs.length === 0 ? (
                        <span className="inline-flex items-center gap-1.5 rounded px-2.5 py-1 text-body-3 font-medium bg-muted text-content-tertiary">
                          No appointments
                        </span>
                      ) : timeline ? (
                        <div className="w-24">
                          <BookingTimeline
                            timeline={timeline}
                            segments={segs.map((s) => ({ key: s.id, name: serviceByID.get(s.service_id)?.name ?? s.service_id, start: s.start_time, end: s.end_time }))}
                            variant="mini"
                            timeZone={timeZone}
                          />
                          <div className="mt-1.5 text-body-3 text-content-secondary tabular-nums">
                            <div className="whitespace-nowrap">{timeline.startLabel} – {timeline.endLabel}</div>
                            {timeline.totalGapMinutes > 0 && (
                              <div className="inline-flex items-center gap-1 text-amber-700 whitespace-nowrap">
                                <span className="size-1 rounded-full bg-amber-500" />
                                {timeline.totalGapMinutes} min gap
                              </div>
                            )}
                          </div>
                        </div>
                      ) : null}
                    </TableCell>

                    <TableCell className="tabular-nums text-content-secondary font-medium">
                      {b.window_start
                        ? new Date(b.window_start).toLocaleString([], { month: "short", day: "numeric", hour: "numeric", minute: "2-digit", timeZone })
                        : "—"}
                    </TableCell>

                    <TableCell>
                      <Badge variant={status.badgeVariant} dotColor={status.dotClass}>
                        {status.label}
                      </Badge>
                    </TableCell>

                    <TableCell className="text-right font-semibold tabular-nums">{formatMoney(b.total_price)}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <BookingWizard open={wizardOpen} onClose={() => setWizardOpen(false)} locationId={selectedLocationId} />

      {rescheduling && (
        <BookingWizard
          open
          onClose={() => setRescheduling(undefined)}
          locationId={selectedLocationId}
          bookingId={rescheduling.bookingId}
          initialServiceIds={rescheduling.serviceIds}
        />
      )}

      <BookingDetailDialog
        open={Boolean(viewingBookingId)}
        onClose={() => setViewingBookingId(undefined)}
        locationId={selectedLocationId}
        bookingId={viewingBookingId}
        onReschedule={openReschedule}
      />
    </div>
  );
}
