import { createFileRoute, Link, Outlet, useMatchRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/audits")({
  component: AuditsShell,
});

function AuditsShell() {
  const match = useMatchRoute();
  const onHistory = match({ to: "/audits/history" });
  return (
    <div className="flex h-full flex-col">
      <div className="flex border-b">
        <TabLink to="/audits" active={!onHistory && !match({ to: "/audits/$id" })}>Geo audit</TabLink>
        <TabLink to="/audits/history" active={Boolean(onHistory)}>History</TabLink>
      </div>
      <div className="flex-1 min-h-0">
        <Outlet />
      </div>
    </div>
  );
}

function TabLink({ to, active, children }: { to: string; active: boolean; children: React.ReactNode }) {
  return (
    <Link
      to={to as never}
      className={
        "px-4 py-2 text-sm border-b-2 " +
        (active ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground")
      }
    >
      {children}
    </Link>
  );
}
