import { useState } from "react";
import { Alert, AlertTitle, AlertDescription, Button } from "@flowposltd/ui";
import { AlertTriangle } from "lucide-react";
import { useLocationContext } from "@/contexts/location-context";
import { TimezoneDialog } from "./timezone-dialog";

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
      <TimezoneDialog open={open} onClose={() => setOpen(false)} location={selectedLocation} />
    </>
  );
}
