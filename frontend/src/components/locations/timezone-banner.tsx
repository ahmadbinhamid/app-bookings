import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Alert, AlertTitle, AlertDescription, Button, Input,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "@flowposltd/ui";
import { AlertTriangle } from "lucide-react";
import { DialogBody } from "@/components/ui/dialog-body";
import { FormField } from "@/components/ui/form-field";
import { setLocationTimezone } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useLocationContext } from "@/contexts/location-context";

// Design resolution: "Until a location's timezone is explicitly set,
// surface a visible warning on that location (UI banner + flag in the API
// response)" — timezone_confirmed is that flag; this is the banner.
export function TimezoneBanner() {
  const { selectedLocation } = useLocationContext();
  const [open, setOpen] = useState(false);

  if (!selectedLocation || selectedLocation.timezone_confirmed) return null;

  return (
    <>
      <Alert className="border-destructive/30 bg-destructive/5">
        <AlertTriangle className="size-4 text-destructive" />
        <AlertTitle>Timezone not set for {selectedLocation.name}</AlertTitle>
        <AlertDescription className="flex items-center justify-between gap-4">
          <span>
            Bookings need a real timezone to work out employee availability correctly.
            Currently defaulting to UTC, which is almost certainly wrong.
          </span>
          <Button size="sm" onClick={() => setOpen(true)}>Set timezone</Button>
        </AlertDescription>
      </Alert>
      <TimezoneDialog open={open} onClose={() => setOpen(false)} />
    </>
  );
}

function TimezoneDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const qc = useQueryClient();
  const { selectedLocation } = useLocationContext();
  const [timezone, setTimezone] = useState("");
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: (tz: string) => setLocationTimezone(selectedLocation!.id, tz),
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

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="w-[calc(100%-2rem)] max-w-lg">
        <DialogHeader>
          <DialogTitle>Set location timezone</DialogTitle>
          <DialogDescription>
            Enter an IANA timezone name for {selectedLocation?.name} — e.g. "Europe/London",
            "America/New_York", "Australia/Sydney". This cannot be undone automatically by
            FlowPOS sync once set.
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
            <Button type="button" variant="secondary" onClick={onClose} disabled={mutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" loading={mutation.isPending}>Save</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
