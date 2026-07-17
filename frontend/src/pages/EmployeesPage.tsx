import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge, Button,
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from "@flowposltd/ui";
import { Users, Settings2 } from "lucide-react";
import { listEmployees } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Employee } from "@/types";
import { EmptyState } from "@/components/ui/empty-state";
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

  if (locationsLoading) return null;
  if (!selectedLocationId) {
    return (
      <div className="p-6">
        <EmptyState
          icon={Users}
          title="No location yet"
          description="Employees are synced from FlowPOS per location — once sync has run, they'll show up here."
        />
      </div>
    );
  }

  return (
    <div className="p-6 flex flex-col gap-5">
      <div>
        <div className="flex items-center gap-2">
          <h1>Employees</h1>
          {employees.length > 0 && <Badge>{employees.length}</Badge>}
        </div>
        <p className="lead mt-0.5">
          Synced from FlowPOS. Manage each employee's working hours and time off here.
        </p>
      </div>

      {isLoading ? (
        <p className="text-sm text-content-secondary">Loading…</p>
      ) : employees.length === 0 ? (
        <EmptyState
          icon={Users}
          title="No employees yet"
          description="Employees appear here automatically once FlowPOS sync has run for this location."
        />
      ) : (
        <Table>
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
                <TableCell className="font-medium text-foreground">{emp.name}</TableCell>
                <TableCell className="text-content-secondary">{emp.email || "—"}</TableCell>
                <TableCell className="text-content-secondary">{emp.phone || "—"}</TableCell>
                <TableCell>
                  <Badge variant={emp.active ? "status-success" : "secondary"}>
                    {emp.active ? "Active" : "Inactive"}
                  </Badge>
                </TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" onClick={() => setManaging(emp)}>
                    <Settings2 className="size-3.5" />
                    Manage
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
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
