import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
  Button, Input,
} from "@flowposltd/ui";
import { Scissors, ArrowRight, RotateCcw, CheckCircle2, Check } from "lucide-react";
import { DialogBody } from "@/components/ui/dialog-body";
import { FormField } from "@/components/ui/form-field";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { BookingTimeline } from "@/components/ui/booking-timeline";
import { DatePopover } from "@/components/ui/calendar-popover";
import { TimePopover } from "@/components/ui/time-popover";
import { ApiError } from "@/lib/api/client";
import { proposeBooking, confirmBooking, rescheduleBooking, listServices, listEmployees } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { serviceVisual } from "@/constants/service-visuals";
import { cn, formatMoney, formatDuration, formatTime, formatTimeRange, buildTimeline, toISODateLocal, combineDateAndMinutes } from "@/utils";
import { useLocationContext } from "@/contexts/location-context";
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

// An hour from now, minutes rounded up to the next quarter hour — a sane
// starting point for "earliest start" before the admin adjusts it.
function defaultEarliest(): { date: string; minutes: number } {
  const d = new Date(Date.now() + 60 * 60 * 1000);
  const raw = d.getHours() * 60 + d.getMinutes();
  return { date: toISODateLocal(d), minutes: Math.ceil(raw / 15) * 15 };
}

export function BookingWizard({ open, onClose, locationId, bookingId, initialServiceIds }: BookingWizardProps) {
  const qc = useQueryClient();
  const isReschedule = Boolean(bookingId);
  const { selectedLocation } = useLocationContext();
  const timeZone = selectedLocation?.timezone;

  const [step, setStep] = useState<Step>("select");
  const [selectedServiceIds, setSelectedServiceIds] = useState<string[]>([]);
  const [earliestDate, setEarliestDate] = useState(() => defaultEarliest().date);
  const [earliestMinutes, setEarliestMinutes] = useState(() => defaultEarliest().minutes);
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [proposeError, setProposeError] = useState<string>("");
  const [customerName, setCustomerName] = useState("");
  const [customerPhone, setCustomerPhone] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");

  useEffect(() => {
    if (open) {
      const def = defaultEarliest();
      setStep("select");
      setSelectedServiceIds(initialServiceIds ?? []);
      setEarliestDate(def.date);
      setEarliestMinutes(def.minutes);
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
        earliest_start: combineDateAndMinutes(earliestDate, earliestMinutes),
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
  const selectedServices = selectedServiceIds.map((id) => serviceByID.get(id)).filter(Boolean) as Service[];
  const totalMinutes = selectedServices.reduce((a, s) => a + s.duration_minutes, 0);
  const totalPrice = selectedServices.reduce((a, s) => a + s.price, 0);

  const proposalTimeline =
    proposal &&
    buildTimeline(
      proposal.segments.map((seg, i) => ({
        key: String(i),
        name: serviceByID.get(seg.service_id)?.name ?? seg.service_id,
        start: seg.start,
        end: seg.end,
      })),
      timeZone
    );

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="w-[calc(100%-2rem)] max-w-xl">
        <DialogHeader>
          <DialogTitle>{isReschedule ? "Reschedule booking" : "New booking"}</DialogTitle>
          <DialogDescription>
            {step === "select"
              ? "Pick the services and the earliest time to look from."
              : "Slots found — review the timeline below before confirming."}
          </DialogDescription>
        </DialogHeader>

        <div className="flex gap-2 -mt-1">
          <div className="h-1 flex-1 rounded-full bg-primary" />
          <div className={cn("h-1 flex-1 rounded-full", step === "review" ? "bg-primary" : "bg-muted")} />
        </div>

        {step === "select" && (
          <>
            <DialogBody className="flex flex-col gap-4">
              <div>
                <div className="flex items-center justify-between mb-2.5">
                  <span className="text-body-2 font-medium text-foreground">
                    Services<span className="text-destructive ml-0.5">*</span>
                  </span>
                  <span className="text-body-3 text-content-tertiary">
                    {selectedServices.length ? `${selectedServices.length} selected` : "Select one or more"}
                  </span>
                </div>
                <div className="flex flex-col gap-2.5 max-h-64 overflow-y-auto">
                  {services.length === 0 && (
                    <p className="text-body-2 text-content-secondary p-2">No services yet at this location.</p>
                  )}
                  {services.map((svc) => {
                    const on = selectedServiceIds.includes(svc.id);
                    const v = serviceVisual(svc.name);
                    return (
                      <div
                        key={svc.id}
                        onClick={() => toggleService(svc.id)}
                        className={cn(
                          "flex items-center gap-3.5 rounded-md border p-3 cursor-pointer transition-colors",
                          on ? "border-primary bg-primary/5" : "border-border hover:bg-muted/50"
                        )}
                      >
                        <div
                          className="flex size-10 shrink-0 items-center justify-center rounded-md"
                          style={{ backgroundColor: v.bg, color: v.fg }}
                        >
                          <Scissors className="size-4.5" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-body-2 text-foreground capitalize">{svc.name}</div>
                          <div className="flex gap-3 mt-0.5 text-body-3 text-content-secondary">
                            <span>{svc.duration_minutes} min</span>
                            <span className="font-medium text-foreground">{formatMoney(svc.price)}</span>
                          </div>
                        </div>
                        <div
                          className={cn(
                            "flex size-5.5 shrink-0 items-center justify-center rounded-md",
                            on ? "bg-primary" : "border-2 border-border"
                          )}
                        >
                          {on && <Check className="size-3.5 text-primary-foreground" strokeWidth={3} />}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div>
                <span className="text-body-2 font-medium text-foreground block mb-2.5">
                  Earliest start<span className="text-destructive ml-0.5">*</span>
                </span>
                <div className="flex flex-col gap-2.5 sm:flex-row">
                  <DatePopover value={earliestDate} onChange={setEarliestDate} className="sm:flex-[1.3]" />
                  <TimePopover value={earliestMinutes} onChange={setEarliestMinutes} className="sm:flex-1" />
                </div>
                <p className="text-body-3 text-content-tertiary mt-1.5">
                  The solver looks for the first available slot at or after this time.
                </p>
              </div>

              {selectedServices.length > 0 && (
                <div className="rounded-md border border-primary/20 bg-primary/5 p-3.5">
                  <div className="flex items-center gap-2 text-body-3 font-medium text-primary">
                    <CheckCircle2 className="size-3.5" />
                    {selectedServices.length} service{selectedServices.length === 1 ? "" : "s"} selected
                  </div>
                  <div className="flex gap-6 mt-3">
                    <div>
                      <div className="text-body-3 text-content-secondary font-medium mb-0.5">Total duration</div>
                      <div className="text-body-1 font-semibold text-foreground">{formatDuration(totalMinutes)}</div>
                    </div>
                    <div>
                      <div className="text-body-3 text-content-secondary font-medium mb-0.5">Total price</div>
                      <div className="text-body-1 font-semibold text-primary">{formatMoney(totalPrice)}</div>
                    </div>
                  </div>
                </div>
              )}

              {proposeError && <p className="text-body-2 text-destructive">{proposeError}</p>}
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

        {step === "review" && proposal && proposalTimeline && (
          <>
            <DialogBody className="flex flex-col gap-4">
              <div className="flex items-start gap-2.5 rounded-md border border-transparent bg-tag-success p-3.5">
                <CheckCircle2 className="size-4.5 shrink-0 text-tag-success-text" />
                <p className="text-body-2 text-tag-success-text">
                  <b>Slots found.</b> First available fit starting {formatTime(proposal.segments[0].start, timeZone)}. Review
                  the timeline below.
                </p>
              </div>

              <BookingTimeline
                timeline={proposalTimeline}
                segments={proposal.segments.map((seg, i) => ({
                  key: String(i),
                  name: serviceByID.get(seg.service_id)?.name ?? seg.service_id,
                  start: seg.start,
                  end: seg.end,
                }))}
                variant="large"
                timeZone={timeZone}
              />

              <div className="flex flex-col gap-2">
                {proposal.segments.map((seg, i) => {
                  const name = serviceByID.get(seg.service_id)?.name ?? seg.service_id;
                  const employee = employeeByID.get(seg.employee_id)?.name ?? seg.employee_id;
                  const v = serviceVisual(name);
                  return (
                    <div key={i} className="flex items-center gap-3 rounded-md border border-border p-3">
                      <span className="size-2.5 rounded-[3px] shrink-0" style={{ backgroundColor: v.solid }} />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-body-2 text-foreground capitalize">{name}</div>
                        <div className="flex items-center gap-1.5 mt-0.5 text-body-3 text-content-secondary">
                          <PersonAvatar name={employee} size="xs" />
                          {employee}
                        </div>
                      </div>
                      <div className="text-body-2 font-medium tabular-nums">{formatTimeRange(seg.start, seg.end, timeZone)}</div>
                    </div>
                  );
                })}
              </div>

              <div className="flex items-center justify-between border-t border-border pt-3">
                <span className="text-body-2 text-content-secondary">Total</span>
                <span className="text-heading-3 font-semibold">{formatMoney(proposal.total_price)}</span>
              </div>

              {!isReschedule && (
                <>
                  <FormField label="Customer name" required>
                    <Input value={customerName} onChange={(e) => setCustomerName(e.target.value)} placeholder="Jane Doe" />
                  </FormField>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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
                <Check className="size-3.5" />
                {isReschedule ? "Confirm reschedule" : "Confirm booking"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
