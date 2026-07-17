/**
 * Holds the session JWT this app uses to authenticate its own /api/v1
 * requests (see backend/internal/auth — HS256, signed with the same secret
 * as FLOWPOS_SIGNING_SECRET).
 *
 * Delivery mechanism: despite @flowposltd/apps-sdk's own docs describing a
 * postMessage handshake, the tenant dashboard's actual embedding code
 * (web/tenant-dashboard/src/pages/apps/EmbedApp.tsx) never postMessages a
 * token to the iframe — it only ever appends it as `?token=` on the iframe
 * `src` URL (same convention the other mini-apps use). So `adoptTokenFromQuery`
 * below is the real, primary delivery path, both embedded and standalone —
 * not just a dev fallback. src/app/use-embed.ts's apps-sdk handshake is kept
 * around in case the host adopts postMessage delivery later, but must not
 * clobber a token already adopted from the query string when it doesn't.
 */
let token: string | null = null;

export function getAuthToken(): string | null {
  return token;
}

export function setAuthToken(next: string | null): void {
  token = next;
}

/**
 * Adopt a `?token=` query param as the session JWT, then scrub it from the
 * URL. Called unconditionally on startup (see main.tsx) since this is how
 * the tenant dashboard actually delivers the token when embedding this app.
 */
export function adoptTokenFromQuery(): void {
  const params = new URLSearchParams(window.location.search);
  const fromQuery = params.get("token");
  if (!fromQuery) return;
  setAuthToken(fromQuery);
  params.delete("token");
  const query = params.toString();
  const url = window.location.pathname + (query ? `?${query}` : "") + window.location.hash;
  window.history.replaceState({}, "", url);
}
