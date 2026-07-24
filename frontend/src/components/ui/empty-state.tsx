import { type LucideIcon } from "lucide-react";
import { cn } from "@/utils";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
  // "compact": a small in-place notice next to still-visible filters/search
  // (e.g. "no results for this search"). "hero": the page's only content,
  // so it can take up more room and carry more visual weight.
  variant?: "compact" | "hero";
}

export function EmptyState({ icon: Icon, title, description, action, className, variant = "compact" }: EmptyStateProps) {
  const hero = variant === "hero";

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center",
        hero ? "gap-4 py-20" : "gap-2.5 py-10",
        className
      )}
    >
      {Icon && (
        <div className={cn("rounded-full", hero ? "bg-primary/10 p-4" : "bg-muted p-2")}>
          <Icon className={cn(hero ? "size-6 text-primary" : "size-4 text-content-secondary")} />
        </div>
      )}
      <div className={cn("flex flex-col", hero ? "gap-1" : "gap-0.5")}>
        <p className={cn("font-medium text-foreground", hero ? "text-heading-3" : "text-body-3")}>{title}</p>
        {description && (
          <p className={cn("text-body-3 text-content-tertiary", hero ? "max-w-sm" : "max-w-2xs")}>{description}</p>
        )}
      </div>
      {action}
    </div>
  );
}
