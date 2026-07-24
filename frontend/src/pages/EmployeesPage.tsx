import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge, Button, Tooltip, TooltipTrigger, TooltipContent,
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from "@flowposltd/ui";
import { Users, CalendarDays, RotateCw } from "lucide-react";
import { listEmployees } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Employee } from "@/types";
import { EmptyState } from "@/components/ui/empty-state";
import { PageFallback } from "@/components/ui/page-fallback";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { EmployeeDetailDialog } from "@/components/employees/employee-detail-dialog";
import { useLocationContext } from "@/contexts/location-context";

// Employees are read-only here — synced from FlowPOS (see
// internal/modules/sync). What an admin manages by hand is their schedule
// and time off, via the detail dialog.
export default function EmployeesPage() {
  const { selectedLocationId, isLoading: locationsLoading } = useLocationContext();
  const [managing, setManaging] = useState<Employee | undefined>();

  const { data: employees = [], isLoading } = useQuery({
    queryKey: selectedLocationId ? queryKeys.employees(selectedLocationId) : ["employees", "none"],
    queryFn: () => listEmployees(selectedLocationId!),
    enabled: Boolean(selectedLocationId),
  });

  if (locationsLoading) return <PageFallback />;
  if (!selectedLocationId) {
    return (
      <div className="p-6">
        <EmptyState
          variant="hero"
          icon={Users}
          title="No location yet"
          description="Employees are synced from FlowPOS per location — once sync has run, they'll show up here."
        />
      </div>
    );
  }

  return (
    <div className="p-4 sm:p-6 flex flex-col gap-5">
      <div>
        <div className="flex items-center gap-2.5">
          <h1>Employees</h1>
          {employees.length > 0 && <Badge>{employees.length}</Badge>}
        </div>
        <p className="lead mt-0.5 flex items-center gap-1.5 flex-wrap">
          Synced from
          <span className="inline-flex items-center gap-1 rounded-md bg-primary/10 px-2 py-0.5 text-body-3 font-medium text-primary">
            <RotateCw className="size-3" />
            FlowPOS
          </span>
          — manage each employee's working hours and time off here.
        </p>
      </div>

      {isLoading ? (
        <p className="text-body-2 text-content-secondary">Loading…</p>
      ) : employees.length === 0 ? (
        <EmptyState
          variant="hero"
          icon={Users}
          title="No employees yet"
          description="Employees appear here automatically once FlowPOS sync has run for this location."
        />
      ) : (
        <div className="rounded-md border border-border bg-card shadow-card overflow-x-auto">
          <Table className="min-w-180">
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Phone</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {employees.map((emp) => (
                <TableRow key={emp.id}>
                  <TableCell>
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="relative shrink-0">
                        <PersonAvatar name={emp.name} seed={emp.name + emp.email} size="sm" />
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <div className="absolute -bottom-1 -right-1 flex size-4.5 items-center justify-center rounded-md bg-primary ring-2 ring-card">
                              <RotateCw className="size-2.5 text-primary-foreground" />
                            </div>
                          </TooltipTrigger>
                          <TooltipContent>
                            Synced from FlowPOS{emp.synced_at && ` · ${new Date(emp.synced_at).toLocaleString()}`}
                          </TooltipContent>
                        </Tooltip>
                      </div>
                      <span className="font-medium text-foreground truncate">{emp.name}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-content-secondary truncate">{emp.email || "—"}</TableCell>
                  <TableCell className="text-content-secondary tabular-nums">{emp.phone || "—"}</TableCell>
                  <TableCell>
                    <Badge variant={emp.active ? "status-success" : "secondary"} dotColor={emp.active ? "bg-emerald-500" : undefined}>
                      {emp.active ? "Active" : "Inactive"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="secondary" size="sm" onClick={() => setManaging(emp)}>
                      <CalendarDays className="size-3.5" />
                      Schedule
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <EmployeeDetailDialog
        open={Boolean(managing)}
        onClose={() => setManaging(undefined)}
        locationId={selectedLocationId}
        employee={managing}
      />
    </div>
  );
}
