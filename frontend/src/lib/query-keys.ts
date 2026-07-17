// Central registry of react-query keys — add one entry per resource as
// features are built, the same way appointments/src/lib/query-keys.ts does.
export const queryKeys = {
  me: () => ["me"] as const,
};
