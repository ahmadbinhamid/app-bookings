import { cn } from "@/utils";

// max-h (not flex-1) is deliberate: DialogContent (@flowposltd/ui) lays out
// with CSS grid, not flex, so a flex-1 height never actually resolves —
// content just grows the whole fixed-position Dialog past the viewport,
// pushing the header/footer off-screen with no way to scroll to them. A
// fixed cap works regardless of the ancestor's display mode.
export function DialogBody({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  // overflow-x-hidden is deliberate, not decorative: per spec, an element with
  // overflow-y set but no explicit overflow-x has its (default "visible")
  // overflow-x computed as "auto" too — so the first child that overflows
  // horizontally by even a pixel (e.g. a centered axis label at the 0/100%
  // edge) silently turns the whole dialog body into a horizontal scroller.
  return <div className={cn("max-h-[60vh] overflow-y-auto overflow-x-hidden py-4", className)} {...props} />;
}
