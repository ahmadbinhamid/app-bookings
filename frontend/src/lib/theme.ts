/**
 * The tenant dashboard delivers its current theme the same way it delivers
 * the auth token (see auth-token.ts): a `?theme=light|dark` query param on
 * the iframe `src` URL, captured once at startup so the app paints in the
 * right theme immediately instead of flashing light before the apps-sdk
 * postMessage handshake (use-embed.ts) catches up with live toggles.
 */
export function adoptThemeFromQuery(): void {
  const params = new URLSearchParams(window.location.search);
  const theme = params.get("theme");
  if (theme !== "light" && theme !== "dark") return;

  document.documentElement.classList.toggle("dark", theme === "dark");

  params.delete("theme");
  const query = params.toString();
  const url = window.location.pathname + (query ? `?${query}` : "") + window.location.hash;
  window.history.replaceState({}, "", url);
}
