import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge, Button,
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem,
} from "@flowposltd/ui";
import { MapPin, Users, AlertTriangle, Clock, MoreHorizontal } from "lucide-react";
import { listLocations } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Location } from "@/types";
import { EmptyState } from "@/components/ui/empty-state";
import { PageFallback } from "@/components/ui/page-fallback";
import { AssignEmployeesDialog } from "@/components/locations/assign-employees-dialog";
import { TimezoneDialog } from "@/components/locations/timezone-dialog";
import { SyncButton } from "@/components/ui/sync-button";

// Locations are read-only here — synced from FlowPOS (see
// internal/modules/sync). What an admin manages by hand is which employees
// belong to each one, and each location's timezone: FlowPOS has no
// employee-location relationship at all, and its own location timezone
// isn't trusted as confirmed truth (see location.Repository.Upsert) — both
// live entirely in app-bookings.
export default function LocationsPage() {
  const [assigningTo, setAssigningTo] = useState<Location | undefined>();
  const [settingTzFor, setSettingTzFor] = useState<Location | undefined>();

  const { data: locations = [], isLoading } = useQuery({
    queryKey: queryKeys.locations(),
    queryFn: listLocations,
  });

  if (isLoading) return <PageFallback />;

  return (
    <div className="p-4 sm:p-6 flex flex-col gap-5">
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div>
          <div className="flex items-center gap-2.5">
            <h1>Locations</h1>
            {locations.length > 0 && <Badge>{locations.length}</Badge>}
          </div>
          <p className="lead mt-0.5">Assign employees to each location — one employee can only belong to one at a time.</p>
        </div>
        <SyncButton />
      </div>

      {locations.length === 0 ? (
        <EmptyState
          variant="hero"
          icon={MapPin}
          title="No locations yet"
          description="Locations appear here automatically once FlowPOS sync has run."
        />
      ) : (
        <div className="rounded-md border border-border bg-card shadow-card overflow-x-auto">
          <Table className="min-w-120">
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Timezone</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {locations.map((loc) => (
                <TableRow key={loc.id}>
                  <TableCell>
                    <div className="flex items-center gap-2 font-medium text-foreground">
                      <MapPin className="size-3.5 shrink-0 text-primary" />
                      {loc.name}
                    </div>
                  </TableCell>
                  <TableCell className="text-content-secondary">
                    <div className="flex items-center gap-1.5">
                      {loc.timezone}
                      {!loc.timezone_confirmed && (
                        <Badge variant="destructive" className="gap-1">
                          <AlertTriangle className="size-3" />
                          Unconfirmed
                        </Badge>
                      )}
                    </div>
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
                        <DropdownMenuItem onSelect={() => setTimeout(() => setSettingTzFor(loc), 0)}>
                          <Clock className="size-3.5 mr-2" />
                          Set timezone
                        </DropdownMenuItem>
                        <DropdownMenuItem onSelect={() => setTimeout(() => setAssigningTo(loc), 0)}>
                          <Users className="size-3.5 mr-2" />
                          Assign employees
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <AssignEmployeesDialog
        open={Boolean(assigningTo)}
        onClose={() => setAssigningTo(undefined)}
        location={assigningTo}
      />
      <TimezoneDialog
        open={Boolean(settingTzFor)}
        onClose={() => setSettingTzFor(undefined)}
        location={settingTzFor}
      />
    </div>
  );
}
