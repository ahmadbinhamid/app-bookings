import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/lib/toast";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, Button, Input,
} from "@flowposltd/ui";
import { Search, Check } from "lucide-react";
import { type Location } from "@/types";
import { listUnassignedEmployees, assignEmployeeLocation } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { EmptyState } from "@/components/ui/empty-state";
import { cn } from "@/utils";

interface AssignEmployeesDialogProps {
  open: boolean;
  onClose: () => void;
  location: Location | undefined;
}

// Assigns one or more employees to `location`. Only ever lists the tenant's
// UNASSIGNED employees — an employee already at a (possibly different)
// location is moved via the per-employee "Assign to location" action on the
// Employees page instead, never picked from here (design resolution: one
// location can have many employees, but one employee only ever belongs to
// one location — this dialog only ever adds, never moves).
export function AssignEmployeesDialog({ open, onClose, location }: AssignEmployeesDialogProps) {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const { data: unassigned = [], isLoading } = useQuery({
    queryKey: queryKeys.unassignedEmployees(),
    queryFn: listUnassignedEmployees,
    enabled: open,
  });

  const assignMut = useMutation({
    mutationFn: async () => {
      await Promise.all([...selected].map((employeeId) => assignEmployeeLocation(employeeId, location!.id)));
    },
    onSuccess: () => {
      toast.success(`${selected.size} employee${selected.size === 1 ? "" : "s"} assigned to ${location!.name}`);
      // Broad match: covers allEmployees(), unassignedEmployees(), and
      // employees(location.id) all at once.
      qc.invalidateQueries({ queryKey: ["employees"] });
      handleClose();
    },
    onError: (err: Error) => toast.error(err.message),
  });

  function handleClose() {
    setSearch("");
    setSelected(new Set());
    onClose();
  }

  function toggle(employeeId: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(employeeId)) next.delete(employeeId);
      else next.add(employeeId);
      return next;
    });
  }

  const q = search.trim().toLowerCase();
  const matched = unassigned.filter(
    (e) => !q || e.name.toLowerCase().includes(q) || (e.email ?? "").toLowerCase().includes(q) || (e.phone ?? "").includes(q)
  );

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent className="w-[calc(100%-2rem)] flex flex-col max-h-[calc(100vh-8rem)]">
        <DialogHeader>
          <DialogTitle>Assign employees to {location?.name}</DialogTitle>
          <DialogDescription>
            Only employees not yet assigned anywhere are shown — an employee already at another
            location must be moved from the Employees page instead.
          </DialogDescription>
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

        <div className="flex-1 overflow-y-auto flex flex-col gap-1 -mx-1 px-1">
          {isLoading && <p className="text-body-2 text-content-secondary p-2">Loading…</p>}
          {!isLoading && matched.length === 0 && (
            <EmptyState
              title={unassigned.length === 0 ? "No unassigned employees" : "No matches"}
              description={
                unassigned.length === 0
                  ? "Every synced employee already belongs to a location."
                  : `No unassigned employees match "${search}".`
              }
            />
          )}
          {matched.map((emp) => {
            const on = selected.has(emp.id);
            return (
              <div
                key={emp.id}
                onClick={() => toggle(emp.id)}
                className={cn(
                  "flex items-center gap-3 rounded-md p-2.5 transition-colors border cursor-pointer",
                  on ? "border-primary/20 bg-primary/5" : "border-transparent hover:bg-muted/60"
                )}
              >
                <PersonAvatar name={emp.name} seed={emp.name + emp.email} size="sm" />
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-body-2 text-foreground truncate">{emp.name}</div>
                  <div className="text-body-3 text-content-tertiary truncate">{emp.email || emp.phone || "—"}</div>
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
          <Button variant="secondary" onClick={handleClose}>Cancel</Button>
          <Button onClick={() => assignMut.mutate()} disabled={selected.size === 0} loading={assignMut.isPending}>
            Assign {selected.size > 0 && selected.size}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
