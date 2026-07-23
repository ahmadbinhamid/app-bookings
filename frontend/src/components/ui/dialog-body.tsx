import { cn } from "@/utils";

// max-h (not flex-1) is deliberate: DialogContent (@flowposltd/ui) lays out
// with CSS grid, not flex, so a flex-1 height never actually resolves —
// content just grows the whole fixed-position Dialog past the viewport,
// pushing the header/footer off-screen with no way to scroll to them. A
// fixed cap works regardless of the ancestor's display mode.
export function DialogBody({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("max-h-[60vh] overflow-y-auto py-4", className)} {...props} />;
}
