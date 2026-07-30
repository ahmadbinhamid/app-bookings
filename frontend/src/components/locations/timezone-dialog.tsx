import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/lib/toast";
import {
  Button, Input,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "@flowposltd/ui";
import { DialogBody } from "@/components/ui/dialog-body";
import { FormField } from "@/components/ui/form-field";
import { setLocationTimezone } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Location } from "@/types";

interface TimezoneDialogProps {
  open: boolean;
  onClose: () => void;
  location: Location | undefined;
}

// Shared by TimezoneBanner (acts on the currently-selected location) and
// LocationsPage (acts on whichever row's "Set timezone" was clicked) — takes
// the location as a prop rather than reading useLocationContext itself so
// either caller can point it at a location that isn't the selected one.
export function TimezoneDialog({ open, onClose, location }: TimezoneDialogProps) {
  const qc = useQueryClient();
  const [timezone, setTimezone] = useState("");
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: (tz: string) => setLocationTimezone(location!.id, tz),
    onSuccess: () => {
      toast.success("Timezone set");
      qc.invalidateQueries({ queryKey: queryKeys.locations() });
      setTimezone("");
      onClose();
    },
    onError: (err: Error) => setError(err.message),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!timezone.trim()) {
      setError("Timezone is required");
      return;
    }
    mutation.mutate(timezone.trim());
  }

  function handleClose() {
    setTimezone("");
    setError("");
    onClose();
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent className="w-[calc(100%-2rem)] max-w-lg">
        <DialogHeader>
          <DialogTitle>Set location timezone</DialogTitle>
          <DialogDescription>
            Enter an IANA timezone name for {location?.name} — e.g. "Europe/London",
            "America/New_York", "Australia/Sydney".
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <DialogBody>
            <FormField label="Timezone" required hint="Must be a real IANA zone name">
              <Input
                value={timezone}
                onChange={(e) => { setTimezone(e.target.value); setError(""); }}
                placeholder="Europe/London"
                error={error}
                autoFocus
              />
            </FormField>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="secondary" onClick={handleClose} disabled={mutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" loading={mutation.isPending}>Save</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
