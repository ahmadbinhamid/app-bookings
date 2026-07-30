import { useState } from "react";
import { Alert, AlertTitle, AlertDescription, Button } from "@flowposltd/ui";
import { Info } from "lucide-react";
import { useLocationContext } from "@/contexts/location-context";
import { TimezoneDialog } from "./timezone-dialog";

// Every newly-synced location starts on app-bookings' own UTC default (see
// sync.Service.syncLocations) — timezone_confirmed is false until an admin
// explicitly sets the real one. This is informational, not an error state:
// UTC is a perfectly working default, just one worth double-checking.
export function TimezoneBanner() {
  const { selectedLocation } = useLocationContext();
  const [open, setOpen] = useState(false);

  if (!selectedLocation || selectedLocation.timezone_confirmed) return null;

  return (
    <>
      <Alert className="border-info-text/30 bg-info-bg">
        <Info className="size-4 text-info-text" />
        <AlertTitle>Timezone set to default (UTC) for {selectedLocation.name}</AlertTitle>
        <AlertDescription className="flex items-center justify-between gap-4">
          <span>
            Bookings and employee availability are calculated using this timezone —
            set the real one if this location isn't in UTC.
          </span>
          <Button size="sm" onClick={() => setOpen(true)}>Set timezone</Button>
        </AlertDescription>
      </Alert>
      <TimezoneDialog open={open} onClose={() => setOpen(false)} location={selectedLocation} />
    </>
  );
}
