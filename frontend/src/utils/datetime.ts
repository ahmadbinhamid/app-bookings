// Every location carries its own IANA timezone (locations.timezone) — "today"
// /"now" for booking and time-off purposes must be read as wall-clock time in
// THAT zone, not the browser's own local calendar/clock, since the admin
// managing a location may be sitting in a different timezone than it.

function zoneOffsetMinutes(instantMs: number, timeZone: string): number {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    hourCycle: "h23",
    year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", second: "2-digit",
  }).formatToParts(new Date(instantMs));
  const get = (type: string) => Number(parts.find((p) => p.type === type)?.value);
  const asUTC = Date.UTC(get("year"), get("month") - 1, get("day"), get("hour"), get("minute"), get("second"));
  return (asUTC - instantMs) / 60000;
}

/** "YYYY-MM-DD" for the given instant, as a wall-clock date in `timeZone`. */
export function toISODateInZone(d: Date, timeZone: string): string {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone, year: "numeric", month: "2-digit", day: "2-digit",
  }).formatToParts(d);
  const get = (type: string) => parts.find((p) => p.type === type)?.value;
  return `${get("year")}-${get("month")}-${get("day")}`;
}

/** Minutes since midnight for the given instant, as wall-clock time in `timeZone`. */
export function minutesOfDayInZone(d: Date, timeZone: string): number {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone, hourCycle: "h23", hour: "2-digit", minute: "2-digit",
  }).formatToParts(d);
  const get = (type: string) => Number(parts.find((p) => p.type === type)?.value);
  return get("hour") * 60 + get("minute");
}

// Combines a "YYYY-MM-DD" date (from a DatePopover) with minutes-since-midnight
// (from a TimePopover), both wall-clock time in `timeZone`, into the true UTC
// instant they represent — accounting for that zone's actual UTC offset
// (which can vary with DST), not just appending "Z" as if the input were
// already UTC. Getting this wrong silently shifts every proposed/confirmed
// time by the zone's offset (e.g. picking "4:30 PM" landing as "11:30 AM"
// for a UTC+5 zone) — this used to go through `new Date(...).toISOString()`,
// which parses a timezone-less string as the BROWSER's local time and was
// exactly that bug. Offset is resolved twice since it can itself differ
// right around a DST transition instant.
export function combineDateAndMinutesInZone(dateISO: string, minutes: number, timeZone: string): string {
  const [y, mo, day] = dateISO.split("-").map(Number);
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  const guessMs = Date.UTC(y, mo - 1, day, h, m, 0);
  const offset1 = zoneOffsetMinutes(guessMs, timeZone);
  const utcMs1 = guessMs - offset1 * 60000;
  const offset2 = zoneOffsetMinutes(utcMs1, timeZone);
  const utcMs = offset2 === offset1 ? utcMs1 : guessMs - offset2 * 60000;
  return new Date(utcMs).toISOString();
}
