import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem, Skeleton } from "@flowposltd/ui";
import { MapPin, AlertTriangle } from "lucide-react";
import { useLocationContext } from "@/contexts/location-context";

// Every domain screen operates against whichever location is picked here —
// see contexts/location-context.tsx.
export function LocationSwitcher() {
  const { locations, selectedLocationId, setSelectedLocationId, isLoading } = useLocationContext();

  // A skeleton matching the trigger's own shape, not null — this sits in
  // the persistent header (outside any page-level Suspense/loading
  // fallback), so returning null here left a blank gap next to "LOCATION"
  // for however long the locations list takes to load, however brief.
  if (isLoading) return <Skeleton className="h-9 w-36 sm:w-56 rounded" />;
  if (locations.length === 0) return null;

  return (
    <Select value={selectedLocationId ?? undefined} onValueChange={setSelectedLocationId}>
      <SelectTrigger className="w-36 sm:w-56">
        <MapPin className="size-3.5 shrink-0 text-primary" />
        <SelectValue placeholder="Select a location" />
      </SelectTrigger>
      <SelectContent>
        {locations.map((loc) => (
          <SelectItem key={loc.id} value={loc.id}>
            <span className="flex items-center gap-1.5">
              {loc.name}
              {!loc.timezone_confirmed && <AlertTriangle className="size-3.5 text-destructive" />}
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
