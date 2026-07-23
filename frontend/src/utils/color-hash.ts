// Deterministic "person" color palette — every employee/customer gets a
// stable background color derived from a hash of their name (+ email when
// disambiguating same-named people), so the same person always renders with
// the same avatar color across the app without persisting a color anywhere.
const AVATAR_PALETTE = [
  "#7C3AED", "#2563EB", "#0EA5E9", "#059669", "#D97706",
  "#DB2777", "#4F46E5", "#0891B2", "#DC2626", "#65A30D",
];

export function hashIndex(seed: string, modulo: number): number {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = seed.charCodeAt(i) + ((hash << 5) - hash);
  }
  return Math.abs(hash) % modulo;
}

export function hashColor(seed: string): string {
  return AVATAR_PALETTE[hashIndex(seed, AVATAR_PALETTE.length)];
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  const value = ((parts[0]?.[0] ?? "") + (parts[1]?.[0] ?? "")).toUpperCase();
  return value || "?";
}
