import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/lib/toast";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, Button, Input,
} from "@flowposltd/ui";
import { Search, Check } from "lucide-react";
import { type Service } from "@/types";
import { listEmployees, listAssignedEmployees, assignEmployee, unassignEmployee } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { cn } from "@/utils";

interface ServiceAssignmentsDialogProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  service: Service | undefined;
}

// Which employees can perform this service — the EMPLOYEE_SERVICE join from
// the app-bookings design doc. Toggling a row assigns/unassigns immediately
// (no separate save step) since each toggle is its own request.
export function ServiceAssignmentsDialog({ open, onClose, locationId, service }: ServiceAssignmentsDialogProps) {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");

  const { data: allEmployees = [] } = useQuery({
    queryKey: queryKeys.employees(locationId),
    queryFn: () => listEmployees(locationId),
    enabled: open,
  });

  const { data: assigned = [], isLoading } = useQuery({
    queryKey: service ? queryKeys.serviceEmployees(locationId, service.id) : ["service-employees", "none"],
    queryFn: () => listAssignedEmployees(locationId, service!.id),
    enabled: open && Boolean(service),
  });

  const assignedIds = new Set(assigned.map((e) => e.id));

  const invalidate = () => {
    if (service) qc.invalidateQueries({ queryKey: queryKeys.serviceEmployees(locationId, service.id) });
  };

  const assignMut = useMutation({
    mutationFn: (employeeId: string) => assignEmployee(locationId, service!.id, employeeId),
    onSuccess: invalidate,
    onError: (err: Error) => toast.error(err.message),
  });

  const unassignMut = useMutation({
    mutationFn: (employeeId: string) => unassignEmployee(locationId, service!.id, employeeId),
    onSuccess: invalidate,
    onError: (err: Error) => toast.error(err.message),
  });

  function toggle(employeeId: string, checked: boolean) {
    if (checked) assignMut.mutate(employeeId);
    else unassignMut.mutate(employeeId);
  }

  // Duplicate display names (e.g. two "Alice Walters" synced from different
  // FlowPOS records) need a second identifying line so rows stay tellable
  // apart — fall back to phone when the name is unique.
  const nameCounts = new Map<string, number>();
  allEmployees.forEach((e) => nameCounts.set(e.name, (nameCounts.get(e.name) ?? 0) + 1));

  const q = search.trim().toLowerCase();
  const matched = allEmployees.filter(
    (e) => !q || e.name.toLowerCase().includes(q) || (e.email ?? "").toLowerCase().includes(q) || (e.phone ?? "").includes(q)
  );

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="w-[calc(100%-2rem)] flex flex-col max-h-[calc(100vh-8rem)]">
        <DialogHeader>
          <DialogTitle>
            Employees for <span className="capitalize">{service?.name}</span>
          </DialogTitle>
          <DialogDescription>Choose who can be booked for this service.</DialogDescription>
        </DialogHeader>

        <div className="relative shrink-0">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 size-4 text-content-tertiary pointer-events-none" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name, email or phone…"
            className="pl-10"
          />
        </div>
        <div className="flex items-center justify-between text-body-3 shrink-0">
          <span className="text-content-tertiary">{matched.length} of {allEmployees.length} employees</span>
          <span className="font-medium text-primary">{assignedIds.size} assigned</span>
        </div>

        <div className="flex-1 overflow-y-auto flex flex-col gap-1 -mx-1 px-1">
          {isLoading && <p className="text-body-2 text-content-secondary p-2">Loading…</p>}
          {!isLoading && matched.length === 0 && (
            <div className="py-10 text-center">
              <p className="text-body-2 font-medium text-foreground">No matches</p>
              <p className="text-body-3 text-content-secondary mt-0.5">No employees match "{search}".</p>
            </div>
          )}
          {matched.map((emp) => {
            const on = assignedIds.has(emp.id);
            const dup = (nameCounts.get(emp.name) ?? 0) > 1;
            const detail = dup ? (emp.email || emp.phone || "—") : (emp.phone || emp.email || "—");
            return (
              <div
                key={emp.id}
                onClick={() => emp.active && toggle(emp.id, !on)}
                className={cn(
                  "flex items-center gap-3 rounded-md p-2.5 transition-colors border",
                  emp.active ? "cursor-pointer" : "opacity-50 cursor-not-allowed",
                  on ? "border-primary/20 bg-primary/5" : "border-transparent hover:bg-muted/60"
                )}
              >
                <PersonAvatar name={emp.name} seed={emp.name + emp.email} size="sm" />
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-body-2 text-foreground truncate">{emp.name}</div>
                  <div className="text-body-3 text-content-tertiary truncate">{detail}{!emp.active && " · inactive"}</div>
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

        <DialogFooter className="shrink-0">
          <Button variant="secondary" onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
