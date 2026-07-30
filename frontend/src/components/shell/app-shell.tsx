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
        <div className="flex items-center justify-between gap-3 px-4 sm:px-8 h-15.5 border-b border-border bg-card/80 backdrop-blur-sm sticky top-0 z-20 shrink-0">
          <SidebarTrigger />
          <LocationSwitcher />
        </div>
        <div className="px-4 sm:px-6 pt-4 shrink-0">
          <TimezoneBanner />
        </div>
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  );
}
