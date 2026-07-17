import { Badge, Button } from "@flowposltd/ui";
import { MoreHorizontal, Pencil, Trash2, Clock, Users } from "lucide-react";
import { type Service } from "@/types";
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent,
  DropdownMenuItem, DropdownMenuSeparator,
} from "@flowposltd/ui";
import { DropdownMenuDangerItem } from "@/components/ui/dropdown-menu";

interface ServiceCardProps {
  service: Service;
  onEdit: () => void;
  onDelete: () => void;
  onManageEmployees: () => void;
}

export function ServiceCard({ service, onEdit, onDelete, onManageEmployees }: ServiceCardProps) {
  const isFree = !service.price || service.price === 0;

  return (
    <div className="group relative flex flex-col rounded-sm border border-border bg-card shadow-card hover:shadow-elevated transition-all duration-200 overflow-hidden">
      <div className={`h-1 w-full ${service.active ? "bg-primary" : "bg-muted"} shrink-0`} />

      <div className="flex flex-col gap-3 p-4 flex-1">
        <div className="flex items-start justify-between gap-2">
          <h4 className="font-semibold text-foreground leading-tight">{service.name}</h4>
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
              <DropdownMenuItem onClick={onEdit}>
                <Pencil className="size-3.5" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onManageEmployees}>
                <Users className="size-3.5" />
                Manage employees
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuDangerItem onClick={onDelete}>
                <Trash2 className="size-3.5" />
                Delete
              </DropdownMenuDangerItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="flex items-center gap-3 text-xs text-content-secondary">
          <span className="flex items-center gap-1">
            <Clock className="size-3 shrink-0" />
            {service.duration_minutes} min
            {service.buffer_minutes > 0 && ` (+${service.buffer_minutes} buffer)`}
          </span>
          {!service.active && <Badge variant="secondary">Inactive</Badge>}
        </div>

        <p className="text-sm text-content-secondary line-clamp-2 flex-1 min-h-[2.5rem]">
          {service.description || <span className="italic opacity-60">No description</span>}
        </p>

        <div className="flex items-center justify-between pt-2 border-t border-border mt-auto">
          {isFree ? (
            <Badge variant="status-success" className="text-xs">Free</Badge>
          ) : (
            <span className="text-base font-bold text-foreground tracking-tight">
              ${service.price.toFixed(2)}
            </span>
          )}
          <button
            onClick={onManageEmployees}
            className="text-xs text-content-secondary hover:text-primary transition-colors font-medium"
          >
            Employees →
          </button>
        </div>
      </div>
    </div>
  );
}

export function ServiceCardSkeleton() {
  return (
    <div className="flex flex-col rounded-sm border border-border bg-card overflow-hidden">
      <div className="h-1 bg-muted animate-pulse" />
      <div className="p-4 flex flex-col gap-3">
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
