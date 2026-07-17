import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
  Button, Input, Checkbox,
} from "@flowposltd/ui";
import { Clock, ArrowRight, RotateCcw } from "lucide-react";
import { DialogBody } from "@/components/ui/dialog-body";
import { FormField } from "@/components/ui/form-field";
import { ApiError } from "@/lib/api/client";
import { listServices, listEmployees, proposeBooking, confirmBooking, rescheduleBooking } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Proposal, type Service, type Employee } from "@/types";

interface BookingWizardProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  /** Reschedule mode when a bookingId is supplied — same propose/review flow,
   *  but the last step calls reschedule instead of confirm, and service
   *  selection is pre-filled from the existing booking's segments. */
  bookingId?: string;
  initialServiceIds?: string[];
}

type Step = "select" | "review";

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function toDatetimeLocalDefault(): string {
  const d = new Date(Date.now() + 60 * 60 * 1000); // an hour from now, as a sane default
  d.setMinutes(0, 0, 0);
  return d.toISOString().slice(0, 16);
}

export function BookingWizard({ open, onClose, locationId, bookingId, initialServiceIds }: BookingWizardProps) {
  const qc = useQueryClient();
  const isReschedule = Boolean(bookingId);

  const [step, setStep] = useState<Step>("select");
  const [selectedServiceIds, setSelectedServiceIds] = useState<string[]>([]);
  const [earliestStart, setEarliestStart] = useState(toDatetimeLocalDefault());
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [proposeError, setProposeError] = useState<string>("");
  const [customerName, setCustomerName] = useState("");
  const [customerPhone, setCustomerPhone] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");

  useEffect(() => {
    if (open) {
      setStep("select");
      setSelectedServiceIds(initialServiceIds ?? []);
      setEarliestStart(toDatetimeLocalDefault());
      setProposal(null);
      setProposeError("");
      setCustomerName("");
      setCustomerPhone("");
      setCustomerEmail("");
    }
  }, [open, initialServiceIds]);

  const { data: servicesPage } = useQuery({
    queryKey: queryKeys.services(locationId, 1, 100),
    queryFn: () => listServices(locationId, 1, 100),
    enabled: open,
  });
  const services = servicesPage?.data ?? [];

  const { data: employees = [] } = useQuery({
    queryKey: queryKeys.employees(locationId),
    queryFn: () => listEmployees(locationId),
    enabled: open,
  });

  const serviceByID = new Map<string, Service>(services.map((s) => [s.id, s]));
  const employeeByID = new Map<string, Employee>(employees.map((e) => [e.id, e]));

  const proposeMut = useMutation({
    mutationFn: () =>
      proposeBooking(locationId, {
        services: selectedServiceIds.map((id) => ({ service_id: id })),
        earliest_start: new Date(earliestStart).toISOString(),
      }),
    onSuccess: (p) => {
      setProposal(p);
      setProposeError("");
      setStep("review");
    },
    onError: (err: ApiError) => {
      if (err.code === "NO_AVAILABILITY") {
        const serviceName = serviceByID.get(err.serviceId ?? "")?.name ?? "a service";
        setProposeError(
          err.reason === "NO_EMPLOYEE_ASSIGNED"
            ? `No employee is assigned to ${serviceName} yet.`
            : `No one qualified for ${serviceName} is free at that time.`
        );
      } else {
        setProposeError(err.message);
      }
    },
  });

  const confirmMut = useMutation({
    mutationFn: () =>
      confirmBooking(locationId, {
        customer_name: customerName,
        customer_phone: customerPhone || undefined,
        customer_email: customerEmail || undefined,
        segments: proposal!.segments,
      }),
    onSuccess: () => {
      toast.success("Booking confirmed");
      qc.invalidateQueries({ queryKey: ["bookings"] });
      onClose();
    },
    onError: (err: ApiError) => {
      if (err.code === undefined && err.message.toLowerCase().includes("no longer available")) {
        toast.error("That slot was just taken — please find a new time.");
        setStep("select");
        setProposal(null);
      } else {
        toast.error(err.message);
      }
    },
  });

  const rescheduleMut = useMutation({
    mutationFn: () => rescheduleBooking(locationId, bookingId!, { segments: proposal!.segments }),
    onSuccess: () => {
      toast.success("Booking rescheduled");
      qc.invalidateQueries({ queryKey: ["bookings"] });
      onClose();
    },
    onError: (err: ApiError) => {
      if (err.message.toLowerCase().includes("no longer available")) {
        toast.error("That slot was just taken — please find a new time.");
        setStep("select");
        setProposal(null);
      } else {
        toast.error(err.message);
      }
    },
  });

  function toggleService(id: string) {
    setSelectedServiceIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]));
  }

  function handleFindTimes() {
    setProposeError("");
    proposeMut.mutate();
  }

  function handleConfirm() {
    if (isReschedule) rescheduleMut.mutate();
    else confirmMut.mutate();
  }

  const confirming = confirmMut.isPending || rescheduleMut.isPending;

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{isReschedule ? "Reschedule booking" : "New booking"}</DialogTitle>
          <DialogDescription>
            {step === "select"
              ? "Pick the services and the earliest time to look from."
              : "Review the proposed times before confirming — nothing is booked yet."}
          </DialogDescription>
        </DialogHeader>

        {step === "select" && (
          <>
            <DialogBody className="flex flex-col gap-4">
              <FormField label="Services" required>
                <div className="flex flex-col gap-1.5 max-h-56 overflow-y-auto rounded-sm border border-border p-2">
                  {services.length === 0 && (
                    <p className="text-sm text-content-secondary p-2">No services yet at this location.</p>
                  )}
                  {services.map((svc) => (
                    <label key={svc.id} className="flex items-center gap-2.5 rounded-sm px-2 py-1.5 hover:bg-muted cursor-pointer">
                      <Checkbox
                        checked={selectedServiceIds.includes(svc.id)}
                        onCheckedChange={() => toggleService(svc.id)}
                      />
                      <span className="flex-1">{svc.name}</span>
                      <span className="text-xs text-content-secondary">{svc.duration_minutes} min · ${svc.price.toFixed(2)}</span>
                    </label>
                  ))}
                </div>
              </FormField>

              <FormField label="Earliest start" required hint="The solver looks for the first available time at or after this">
                <Input type="datetime-local" value={earliestStart} onChange={(e) => setEarliestStart(e.target.value)} />
              </FormField>

              {proposeError && <p className="text-sm text-destructive">{proposeError}</p>}
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
              <Button
                onClick={handleFindTimes}
                loading={proposeMut.isPending}
                disabled={selectedServiceIds.length === 0}
              >
                Find times
                <ArrowRight className="size-3.5" />
              </Button>
            </DialogFooter>
          </>
        )}

        {step === "review" && proposal && (
          <>
            <DialogBody className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                {proposal.segments.map((seg, i) => {
                  const prevEnd = i > 0 ? proposal.segments[i - 1].end : null;
                  const gapMinutes = prevEnd ? Math.round((new Date(seg.start).getTime() - new Date(prevEnd).getTime()) / 60000) : 0;
                  return (
                    <div key={i}>
                      {gapMinutes > 0 && (
                        <div className="flex items-center gap-2 pl-3 py-1 text-xs text-content-secondary italic">
                          <Clock className="size-3" />
                          {gapMinutes} min gap
                        </div>
                      )}
                      <div className="flex items-center justify-between rounded-sm border border-border p-3">
                        <div>
                          <p className="font-medium text-foreground">{serviceByID.get(seg.service_id)?.name ?? seg.service_id}</p>
                          <p className="text-xs text-content-secondary">{employeeByID.get(seg.employee_id)?.name ?? seg.employee_id}</p>
                        </div>
                        <div className="text-right">
                          <p className="tabular-nums font-medium">{formatTime(seg.start)} – {formatTime(seg.end)}</p>
                          <p className="text-xs text-content-secondary">${seg.price.toFixed(2)}</p>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="flex items-center justify-between border-t border-border pt-3">
                <span className="text-sm text-content-secondary">Total</span>
                <span className="font-bold">${proposal.total_price.toFixed(2)}</span>
              </div>

              {!isReschedule && (
                <>
                  <FormField label="Customer name" required>
                    <Input value={customerName} onChange={(e) => setCustomerName(e.target.value)} placeholder="Jane Doe" />
                  </FormField>
                  <div className="grid grid-cols-2 gap-3">
                    <FormField label="Phone">
                      <Input value={customerPhone} onChange={(e) => setCustomerPhone(e.target.value)} placeholder="Optional" />
                    </FormField>
                    <FormField label="Email">
                      <Input value={customerEmail} onChange={(e) => setCustomerEmail(e.target.value)} placeholder="Optional" />
                    </FormField>
                  </div>
                </>
              )}
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="secondary" onClick={() => setStep("select")}>
                <RotateCcw className="size-3.5" />
                Back
              </Button>
              <Button
                onClick={handleConfirm}
                loading={confirming}
                disabled={!isReschedule && !customerName.trim()}
              >
                {isReschedule ? "Confirm reschedule" : "Confirm booking"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
