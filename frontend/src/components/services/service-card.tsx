import { useQuery } from "@tanstack/react-query";
import { Badge, Button } from "@flowposltd/ui";
import { MoreHorizontal, Pencil, Trash2, Clock, Scissors, ChevronRight, Users } from "lucide-react";
import { type Service } from "@/types";
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent,
  DropdownMenuItem, DropdownMenuSeparator,
} from "@flowposltd/ui";
import { DropdownMenuDangerItem } from "@/components/ui/dropdown-menu";
import { PersonAvatar } from "@/components/ui/person-avatar";
import { listAssignedEmployees } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { serviceVisual } from "@/constants/service-visuals";
import { cn, formatMoney } from "@/utils";

interface ServiceCardProps {
  service: Service;
  locationId: string;
  onEdit: () => void;
  onDelete: () => void;
  onManageEmployees: () => void;
}

export function ServiceCard({ service, locationId, onEdit, onDelete, onManageEmployees }: ServiceCardProps) {
  const isFree = !service.price || service.price === 0;
  const v = serviceVisual(service.name);

  const { data: staff = [] } = useQuery({
    queryKey: queryKeys.serviceEmployees(locationId, service.id),
    queryFn: () => listAssignedEmployees(locationId, service.id),
  });

  return (
    <div className="group relative flex flex-col rounded-md border border-border bg-card shadow-card hover:shadow-elevated transition-all duration-200 overflow-hidden">
      <div className="h-1.5 w-full shrink-0" style={{ backgroundColor: v.solid }} />

      <div className="flex flex-col gap-3.5 p-5 flex-1">
        <div className="flex items-start gap-3.5">
          <div
            className="flex size-11 shrink-0 items-center justify-center rounded-md"
            style={{ backgroundColor: v.bg, color: v.fg }}
          >
            <Scissors className="size-5" />
          </div>
          <div className="flex-1 min-w-0">
            <h4 className="font-semibold text-foreground capitalize leading-tight truncate">{service.name}</h4>
            <div className="flex items-center gap-1.5 mt-1 text-body-3 text-content-secondary font-medium">
              <Clock className="size-3.5 shrink-0" />
              {service.duration_minutes} min
              {service.buffer_minutes > 0 && ` (+${service.buffer_minutes} buffer)`}
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-7 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <MoreHorizontal className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {/* Edit/Manage employees/Delete all open a Dialog. Let the
                  DropdownMenu close normally (no preventDefault — that would
                  suppress Radix's own close behavior and leave it stuck open
                  under the Dialog), but defer the Dialog-opening callback to a
                  macrotask so it runs after the DropdownMenu's close/unmount
                  has fully committed. Without this, the two Radix modals'
                  overlapping mount/unmount can leave document.body's
                  pointer-events lock stuck, silently breaking this card's
                  hover-revealed trigger — and, worse, the whole page, since
                  that lock lands on document.body, not just this card. */}
              <DropdownMenuItem onSelect={() => setTimeout(onEdit, 0)}>
                <Pencil className="size-3.5" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setTimeout(onManageEmployees, 0)}>
                <Users className="size-3.5" />
                Manage employees
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuDangerItem onSelect={() => setTimeout(onDelete, 0)}>
                <Trash2 className="size-3.5" />
                Delete
              </DropdownMenuDangerItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {!service.active && <Badge variant="secondary" className="w-fit">Inactive</Badge>}

        <p className={cn("text-body-2 flex-1 min-h-10 line-clamp-2", service.description ? "text-content-secondary" : "italic text-content-tertiary")}>
          {service.description || "No description added"}
        </p>

        <div className="flex items-center justify-between pt-3.5 border-t border-border mt-auto">
          {isFree ? (
            <Badge variant="status-success" className="text-body-3">Free</Badge>
          ) : (
            <span className="text-heading-2 font-semibold text-foreground">{formatMoney(service.price)}</span>
          )}
          <button
            onClick={onManageEmployees}
            className="flex items-center gap-2 rounded-md border border-border px-3 py-1.5 text-body-3 font-medium text-primary hover:bg-primary/5 hover:border-primary/30 transition-colors"
          >
            {staff.length > 0 && (
              <div className="flex">
                {staff.slice(0, 3).map((e, i) => (
                  <PersonAvatar
                    key={e.id}
                    name={e.name}
                    seed={e.name + e.email}
                    size="xs"
                    className={cn("ring-2 ring-card", i > 0 && "-ml-2")}
                  />
                ))}
              </div>
            )}
            {staff.length} staff
            <ChevronRight className="size-3.5" />
          </button>
        </div>
      </div>
    </div>
  );
}

export function ServiceCardSkeleton() {
  return (
    <div className="flex flex-col rounded-md border border-border bg-card overflow-hidden">
      <div className="h-1.5 bg-muted animate-pulse" />
      <div className="p-5 flex flex-col gap-3.5">
        <div className="h-4 w-2/3 bg-muted animate-pulse rounded" />
        <div className="h-3 w-1/3 bg-muted animate-pulse rounded" />
        <div className="space-y-1.5">
          <div className="h-3 bg-muted animate-pulse rounded" />
          <div className="h-3 w-4/5 bg-muted animate-pulse rounded" />
        </div>
        <div className="h-px bg-muted mt-1" />
        <div className="h-5 w-1/4 bg-muted animate-pulse rounded" />
      </div>
    </div>
  );
}
