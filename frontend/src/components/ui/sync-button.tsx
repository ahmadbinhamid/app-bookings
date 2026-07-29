import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@flowposltd/ui";
import { RotateCw } from "lucide-react";
import { triggerSync } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/utils";

// Manual on-demand FlowPOS sync (POST /sync/trigger) — the background
// scheduler only re-syncs every SYNC_INTERVAL_MINUTES (an hour by default),
// so a tenant that just installed, or an admin who knows FlowPOS-side data
// just changed, shouldn't have to wait for that. Shared by the Locations
// and Employees pages, since both read data this same sync produces.
export function SyncButton({ className }: { className?: string }) {
  const qc = useQueryClient();

  const syncMut = useMutation({
    mutationFn: triggerSync,
    onSuccess: (summary) => {
      toast.success(
        `Synced ${summary.locations_synced} location${summary.locations_synced === 1 ? "" : "s"} and ` +
          `${summary.employees_synced} employee${summary.employees_synced === 1 ? "" : "s"}`
      );
      qc.invalidateQueries({ queryKey: queryKeys.locations() });
      qc.invalidateQueries({ queryKey: ["employees"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <Button variant="secondary" size="sm" className={className} onClick={() => syncMut.mutate()} disabled={syncMut.isPending}>
      <RotateCw className={cn("size-3.5", syncMut.isPending && "animate-spin")} />
      Sync now
    </Button>
  );
}
