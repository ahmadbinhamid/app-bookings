import { Outlet } from "react-router-dom";
import { AppNav } from "./nav";

export function AppShell() {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppNav />
      <main className="flex-1 min-w-0 h-full overflow-y-auto bg-background">
        <Outlet />
      </main>
    </div>
  );
}
