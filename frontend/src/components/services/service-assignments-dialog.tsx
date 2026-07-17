import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, Button, Checkbox,
} from "@flowposltd/ui";
import { DialogBody } from "@/components/ui/dialog-body";
import { type Service } from "@/types";
import { listEmployees, listAssignedEmployees, assignEmployee, unassignEmployee } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

interface ServiceAssignmentsDialogProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  service: Service | undefined;
}

// Which employees can perform this service — the EMPLOYEE_SERVICE join from
// the app-bookings design doc. Toggling a checkbox assigns/unassigns
// immediately (no separate save step) since each toggle is its own request.
export function ServiceAssignmentsDialog({ open, onClose, locationId, service }: ServiceAssignmentsDialogProps) {
  const qc = useQueryClient();

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

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Employees for {service?.name}</DialogTitle>
          <DialogDescription>
            Choose which employees can be booked for this service.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="flex flex-col gap-2">
          {isLoading && <p className="text-sm text-content-secondary">Loading…</p>}
          {!isLoading && allEmployees.length === 0 && (
            <p className="text-sm text-content-secondary">
              No employees synced for this location yet.
            </p>
          )}
          {allEmployees.map((emp) => (
            <label
              key={emp.id}
              className="flex items-center gap-2.5 rounded-sm px-2 py-1.5 hover:bg-muted cursor-pointer"
            >
              <Checkbox
                checked={assignedIds.has(emp.id)}
                onCheckedChange={(checked) => toggle(emp.id, checked === true)}
                disabled={!emp.active}
              />
              <span className={emp.active ? "" : "text-content-secondary italic"}>
                {emp.name}
                {!emp.active && " (inactive)"}
              </span>
            </label>
          ))}
        </DialogBody>
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
