import { Skeleton } from "@flowposltd/ui";

// Shared between App.tsx's route-level Suspense (covers the lazy-loaded
// page chunk itself) and each page's own `locationsLoading` branch (covers
// the async location-context fetch that follows) — without this in both
// spots, the chunk skeleton flashes in, then the page returns null while
// locations load, producing a blank screen in between the two loading
// phases instead of one continuous skeleton.
export function PageFallback() {
  return (
    <div className="p-6 flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-9 w-32" />
      </div>
      <Skeleton className="h-96 w-full" />
    </div>
  );
}
