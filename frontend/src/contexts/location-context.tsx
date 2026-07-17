import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { listLocations } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Location } from "@/types";

interface LocationContextValue {
  locations: Location[];
  isLoading: boolean;
  selectedLocationId: string | null;
  selectedLocation: Location | undefined;
  setSelectedLocationId: (id: string) => void;
}

const LocationContext = createContext<LocationContextValue | undefined>(undefined);

// Every domain screen (services, employees, schedules, time off) operates
// against one "current" location — a multi-location tenant switches it here
// once, rather than every page re-implementing its own location picker.
export function LocationProvider({ children }: { children: ReactNode }) {
  const { data: locations = [], isLoading } = useQuery({
    queryKey: queryKeys.locations(),
    queryFn: listLocations,
  });
  const [selectedLocationId, setSelectedLocationId] = useState<string | null>(null);

  // Default to the first location once the list loads, unless the admin
  // already picked one.
  useEffect(() => {
    if (!selectedLocationId && locations.length > 0) {
      setSelectedLocationId(locations[0].id);
    }
  }, [locations, selectedLocationId]);

  const selectedLocation = locations.find((l) => l.id === selectedLocationId);

  return (
    <LocationContext.Provider
      value={{ locations, isLoading, selectedLocationId, selectedLocation, setSelectedLocationId }}
    >
      {children}
    </LocationContext.Provider>
  );
}

export function useLocationContext() {
  const ctx = useContext(LocationContext);
  if (!ctx) throw new Error("useLocationContext must be used within a LocationProvider");
  return ctx;
}
