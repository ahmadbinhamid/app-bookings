import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";
import { adoptTokenFromQuery } from "@/lib/auth-token";
import { adoptThemeFromQuery } from "@/lib/theme";

// The tenant dashboard delivers this app's session token via `?token=` on
// the iframe `src` URL (see EmbedApp.tsx in web/tenant-dashboard) — both when
// embedded and when run standalone for local dev, so this always runs.
adoptTokenFromQuery();
adoptThemeFromQuery();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
