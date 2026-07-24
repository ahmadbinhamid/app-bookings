import { useLocationContext } from "@/contexts/location-context";

// Every appointment-time display must render in the LOCATION's own IANA
// zone, never whoever's viewing device happens to be set to — see
// formatTime/formatTimeRange's timeZone param. This is the one place that
// resolves it, so call sites pull from here instead of each repeating
// `selectedLocation?.timezone` inline.
export function useLocationTimezone(): string | undefined {
  const { selectedLocation } = useLocationContext();
  return selectedLocation?.timezone;
}
