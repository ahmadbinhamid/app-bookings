import { Outlet } from "react-router-dom";
import { AppNav } from "./nav";
import { LocationSwitcher } from "@/components/locations/location-switcher";
import { TimezoneBanner } from "@/components/locations/timezone-banner";

export function AppShell() {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppNav />
      <main className="flex-1 min-w-0 h-full overflow-y-auto bg-background">
        <div className="flex items-center justify-between gap-4 px-6 py-3 border-b border-border">
          <span className="text-xs font-medium text-content-secondary uppercase tracking-wide">Location</span>
          <div className="flex-1" />
          <LocationSwitcher />
        </div>
        <div className="px-6 pt-4">
          <TimezoneBanner />
        </div>
        <Outlet />
      </main>
    </div>
  );
}
