import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Toaster } from "sonner";
import { Skeleton } from "@flowposltd/ui";
import { AppProviders } from "@/components/providers/app-providers";
import { AppShell } from "@/components/shell/app-shell";
import { useEmbed } from "@/app/use-embed";

const HomePage = lazy(() => import("./pages/HomePage"));
const ServicesPage = lazy(() => import("./pages/ServicesPage"));
const EmployeesPage = lazy(() => import("./pages/EmployeesPage"));
const BookingsPage = lazy(() => import("./pages/BookingsPage"));

function PageFallback() {
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

export default function App() {
  // Handshake + theme sync + auto-resize with the tenant-dashboard iframe
  // host (no-op when run standalone, outside an iframe).
  useEmbed();

  return (
    <BrowserRouter>
      <AppProviders>
        <Routes>
          <Route path="/" element={<AppShell />}>
            <Route
              index
              element={
                <Suspense fallback={<PageFallback />}>
                  <BookingsPage />
                </Suspense>
              }
            />
            <Route
              path="debug"
              element={
                <Suspense fallback={<PageFallback />}>
                  <HomePage />
                </Suspense>
              }
            />
            <Route
              path="services"
              element={
                <Suspense fallback={<PageFallback />}>
                  <ServicesPage />
                </Suspense>
              }
            />
            <Route
              path="employees"
              element={
                <Suspense fallback={<PageFallback />}>
                  <EmployeesPage />
                </Suspense>
              }
            />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </AppProviders>
      <Toaster richColors position="bottom-right" />
    </BrowserRouter>
  );
}
