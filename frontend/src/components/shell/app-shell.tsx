import { Outlet } from "react-router-dom";
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@flowposltd/ui";
import { AppNav } from "./nav";
import { LocationSwitcher } from "@/components/locations/location-switcher";
import { TimezoneBanner } from "@/components/locations/timezone-banner";

export function AppShell() {
  return (
    <SidebarProvider>
      <AppNav />
      <SidebarInset className="h-svh overflow-y-auto">
        <div className="flex items-center gap-3 px-4 sm:gap-3.5 sm:px-8 h-15.5 border-b border-border bg-card/80 backdrop-blur-sm sticky top-0 z-20">
          <SidebarTrigger />
          <span className="hidden sm:inline text-body-3 font-medium tracking-[0.08em] text-content-tertiary">
            LOCATION
          </span>
          <LocationSwitcher />
        </div>
        <div className="px-4 sm:px-6 pt-4">
          <TimezoneBanner />
        </div>
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  );
}
