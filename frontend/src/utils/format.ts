export function formatMoney(amount: number): string {
  return `$${amount.toFixed(2)}`;
}

export function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h && m) return `${h}h ${m}m`;
  if (h) return `${h}h`;
  return `${m} min`;
}

// timeZone should always be the booking location's own IANA zone — appointment
// times are meaningless in whoever's viewing device happens to be set to, and
// silently fall back to it (via undefined) only when no location is in scope.
export function formatTime(iso: string, timeZone?: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: "numeric", minute: "2-digit", timeZone });
}

export function formatTimeRange(startIso: string, endIso: string, timeZone?: string): string {
  return `${formatTime(startIso, timeZone)} – ${formatTime(endIso, timeZone)}`;
}

export function formatShortDate(iso: string): string {
  return new Date(iso).toLocaleDateString([], { month: "short", day: "numeric" });
}
