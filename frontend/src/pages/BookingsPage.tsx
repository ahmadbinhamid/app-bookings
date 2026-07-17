import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge, Button, Input,
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from "@flowposltd/ui";
import { Plus, CalendarDays, X } from "lucide-react";
import { listBookings, listEmployees, listServices } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { EmptyState } from "@/components/ui/empty-state";
import { FormField } from "@/components/ui/form-field";
import { BookingWizard } from "@/components/bookings/booking-wizard";
import { BookingDetailDialog } from "@/components/bookings/booking-detail-dialog";
import { useLocationContext } from "@/contexts/location-context";

// "YYYY-MM-DD" (from a native date input) → start/end-of-day ISO instants
// in the browser's local time, matching what the backend's from/to query
// params expect (RFC3339).
function startOfDayISO(dateStr: string): string {
  return new Date(`${dateStr}T00:00:00`).toISOString();
}
function endOfDayISO(dateStr: string): string {
  return new Date(`${dateStr}T23:59:59.999`).toISOString();
}

const STATUS_VARIANT: Record<string, "status-success" | "secondary" | "destructive"> = {
  confirmed: "status-success",
  completed: "secondary",
  cancelled: "destructive",
  pending: "secondary",
};

export default function BookingsPage() {
  const { selectedLocationId, isLoading: locationsLoading } = useLocationContext();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [rescheduling, setRescheduling] = useState<{ bookingId: string; serviceIds: string[] } | undefined>();
  const [viewingBookingId, setViewingBookingId] = useState<string | undefined>();
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");

  const from = fromDate ? startOfDayISO(fromDate) : undefined;
  const to = toDate ? endOfDayISO(toDate) : undefined;

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

  if (locationsLoading) return null;
  if (!selectedLocationId) {
    return (
      <div className="p-6">
        <EmptyState icon={CalendarDays} title="No location yet" description="Bookings are managed per location." />
      </div>
    );
  }

  function openReschedule(bookingId: string, serviceIds: string[]) {
    setViewingBookingId(undefined);
    setRescheduling({ bookingId, serviceIds });
  }

  return (
    <div className="p-6 flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1>Bookings</h1>
            {bookings.length > 0 && <Badge>{bookings.length}</Badge>}
          </div>
          <p className="lead mt-0.5">Every booking for this location — upcoming and past.</p>
        </div>
        <Button onClick={() => setWizardOpen(true)}>
          <Plus className="size-4" />
          New Booking
        </Button>
      </div>

      <div className="flex items-end gap-3">
        <FormField label="From" className="w-40">
          <Input type="date" value={fromDate} onChange={(e) => setFromDate(e.target.value)} />
        </FormField>
        <FormField label="To" className="w-40">
          <Input type="date" value={toDate} onChange={(e) => setToDate(e.target.value)} />
        </FormField>
        {(fromDate || toDate) && (
          <Button variant="ghost" size="sm" onClick={() => { setFromDate(""); setToDate(""); }}>
            <X className="size-3.5" />
            Clear
          </Button>
        )}
      </div>

      {isLoading ? (
        <p className="text-sm text-content-secondary">Loading…</p>
      ) : bookings.length === 0 ? (
        <EmptyState
          icon={CalendarDays}
          title="No bookings yet"
          description="Create a booking to get started."
          action={<Button size="sm" onClick={() => setWizardOpen(true)}><Plus className="size-3.5" />New Booking</Button>}
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Customer</TableHead>
              <TableHead>Services</TableHead>
              <TableHead>Employees</TableHead>
              <TableHead>When</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Total</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {bookings.map((b) => {
              const segs = (b.segments ?? []).filter((s) => s.status !== "cancelled");
              const serviceNames = [...new Set(segs.map((s) => serviceByID.get(s.service_id)?.name ?? s.service_id))];
              const employeeNames = [...new Set(segs.map((s) => employeeByID.get(s.employee_id)?.name ?? s.employee_id))];
              return (
                <TableRow key={b.id} className="cursor-pointer" onClick={() => setViewingBookingId(b.id)}>
                  <TableCell className="font-medium text-foreground">{b.customer_name}</TableCell>
                  <TableCell className="text-content-secondary">{serviceNames.join(", ") || "—"}</TableCell>
                  <TableCell className="text-content-secondary">{employeeNames.join(", ") || "—"}</TableCell>
                  <TableCell className="tabular-nums text-content-secondary">
                    {b.window_start ? new Date(b.window_start).toLocaleString([], { dateStyle: "medium", timeStyle: "short" }) : "—"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANT[b.status] ?? "secondary"}>{b.status}</Badge>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">${b.total_price.toFixed(2)}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
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
