import { type LucideIcon } from "lucide-react";
import { cn } from "@/utils";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 py-14 text-center",
        className
      )}
    >
      {Icon && (
        <div className="rounded-full bg-muted p-3">
          <Icon className="size-6 text-content-secondary" />
        </div>
      )}
      <div className="flex flex-col gap-1">
        <p className="text-body-2 font-medium text-foreground">{title}</p>
        {description && <p className="text-body-3 text-content-secondary max-w-xs">{description}</p>}
      </div>
      {action}
    </div>
  );
}
