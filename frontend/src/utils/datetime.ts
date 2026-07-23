export function toISODateLocal(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

// Combines a "YYYY-MM-DD" date (from a DatePopover) with minutes-since-midnight
// (from a TimePopover) into a full ISO instant in the browser's local time.
export function combineDateAndMinutes(dateISO: string, minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return new Date(`${dateISO}T${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:00`).toISOString();
}
