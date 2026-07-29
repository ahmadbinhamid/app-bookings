// Every location is currently hardcoded to UTC (per-location timezone
// selection isn't built yet — see the backend's sync.Service.syncLocations),
// so "today"/"now" for booking and time-off purposes must be read in UTC
// terms, not the browser's own local calendar/clock — otherwise near a day
// boundary the two disagree by up to a full day.
export function toISODateUTC(d: Date): string {
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
}

// Combines a "YYYY-MM-DD" date (from a DatePopover) with minutes-since-midnight
// (from a TimePopover) into a full ISO instant, treating the input as
// wall-clock time in the LOCATION's own timezone — currently always UTC (see
// toISODateUTC above) — rather than the browser's local timezone. Getting
// this wrong silently shifts every proposed/confirmed time by the admin's
// own UTC offset (e.g. picking "4:30 PM" landing as "11:30 AM" for a UTC+5
// admin) — this used to go through `new Date(...).toISOString()`, which
// parses a timezone-less string as local time and was exactly that bug.
export function combineDateAndMinutes(dateISO: string, minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return `${dateISO}T${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:00.000Z`;
}
