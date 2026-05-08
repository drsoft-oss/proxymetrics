import { useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { AuditSetupForm } from "@/components/audits/AuditSetupForm";
import { AuditActiveRun } from "@/components/audits/AuditActiveRun";

export const Route = createFileRoute("/audits/")({
  component: GeoAuditTab,
  validateSearch: (s: Record<string, unknown>) => ({ run: typeof s.run === "string" ? s.run : undefined }),
});

function GeoAuditTab() {
  const { run } = Route.useSearch();
  const nav = useNavigate({ from: "/audits/" });
  const [, force] = useState(0);

  if (run) {
    return <AuditActiveRun runId={run} onCancelled={() => force((n) => n + 1)} />;
  }
  return (
    <div className="p-4">
      <AuditSetupForm onStarted={(id) => nav({ search: { run: id } })} />
    </div>
  );
}
