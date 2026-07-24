import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge, Button, Input } from "@flowposltd/ui";
import { Search, X, ChevronLeft, ChevronRight, Plus, Scissors } from "lucide-react";
import { listServices, deleteService } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { type Service } from "@/types";
import { ServiceCard, ServiceCardSkeleton } from "@/components/services/service-card";
import { ServiceForm } from "@/components/services/service-form";
import { ServiceAssignmentsDialog } from "@/components/services/service-assignments-dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { useLocationContext } from "@/contexts/location-context";

const DEFAULT_LIMIT = 18; // 3 cols × 6 rows

export default function ServicesPage() {
  const { selectedLocationId, isLoading: locationsLoading } = useLocationContext();
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Service | undefined>();
  const [assignmentsFor, setAssignmentsFor] = useState<Service | undefined>();

  const { data, isLoading } = useQuery({
    queryKey: selectedLocationId ? queryKeys.services(selectedLocationId, page, DEFAULT_LIMIT, search || undefined) : ["services", "none"],
    queryFn: () => listServices(selectedLocationId!, page, DEFAULT_LIMIT, search || undefined),
    enabled: Boolean(selectedLocationId),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteService(selectedLocationId!, id),
    onSuccess: () => {
      toast.success("Service deleted");
      qc.invalidateQueries({ queryKey: ["services"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  // Deleting the last item on the last page would otherwise strand the view
  // on an empty page even though earlier pages still have services.
  useEffect(() => {
    if (data?.meta && data.meta.total_pages > 0 && page > data.meta.total_pages) {
      setPage(data.meta.total_pages);
    }
  }, [data?.meta, page]);

  if (locationsLoading) return null;
  if (!selectedLocationId) {
    return (
      <div className="p-6">
        <EmptyState
          variant="hero"
          icon={Scissors}
          title="No location yet"
          description="Services are managed per location — once FlowPOS sync has run, a location will show up here."
        />
      </div>
    );
  }

  const services = data?.data ?? [];
  const meta = data?.meta;
  const totalPages = meta?.total_pages ?? 1;
  const totalCount = meta?.total ?? 0;
  // Nothing to search or page through when the location has zero services —
  // only show the toolbar once there's something (or a search) for it to act on.
  const showToolbar = isLoading || Boolean(search) || totalCount > 0;
  const isTrueEmpty = !isLoading && !search && totalCount === 0;

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    setSearch(draft);
    setPage(1);
  }

  function handleClear() {
    setDraft("");
    setSearch("");
    setPage(1);
  }

  function openCreate() {
    setEditing(undefined);
    setFormOpen(true);
  }

  function openEdit(svc: Service) {
    setEditing(svc);
    setFormOpen(true);
  }

  function handleDelete(svc: Service) {
    if (!confirm(`Delete "${svc.name}"?`)) return;
    deleteMut.mutate(svc.id);
  }

  return (
    <div className="p-4 sm:p-6 flex flex-col gap-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1>Services</h1>
            {meta && meta.total > 0 && <Badge>{meta.total}</Badge>}
          </div>
          <p className="lead mt-0.5">Manage the services available for booking at this location.</p>
        </div>
        <Button onClick={openCreate} className="self-start">
          <Plus className="size-4" />
          New Service
        </Button>
      </div>

      {showToolbar && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <form onSubmit={handleSearch} className="flex items-center gap-2 flex-1 min-w-0">
            <div className="relative w-full sm:max-w-xs">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-content-secondary pointer-events-none z-10" />
              <Input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Search services…"
                className="pl-9"
              />
              {draft && (
                <button
                  type="button"
                  onClick={handleClear}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-content-secondary hover:text-foreground transition-colors z-10"
                >
                  <X className="size-3.5" />
                </button>
              )}
            </div>
            <Button type="submit" variant="secondary" size="sm">Search</Button>
          </form>

          {meta && (
            <div className="flex items-center gap-1.5 text-body-3 text-content-secondary shrink-0">
              <span>{meta.total} result{meta.total !== 1 ? "s" : ""}</span>
              <Button variant="ghost" size="icon" className="size-7" onClick={() => setPage(page - 1)} disabled={page <= 1}>
                <ChevronLeft className="size-4" />
              </Button>
              <span className="font-medium">{page} / {totalPages}</span>
              <Button variant="ghost" size="icon" className="size-7" onClick={() => setPage(page + 1)} disabled={page >= totalPages}>
                <ChevronRight className="size-4" />
              </Button>
            </div>
          )}
        </div>
      )}

      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <ServiceCardSkeleton key={i} />
          ))}
        </div>
      ) : services.length === 0 ? (
        isTrueEmpty ? (
          <EmptyState
            variant="hero"
            icon={Scissors}
            title="No services yet"
            description="Add your first service to start accepting bookings."
            action={
              <Button onClick={openCreate}>
                <Plus className="size-4" />
                New Service
              </Button>
            }
          />
        ) : (
          <EmptyState
            icon={Scissors}
            title={`No services match "${search}"`}
            description="Try a different search term."
            action={
              <Button size="sm" variant="secondary" onClick={handleClear}>
                Clear search
              </Button>
            }
          />
        )
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {services.map((svc) => (
            <ServiceCard
              key={svc.id}
              service={svc}
              locationId={selectedLocationId}
              onEdit={() => openEdit(svc)}
              onDelete={() => handleDelete(svc)}
              onManageEmployees={() => setAssignmentsFor(svc)}
            />
          ))}

          <button
            onClick={openCreate}
            className="flex flex-col items-center justify-center gap-2 rounded-md border-2 border-dashed border-border hover:border-primary hover:bg-primary/5 transition-all duration-200 min-h-40 p-6 text-content-secondary hover:text-primary"
          >
            <div className="rounded-full border-2 border-current p-2">
              <Plus className="size-5" />
            </div>
            <span className="text-body-2 font-medium">Add Service</span>
          </button>
        </div>
      )}

      <ServiceForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        locationId={selectedLocationId}
        service={editing}
        page={page}
        limit={DEFAULT_LIMIT}
        search={search}
      />
      <ServiceAssignmentsDialog
        open={Boolean(assignmentsFor)}
        onClose={() => setAssignmentsFor(undefined)}
        locationId={selectedLocationId}
        service={assignmentsFor}
      />
    </div>
  );
}
