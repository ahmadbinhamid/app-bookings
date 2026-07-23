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

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

export function formatTimeRange(startIso: string, endIso: string): string {
  return `${formatTime(startIso)} – ${formatTime(endIso)}`;
}

export function formatShortDate(iso: string): string {
  return new Date(iso).toLocaleDateString([], { month: "short", day: "numeric" });
}
