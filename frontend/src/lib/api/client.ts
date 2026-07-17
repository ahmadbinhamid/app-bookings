import axios from "axios";
import { getAuthToken } from "@/lib/auth-token";

export const apiClient = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
  headers: { "Content-Type": "application/json" },
});

// Request interceptor: attach the session JWT (see lib/auth-token.ts) to
// every call. There's nothing else to add yet (no tenant header, no
// multi-location context) — this is the one place to add it later.
apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor: normalize every backend error into a plain Error
// with a readable message, and react to expired/invalid sessions. The
// backend always responds with either {"error": "..."} or, for field
// validation failures, {"errors": {field: message}}.
apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err.response?.status;

    if (status === 401) {
      // The session JWT is missing/expired. There's no client-side refresh
      // flow for this token (it's minted by the tenant dashboard) — the
      // dashboard re-embeds this app with a fresh `?token=` when needed, so
      // there's nothing to do here beyond surfacing a clear message.
    }

    const data = err.response?.data;
    const message =
      data?.error ??
      (data?.errors && Object.values(data.errors)[0]) ??
      err.message ??
      "Something went wrong";
    return Promise.reject(new Error(message));
  }
);
