import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem, Skeleton } from "@flowposltd/ui";
import { MapPin } from "lucide-react";
import { useLocationContext } from "@/contexts/location-context";

// Every domain screen operates against whichever location is picked here —
// see contexts/location-context.tsx.
export function LocationSwitcher() {
  const { locations, selectedLocationId, setSelectedLocationId, isLoading } = useLocationContext();

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
            {loc.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
