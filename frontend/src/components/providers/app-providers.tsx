import { type ReactNode } from "react";
import { QueryClient, QueryClientProvider, QueryCache, MutationCache } from "@tanstack/react-query";
import { TooltipProvider } from "@flowposltd/ui";
import { toast } from "@/lib/toast";
import { LocationProvider } from "@/contexts/location-context";

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong";
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error) => toast.error(errorMessage(error)),
  }),
  mutationCache: new MutationCache({
    // A catch-all safety net so a mutation that forgets to handle its own
    // error still surfaces *something* — but a mutation whose own onError
    // already shows the error (inline in a dialog, or its own explicit
    // toast) sets meta: { skipErrorToast: true } to opt out, so the error
    // isn't shown twice.
    onError: (error, _variables, _context, mutation) => {
      if (mutation.meta?.skipErrorToast) return;
      toast.error(errorMessage(error));
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={300}>
        <LocationProvider>{children}</LocationProvider>
      </TooltipProvider>
    </QueryClientProvider>
  );
}
