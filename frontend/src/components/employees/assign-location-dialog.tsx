import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
  Button, Select, SelectTrigger, SelectValue, SelectContent, SelectItem, Alert, AlertDescription,
} from "@flowposltd/ui";
import { MapPin } from "lucide-react";
import { type Employee } from "@/types";
import { assignEmployeeLocation } from "@/lib/api";
import { type ApiError } from "@/lib/api/client";
import { useLocationContext } from "@/contexts/location-context";

interface AssignLocationDialogProps {
  open: boolean;
  onClose: () => void;
  employee: Employee | undefined;
}

// Moves (or makes, for a not-yet-assigned employee) a single employee's
// location assignment — one location at a time (design resolution: an
// employee can't work at two locations at once). The backend blocks the
// move with a 409 (code HAS_FUTURE_BOOKINGS) while the employee still has
// upcoming bookings at their current location; that's surfaced below rather
// than silently orphaning those bookings.
export function AssignLocationDialog({ open, onClose, employee }: AssignLocationDialogProps) {
  const qc = useQueryClient();
  const { locations } = useLocationContext();
  const [locationId, setLocationId] = useState<string | undefined>(employee?.location_id ?? undefined);
  const [blockedReason, setBlockedReason] = useState<string | undefined>();

  // Re-seed the selection whenever a different employee opens the dialog —
  // there's no key-based remount here (the dialog is a single shared
  // instance on the page), so this runs the sync manually on open.
  if (open && locationId === undefined && employee?.location_id) {
    setLocationId(employee.location_id);
  }

  const assignMut = useMutation({
    mutationFn: () => assignEmployeeLocation(employee!.id, locationId!),
    onSuccess: () => {
      toast.success(`${employee!.name} assigned to ${locations.find((l) => l.id === locationId)?.name ?? "the location"}`);
      // Broad match: covers allEmployees(), unassignedEmployees(), and
      // employees(locationId) for both the old and new location at once.
      qc.invalidateQueries({ queryKey: ["employees"] });
      handleClose();
    },
    onError: (err: ApiError) => {
      if (err.code === "HAS_FUTURE_BOOKINGS") {
        setBlockedReason(err.message);
      } else {
        toast.error(err.message);
      }
    },
  });

  function handleClose() {
    setLocationId(undefined);
    setBlockedReason(undefined);
    onClose();
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent className="w-[calc(100%-2rem)] sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Assign to location</DialogTitle>
          <DialogDescription>
            {employee?.name} can only be assigned to one location at a time.
          </DialogDescription>
        </DialogHeader>

        <Select
          value={locationId}
          onValueChange={(v) => {
            setLocationId(v);
            setBlockedReason(undefined);
          }}
        >
          <SelectTrigger>
            <MapPin className="size-3.5 shrink-0 text-primary" />
            <SelectValue placeholder="Select a location" />
          </SelectTrigger>
          <SelectContent>
            {locations.map((loc) => (
              <SelectItem key={loc.id} value={loc.id}>{loc.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        {blockedReason && (
          <Alert variant="destructive">
            <AlertDescription>{blockedReason}</AlertDescription>
          </Alert>
        )}

        <DialogFooter>
          <Button variant="secondary" onClick={handleClose}>Cancel</Button>
          <Button
            onClick={() => assignMut.mutate()}
            disabled={!locationId || locationId === employee?.location_id}
            loading={assignMut.isPending}
          >
            Assign
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
