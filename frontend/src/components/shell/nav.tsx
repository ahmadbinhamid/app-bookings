import { NavLink } from "react-router-dom";
import { CalendarDays } from "lucide-react";
import { cn } from "@/utils";

// One entry per top-level page. Add to this list as real features (employees,
// services, bookings...) get built — see app-bookings design doc for the
// planned domain.
const NAV = [{ to: "/", label: "Home", icon: CalendarDays }];

export function AppNav() {
  return (
    <aside className="flex w-56 shrink-0 flex-col border-r border-border bg-secondary">
      <div className="flex items-center gap-2.5 px-4 py-4 border-b border-border">
        <div className="flex size-7 items-center justify-center rounded-sm bg-primary text-primary-foreground text-sm font-bold shrink-0">
          B
        </div>
        <span className="text-sm font-semibold text-foreground">App Booking</span>
      </div>

      <nav className="flex flex-col gap-0.5 p-2 flex-1">
        {NAV.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end
            className={({ isActive }) =>
              cn(
                "flex items-center gap-2.5 rounded-sm px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary/10 text-primary"
                  : "text-content-secondary hover:bg-muted hover:text-foreground"
              )
            }
          >
            <Icon className="size-4 shrink-0" />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
