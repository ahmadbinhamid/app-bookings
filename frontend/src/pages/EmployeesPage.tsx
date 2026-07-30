import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge, Button, Tooltip, TooltipTrigger, TooltipContent,
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem,
} from "@flowposltd/ui";
import { Users, CalendarDays, RotateCw, MoreHorizontal, MapPin } from "lucide-react";
import { listAllEmployees } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Employee } from "@/types";
import { EmptyState } from "@/components/ui/empty-state";
import { PageFallback } from "@/components/ui/page-fallback";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { EmployeeDetailDialog } from "@/components/employees/employee-detail-dialog";
import { AssignLocationDialog } from "@/components/employees/assign-location-dialog";
import { SyncButton } from "@/components/ui/sync-button";
import { useLocationContext } from "@/contexts/location-context";
import { formatDateTime } from "@/utils";

export default function EmployeesPage() {
  const { locations, isLoading: locationsLoading } = useLocationContext();
  const [managing, setManaging] = useState<Employee | undefined>();
  const [assigning, setAssigning] = useState<Employee | undefined>();

  const { data: employees = [], isLoading } = useQuery({
    queryKey: queryKeys.allEmployees(),
    queryFn: listAllEmployees,
  });

  const locationById = new Map(locations.map((l) => [l.id, l]));

  if (locationsLoading || isLoading) return <PageFallback />;

  return (
    <div className="p-4 sm:p-6 flex flex-col gap-5">
      <div className="flex items-start justify-between gap-3 flex-wrap">
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
            — assign each to a location, then manage their working hours and time off here.
          </p>
        </div>
        <SyncButton />
      </div>

      {employees.length === 0 ? (
        <EmptyState
          variant="hero"
          icon={Users}
          title="No employees yet"
          description="Employees appear here automatically once FlowPOS sync has run."
        />
      ) : (
        <div className="rounded-md border border-border bg-card shadow-card overflow-x-auto">
          <Table className="min-w-180">
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Location</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Phone</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {employees.map((emp) => {
                const location = emp.location_id ? locationById.get(emp.location_id) : undefined;
                return (
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
                              {/* Explicit UTC, not a per-location timezone: this list spans
                                  employees across every location (or none at all), so there's
                                  no single "right" location timezone to convert this into. */}
                              Synced from FlowPOS{emp.synced_at && ` · ${formatDateTime(emp.synced_at, "UTC")} UTC`}
                            </TooltipContent>
                          </Tooltip>
                        </div>
                        <span className="font-medium text-foreground truncate">{emp.name}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      {location ? (
                        <span className="inline-flex items-center gap-1.5 text-content-secondary">
                          <MapPin className="size-3.5 shrink-0 text-primary" />
                          {location.name}
                        </span>
                      ) : (
                        <Badge variant="secondary">Unassigned</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-content-secondary truncate">{emp.email || "—"}</TableCell>
                    <TableCell className="text-content-secondary tabular-nums">{emp.phone || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={emp.active ? "status-success" : "secondary"} dotColor={emp.active ? "bg-emerald-500" : undefined}>
                        {emp.active ? "Active" : "Inactive"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="size-8">
                            <MoreHorizontal className="size-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {/* Deferred to a macrotask so the DropdownMenu's own
                              close/unmount fully commits before the Dialog
                              mounts — see service-card.tsx for why (overlapping
                              Radix modal teardown otherwise leaves
                              pointer-events locked). */}
                          {emp.location_id ? (
                            <DropdownMenuItem onSelect={() => setTimeout(() => setManaging(emp), 0)}>
                              <CalendarDays className="size-3.5 mr-2" />
                              Schedule
                            </DropdownMenuItem>
                          ) : (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                {/* disabled items don't fire onSelect, and still need
                                    a wrapping element for the tooltip to anchor to */}
                                <div>
                                  <DropdownMenuItem disabled>
                                    <CalendarDays className="size-3.5 mr-2" />
                                    Schedule
                                  </DropdownMenuItem>
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>Assign a location first</TooltipContent>
                            </Tooltip>
                          )}
                          <DropdownMenuItem onSelect={() => setTimeout(() => setAssigning(emp), 0)}>
                            <MapPin className="size-3.5 mr-2" />
                            {emp.location_id ? "Change location" : "Assign to location"}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {managing?.location_id && (
        <EmployeeDetailDialog
          open={Boolean(managing)}
          onClose={() => setManaging(undefined)}
          locationId={managing.location_id}
          employee={managing}
        />
      )}

      <AssignLocationDialog
        open={Boolean(assigning)}
        onClose={() => setAssigning(undefined)}
        employee={assigning}
      />
    </div>
  );
}
