// ui-kit's own DropdownMenu* components cover everything except a
// destructive-styled item (for "Delete" actions) — this is the one addition
// on top of it, not a full local reimplementation.
import { DropdownMenuItem } from "@flowposltd/ui";
import { cn } from "@/utils";

export function DropdownMenuDangerItem({
  className,
  ...props
}: React.ComponentProps<typeof DropdownMenuItem>) {
  return (
    <DropdownMenuItem
      className={cn(
        "text-destructive data-highlighted:bg-destructive/10 data-highlighted:text-destructive",
        className
      )}
      {...props}
    />
  );
}
