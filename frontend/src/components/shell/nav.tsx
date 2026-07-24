import { NavLink } from "react-router-dom";
import { CalendarDays, Scissors, Users } from "lucide-react";
import {
  Sidebar, SidebarContent, SidebarGroup, SidebarGroupContent, SidebarGroupLabel,
  SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar,
} from "@flowposltd/ui";
import { cn } from "@/utils";

// One entry per top-level page — see app-bookings design doc for the domain.
const NAV = [
  { to: "/", label: "Bookings", icon: CalendarDays },
  { to: "/services", label: "Services", icon: Scissors },
  { to: "/employees", label: "Employees", icon: Users },
];

export function AppNav() {
  const { setOpenMobile } = useSidebar();

  return (
    <Sidebar collapsible="icon" className="border-r border-sidebar-border bg-card">
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel className="uppercase tracking-wider">Manage</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {NAV.map(({ to, label, icon: Icon }) => (
                <SidebarMenuItem key={to}>
                  <SidebarMenuButton asChild tooltip={label} className="hover:bg-transparent">
                    <NavLink to={to} end onClick={() => setOpenMobile(false)}>
                      {({ isActive }) => (
                        <div
                          className={cn(
                            "flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 transition-colors",
                            isActive
                              ? "bg-primary/10 font-medium text-primary"
                              : "text-content-secondary hover:bg-muted hover:text-foreground"
                          )}
                        >
                          <Icon className="size-4 shrink-0" />
                          <span className="truncate">{label}</span>
                        </div>
                      )}
                    </NavLink>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}
