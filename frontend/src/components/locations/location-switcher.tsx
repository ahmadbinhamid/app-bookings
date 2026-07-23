import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@flowposltd/ui";
import { MapPin, AlertTriangle } from "lucide-react";
import { useLocationContext } from "@/contexts/location-context";

// Every domain screen operates against whichever location is picked here —
// see contexts/location-context.tsx.
export function LocationSwitcher() {
  const { locations, selectedLocationId, setSelectedLocationId, isLoading } = useLocationContext();

  if (isLoading || locations.length === 0) return null;

  return (
    <Select value={selectedLocationId ?? undefined} onValueChange={setSelectedLocationId}>
      <SelectTrigger className="w-36 sm:w-56 rounded-full">
        <MapPin className="size-3.5 shrink-0 text-primary" />
        <SelectValue placeholder="Select a location" />
      </SelectTrigger>
      <SelectContent>
        {locations.map((loc) => (
          <SelectItem key={loc.id} value={loc.id}>
            <span className="flex items-center gap-1.5">
              {loc.name}
              {!loc.timezone_confirmed && (
                <AlertTriangle className="size-3.5 text-destructive" />
              )}
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
