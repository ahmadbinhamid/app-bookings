import { useQuery } from "@tanstack/react-query";
import { Button, Skeleton } from "@flowposltd/ui";
import { getMe } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

// Placeholder landing page — proves the wiring (ui-kit, react-query, the
// axios client + its interceptors, the JWT round-trip via /api/v1/me) all
// works end to end. Replace with real pages as features are built.
export default function HomePage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.me(),
    queryFn: getMe,
  });

  return (
    <div className="p-6 flex flex-col gap-5 max-w-xl">
      <div>
        <h1>App Booking</h1>
        <p className="lead">
          Boilerplate is wired up — ui-kit components, react-query, and the
          authenticated API client all work. Build the real pages here.
        </p>
      </div>

      <div className="rounded-md border border-border bg-card p-4 flex flex-col gap-3">
        <h3>Session check (GET /api/v1/me)</h3>

        {isLoading && <Skeleton className="h-5 w-64" />}

        {isError && (
          <p className="text-sm text-destructive">
            Could not reach the API — is the backend running?
          </p>
        )}

        {data && (
          <p className="text-sm">
            {data.installation
              ? `Installed for tenant ${data.installation.tenant_id} (installed: ${String(
                  data.installation.installed
                )})`
              : "No installation found for this tenant yet."}
          </p>
        )}

        <div>
          <Button size="sm" variant="secondary" onClick={() => refetch()}>
            Refetch
          </Button>
        </div>
      </div>
    </div>
  );
}
