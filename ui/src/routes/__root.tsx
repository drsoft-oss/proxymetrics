import { createRootRoute, Outlet } from "@tanstack/react-router";
import { Topbar } from "@/components/shell/Topbar";
import { Sidebar } from "@/components/shell/Sidebar";
import { LiveActivityRail } from "@/components/overview/LiveActivityRail";
import { FilterBar } from "@/components/filter-bar/FilterBar";

export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  return (
    <div className="grid h-screen grid-cols-[220px_1fr_320px] grid-rows-[48px_1fr] bg-background text-foreground">
      <Topbar />
      <aside className="border-r border-border bg-card overflow-y-auto" data-testid="sidebar">
        <Sidebar />
      </aside>
      <main className="flex min-h-0 flex-col overflow-hidden">
        <FilterBar />
        <div className="flex-1 min-h-0 overflow-y-auto">
          <Outlet />
        </div>
      </main>
      <aside className="border-l border-border bg-card overflow-hidden" data-testid="activity-rail">
        <LiveActivityRail />
      </aside>
    </div>
  );
}
